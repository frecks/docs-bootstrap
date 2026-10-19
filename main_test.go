package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			repo, path := sampleRepo(t)
			if code, out := install(t, repo, "--profile", name); code != 0 {
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
		})
	}
}

func TestProfilesDifferWhereIntended(t *testing.T) {
	repo, path := sampleRepo(t)
	install(t, repo, "--profile", "infrastructure", "--noun", "service")
	guide := read(t, path("docs/standards/style-guide.md"))
	if !strings.Contains(guide, "## Say where a command runs") || !strings.Contains(guide, "this service") {
		t.Error("infrastructure style guide is missing its section or noun")
	}

	repo, path = sampleRepo(t)
	install(t, repo, "--profile", "software")
	if strings.Contains(read(t, path("docs/standards/style-guide.md")), "## Say where a command runs") {
		t.Error("software style guide has the infrastructure section")
	}
	if strings.Contains(read(t, path(".agents/skills/docs-write/SKILL.md")), "Name the host") {
		t.Error("software docs-write skill has the infrastructure rule")
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

func TestOptionsAreRemembered(t *testing.T) {
	repo, path := sampleRepo(t)
	install(t, repo, "--profile", "infrastructure", "--style", "Lab")
	install(t, repo)
	options := readJSON(t, path(".docs-bootstrap.json"))["options"].(map[string]any)
	if options["profile"] != "infrastructure" || options["style"] != "Lab" {
		t.Errorf("options = %v", options)
	}
}

func TestOptionsMayFollowOrPrecedeTarget(t *testing.T) {
	repo, path := sampleRepo(t)
	var out bytes.Buffer
	if code := run([]string{"--style", "Lab", repo, "--profile", "infrastructure"}, &out, &out); code != 0 {
		t.Fatalf("exit %d:\n%s", code, out.String())
	}
	options := readJSON(t, path(".docs-bootstrap.json"))["options"].(map[string]any)
	if options["profile"] != "infrastructure" || options["style"] != "Lab" {
		t.Errorf("options = %v", options)
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

func TestExistingConfigIsAConflict(t *testing.T) {
	repo, path := sampleRepo(t)
	write(t, path(".vale.ini"), "StylesPath = styles\n")
	code, out := install(t, repo, "--diff")
	if code != 1 || !strings.Contains(out, "Conflicts") {
		t.Errorf("exit %d:\n%s", code, out)
	}
	if !strings.Contains(out, "--- .vale.ini (yours)\n+++ .vale.ini (docs-bootstrap)\n@@ -1 +1,") {
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
	if code, _ := install(t, repo, "--replace", "docs/README.md"); code == 0 {
		t.Error("exit 0, want an error")
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
	vars := map[string]string{"NOUN": "project"}
	tests := []struct{ in, want string }{
		{"this {{NOUN}}", "this project"},
		{"a{{#software}}b{{/software}}{{#infrastructure}}c{{/infrastructure}}d", "abd"},
		{"a\n{{#software}}\nb\n{{/software}}\n{{#infrastructure}}\nc\n{{/infrastructure}}\nd\n", "a\nb\nd\n"},
	}
	for _, test := range tests {
		if got, err := render(test.in, vars, "software"); err != nil || got != test.want {
			t.Errorf("render(%q) = %q, %v; want %q", test.in, got, err, test.want)
		}
	}
	for _, in := range []string{"{{UNKNOWN}}", "{{#software}}unclosed", "{{ NOUN }}"} {
		if _, err := render(in, vars, "software"); err == nil {
			t.Errorf("render(%q) didn't fail", in)
		}
	}
}

func TestUnifiedDiff(t *testing.T) {
	a := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n"
	b := "1\nTWO\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13\n"
	want := "--- a\n+++ b\n" +
		"@@ -1,5 +1,5 @@\n 1\n-2\n+TWO\n 3\n 4\n 5\n" +
		"@@ -10,3 +10,4 @@\n 10\n 11\n 12\n+13\n"
	if got := unifiedDiff(a, b, "a", "b"); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if got := unifiedDiff(a, a, "a", "b"); got != "" {
		t.Errorf("same text gave a diff:\n%s", got)
	}
}
