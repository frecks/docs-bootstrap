#!/usr/bin/env python3
"""Set up a repository with the documentation system.

Usage: bootstrap.py TARGET [options]

Installs docs standards, Vale and markdownlint configuration, lint scripts,
a Claude Code hook, and the docs-write and docs-review agent skills into
TARGET, a new or existing repository. Run it again to upgrade: files you
haven't edited are updated, and files you have edited are kept and reported.

See docs/ in this repository for how to use it.
"""

import argparse
import datetime
import difflib
import hashlib
import json
import os
import re
import sys

VERSION = '0.1.0'
MANIFEST = '.docs-bootstrap.json'
HERE = os.path.dirname(os.path.abspath(__file__))

PROFILES = {
    'infrastructure': {
        'NOUN': 'environment',
        'AREA': 'system',
        'EXAMPLE_AREA': 'network',
        'AUDIENCE': "an operator who knows Linux and networking, but doesn't know this {NOUN} or remember why it's set up this way",
    },
    'software': {
        'NOUN': 'project',
        'AREA': 'component',
        'EXAMPLE_AREA': 'api',
        'AUDIENCE': "a developer who knows the languages and tools this {NOUN} uses, but doesn't know this codebase or why it's built this way",
    },
}

# Created once, then owned by the repository. Never updated.
SEED = {
    'docs/README.md',
    'docs/documentation/check-documentation.md',
    'docs/documentation/why-the-documentation-system-works-this-way.md',
    '.vale/styles/config/vocabularies/{{STYLE}}/accept.txt',
}

# Merged into existing files instead of copied, by the named Installer method.
MERGED = {
    'AGENTS.md': 'merge_agents_md',
    '.gitignore': 'merge_gitignore',
    '.claude/settings.json': 'merge_settings',
}

SKILLS = ['docs-write', 'docs-review']
BLOCK_BEGIN = '<!-- docs-bootstrap:begin -->'
BLOCK_END = '<!-- docs-bootstrap:end -->'


def render(text, variables, profile):
    """Fill {{VAR}} placeholders and keep only this profile's {{#p}}…{{/p}} sections."""
    # A marker alone on its line takes the line with it.
    text = re.sub(r'^[ \t]*(\{\{[#/]\w+\}\})[ \t]*\n', r'\1', text, flags=re.MULTILINE)
    text = re.sub(
        r'\{\{#(\w+)\}\}(.*?)\{\{/\1\}\}',
        lambda m: m.group(2) if m.group(1) == profile else '',
        text,
        flags=re.DOTALL,
    )
    text = re.sub(r'\{\{(\w+)\}\}', lambda m: variables[m.group(1)], text)
    if '{{' in text:
        raise ValueError('unrendered template marker: ' + text[text.index('{{'):][:40])
    return text


def sha(text):
    return hashlib.sha256(text.encode('utf-8')).hexdigest()


def read(path):
    with open(path, encoding='utf-8') as f:
        return f.read()


class Installer:
    def __init__(self, target, options, template_dir, dry_run, show_diff, replace=()):
        self.target = target
        self.options = options
        self.template_dir = template_dir
        self.dry_run = dry_run
        self.show_diff = show_diff
        self.replace = {os.path.normpath(p) for p in replace}
        self.variables = variables_for(options)
        self.profile = options['profile']
        self.old_hashes = {}
        self.new_hashes = {}
        self.report = {'created': [], 'updated': [], 'merged': [], 'kept': [], 'conflict': []}
        manifest = os.path.join(target, MANIFEST)
        if os.path.exists(manifest):
            self.old_hashes = json.loads(read(manifest)).get('files', {})

    def path(self, rel):
        return os.path.join(self.target, rel)

    def write(self, rel, text, mode=None):
        if self.dry_run:
            return
        full = self.path(rel)
        os.makedirs(os.path.dirname(full) or '.', exist_ok=True)
        with open(full, 'w', encoding='utf-8') as f:
            f.write(text)
        if mode is not None:
            os.chmod(full, mode)

    def run(self):
        for rel_template, mode in self.template_files():
            rel = render(rel_template, self.variables, self.profile)
            text = render(read(os.path.join(self.template_dir, rel_template)), self.variables, self.profile)
            if rel_template in MERGED:
                getattr(self, MERGED[rel_template])(rel, text)
            elif rel_template in SEED:
                self.seed(rel, text, mode)
            else:
                self.managed(rel, text, mode)
        self.link_claude_md()
        for skill in SKILLS:
            self.symlink('.claude/skills/' + skill, '../../.agents/skills/' + skill)
        self.save_manifest()
        return self.report

    def managed_paths(self):
        return {render(rel, self.variables, self.profile) for rel, _ in self.template_files()
                if rel not in SEED and rel not in MERGED}

    def template_files(self):
        for directory, _, files in os.walk(self.template_dir):
            for name in sorted(files):
                full = os.path.join(directory, name)
                yield os.path.relpath(full, self.template_dir), os.stat(full).st_mode & 0o777

    def seed(self, rel, text, mode):
        if not os.path.exists(self.path(rel)):
            self.write(rel, text, mode)
            self.report['created'].append(rel)

    def managed(self, rel, text, mode):
        full = self.path(rel)
        new_hash = sha(text)
        if rel in self.replace:
            if not os.path.exists(full) or read(full) != text:
                self.write(rel, text, mode)
                self.report['updated'].append(rel + ' (replaced)')
            self.new_hashes[rel] = new_hash
            return
        if not os.path.exists(full):
            if rel in self.old_hashes:
                # Installed before and deleted since: respect the deletion.
                self.new_hashes[rel] = self.old_hashes[rel]
                self.report['kept'].append(rel + ' (deleted locally)')
                return
            self.write(rel, text, mode)
            self.new_hashes[rel] = new_hash
            self.report['created'].append(rel)
            return
        current = read(full)
        if current == text:
            self.new_hashes[rel] = new_hash
            if mode is not None and not self.dry_run:
                os.chmod(full, mode)
            return
        if self.old_hashes.get(rel) == sha(current):
            self.write(rel, text, mode)
            self.new_hashes[rel] = new_hash
            self.report['updated'].append(rel)
            return
        # Edited locally, or present before the system was installed.
        if rel in self.old_hashes:
            self.new_hashes[rel] = self.old_hashes[rel]
        self.report['conflict'].append(rel)
        if self.show_diff:
            sys.stdout.writelines(difflib.unified_diff(
                current.splitlines(True), text.splitlines(True), rel + ' (yours)', rel + ' (docs-bootstrap)'))

    def merge_agents_md(self, rel, block):
        block = BLOCK_BEGIN + '\n' + block.rstrip('\n') + '\n' + BLOCK_END
        full = self.path(rel)
        if not os.path.exists(full):
            self.write(rel, '# {}\n\n{}\n'.format(self.variables['PROJECT'], block))
            self.report['created'].append(rel)
            return
        current = read(full)
        pattern = re.compile(re.escape(BLOCK_BEGIN) + '.*?' + re.escape(BLOCK_END), re.DOTALL)
        if pattern.search(current):
            merged = pattern.sub(lambda m: block, current)
        else:
            merged = current.rstrip('\n') + '\n\n' + block + '\n'
        if merged != current:
            self.write(rel, merged)
            self.report['merged'].append(rel)

    def merge_gitignore(self, rel, text):
        full = self.path(rel)
        current = read(full) if os.path.exists(full) else ''
        existing = {line.strip() for line in current.splitlines()}
        missing = [line for line in text.splitlines() if line.strip() and not line.startswith('#') and line.strip() not in existing]
        if not missing:
            return
        addition = text if not current else '\n' + text
        self.write(rel, current + ('' if current.endswith('\n') or not current else '\n') + addition)
        self.report['merged' if current else 'created'].append(rel)

    def merge_settings(self, rel, text):
        ours = json.loads(text)['hooks']['PostToolUse'][0]
        full = self.path(rel)
        if not os.path.exists(full):
            self.write(rel, text)
            self.report['created'].append(rel)
            return
        try:
            settings = json.loads(read(full))
        except json.JSONDecodeError:
            self.report['conflict'].append(rel + ' (not valid JSON; add the hook by hand)')
            return
        entries = settings.setdefault('hooks', {}).setdefault('PostToolUse', [])
        command = ours['hooks'][0]['command']
        if any(h.get('command') == command for e in entries for h in e.get('hooks', [])):
            return
        entries.append(ours)
        self.write(rel, json.dumps(settings, indent=2, ensure_ascii=False) + '\n')
        self.report['merged'].append(rel)

    def link_claude_md(self):
        rel = 'CLAUDE.md'
        full = self.path(rel)
        if os.path.islink(full):
            return
        if not os.path.exists(full):
            if not self.dry_run:
                os.symlink('AGENTS.md', full)
            self.report['created'].append(rel + ' -> AGENTS.md')
            return
        current = read(full)
        if not re.search(r'^@AGENTS\.md\s*$', current, re.MULTILINE):
            self.write(rel, current.rstrip('\n') + '\n\n@AGENTS.md\n')
            self.report['merged'].append(rel + ' (imports AGENTS.md)')

    def symlink(self, rel, target):
        full = self.path(rel)
        if os.path.islink(full) and os.readlink(full) == target:
            return
        if os.path.lexists(full):
            self.report['conflict'].append(rel + ' (expected a link to ' + target + ')')
            return
        if not self.dry_run:
            os.makedirs(os.path.dirname(full), exist_ok=True)
            os.symlink(target, full)
        self.report['created'].append(rel + ' -> ' + target)

    def save_manifest(self):
        manifest = {'version': VERSION, 'options': self.options, 'files': dict(sorted(self.new_hashes.items()))}
        self.write(MANIFEST, json.dumps(manifest, indent=2, ensure_ascii=False) + '\n')


def variables_for(options):
    profile = PROFILES[options['profile']]
    variables = {
        'PROJECT': options['name'],
        'STYLE': options['style'],
        'NOUN': options['noun'],
        'AREA': profile['AREA'],
        'EXAMPLE_AREA': profile['EXAMPLE_AREA'],
        'DATE': options['date'],
    }
    variables['AUDIENCE'] = options['audience'] or profile['AUDIENCE'].format(NOUN=options['noun'])
    return variables


def resolve_options(args, target):
    """Combine command-line options with those saved by a previous run."""
    saved = {}
    manifest = os.path.join(target, MANIFEST)
    if os.path.exists(manifest):
        saved = json.loads(read(manifest)).get('options', {})
    base = os.path.basename(os.path.abspath(target))
    options = {
        'profile': args.profile or saved.get('profile') or 'software',
        'name': args.name or saved.get('name') or re.sub(r'[-_]+', ' ', base).capitalize(),
        'audience': args.audience or saved.get('audience'),
        'date': saved.get('date') or datetime.date.today().isoformat(),
    }
    options['style'] = args.style or saved.get('style') or re.sub(r'[^A-Za-z0-9]', '', options['name'].title()) or 'House'
    options['noun'] = args.noun or saved.get('noun') or PROFILES[options['profile']]['NOUN']
    if not re.fullmatch(r'[A-Za-z][A-Za-z0-9]*', options['style']):
        sys.exit('error: --style must be letters and digits, starting with a letter')
    return options


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.split('\n\n')[1], formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('target', help='repository to set up; created if missing')
    parser.add_argument('--profile', choices=sorted(PROFILES), help='rule set: software (default) or infrastructure')
    parser.add_argument('--name', help='project name for titles (default: from the directory name)')
    parser.add_argument('--style', help='Vale style and vocabulary name (default: from the project name)')
    parser.add_argument('--noun', help='what the docs describe, as in "this NOUN" (default: project or environment)')
    parser.add_argument('--audience', help='who the docs are for, completing "Write for …"')
    parser.add_argument('--dry-run', action='store_true', help="show what would change, but don't write")
    parser.add_argument('--diff', action='store_true', help='print a diff for each conflicting file')
    parser.add_argument('--replace', action='append', default=[], metavar='PATH',
                        help='take the new version of this managed file, replacing local edits (repeatable)')
    parser.add_argument('--template-dir', default=os.path.join(HERE, 'template'), help=argparse.SUPPRESS)
    args = parser.parse_args(argv)

    target = os.path.abspath(args.target)
    if not args.dry_run:
        os.makedirs(target, exist_ok=True)
    options = resolve_options(args, target)
    installer = Installer(target, options, args.template_dir, args.dry_run, args.diff, args.replace)
    unknown = installer.replace - installer.managed_paths()
    if unknown:
        sys.exit('error: --replace takes a managed file, such as docs/standards/style-guide.md: ' + ', '.join(sorted(unknown)))
    report = installer.run()

    print('{}docs-bootstrap {} ({} profile, style {}) in {}'.format(
        'Dry run: ' if args.dry_run else '', VERSION, options['profile'], options['style'], target))
    labels = {'created': 'Created', 'updated': 'Updated', 'merged': 'Merged into', 'kept': 'Not restored', 'conflict': 'Conflicts, left unchanged'}
    for key, label in labels.items():
        if report[key]:
            print('\n{} ({}):'.format(label, len(report[key])))
            for rel in report[key]:
                print('  ' + rel)
    if not any(report[key] for key in ('created', 'updated', 'merged', 'conflict')):
        print('\nAlready up to date.')
    if report['conflict']:
        print('\nYour version of each conflicting file is kept. To compare, run again with --diff. '
              'To take the new version, run again with --replace PATH.')
    print('\nNext: run scripts/docs-lint.sh in the repository. See docs/documentation/check-documentation.md there.')
    return 1 if report['conflict'] else 0


if __name__ == '__main__':
    sys.exit(main())
