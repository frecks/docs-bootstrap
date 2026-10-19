"""Tests for bootstrap.py. Run from the repository root:

    python3 -m unittest discover -s tests
"""

import contextlib
import io
import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, ROOT)

import bootstrap  # noqa: E402


def run(target, *args, template_dir=None):
    """Run the installer quietly; return (exit code, output)."""
    argv = [target, *args]
    if template_dir:
        argv += ['--template-dir', template_dir]
    out = io.StringIO()
    with contextlib.redirect_stdout(out):
        code = bootstrap.main(argv)
    return code, out.getvalue()


def read(path):
    with open(path, encoding='utf-8') as f:
        return f.read()


def write(path, text):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, 'w', encoding='utf-8') as f:
        f.write(text)


class BootstrapTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp()
        self.repo = os.path.join(self.tmp, 'sample-repo')

    def tearDown(self):
        shutil.rmtree(self.tmp)

    def path(self, rel):
        return os.path.join(self.repo, rel)

    def all_text(self):
        for directory, _, files in os.walk(self.repo):
            for name in files:
                full = os.path.join(directory, name)
                if not os.path.islink(full):
                    yield full, read(full)

    def test_fresh_install_renders_every_file(self):
        for profile in bootstrap.PROFILES:
            with self.subTest(profile=profile):
                shutil.rmtree(self.repo, ignore_errors=True)
                code, _ = run(self.repo, '--profile', profile)
                self.assertEqual(code, 0)
                for full, text in self.all_text():
                    self.assertNotIn('{{', text, full)
                    if full.endswith('.md'):
                        # Pages stay free of Vale configuration comments.
                        self.assertNotIn('<!-- vale', text, full)
                self.assertTrue(os.access(self.path('scripts/docs-lint.sh'), os.X_OK))
                self.assertEqual(os.readlink(self.path('CLAUDE.md')), 'AGENTS.md')
                self.assertEqual(os.readlink(self.path('.claude/skills/docs-write')), '../../.agents/skills/docs-write')
                self.assertTrue(os.path.isdir(self.path('.vale/styles/SampleRepo')))

    def test_profiles_differ_where_intended(self):
        run(self.repo, '--profile', 'infrastructure', '--noun', 'service')
        guide = read(self.path('docs/standards/style-guide.md'))
        self.assertIn('## Say where a command runs', guide)
        self.assertIn('this service', guide)
        shutil.rmtree(self.repo)
        run(self.repo, '--profile', 'software')
        guide = read(self.path('docs/standards/style-guide.md'))
        self.assertNotIn('## Say where a command runs', guide)
        self.assertNotIn('Name the host', read(self.path('.agents/skills/docs-write/SKILL.md')))

    def test_install_passes_its_own_structure_checks(self):
        run(self.repo)
        result = subprocess.run([self.path('scripts/docs-lint.sh')], capture_output=True, text=True)
        for section in ('== Placement and type\nOK', '== Template instructions\nOK', '== Links\nOK'):
            self.assertIn(section, result.stdout)

    def test_second_run_changes_nothing(self):
        run(self.repo)
        code, out = run(self.repo)
        self.assertEqual(code, 0)
        self.assertIn('Already up to date.', out)

    def test_options_are_remembered(self):
        run(self.repo, '--profile', 'infrastructure', '--style', 'Lab')
        run(self.repo)
        options = json.loads(read(self.path('.docs-bootstrap.json')))['options']
        self.assertEqual(options['profile'], 'infrastructure')
        self.assertEqual(options['style'], 'Lab')

    def test_existing_files_are_merged(self):
        write(self.path('.gitignore'), 'node_modules/\n')
        write(self.path('AGENTS.md'), '# Sample\n\nBuild with make.\n')
        write(self.path('CLAUDE.md'), 'Use tabs.\n')
        other_hook = {'matcher': 'Bash', 'hooks': [{'type': 'command', 'command': 'echo hi'}]}
        write(self.path('.claude/settings.json'), json.dumps({'model': 'opus', 'hooks': {'PostToolUse': [other_hook]}}))
        run(self.repo)

        self.assertTrue(read(self.path('.gitignore')).startswith('node_modules/\n'))
        self.assertIn('.vale/styles/Google/', read(self.path('.gitignore')))
        agents = read(self.path('AGENTS.md'))
        self.assertTrue(agents.startswith('# Sample\n\nBuild with make.\n'))
        self.assertIn(bootstrap.BLOCK_BEGIN, agents)
        self.assertEqual(read(self.path('CLAUDE.md')), 'Use tabs.\n\n@AGENTS.md\n')
        settings = json.loads(read(self.path('.claude/settings.json')))
        self.assertEqual(settings['model'], 'opus')
        self.assertEqual(len(settings['hooks']['PostToolUse']), 2)

        # Merging again adds nothing.
        before = {rel: read(self.path(rel)) for rel in ('.gitignore', 'AGENTS.md', 'CLAUDE.md', '.claude/settings.json')}
        run(self.repo)
        for rel, text in before.items():
            self.assertEqual(read(self.path(rel)), text, rel)

    def test_existing_config_is_a_conflict(self):
        write(self.path('.vale.ini'), 'StylesPath = styles\n')
        code, out = run(self.repo)
        self.assertEqual(code, 1)
        self.assertIn('Conflicts', out)
        self.assertEqual(read(self.path('.vale.ini')), 'StylesPath = styles\n')

    def test_upgrade_updates_unedited_files_and_keeps_edited_ones(self):
        template = os.path.join(self.tmp, 'template')
        shutil.copytree(os.path.join(ROOT, 'template'), template, symlinks=True)
        run(self.repo, template_dir=template)

        # Edit one managed file in the repository, and seed file content.
        guide = self.path('docs/standards/style-guide.md')
        write(guide, read(guide) + '\nLocal rule.\n')
        write(self.path('docs/README.md'), '# Ours\n')

        # A new version of the template changes both, plus another file.
        for rel in ('docs/standards/style-guide.md', 'docs/standards/structure.md', 'docs/README.md'):
            full = os.path.join(template, rel)
            write(full, read(full) + '\nUpstream change.\n')

        code, out = run(self.repo, template_dir=template)
        self.assertEqual(code, 1)
        self.assertIn('Local rule.', read(guide))
        self.assertNotIn('Upstream change.', read(guide))
        self.assertIn('Upstream change.', read(self.path('docs/standards/structure.md')))
        self.assertEqual(read(self.path('docs/README.md')), '# Ours\n')
        self.assertIn('docs/standards/style-guide.md', out.split('Conflicts')[1])

        # Taking the new version replaces the local edit, and later upgrades
        # treat the file as unedited again.
        code, _ = run(self.repo, '--replace', 'docs/standards/style-guide.md', template_dir=template)
        self.assertEqual(code, 0)
        self.assertIn('Upstream change.', read(guide))
        self.assertNotIn('Local rule.', read(guide))
        code, out = run(self.repo, template_dir=template)
        self.assertEqual(code, 0)
        self.assertIn('Already up to date.', out)

    def test_deleted_managed_file_stays_deleted(self):
        run(self.repo)
        os.remove(self.path('docs/standards/templates/tutorial.md'))
        code, out = run(self.repo)
        self.assertEqual(code, 0)
        self.assertFalse(os.path.exists(self.path('docs/standards/templates/tutorial.md')))
        self.assertIn('deleted locally', out)

    def test_replace_rejects_unmanaged_paths_before_writing(self):
        with self.assertRaises(SystemExit):
            run(self.repo, '--replace', 'docs/README.md')
        self.assertFalse(os.path.exists(self.path('.vale.ini')))

    def test_dry_run_writes_nothing(self):
        code, out = run(self.repo, '--dry-run')
        self.assertEqual(code, 0)
        self.assertIn('Dry run', out)
        self.assertFalse(os.path.exists(self.repo))


if __name__ == '__main__':
    unittest.main()
