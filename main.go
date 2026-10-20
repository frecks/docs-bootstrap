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
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	version      = "0.4.0"
	manifestName = ".docs-bootstrap.json"
	blockBegin   = "<!-- docs-bootstrap:begin -->"
	blockEnd     = "<!-- docs-bootstrap:end -->"
)

//go:embed all:template
var embedded embed.FS

// Created once, then owned by the repository. Updated only while unedited.
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
	placeholder = regexp.MustCompile(`\{\{(\w+)\}\}`)
	agentsBlock = regexp.MustCompile(`(?s)` + regexp.QuoteMeta(blockBegin) + `.*?` + regexp.QuoteMeta(blockEnd))
	importsMD   = regexp.MustCompile(`(?m)^@AGENTS\.md\s*$`)
	separators  = regexp.MustCompile(`[-_]+`)
	alnumWords  = regexp.MustCompile(`[A-Za-z0-9]+`)
	validStyle  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
)

// render fills {{VAR}} placeholders.
func render(text string, vars map[string]string) (string, error) {
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
	return normalizeNewlines(string(data)), err == nil, err
}

// read returns a repository file's contents with \n line endings, and
// whether it exists. It records whether the file had CRLF line endings, so
// that write can keep them.
func (in *installer) read(rel string) (string, bool, error) {
	data, err := os.ReadFile(in.path(rel))
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	in.crlf[rel] = bytes.Contains(data, []byte("\r\n"))
	return normalizeNewlines(string(data)), err == nil, err
}

// normalizeNewlines turns \r\n and \r line endings into \n, so a checkout
// with CRLF line endings matches the template.
func normalizeNewlines(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
}

type options struct {
	Name string `json:"name"`
	Date string `json:"date"`
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

// reportSections are the report's sections, in the order they're printed.
var reportSections = []struct{ key, label string }{
	{"created", "Created"},
	{"updated", "Updated"},
	{"merged", "Merged into"},
	{"kept", "Not restored"},
	{"conflict", "Conflicts, left unchanged"},
}

type installer struct {
	target    string
	options   options
	files     []templateFile
	dryRun    bool
	showDiff  bool
	replace   map[string]bool
	vars      map[string]string
	oldHashes map[string]string
	newHashes map[string]string
	crlf      map[string]bool // files read with CRLF line endings
	report    map[string][]string
	out       io.Writer
}

// templateFile is a file in the template, rendered for this repository.
type templateFile struct {
	template string // path in the template
	rel      string // path in the repository
	text     string
	mode     fs.FileMode
}

func newInstaller(target string, opts options, template fs.FS, dryRun, showDiff bool, replace []string, oldHashes map[string]string, out io.Writer) (*installer, error) {
	in := &installer{
		target:    target,
		options:   opts,
		dryRun:    dryRun,
		showDiff:  showDiff,
		replace:   map[string]bool{},
		vars:      variablesFor(opts),
		oldHashes: oldHashes,
		newHashes: map[string]string{},
		crlf:      map[string]bool{},
		report:    map[string][]string{},
		out:       out,
	}
	if in.oldHashes == nil {
		in.oldHashes = map[string]string{}
	}
	for _, p := range replace {
		// Template paths always use slashes.
		in.replace[path.Clean(filepath.ToSlash(p))] = true
	}
	var err error
	in.files, err = loadTemplate(template, in.vars)
	return in, err
}

// loadTemplate reads and renders every file in the template.
func loadTemplate(template fs.FS, vars map[string]string) ([]templateFile, error) {
	var files []templateFile
	err := fs.WalkDir(template, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		raw, err := fs.ReadFile(template, name)
		if err != nil {
			return err
		}
		f := templateFile{template: name, mode: 0o644}
		if f.rel, err = render(name, vars); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if f.text, err = render(normalizeNewlines(string(raw)), vars); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		// Scripts are executable. The embedded template doesn't keep file
		// modes, so tell them by their #! line.
		if strings.HasPrefix(string(raw), "#!") {
			f.mode = 0o755
		}
		files = append(files, f)
		return nil
	})
	return files, err
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
	if in.crlf[rel] {
		text = strings.ReplaceAll(text, "\n", "\r\n")
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
	for _, f := range in.files {
		var err error
		if merge, ok := merged[f.template]; ok {
			err = merge(in, f.rel, f.text)
		} else if seed[f.template] {
			err = in.seed(f.rel, f.text, f.mode)
		} else {
			err = in.managed(f.rel, f.text, f.mode)
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

// replaceablePaths lists the managed and seed files, which --replace accepts.
func (in *installer) replaceablePaths() map[string]bool {
	paths := map[string]bool{}
	for _, f := range in.files {
		if merged[f.template] == nil {
			paths[f.rel] = true
		}
	}
	return paths
}

// seed creates a file that the repository owns from then on. An upgrade
// updates it only while it matches the version the installer last wrote, so a
// repository that hasn't edited it still gets fixes.
func (in *installer) seed(rel, text string, mode fs.FileMode) error {
	current, exists, err := in.read(rel)
	if err != nil {
		return err
	}
	newHash := sha(text)
	oldHash, installed := in.oldHashes[rel]
	switch {
	case in.replace[rel]:
		in.newHashes[rel] = newHash
		if !exists || current != text {
			in.add("updated", rel+" (replaced)")
			return in.write(rel, text, mode)
		}
	case !exists:
		in.newHashes[rel] = newHash
		in.add("created", rel)
		return in.write(rel, text, mode)
	case current == text:
		in.newHashes[rel] = newHash
	case installed && oldHash == sha(current):
		in.newHashes[rel] = newHash
		in.add("updated", rel)
		return in.write(rel, text, mode)
	}
	// Edited here, or present before the install: the repository owns it, so
	// the installer stops tracking it.
	return nil
}

func (in *installer) managed(rel, text string, mode fs.FileMode) error {
	full := in.path(rel)
	newHash := sha(text)
	current, exists, err := in.read(rel)
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
		return in.diff(rel, current, text)
	}
	return nil
}

// diff prints the changes from current, the repository's version of rel, to text.
func (in *installer) diff(rel, current, text string) error {
	dir, err := os.MkdirTemp("", "docs-bootstrap-diff-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	// Copies under these names make the diff headers say which is which.
	yours, ours := filepath.Join("yours", rel), filepath.Join("docs-bootstrap", rel)
	for name, data := range map[string][]byte{yours: []byte(current), ours: []byte(text)} {
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o777); err != nil {
			return err
		}
		if err := os.WriteFile(full, data, 0o666); err != nil {
			return err
		}
	}
	fmt.Fprintln(in.out)
	// Compare the plain text, whatever the user's diff drivers and textconv
	// settings. --color=auto overrides a color.ui=always setting, which would
	// put escape codes in redirected output.
	cmd := exec.Command("git", "--no-pager", "diff", "--no-index", "--no-ext-diff", "--no-textconv", "--no-prefix", "--color=auto", "--", yours, ours)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = in.out, &stderr
	// git diff exits with status 1 when the files differ.
	if err := cmd.Run(); err != nil && !(errors.As(err, new(*exec.ExitError)) && cmd.ProcessState.ExitCode() == 1) {
		return fmt.Errorf("git diff %s: %w: %s", rel, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// markersPaired reports whether each begin marker in text is followed by an
// end marker before the next begin marker.
func markersPaired(text string) bool {
	for {
		b, e := strings.Index(text, blockBegin), strings.Index(text, blockEnd)
		if b < 0 || e < 0 {
			return b < 0 && e < 0
		}
		if e < b || strings.Contains(text[b+len(blockBegin):e], blockBegin) {
			return false
		}
		text = text[e+len(blockEnd):]
	}
}

func (in *installer) mergeAgentsMD(rel, block string) error {
	block = blockBegin + "\n" + strings.TrimRight(block, "\n") + "\n" + blockEnd
	current, exists, err := in.read(rel)
	if err != nil {
		return err
	}
	if !exists {
		in.add("created", rel)
		return in.write(rel, fmt.Sprintf("# %s\n\n%s\n", in.vars["PROJECT"], block), 0)
	}
	// Replacing between unpaired markers would delete the text between them.
	if !markersPaired(current) {
		in.add("conflict", rel+" (a docs-bootstrap:begin or :end comment is missing; fix it by hand)")
		return nil
	}
	if strings.Count(current, blockBegin) > 1 {
		in.add("conflict", rel+" (more than one docs-bootstrap section; delete all but one)")
		return nil
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
	current, exists, err := in.read(rel)
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	for _, line := range strings.Split(current, "\n") {
		existing[strings.TrimSpace(line)] = true
	}
	// Add each missing pattern with the comments just above it in the template.
	var missing, comments []string
	for _, line := range strings.Split(text, "\n") {
		switch trimmed := strings.TrimSpace(line); {
		case trimmed == "":
			comments = nil
		case strings.HasPrefix(trimmed, "#"):
			comments = append(comments, line)
		default:
			if !existing[trimmed] {
				if len(missing) > 0 && len(comments) > 0 {
					missing = append(missing, "")
				}
				missing = append(missing, comments...)
				missing = append(missing, line)
			}
			comments = nil
		}
	}
	if len(missing) == 0 {
		return nil
	}
	addition := strings.Join(missing, "\n") + "\n"
	if !exists {
		in.add("created", rel)
		return in.write(rel, addition, 0)
	}
	if current != "" {
		addition = "\n" + addition
		if !strings.HasSuffix(current, "\n") {
			current += "\n"
		}
	}
	in.add("merged", rel)
	return in.write(rel, current+addition, 0)
}

func (in *installer) mergeSettings(rel, text string) error {
	template, err := decodeJSON(text)
	if err != nil {
		return fmt.Errorf("template %s: %w", rel, err)
	}
	// Each hook event in the template, such as PostToolUse, has one entry.
	templateHooks, ok := get(template, "hooks").(*object)
	if !ok || len(templateHooks.keys) == 0 {
		return fmt.Errorf("template %s: hooks is missing", rel)
	}
	for _, event := range templateHooks.keys {
		if _, ok := get(index(get(index(templateHooks.values[event], 0), "hooks"), 0), "command").(string); !ok {
			return fmt.Errorf("template %s: hooks.%s[0].hooks[0].command is missing", rel, event)
		}
	}
	current, exists, err := in.read(rel)
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
	changed := false
	for _, event := range templateHooks.keys {
		ours := index(templateHooks.values[event], 0)
		command := get(index(get(ours, "hooks"), 0), "command")
		if _, ok := hooks.values[event]; !ok {
			hooks.set(event, []any{})
		}
		entries, ok := hooks.values[event].([]any)
		if !ok {
			in.add("conflict", rel+" (hooks."+event+" isn't a JSON array; add the hook by hand)")
			return nil
		}
		if !hasCommand(entries, command) {
			hooks.set(event, append(entries, ours))
			changed = true
		}
	}
	if !changed {
		return nil
	}
	out, err := marshal(settings)
	if err != nil {
		return err
	}
	in.add("merged", rel)
	return in.write(rel, out, 0)
}

// hasCommand reports whether a hook in entries runs command.
func hasCommand(entries []any, command any) bool {
	for _, entry := range entries {
		if list, ok := get(entry, "hooks").([]any); ok {
			for _, hook := range list {
				if get(hook, "command") == command {
					return true
				}
			}
		}
	}
	return false
}

func (in *installer) linkClaudeMD() error {
	rel := "CLAUDE.md"
	full := in.path(rel)
	if info, err := os.Lstat(full); err == nil && info.Mode()&fs.ModeSymlink != 0 {
		return nil
	}
	current, exists, err := in.read(rel)
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
	return map[string]string{
		"PROJECT": opts.Name,
		"STYLE":   styleName(opts.Name),
		"DATE":    opts.Date,
	}
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

// capitalize upper-cases the first letter.
func capitalize(s string) string {
	r := []rune(s)
	if len(r) > 0 {
		r[0] = unicode.ToUpper(r[0])
	}
	return string(r)
}

// styleName returns the Vale style name for a project: the words of name,
// each with its first letter upper-cased, without anything that isn't a
// letter or digit. If that doesn't start with a letter, it's HouseRules.
func styleName(name string) string {
	var b strings.Builder
	for _, word := range alnumWords.FindAllString(name, -1) {
		b.WriteString(capitalize(word))
	}
	if !validStyle.MatchString(b.String()) {
		// Such as a name that starts with a digit.
		return "HouseRules"
	}
	return b.String()
}

// usageError is an error in the command line.
type usageError struct{ error }

// resolveOptions combines the --name option with the options saved by a
// previous run.
func resolveOptions(name, target string, saved options) (options, error) {
	base := filepath.Base(target)
	if base == string(filepath.Separator) {
		base = ""
	}
	opts := options{
		Name: first(name, saved.Name, capitalize(strings.TrimSpace(separators.ReplaceAllString(base, " ")))),
		Date: first(saved.Date, time.Now().Format("2006-01-02")),
	}
	if opts.Name == "" {
		return opts, usageError{errors.New("TARGET has no directory name; give --name")}
	}
	return opts, nil
}

// takesValue reports whether arg is a flag, such as --name, whose value is
// the next argument.
func takesValue(set *flag.FlagSet, arg string) bool {
	name := strings.TrimLeft(arg, "-")
	if !strings.HasPrefix(arg, "-") || strings.Contains(name, "=") {
		return false
	}
	f := set.Lookup(name)
	if f == nil {
		return false
	}
	b, ok := f.Value.(interface{ IsBoolFlag() bool })
	return !ok || !b.IsBoolFlag()
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
  --name NAME      project name, used in titles and to name the Vale style
                   (default: from the directory name)
  --dry-run        show what would change, but don't write
  --diff           print a diff for each conflicting file
  --replace PATH   take the new version of this managed or seed file,
                   replacing local edits (repeatable)
  --version        print the version and exit
`

// run runs the installer with the given arguments, and returns the exit status.
func run(args []string, stdout, stderr io.Writer) int {
	set := flag.NewFlagSet("docs-bootstrap", flag.ContinueOnError)
	set.SetOutput(stderr)
	set.Usage = func() {}
	var replace stringList
	name := set.String("name", "", "")
	dryRun := set.Bool("dry-run", false, "")
	showDiff := set.Bool("diff", false, "")
	set.Var(&replace, "replace", "")
	templateDir := set.String("template-dir", "", "")
	showVersion := set.Bool("version", false, "")

	// Options may come before or after TARGET, so sort the arguments into
	// flags, with their values, and positional arguments.
	var flagArgs, positional []string
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--":
			positional = append(positional, args[i+1:]...)
			i = len(args)
		case len(arg) > 1 && arg[0] == '-':
			flagArgs = append(flagArgs, arg)
			if takesValue(set, arg) && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		default:
			positional = append(positional, arg)
		}
	}
	if err := set.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usage)
			return 0
		}
		fmt.Fprint(stderr, usage)
		return 2
	}
	if *showVersion {
		fmt.Fprintf(stdout, "docs-bootstrap %s\n", version)
		return 0
	}
	fail := func(code int, format string, a ...any) int {
		fmt.Fprintf(stderr, "error: "+format+"\n", a...)
		return code
	}
	if len(positional) != 1 {
		fmt.Fprint(stderr, usage)
		return fail(2, "give one TARGET")
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
	saved, err := readManifest(target)
	if err != nil {
		return fail(1, "%v", err)
	}
	opts, err := resolveOptions(*name, target, saved.Options)
	if err != nil {
		if errors.As(err, new(usageError)) {
			return fail(2, "%v", err)
		}
		return fail(1, "%v", err)
	}
	if *showDiff {
		if _, err := exec.LookPath("git"); err != nil {
			return fail(1, "--diff needs git, which isn't on PATH")
		}
	}
	in, err := newInstaller(target, opts, template, *dryRun, *showDiff, replace, saved.Files, stdout)
	if err != nil {
		return fail(1, "%v", err)
	}
	replaceable := in.replaceablePaths()
	var unknown []string
	for p := range in.replace {
		if !replaceable[p] {
			unknown = append(unknown, p)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fail(2, "--replace takes a managed or seed file, such as docs/standards/style-guide.md: %s", strings.Join(unknown, ", "))
	}

	// Create the target only once the command line is known to be valid.
	if !*dryRun {
		if err := os.MkdirAll(target, 0o777); err != nil {
			return fail(1, "%v", err)
		}
	}

	// The header comes first, so that any --diff output follows it.
	dry := ""
	if *dryRun {
		dry = "Dry run: "
	}
	fmt.Fprintf(stdout, "%sdocs-bootstrap %s (style %s) in %s\n", dry, version, styleName(opts.Name), target)
	if err := in.run(); err != nil {
		return fail(1, "%v", err)
	}
	report := in.report
	for _, section := range reportSections {
		if entries := report[section.key]; len(entries) > 0 {
			fmt.Fprintf(stdout, "\n%s (%d):\n", section.label, len(entries))
			for _, rel := range entries {
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
