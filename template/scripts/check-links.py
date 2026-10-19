#!/usr/bin/env python3
"""Report relative links in Markdown files that point to missing files.

Usage: scripts/check-links.py FILE...

Checks inline links and images, such as [text](../guides/setup.md), and
reference definitions, such as [setup]: ../guides/setup.md. Skips external
URLs, same-page anchors, code, and HTML comments. Doesn't check anchors in
other pages. Exits 1 if any link is broken.
"""

import os
import re
import sys

INLINE = re.compile(r'!?\[[^\]]*\]\(\s*(<[^>]*>|[^)\s]+)')
REFERENCE = re.compile(r'^\s{0,3}\[[^\]]+\]:\s*(<[^>]*>|\S+)')
EXTERNAL = re.compile(r'^[a-z][a-z0-9+.-]*:', re.IGNORECASE)
FENCE = re.compile(r'^\s{0,3}(`{3,}|~{3,})')


def links(text):
    """Yield (line number, target) for each link outside code and comments."""
    text = re.sub(r'<!--.*?-->', lambda m: '\n' * m.group().count('\n'), text, flags=re.DOTALL)
    fence = None
    for number, line in enumerate(text.splitlines(), 1):
        match = FENCE.match(line)
        if fence is None:
            if match:
                fence = match.group(1)
                continue
        else:
            # A fence closes with the same character, at least as many of them,
            # and nothing after.
            if match and match.group(1).startswith(fence) and not line.strip().strip(fence[0]):
                fence = None
            continue
        line = re.sub(r'(`+).*?\1', '', line)
        for pattern in (INLINE, REFERENCE):
            for match in pattern.finditer(line):
                yield number, match.group(1).strip('<>')


def main(paths):
    broken = 0
    for path in paths:
        with open(path, encoding='utf-8') as f:
            text = f.read()
        for number, target in links(text):
            if EXTERNAL.match(target) or target.startswith('#'):
                continue
            file_part = target.split('#', 1)[0]
            if not os.path.exists(os.path.join(os.path.dirname(path), file_part)):
                print(f'{path}:{number}: link to missing file: {target}')
                broken += 1
    return 1 if broken else 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
