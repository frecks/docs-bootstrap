package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// install runs the installer quietly, and returns its exit status and output.
func install(t *testing.T, target string, args ...string) (int, string) {
	t.Helper()
	var out bytes.Buffer
	code := run(append([]string{target}, args...), &out, &out)
	return code, out.String()
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o666); err != nil {
		t.Fatal(err)
	}
}

func readlink(t *testing.T, path string) string {
	t.Helper()
	link, err := os.Readlink(path)
	if err != nil {
		t.Fatal(err)
	}
	return link
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(read(t, path)), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// sampleRepo returns the path of a repository to install into, and a
// function that returns paths inside it.
func sampleRepo(t *testing.T) (string, func(string) string) {
	repo := filepath.Join(t.TempDir(), "sample-repo")
	return repo, func(rel string) string { return filepath.Join(repo, rel) }
}

// copyDir copies the files in dir to a new temporary directory.
func copyDir(t *testing.T, dir string) string {
	t.Helper()
	dest := t.TempDir()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dest, rel), 0o777)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dest, rel), data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
	return dest
}

func TestFreshInstallRendersEveryFile(t *testing.T) {
	repo, path := sampleRepo(t)
	if code, out := install(t, repo); code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	err := filepath.WalkDir(repo, func(full string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() && strings.Contains(read(t, full), "{{") {
			t.Errorf("%s: unrendered template marker", full)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path("scripts/docs-lint.sh"))
	if err != nil || info.Mode()&0o100 == 0 {
		t.Errorf("scripts/docs-lint.sh isn't executable: %v", err)
	}
	if link := readlink(t, path("CLAUDE.md")); link != "AGENTS.md" {
		t.Errorf("CLAUDE.md links to %q", link)
	}
	if link := readlink(t, path(".claude/skills/docs-write")); link != "../../.agents/skills/docs-write" {
		t.Errorf(".claude/skills/docs-write links to %q", link)
	}
	if info, err := os.Stat(path(".vale/styles/SampleRepo")); err != nil || !info.IsDir() {
		t.Errorf(".vale/styles/SampleRepo isn't a directory: %v", err)
	}
}

func TestInstallPassesItsOwnStructureChecks(t *testing.T) {
	repo, path := sampleRepo(t)
	install(t, repo)
	out, _ := exec.Command(path("scripts/docs-lint.sh")).Output()
	for _, section := range []string{"== Placement and type\nOK", "== Template instructions\nOK", "== Links\nOK"} {
		if !strings.Contains(string(out), section) {
			t.Errorf("missing %q in:\n%s", section, out)
		}
	}
}

func TestSecondRunChangesNothing(t *testing.T) {
	repo, _ := sampleRepo(t)
	install(t, repo)
	code, out := install(t, repo)
	if code != 0 || !strings.Contains(out, "Already up to date.") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestNameIsRemembered(t *testing.T) {
	repo, path := sampleRepo(t)
	install(t, repo, "--name", "Acme lab")
	install(t, repo)
	options := readJSON(t, path(".docs-bootstrap.json"))["options"].(map[string]any)
	if options["name"] != "Acme lab" || len(options) != 2 {
		t.Errorf("options = %v, want only the name and date", options)
	}
	if info, err := os.Stat(path(".vale/styles/AcmeLab")); err != nil || !info.IsDir() {
		t.Errorf(".vale/styles/AcmeLab isn't a directory: %v", err)
	}
}

func TestNamesKeepTheirCase(t *testing.T) {
	tests := []struct{ dir, name, style string }{
		{"sample-repo", "Sample repo", "SampleRepo"},
		{"my-API_server", "My API server", "MyAPIServer"},
		{"acme2go", "Acme2go", "Acme2go"},
		{"-x-", "X", "X"},
		{"2048-game", "2048 game", "HouseRules"},
	}
	for _, test := range tests {
		opts, err := resolveOptions("", filepath.Join("/tmp", test.dir), options{})
		if style := styleName(opts.Name); err != nil || opts.Name != test.name || style != test.style {
			t.Errorf("%s: name %q, style %q, %v; want %q, %q", test.dir, opts.Name, style, err, test.name, test.style)
		}
	}
	if _, err := resolveOptions("", "/", options{}); err == nil {
		t.Error("a root target without --name didn't fail")
	}
}

func TestVersion(t *testing.T) {
	var out bytes.Buffer
	if code := run([]string{"--version"}, &out, &out); code != 0 || out.String() != "docs-bootstrap "+version+"\n" {
		t.Errorf("exit %d: %q", code, out.String())
	}
}

func TestCommandLineErrorsExitTwo(t *testing.T) {
	repo, _ := sampleRepo(t)
	for _, args := range [][]string{{"--replace", "docs/README.md"}, {"--profile", "software"}, {"--bogus"}} {
		if code, _ := install(t, repo, args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
}

func TestOptionsMayFollowOrPrecedeTarget(t *testing.T) {
	repo, path := sampleRepo(t)
	var out bytes.Buffer
	if code := run([]string{"--name", "Lab", repo, "--dry-run"}, &out, &out); code != 0 {
		t.Fatalf("exit %d:\n%s", code, out.String())
	}
	if !strings.HasPrefix(out.String(), "Dry run: ") || !strings.Contains(out.String(), "(style Lab)") {
		t.Errorf("options before and after TARGET weren't both used:\n%s", out.String())
	}
	if _, err := os.Stat(path(".")); err == nil {
		t.Error("dry run created the target")
	}
}

func TestArgumentsAfterDoubleDashArePositional(t *testing.T) {
	repo, _ := sampleRepo(t)
	if code, _ := install(t, "--", repo, "--dry-run"); code != 2 {
		t.Errorf("exit %d, want 2 for two targets", code)
	}
}

func TestDoubleDashAfterAFlagValueEndsOptions(t *testing.T) {
	repo, _ := sampleRepo(t)
	if code, _ := install(t, "--name", "--diff", "--", repo, "--dry-run"); code != 2 {
		t.Errorf("exit %d, want 2 for two targets", code)
	}
}

func TestUsageErrorsCreateNothing(t *testing.T) {
	repo, _ := sampleRepo(t)
	for _, args := range [][]string{{"--profile", "software"}, {"--replace", "README.md"}} {
		install(t, repo, args...)
		if _, err := os.Stat(repo); err == nil {
			t.Errorf("%v created the target", args)
		}
	}
	// A root TARGET without --name fails after the manifest is read, but
	// before anything is written.
	if code, out := install(t, "/"); code != 2 || !strings.Contains(out, "give --name") {
		t.Errorf("root TARGET: exit %d:\n%s", code, out)
	}
}

func TestEmptyGitignoreIsMerged(t *testing.T) {
	repo, path := sampleRepo(t)
	write(t, path(".gitignore"), "")
	_, out := install(t, repo)
	if _, merged, _ := strings.Cut(out, "Merged into"); !strings.Contains(merged, ".gitignore") {
		t.Errorf(".gitignore isn't listed as merged:\n%s", out)
	}
	if text := read(t, path(".gitignore")); !strings.HasPrefix(text, "#") {
		t.Errorf(".gitignore starts with a blank line:\n%q", text)
	}
}

func TestDuplicateAgentsSectionsAreAConflict(t *testing.T) {
	repo, path := sampleRepo(t)
	section := blockBegin + "\nOld.\n" + blockEnd + "\n"
	text := "# Sample\n\n" + section + "\n" + section
	write(t, path("AGENTS.md"), text)
	if code, out := install(t, repo); code != 1 || !strings.Contains(out, "more than one docs-bootstrap section") {
		t.Errorf("exit %d:\n%s", code, out)
	}
	if read(t, path("AGENTS.md")) != text {
		t.Error("AGENTS.md changed")
	}
}

func TestDoubleDashMayBeAFlagValue(t *testing.T) {
	repo, path := sampleRepo(t)
	if code, out := install(t, "--name", "--", repo); code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if options := readJSON(t, path(".docs-bootstrap.json"))["options"].(map[string]any); options["name"] != "--" {
		t.Errorf("name = %v, want --", options["name"])
	}
}

func TestCRLFFilesKeepTheirLineEndings(t *testing.T) {
	repo, path := sampleRepo(t)
	write(t, path("AGENTS.md"), "# Sample\r\n\r\nBuild with make.\r\n")
	install(t, repo)
	agents := read(t, path("AGENTS.md"))
	if strings.Count(agents, "\n") != strings.Count(agents, "\r\n") {
		t.Errorf("AGENTS.md has LF line endings:\n%q", agents)
	}

	// A conflict's diff shows only the lines that changed.
	vale := path(".vale.ini")
	write(t, vale, strings.Replace(strings.ReplaceAll(read(t, vale), "\n", "\r\n"), "StylesPath", "StylesPath2", 1))
	_, out := install(t, repo, "--diff")
	// One line for the --- header, and one for the change.
	if n := strings.Count(out, "\n-"); n != 2 || strings.Contains(out, "\r") {
		t.Errorf("diff has %d lines starting with -, want 2:\n%q", n, out)
	}
}

func TestUnpairedAgentsMarkersAreAConflict(t *testing.T) {
	repo, path := sampleRepo(t)
	text := "# Sample\n\n" + blockBegin + "\nOld docs section.\n\nMy notes.\n"
	write(t, path("AGENTS.md"), text)
	if code, out := install(t, repo); code != 1 || !strings.Contains(out, "AGENTS.md (a docs-bootstrap:begin") {
		t.Errorf("exit %d:\n%s", code, out)
	}
	if read(t, path("AGENTS.md")) != text {
		t.Error("AGENTS.md changed")
	}
}

func TestBrokenTemplateSettingsFail(t *testing.T) {
	template := t.TempDir()
	write(t, filepath.Join(template, ".claude/settings.json"), `{"hooks": {}}`)
	repo, path := sampleRepo(t)
	write(t, path(".claude/settings.json"), `{}`)
	if code, out := install(t, repo, "--template-dir", template); code != 1 || !strings.Contains(out, "command is missing") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestCRLFCheckoutIsUnedited(t *testing.T) {
	repo, path := sampleRepo(t)
	install(t, repo)
	guide := path("docs/standards/style-guide.md")
	write(t, guide, strings.ReplaceAll(read(t, guide), "\n", "\r\n"))
	if code, out := install(t, repo); code != 0 || !strings.Contains(out, "Already up to date.") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestExistingFilesAreMerged(t *testing.T) {
	repo, path := sampleRepo(t)
	write(t, path(".gitignore"), "node_modules/\n")
	write(t, path("AGENTS.md"), "# Sample\n\nBuild with make.\n")
	write(t, path("CLAUDE.md"), "Use tabs.\n")
	write(t, path(".claude/settings.json"),
		`{"model": "opus", "env": {"A": "<b>"}, "hooks": {"PostToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "echo hi"}]}]}}`)
	install(t, repo)

	gitignore := read(t, path(".gitignore"))
	if !strings.HasPrefix(gitignore, "node_modules/\n") || !strings.Contains(gitignore, ".vale/styles/Google/") {
		t.Errorf(".gitignore:\n%s", gitignore)
	}
	agents := read(t, path("AGENTS.md"))
	if !strings.HasPrefix(agents, "# Sample\n\nBuild with make.\n") || !strings.Contains(agents, blockBegin) {
		t.Errorf("AGENTS.md:\n%s", agents)
	}
	if claude := read(t, path("CLAUDE.md")); claude != "Use tabs.\n\n@AGENTS.md\n" {
		t.Errorf("CLAUDE.md:\n%s", claude)
	}
	settings := read(t, path(".claude/settings.json"))
	if !strings.HasPrefix(settings, "{\n  \"model\": \"opus\",\n  \"env\": {\n    \"A\": \"<b>\"\n  },\n  \"hooks\"") {
		t.Errorf("settings.json lost its order or escaped its text:\n%s", settings)
	}
	hooks := readJSON(t, path(".claude/settings.json"))["hooks"].(map[string]any)
	if n := len(hooks["PostToolUse"].([]any)); n != 2 {
		t.Errorf("PostToolUse has %d entries, want 2", n)
	}

	// Merging again adds nothing.
	files := []string{".gitignore", "AGENTS.md", "CLAUDE.md", ".claude/settings.json"}
	before := map[string]string{}
	for _, rel := range files {
		before[rel] = read(t, path(rel))
	}
	install(t, repo)
	for _, rel := range files {
		if read(t, path(rel)) != before[rel] {
			t.Errorf("%s changed on the second run", rel)
		}
	}
}

func TestGitignoreGetsOnlyMissingLines(t *testing.T) {
	template := t.TempDir()
	write(t, filepath.Join(template, ".gitignore"), "# Vale\n.vale/styles/Google/\n\n# Caches\n.cache/\n")
	repo, path := sampleRepo(t)
	write(t, path(".gitignore"), "node_modules/\n.vale/styles/Google/\n")
	install(t, repo, "--template-dir", template)
	if got, want := read(t, path(".gitignore")), "node_modules/\n.vale/styles/Google/\n\n# Caches\n.cache/\n"; got != want {
		t.Errorf(".gitignore:\n%s\nwant:\n%s", got, want)
	}
}

func TestExistingConfigIsAConflict(t *testing.T) {
	repo, path := sampleRepo(t)
	write(t, path(".vale.ini"), "StylesPath = styles\n")
	code, out := install(t, repo, "--diff")
	if code != 1 || !strings.Contains(out, "Conflicts") {
		t.Errorf("exit %d:\n%s", code, out)
	}
	if !strings.Contains(out, "--- yours/.vale.ini\n+++ docs-bootstrap/.vale.ini\n@@ -1 +1,") {
		t.Errorf("missing diff in:\n%s", out)
	}
	if text := read(t, path(".vale.ini")); text != "StylesPath = styles\n" {
		t.Errorf(".vale.ini changed:\n%s", text)
	}
}

func TestUpgradeUpdatesUneditedFilesAndKeepsEditedOnes(t *testing.T) {
	template := copyDir(t, "template")
	repo, path := sampleRepo(t)
	install(t, repo, "--template-dir", template)

	// Edit one managed file in the repository, and seed file content.
	guide := path("docs/standards/style-guide.md")
	write(t, guide, read(t, guide)+"\nLocal rule.\n")
	write(t, path("docs/README.md"), "# Ours\n")

	// A new version of the template changes both, plus another file.
	for _, rel := range []string{"docs/standards/style-guide.md", "docs/standards/structure.md", "docs/README.md"} {
		full := filepath.Join(template, rel)
		write(t, full, read(t, full)+"\nUpstream change.\n")
	}

	code, out := install(t, repo, "--template-dir", template)
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if text := read(t, guide); !strings.Contains(text, "Local rule.") || strings.Contains(text, "Upstream change.") {
		t.Error("edited style guide wasn't kept")
	}
	if !strings.Contains(read(t, path("docs/standards/structure.md")), "Upstream change.") {
		t.Error("unedited structure.md wasn't updated")
	}
	if text := read(t, path("docs/README.md")); text != "# Ours\n" {
		t.Errorf("seed file changed:\n%s", text)
	}
	if _, conflicts, _ := strings.Cut(out, "Conflicts"); !strings.Contains(conflicts, "docs/standards/style-guide.md") {
		t.Errorf("style guide isn't listed as a conflict:\n%s", out)
	}

	// Taking the new version replaces the local edit, and later upgrades
	// treat the file as unedited again.
	if code, out := install(t, repo, "--replace", "docs/standards/style-guide.md", "--template-dir", template); code != 0 {
		t.Errorf("exit %d:\n%s", code, out)
	}
	if text := read(t, guide); strings.Contains(text, "Local rule.") || !strings.Contains(text, "Upstream change.") {
		t.Error("--replace didn't take the new version")
	}
	if code, out := install(t, repo, "--template-dir", template); code != 0 || !strings.Contains(out, "Already up to date.") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestDeletedManagedFileStaysDeleted(t *testing.T) {
	repo, path := sampleRepo(t)
	install(t, repo)
	tutorial := path("docs/standards/templates/tutorial.md")
	if err := os.Remove(tutorial); err != nil {
		t.Fatal(err)
	}
	code, out := install(t, repo)
	if code != 0 || !strings.Contains(out, "deleted locally") {
		t.Errorf("exit %d:\n%s", code, out)
	}
	if _, err := os.Stat(tutorial); err == nil {
		t.Error("deleted file was restored")
	}
}

func TestReplaceRejectsUnmanagedPathsBeforeWriting(t *testing.T) {
	repo, path := sampleRepo(t)
	if code, _ := install(t, repo, "--replace", "docs/README.md"); code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if _, err := os.Stat(path(".vale.ini")); err == nil {
		t.Error("installer wrote files before rejecting --replace")
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	repo, _ := sampleRepo(t)
	code, out := install(t, repo, "--dry-run")
	if code != 0 || !strings.Contains(out, "Dry run") {
		t.Errorf("exit %d:\n%s", code, out)
	}
	if _, err := os.Stat(repo); err == nil {
		t.Error("dry run created the target")
	}
}

func TestRender(t *testing.T) {
	vars := map[string]string{"PROJECT": "Acme"}
	if got, err := render("# {{PROJECT}} documentation", vars); err != nil || got != "# Acme documentation" {
		t.Errorf("render = %q, %v", got, err)
	}
	for _, in := range []string{"{{UNKNOWN}}", "{{#software}}a{{/software}}", "{{ PROJECT }}"} {
		if _, err := render(in, vars); err == nil {
			t.Errorf("render(%q) didn't fail", in)
		}
	}
}
