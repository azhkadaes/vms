"""Source inventory helpers for check.py; no application, configuration or database access."""
from pathlib import Path
import re


def inventory_tables(files):
    tables = []
    for name, sql in files:
        assert '-- +migrate Up' in sql and '-- +migrate Down' in sql, 'Migration convention drift'
        up = sql.split('-- +migrate Down')[0]
        up = re.sub(r'/\*.*?\*/|--[^\n]*', '', up, flags=re.S)
        starts = list(re.finditer(r'\bCREATE\s+(?:(?:GLOBAL|LOCAL|TEMPORARY|TEMP|UNLOGGED)\s+)*TABLE\b', up, re.I))
        for start in starts:
            match = re.match(r'CREATE\s+(?:UNLOGGED\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-z_][a-z0-9_]*\.[a-z_][a-z0-9_]*)\s*\(', up[start.start():], re.I)
            assert match, 'Unsupported table syntax requires explicit review: ' + name
            tables.append({'table': match[1].lower(), 'migration': name})
    return tables


def validate_local_links(directory):
    count = 0
    for file in directory.glob('*.md'):
        raw = file.read_text(encoding='utf-8')
        assert not re.search(r'[\x00-\x08\x0b\x0c\x0e-\x1f]', raw), 'Control character in ' + file.name
        prose = re.sub(r'```.*?```', '', raw, flags=re.S)
        for href in re.findall(r'\[[^\]]*\]\(([^)]+)\)', prose):
            if href.startswith(('http:', 'https:', '#', 'mailto:')):
                continue
            target = href.split('#', 1)[0]
            if target:
                assert (file.parent / target).exists(), f'Broken local link in {file.name}: {target}'
                count += 1
    return count
