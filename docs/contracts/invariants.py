"""Cross-artifact contract checks; no application, configuration or database access."""
from pathlib import Path
import re


def interface_index(manifest):
    combined = {}
    for group in ['sourceInterfaces', 'proposedInterfaces']:
        for path, methods in manifest[group].items():
            key = (path.split('/')[1], path.split(':')[-1])
            entry = combined.setdefault(key, {})
            for method, signature in methods.items():
                assert method not in entry or entry[method] == signature, f'Conflicting interface: {key}.{method}'
                entry[method] = signature
    return combined


def method_signature(index, layer, symbol, seen=()):
    interface, method = symbol.split('.')
    key = (layer, interface)
    assert key not in seen, f'Interface embedding cycle: {symbol}'
    methods = index.get(key, {})
    if method in methods:
        return methods[method]
    for embedded in methods:
        if embedded.startswith('@'):
            try:
                return method_signature(index, layer, embedded[1:] + '.' + method, (*seen, key))
            except AssertionError:
                continue
    raise AssertionError(f'Unmapped {layer} interface method: {symbol}')


def validate_mappings(root, spec, manifest):
    index = interface_index(manifest)
    aliases = {'PublicCatalog': 'Catalogs'}
    query_dtos = {'BookingQuery', 'EmployeeQuery', 'ReportQuery', 'ReportExportQuery', 'AttachmentQuery'}
    transport_results = set()
    operation_ids = {entry['id'] for entry in manifest['operations']}
    for entry in manifest['operations']:
        signature = method_signature(index, 'usecase', entry['usecase'])
        for symbol in entry['repositories']:
            method_signature(index, 'repository', symbol)
        serializer = entry['serializer']
        if serializer is None:
            assert entry['id'] in {'deleteSession', 'staffAttachment'}, 'Missing projection mapping: ' + entry['id']
        else:
            projection = method_signature(index, 'serializer', serializer)
            dto = 'Session' if entry['id'] in transport_results else entry['responseDto'].removeprefix('[]')
            assert 'dto.' + dto in projection, f'Serializer/response DTO mismatch: {entry["id"]}'
        if entry['responseDto'] and entry['id'] not in transport_results | {'staffAttachment'}:
            assert 'dto.' + entry['responseDto'].removeprefix('[]') in signature, 'Usecase/response DTO mismatch: ' + entry['id']
        op = spec['paths'][entry['path']][entry['method'].lower()]
        request = entry['requestDto']
        if 'requestBody' in op:
            assert request and request not in query_dtos, 'Missing body DTO: ' + entry['id']
            for content in op['requestBody']['content'].values():
                assert content['schema'] == {'$ref': '#/components/schemas/' + aliases.get(request, request)}, 'Body schema/DTO drift: ' + entry['id']
        else:
            assert request is None or request in query_dtos, 'Unexpected body DTO: ' + entry['id']
        for status, response in op['responses'].items():
            if not status.startswith('2'):
                continue
            for media, content in response.get('content', {}).items():
                if 'json' not in media:
                    assert entry['responseDto'] == 'PrivateFile'
                    continue
                dto = entry['responseDto']
                schema = content['schema']
                if dto.startswith('[]'):
                    assert schema['type'] == 'array', 'Array DTO drift: ' + entry['id']
                    dto, schema = dto[2:], schema['items']
                assert schema == {'$ref': '#/components/schemas/' + aliases.get(dto, dto)}, 'Response schema/DTO drift: ' + entry['id']
        evidence = entry['evidence']
        assert evidence['consumers'], 'No consumer evidence: ' + entry['id']
        for path in [evidence['requirement'], evidence['decisions'], *evidence['consumers']]:
            assert (root / path).is_file(), 'Missing evidence: ' + path
        assert bool(evidence['source']) == entry['implemented'], 'Source status/evidence drift: ' + entry['id']
        if evidence['source']:
            assert (root / evidence['source']).is_file()
        assert op['x-vms-evidence'] == evidence
        assert {status for status in op['responses'] if not status.startswith('2')} == set(op['x-error-conditions']), 'Undocumented error condition: ' + entry['id']
    assert spec['x-contract-issues'] == manifest['contractIssues']
    for issue in manifest['contractIssues']:
        assert set(issue['operations']) <= operation_ids and issue['decisionNeeded'] and issue['blocks']
        assert all((root / path).is_file() for path in issue['evidence'])


def validate_correlations(query, body):
    view = query['view']
    defaults = {'list': 7, 'reception': 25, 'dashboard': 50}
    assert body['page'] == query.get('page', 1), 'Page echo mismatch'
    assert body['pageSize'] == query.get('pageSize', defaults[view]), 'Page-size echo mismatch'
    assert len(body['items']) <= body['pageSize']
    assert len(body['items']) <= body['total'] <= body['totalScoped']
    if view == 'dashboard':
        assert 'metrics' in body, 'Dashboard metrics missing'


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
