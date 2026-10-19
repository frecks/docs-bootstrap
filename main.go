// Command docs-bootstrap sets up a repository with the documentation system.
//
// Usage: docs-bootstrap TARGET [options]
//
// It installs docs standards, Vale and markdownlint configuration, lint
// scripts, a Claude Code hook, and the docs-write and docs-review agent skills
// into TARGET, a new or existing repository. Run it again to upgrade: files
// you haven't edited are updated, and files you have edited are kept and
// reported.
//
// See docs/ in this repository for how to use it.
package main

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	version      = "0.1.1"
	manifestName = ".docs-bootstrap.json"
	blockBegin   = "<!-- docs-bootstrap:begin -->"
	blockEnd     = "<!-- docs-bootstrap:end -->"
)

//go:embed all:template
var embedded embed.FS

type profile struct {
	Noun, Area, ExampleArea, Audience string
}

var profiles = map[string]profile{
	"infrastructure": {
		Noun:        "environment",
		Area:        "system",
		ExampleArea: "network",
		Audience:    "an operator who knows Linux and networking, but doesn't know this {NOUN} or remember why it's set up this way",
	},
	"software": {
		Noun:        "project",
		Area:        "component",
		ExampleArea: "api",
		Audience:    "a developer who knows the languages and tools this {NOUN} uses, but doesn't know this codebase or why it's built this way",
	},
}

// Created once, then owned by the repository. Never updated.
var seed = map[string]bool{
	"docs/README.md": true,
	"docs/documentation/check-documentation.md":                         true,
	"docs/documentation/why-the-documentation-system-works-this-way.md": true,
	".vale/styles/config/vocabularies/{{STYLE}}/accept.txt":             true,
}

// Merged into existing files instead of copied, by the named installer method.
var merged = map[string]func(*installer, string, string) error{
	"AGENTS.md":             (*installer).mergeAgentsMD,
	".gitignore":            (*installer).mergeGitignore,
	".claude/settings.json": (*installer).mergeSettings,
}

var skills = []string{"docs-write", "docs-review"}

var (
	markerLine  = regexp.MustCompile(`(?m)^[ \t]*(\{\{[#/]\w+\}\})[ \t]*\n`)
	sectionOpen = regexp.MustCompile(`\{\{#(\w+)\}\}`)
	placeholder = regexp.MustCompile(`\{\{(\w+)\}\}`)
	agentsBlock = regexp.MustCompile(`(?s)` + regexp.QuoteMeta(blockBegin) + `.*?` + regexp.QuoteMeta(blockEnd))
	importsMD   = regexp.MustCompile(`(?m)^@AGENTS\.md\s*$`)
	separators  = regexp.MustCompile(`[-_]+`)
	notAlnum    = regexp.MustCompile(`[^A-Za-z0-9]`)
	validStyle  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
)

// render fills {{VAR}} placeholders and keeps only this profile's {{#p}}…{{/p}} sections.
func render(text string, vars map[string]string, profile string) (string, error) {
	// A marker alone on its line takes the line with it.
	text = markerLine.ReplaceAllString(text, "$1")
	text = renderSections(text, profile)
	var unknown string
	text = placeholder.ReplaceAllStringFunc(text, func(m string) string {
		value, ok := vars[m[2:len(m)-2]]
		if !ok {
			if unknown == "" {
				unknown = m
			}
			return m
		}
		return value
	})
	if unknown != "" {
		return "", fmt.Errorf("unknown template placeholder: %s", unknown)
	}
	if i := strings.Index(text, "{{"); i >= 0 {
		rest := []rune(text[i:])
		return "", fmt.Errorf("unrendered template marker: %s", string(rest[:min(len(rest), 40)]))
	}
	return text, nil
}

func renderSections(text, profile string) string {
	var b strings.Builder
	for {
		loc := sectionOpen.FindStringSubmatchIndex(text)
		if loc == nil {
			break
		}
		name := text[loc[2]:loc[3]]
		end := "{{/" + name + "}}"
		j := strings.Index(text[loc[1]:], end)
		if j < 0 {
			// Unclosed: leave the marker for render to report.
			b.WriteString(text[:loc[1]])
			text = text[loc[1]:]
			continue
		}
		b.WriteString(text[:loc[0]])
		if name == profile {
			b.WriteString(text[loc[1] : loc[1]+j])
		}
		text = text[loc[1]+j+len(end):]
	}
	b.WriteString(text)
	return b.String()
}

func sha(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// readIfExists returns a file's contents, and whether it exists.
func readIfExists(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	return string(data), err == nil, err
}

type options struct {
	Profile  string  `json:"profile"`
	Name     string  `json:"name"`
	Audience *string `json:"audience"`
	Date     string  `json:"date"`
	Style    string  `json:"style"`
	Noun     string  `json:"noun"`
}

type manifest struct {
	Version string            `json:"version"`
	Options options           `json:"options"`
	Files   map[string]string `json:"files"`
}

func readManifest(target string) (manifest, error) {
	var m manifest
	text, ok, err := readIfExists(filepath.Join(target, manifestName))
	if err != nil || !ok {
		return m, err
	}
	if err := json.Unmarshal([]byte(text), &m); err != nil {
		return m, fmt.Errorf("%s: %w", manifestName, err)
	}
	return m, nil
}

// marshal formats v like the manifest and settings files: two-space indents,
// a final newline, and no escaping of HTML characters.
func marshal(v any) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(v)
	return b.String(), err
}

var reportKeys = []string{"created", "updated", "merged", "kept", "conflict"}

type installer struct {
	target    string
	options   options
	template  fs.FS
	dryRun    bool
	showDiff  bool
	replace   map[string]bool
	vars      map[string]string
	profile   string
	oldHashes map[string]string
	newHashes map[string]string
	report    map[string][]string
	out       io.Writer
}

func newInstaller(target string, opts options, template fs.FS, dryRun, showDiff bool, replace []string, oldHashes map[string]string, out io.Writer) *installer {
	in := &installer{
		target:    target,
		options:   opts,
		template:  template,
		dryRun:    dryRun,
		showDiff:  showDiff,
		replace:   map[string]bool{},
		vars:      variablesFor(opts),
		profile:   opts.Profile,
		oldHashes: oldHashes,
		newHashes: map[string]string{},
		report:    map[string][]string{},
		out:       out,
	}
	if in.oldHashes == nil {
		in.oldHashes = map[string]string{}
	}
	for _, p := range replace {
		in.replace[filepath.Clean(p)] = true
	}
	return in
}

func (in *installer) path(rel string) string {
	return filepath.Join(in.target, rel)
}

func (in *installer) add(key, entry string) {
	in.report[key] = append(in.report[key], entry)
}

// write writes a file, and sets its mode unless mode is 0.
func (in *installer) write(rel, text string, mode fs.FileMode) error {
	if in.dryRun {
		return nil
	}
	full := in.path(rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o777); err != nil {
		return err
	}
	if err := os.WriteFile(full, []byte(text), 0o666); err != nil {
		return err
	}
	if mode != 0 {
		return os.Chmod(full, mode)
	}
	return nil
}

func (in *installer) run() error {
	files, err := in.templateFiles()
	if err != nil {
		return err
	}
	for _, relTemplate := range files {
		rel, err := render(relTemplate, in.vars, in.profile)
		if err != nil {
			return fmt.Errorf("%s: %w", relTemplate, err)
		}
		raw, err := fs.ReadFile(in.template, relTemplate)
		if err != nil {
			return err
		}
		text, err := render(string(raw), in.vars, in.profile)
		if err != nil {
			return fmt.Errorf("%s: %w", relTemplate, err)
		}
		// Scripts are executable. The embedded template doesn't keep file
		// modes, so tell them by their #! line.
		mode := fs.FileMode(0o644)
		if strings.HasPrefix(string(raw), "#!") {
			mode = 0o755
		}
		if merge, ok := merged[relTemplate]; ok {
			err = merge(in, rel, text)
		} else if seed[relTemplate] {
			err = in.seed(rel, text, mode)
		} else {
			err = in.managed(rel, text, mode)
		}
		if err != nil {
			return err
		}
	}
	if err := in.linkClaudeMD(); err != nil {
		return err
	}
	for _, skill := range skills {
		if err := in.symlink(".claude/skills/"+skill, "../../.agents/skills/"+skill); err != nil {
			return err
		}
	}
	return in.saveManifest()
}

func (in *installer) managedPaths() (map[string]bool, error) {
	files, err := in.templateFiles()
	if err != nil {
		return nil, err
	}
	paths := map[string]bool{}
	for _, rel := range files {
		if seed[rel] || merged[rel] != nil {
			continue
		}
		rendered, err := render(rel, in.vars, in.profile)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		paths[rendered] = true
	}
	return paths, nil
}

func (in *installer) templateFiles() ([]string, error) {
	var files []string
	err := fs.WalkDir(in.template, ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			files = append(files, path)
		}
		return err
	})
	return files, err
}

func (in *installer) seed(rel, text string, mode fs.FileMode) error {
	if _, err := os.Stat(in.path(rel)); err == nil {
		return nil
	}
	in.add("created", rel)
	return in.write(rel, text, mode)
}

func (in *installer) managed(rel, text string, mode fs.FileMode) error {
	full := in.path(rel)
	newHash := sha(text)
	current, exists, err := readIfExists(full)
	if err != nil {
		return err
	}
	if in.replace[rel] {
		in.newHashes[rel] = newHash
		if !exists || current != text {
			in.add("updated", rel+" (replaced)")
			return in.write(rel, text, mode)
		}
		return nil
	}
	oldHash, installed := in.oldHashes[rel]
	if !exists {
		if installed {
			// Installed before and deleted since: respect the deletion.
			in.newHashes[rel] = oldHash
			in.add("kept", rel+" (deleted locally)")
			return nil
		}
		in.newHashes[rel] = newHash
		in.add("created", rel)
		return in.write(rel, text, mode)
	}
	if current == text {
		in.newHashes[rel] = newHash
		if !in.dryRun {
			return os.Chmod(full, mode)
		}
		return nil
	}
	if installed && oldHash == sha(current) {
		in.newHashes[rel] = newHash
		in.add("updated", rel)
		return in.write(rel, text, mode)
	}
	// Edited locally, or present before the system was installed.
	if installed {
		in.newHashes[rel] = oldHash
	}
	in.add("conflict", rel)
	if in.showDiff {
		_, err := io.WriteString(in.out, unifiedDiff(current, text, rel+" (yours)", rel+" (docs-bootstrap)"))
		return err
	}
	return nil
}

func (in *installer) mergeAgentsMD(rel, block string) error {
	block = blockBegin + "\n" + strings.TrimRight(block, "\n") + "\n" + blockEnd
	current, exists, err := readIfExists(in.path(rel))
	if err != nil {
		return err
	}
	if !exists {
		in.add("created", rel)
		return in.write(rel, fmt.Sprintf("# %s\n\n%s\n", in.vars["PROJECT"], block), 0)
	}
	var result string
	if agentsBlock.MatchString(current) {
		result = agentsBlock.ReplaceAllLiteralString(current, block)
	} else {
		result = strings.TrimRight(current, "\n") + "\n\n" + block + "\n"
	}
	if result == current {
		return nil
	}
	in.add("merged", rel)
	return in.write(rel, result, 0)
}

func (in *installer) mergeGitignore(rel, text string) error {
	current, _, err := readIfExists(in.path(rel))
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	for _, line := range strings.Split(current, "\n") {
		existing[strings.TrimSpace(line)] = true
	}
	missing := false
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "#") && !existing[strings.TrimSpace(line)] {
			missing = true
		}
	}
	if !missing {
		return nil
	}
	if current == "" {
		in.add("created", rel)
		return in.write(rel, text, 0)
	}
	if !strings.HasSuffix(current, "\n") {
		current += "\n"
	}
	in.add("merged", rel)
	return in.write(rel, current+"\n"+text, 0)
}

func (in *installer) mergeSettings(rel, text string) error {
	template, err := decodeJSON(text)
	if err != nil {
		return fmt.Errorf("template %s: %w", rel, err)
	}
	ours := index(get(get(template, "hooks"), "PostToolUse"), 0)
	command := get(index(get(ours, "hooks"), 0), "command")
	current, exists, err := readIfExists(in.path(rel))
	if err != nil {
		return err
	}
	if !exists {
		in.add("created", rel)
		return in.write(rel, text, 0)
	}
	value, err := decodeJSON(current)
	if err != nil {
		in.add("conflict", rel+" (not valid JSON; add the hook by hand)")
		return nil
	}
	settings, ok := value.(*object)
	if !ok {
		in.add("conflict", rel+" (not a JSON object; add the hook by hand)")
		return nil
	}
	if _, ok := settings.values["hooks"]; !ok {
		settings.set("hooks", newObject())
	}
	hooks, ok := settings.values["hooks"].(*object)
	if !ok {
		in.add("conflict", rel+" (hooks isn't a JSON object; add the hook by hand)")
		return nil
	}
	if _, ok := hooks.values["PostToolUse"]; !ok {
		hooks.set("PostToolUse", []any{})
	}
	entries, ok := hooks.values["PostToolUse"].([]any)
	if !ok {
		in.add("conflict", rel+" (hooks.PostToolUse isn't a JSON array; add the hook by hand)")
		return nil
	}
	for _, entry := range entries {
		if list, ok := get(entry, "hooks").([]any); ok {
			for _, hook := range list {
				if get(hook, "command") == command {
					return nil
				}
			}
		}
	}
	hooks.set("PostToolUse", append(entries, ours))
	out, err := marshal(settings)
	if err != nil {
		return err
	}
	in.add("merged", rel)
	return in.write(rel, out, 0)
}

func (in *installer) linkClaudeMD() error {
	rel := "CLAUDE.md"
	full := in.path(rel)
	if info, err := os.Lstat(full); err == nil && info.Mode()&fs.ModeSymlink != 0 {
		return nil
	}
	current, exists, err := readIfExists(full)
	if err != nil {
		return err
	}
	if !exists {
		in.add("created", rel+" -> AGENTS.md")
		if in.dryRun {
			return nil
		}
		return os.Symlink("AGENTS.md", full)
	}
	if importsMD.MatchString(current) {
		return nil
	}
	in.add("merged", rel+" (imports AGENTS.md)")
	return in.write(rel, strings.TrimRight(current, "\n")+"\n\n@AGENTS.md\n", 0)
}

func (in *installer) symlink(rel, target string) error {
	full := in.path(rel)
	if info, err := os.Lstat(full); err == nil {
		if link, err := os.Readlink(full); err == nil && info.Mode()&fs.ModeSymlink != 0 && link == target {
			return nil
		}
		in.add("conflict", rel+" (expected a link to "+target+")")
		return nil
	}
	in.add("created", rel+" -> "+target)
	if in.dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o777); err != nil {
		return err
	}
	return os.Symlink(target, full)
}

func (in *installer) saveManifest() error {
	out, err := marshal(manifest{Version: version, Options: in.options, Files: in.newHashes})
	if err != nil {
		return err
	}
	return in.write(manifestName, out, 0)
}

func variablesFor(opts options) map[string]string {
	p := profiles[opts.Profile]
	vars := map[string]string{
		"PROJECT":      opts.Name,
		"STYLE":        opts.Style,
		"NOUN":         opts.Noun,
		"AREA":         p.Area,
		"EXAMPLE_AREA": p.ExampleArea,
		"DATE":         opts.Date,
		"AUDIENCE":     strings.ReplaceAll(p.Audience, "{NOUN}", opts.Noun),
	}
	if opts.Audience != nil && *opts.Audience != "" {
		vars["AUDIENCE"] = *opts.Audience
	}
	return vars
}

// first returns the first value that isn't empty.
func first(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// capitalize upper-cases the first letter, and lower-cases the rest.
func capitalize(s string) string {
	r := []rune(strings.ToLower(s))
	if len(r) > 0 {
		r[0] = unicode.ToUpper(r[0])
	}
	return string(r)
}

// title upper-cases the first letter of each word, and lower-cases the rest.
func title(s string) string {
	r := []rune(s)
	inWord := false
	for i, c := range r {
		if inWord {
			r[i] = unicode.ToLower(c)
		} else {
			r[i] = unicode.ToUpper(c)
		}
		inWord = unicode.IsLetter(c)
	}
	return string(r)
}

type flags struct {
	profile, name, style, noun, audience string
}

// resolveOptions combines command-line options with those saved by a previous run.
func resolveOptions(f flags, target string, saved options) (options, error) {
	base := filepath.Base(target)
	opts := options{
		Profile: first(f.profile, saved.Profile, "software"),
		Name:    first(f.name, saved.Name, capitalize(separators.ReplaceAllString(base, " "))),
		Date:    first(saved.Date, time.Now().Format("2006-01-02")),
	}
	if audience := first(f.audience, deref(saved.Audience)); audience != "" {
		opts.Audience = &audience
	}
	opts.Style = first(f.style, saved.Style, notAlnum.ReplaceAllString(title(opts.Name), ""), "House")
	opts.Noun = first(f.noun, saved.Noun, profiles[opts.Profile].Noun)
	if _, ok := profiles[opts.Profile]; !ok {
		return opts, fmt.Errorf("unknown profile %q in %s", opts.Profile, manifestName)
	}
	if !validStyle.MatchString(opts.Style) {
		return opts, errors.New("--style must be letters and digits, starting with a letter")
	}
	return opts, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type stringList []string

func (l *stringList) String() string     { return strings.Join(*l, ", ") }
func (l *stringList) Set(v string) error { *l = append(*l, v); return nil }

const usage = `Usage: docs-bootstrap TARGET [options]

Installs docs standards, Vale and markdownlint configuration, lint scripts,
a Claude Code hook, and the docs-write and docs-review agent skills into
TARGET, a new or existing repository. Run it again to upgrade: files you
haven't edited are updated, and files you have edited are kept and reported.

Options:
  --profile NAME   rule set: software (default) or infrastructure
  --name NAME      project name for titles (default: from the directory name)
  --style NAME     Vale style and vocabulary name (default: from the project name)
  --noun NOUN      what the docs describe, as in "this NOUN" (default: project or environment)
  --audience TEXT  who the docs are for, completing "Write for …"
  --dry-run        show what would change, but don't write
  --diff           print a diff for each conflicting file
  --replace PATH   take the new version of this managed file, replacing local
                   edits (repeatable)
`

// run runs the installer with the given arguments, and returns the exit status.
func run(args []string, stdout, stderr io.Writer) int {
	set := flag.NewFlagSet("docs-bootstrap", flag.ContinueOnError)
	set.SetOutput(stderr)
	set.Usage = func() {}
	var f flags
	var replace stringList
	set.StringVar(&f.profile, "profile", "", "")
	set.StringVar(&f.name, "name", "", "")
	set.StringVar(&f.style, "style", "", "")
	set.StringVar(&f.noun, "noun", "", "")
	set.StringVar(&f.audience, "audience", "", "")
	dryRun := set.Bool("dry-run", false, "")
	showDiff := set.Bool("diff", false, "")
	set.Var(&replace, "replace", "")
	templateDir := set.String("template-dir", "", "")

	// Options may come before or after TARGET.
	var positional []string
	for rest := args; ; rest = set.Args()[1:] {
		if err := set.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				fmt.Fprint(stdout, usage)
				return 0
			}
			fmt.Fprint(stderr, usage)
			return 2
		}
		if set.NArg() == 0 {
			break
		}
		positional = append(positional, set.Arg(0))
	}
	fail := func(code int, format string, a ...any) int {
		fmt.Fprintf(stderr, "error: "+format+"\n", a...)
		return code
	}
	if len(positional) != 1 {
		fmt.Fprint(stderr, usage)
		return fail(2, "give one TARGET")
	}
	if _, ok := profiles[f.profile]; f.profile != "" && !ok {
		return fail(2, "--profile must be software or infrastructure")
	}

	var template fs.FS
	if *templateDir != "" {
		template = os.DirFS(*templateDir)
	} else {
		sub, err := fs.Sub(embedded, "template")
		if err != nil {
			return fail(1, "%v", err)
		}
		template = sub
	}

	target, err := filepath.Abs(positional[0])
	if err != nil {
		return fail(1, "%v", err)
	}
	if !*dryRun {
		if err := os.MkdirAll(target, 0o777); err != nil {
			return fail(1, "%v", err)
		}
	}
	saved, err := readManifest(target)
	if err != nil {
		return fail(1, "%v", err)
	}
	opts, err := resolveOptions(f, target, saved.Options)
	if err != nil {
		return fail(1, "%v", err)
	}
	in := newInstaller(target, opts, template, *dryRun, *showDiff, replace, saved.Files, stdout)
	managed, err := in.managedPaths()
	if err != nil {
		return fail(1, "%v", err)
	}
	var unknown []string
	for p := range in.replace {
		if !managed[p] {
			unknown = append(unknown, p)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fail(1, "--replace takes a managed file, such as docs/standards/style-guide.md: %s", strings.Join(unknown, ", "))
	}
	if err := in.run(); err != nil {
		return fail(1, "%v", err)
	}
	report := in.report

	dry := ""
	if *dryRun {
		dry = "Dry run: "
	}
	fmt.Fprintf(stdout, "%sdocs-bootstrap %s (%s profile, style %s) in %s\n", dry, version, opts.Profile, opts.Style, target)
	labels := map[string]string{
		"created":  "Created",
		"updated":  "Updated",
		"merged":   "Merged into",
		"kept":     "Not restored",
		"conflict": "Conflicts, left unchanged",
	}
	for _, key := range reportKeys {
		if len(report[key]) > 0 {
			fmt.Fprintf(stdout, "\n%s (%d):\n", labels[key], len(report[key]))
			for _, rel := range report[key] {
				fmt.Fprintf(stdout, "  %s\n", rel)
			}
		}
	}
	if len(report["created"])+len(report["updated"])+len(report["merged"])+len(report["conflict"]) == 0 {
		fmt.Fprint(stdout, "\nAlready up to date.\n")
	}
	if len(report["conflict"]) > 0 {
		fmt.Fprint(stdout, "\nYour version of each conflicting file is kept. To compare, run again with --diff. "+
			"To take the new version, run again with --replace PATH.\n")
	}
	fmt.Fprint(stdout, "\nNext: run scripts/docs-lint.sh in the repository. See docs/documentation/check-documentation.md there.\n")
	if len(report["conflict"]) > 0 {
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
