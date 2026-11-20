"""Offline contract gate for the 40-operation frontend-driven REST contract.

No server, database, containers or network. It keeps four artifacts in step:

- manifest.json: every operation of FRONTEND-DRIVEN-REST-API-2026-11-16.md (number, method,
  path, implemented/missing) plus the reviewed Go and migration inventory of the backend;
- source-baseline.json: OpenAPI shapes of the implemented operations, matching the server,
  including the error body and error-code catalogue;
- the Go source of the target backend: routes, handlers, interfaces, DTOs, error catalogue;
- the served backend docs/openapi.json, generated from the first two by `make swagger`.

Usage, from a backend checkout (the main one or a worktree):
  python ../docs/contracts/check.py --backend . [--runtime] [--consumers] [--strict]
  python ../docs/contracts/check.py --backend . --write-inventory   # after reviewing Go/SQL changes

--backend DIR        backend checkout to check; default ../backend next to docs/.
--contracts DIR      directory holding manifest.json and source-baseline.json; default this
                     one. A worktree drafts its contract updates in a copy and checks them here.
--runtime            run the contract capture tests and validate real Fiber responses.
--consumers          run the frontend parsers and request inventory (check-consumers.mjs).
--strict             exit 1 while any operation is missing or any known gap remains.
--write-inventory    rewrite manifest sourceInterfaces, sourceDTOs and migrationTables from DIR
                     before checking. Review the manifest diff: it is the unreviewed drift.
--output PATH        results file; default validation-results.json here, written only when
                     both directories are the defaults.
"""
from pathlib import Path
import argparse
import copy
import json
import os
import re
import subprocess
import sys
import tempfile

sys.dont_write_bytecode = True
from jsonschema import Draft202012Validator, FormatChecker, ValidationError, validators
from invariants import inventory_tables, validate_local_links

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[1]
DEFAULT_BACKEND = ROOT / 'backend'
OPENAPI_SCHEMA = ROOT / 'docs/planning-evidence/openapi-3.1-schema-2025-09-15.json'
METHODS = {'get', 'post', 'put', 'patch', 'delete'}
OPERATION_COUNT = 40


def read(path):
    return json.loads(Path(path).read_text(encoding='utf-8-sig'))


def write_json(path, value):
    Path(path).write_text(json.dumps(value, indent=2, ensure_ascii=False) + '\n', encoding='utf-8', newline='\n')


def utf8_limit(validator, maximum, instance, schema):
    if isinstance(instance, str) and len(instance.encode('utf-8')) > maximum:
        yield ValidationError('UTF-8 byte limit exceeded')


Validator = validators.extend(Draft202012Validator, {'x-max-utf8-bytes': utf8_limit})


def validate(spec, schema, value):
    Validator({'components': spec['components'], **schema}, format_checker=FormatChecker()).validate(value)


def run(args, cwd, env=None):
    result = subprocess.run(args, cwd=cwd, env=env, capture_output=True, text=True, encoding='utf-8', errors='replace')
    if result.returncode:
        raise AssertionError('Command failed: ' + ' '.join(map(str, args)) + '\n' + result.stdout[-4000:] + result.stderr[-4000:])
    return result.stdout


def assert_equal(actual, expected, message):
    if actual != expected:
        raise AssertionError(message)


def resolve(spec, value):
    while isinstance(value, dict) and '$ref' in value:
        target = spec
        for segment in value['$ref'][2:].split('/'):
            target = target[segment.replace('~1', '/').replace('~0', '~')]
        value = target
    return value


def operation_key(entry):
    return (entry['method'], entry['path'])


# ----------------------------------------------------------------------------- manifest

def contract_headings(manifest):
    """Operation headings of the human contract: `### NN. METHOD /v1/path`."""
    text = (HERE / manifest['contract']['operations']).read_text(encoding='utf-8')
    return [(int(number), method, path) for number, method, path in re.findall(r'^### (\d{2})\. (GET|POST|PUT|PATCH|DELETE) (/v1/\S+)$', text, re.M)]


BASE_KEYS = {'number', 'id', 'method', 'path', 'summary', 'ticket', 'status'}
IMPLEMENTED_KEYS = BASE_KEYS | {'handler', 'calls', 'usecase', 'requestDto', 'responseDto'}


def validate_manifest(manifest):
    for name in manifest['contract'].values():
        assert (HERE / name).is_file(), 'Contract reference missing: ' + name
    operations = manifest['operations']
    headings = contract_headings(manifest)
    assert_equal(len(headings), OPERATION_COUNT, f'The operation reference no longer has {OPERATION_COUNT} numbered operations; renumber deliberately.')
    assert_equal([(e['number'], e['method'], '/v1' + e['path']) for e in operations], headings, 'Manifest operations differ from the numbered operation reference (number, method or path).')
    assert_equal(len({e['id'] for e in operations}), len(operations), 'Duplicate operation ID in the manifest.')
    assert_equal(len({operation_key(e) for e in operations}), len(operations), 'Duplicate method and path in the manifest.')
    for entry in operations:
        assert re.fullmatch(r'[a-z][A-Za-z]+', entry['id']), 'Operation ID must be lowerCamel: ' + entry['id']
        assert entry['summary'].strip() and entry['ticket'].strip(), 'Operation summary and owning ticket required: ' + entry['id']
        if entry['status'] == 'implemented':
            assert_equal(set(entry), IMPLEMENTED_KEYS, 'Implemented operation needs exactly handler, calls, usecase, requestDto, responseDto: ' + entry['id'])
            assert entry['calls'], 'Implemented operation must name its handler delegation: ' + entry['id']
        else:
            assert_equal(entry['status'], 'missing', 'Operation status must be implemented or missing: ' + entry['id'])
            assert_equal(set(entry), BASE_KEYS, 'A missing operation carries no source mapping: ' + entry['id'])
    numbers = {e['number'] for e in operations}
    for call in manifest['frontendRequestCalls']:
        assert set(call['operations']) <= numbers, 'Frontend request call maps to an unknown operation: ' + call['call']
    return [e for e in operations if e['status'] == 'implemented']


# ----------------------------------------------------------------------------- baseline

def validate_baseline(baseline, manifest):
    spec = baseline['spec']
    Draft202012Validator(read(OPENAPI_SCHEMA)).validate(spec)
    for schema in spec['components']['schemas'].values():
        Draft202012Validator.check_schema(schema)

    def refs(value):
        if isinstance(value, dict):
            if '$ref' in value:
                assert value['$ref'].startswith('#/'), 'External ref requires an explicit offline dependency.'
                resolve(spec, {'$ref': value['$ref']})
            for item in value.values():
                refs(item)
        elif isinstance(value, list):
            for item in value:
                refs(item)
    refs(spec)

    implemented = {operation_key(e): e for e in manifest['operations'] if e['status'] == 'implemented'}
    documented = {(method.upper(), path): op for path, item in spec['paths'].items() for method, op in item.items() if method in METHODS}
    assert_equal(set(documented), set(implemented), 'Source baseline operations differ from the operations the manifest marks implemented.')
    catalogue = spec['x-error-catalogue']
    error = spec['components']['schemas']['ApiError']
    assert_equal(error['properties']['code']['enum'], list(catalogue), 'ApiError.code enum differs from x-error-catalogue.')
    for code, entry in catalogue.items():
        assert f"Error{entry['status']}" in spec['components']['responses'], 'Catalogue status without an error response: ' + code
    for name, response in spec['components']['responses'].items():
        status = int(name.removeprefix('Error'))
        content = response['content']['application/json']
        assert_equal(content['schema'], {'$ref': '#/components/schemas/ApiError'}, 'Error responses use the ApiError body: ' + name)
        codes = {code for code, entry in catalogue.items() if entry['status'] == status}
        assert codes <= set(content['examples']), f'Every catalogue code of {name} needs an example named after it: {sorted(codes - set(content["examples"]))}'
        for example_name, example in content['examples'].items():
            value = example['value']
            validate(spec, content['schema'], value)
            assert value['code'] in codes, f'{name} example {example_name} uses a code of another status'
            messages = [catalogue[value['code']]['message']] if example_name in codes else catalogue[value['code']].get('variants', [])
            assert value['message'] in messages, f'{name} example {example_name} must use the catalogue message (or, under another name, a documented variant)'
    for key, op in documented.items():
        entry = implemented[key]
        assert_equal((op['operationId'], op['x-operation-number']), (entry['id'], entry['number']), f'Baseline operation ID or number drift: {key}')
        assert op.get('x-reject-unknown-query') is True, f'Unknown query keys must be rejected: {key}'
        errors = {status for status in op['responses'] if not status.startswith('2')}
        assert_equal(set(op['x-error-conditions']), errors, f'Every error status needs its condition: {key}')
        for status in errors:
            assert_equal(op['responses'][status], {'$ref': f'#/components/responses/Error{status}'}, f'Error status must use the shared error response: {key} {status}')
        for content in op.get('requestBody', {}).get('content', {}).values():
            if 'example' in content:
                validate(spec, content['schema'], content['example'])
        for status, response in op['responses'].items():
            for media, content in resolve(spec, response).get('content', {}).items():
                if 'example' in content and 'json' in media:
                    validate(spec, content['schema'], content['example'])
    return documented


# ----------------------------------------------------------------------------- Go source

def inspect_routes(go):
    assert not go.get('unresolvedRoutes'), 'Dynamic/unresolved route registration requires an inspector change and review: ' + '; '.join(go.get('unresolvedRoutes') or [])
    routes = {}
    for route in go['routes']:
        key = (route['Method'], re.sub(r':([A-Za-z]+)', r'{\1}', route['Path'].removeprefix('/v1')))
        assert key not in routes, f'Duplicate route: {key}'
        routes[key] = route
    return routes


def go_base(type_name):
    """Strip pointer, slice and dto.Optional wrappers: '[]*Officer' -> 'Officer'."""
    while True:
        stripped = type_name.lstrip('*')
        if stripped.startswith('[]'):
            stripped = stripped[2:]
        match = re.fullmatch(r'(?:dto\.)?Optional\[(.+)\]', stripped)
        if match:
            stripped = match[1]
        if stripped == type_name:
            return type_name
        type_name = stripped


def json_type(schema):
    if 'type' in schema:
        kind = schema['type']
        if isinstance(kind, list) and len(kind) == 2 and 'null' in kind:
            return next(value for value in kind if value != 'null')
        return kind
    for key in ('const', 'enum'):
        if key in schema:
            value = schema[key][0] if key == 'enum' else schema[key]
            return {bool: 'boolean', int: 'integer', float: 'number', str: 'string'}[type(value)]
    return 'object' if 'properties' in schema else None


def check_dto(spec, dtos, dto, schema, trail):
    """Go DTO fields against a JSON schema, recursing into nested pkg/dto types."""
    schema = resolve(spec, schema)
    if dto.startswith('[]'):
        assert_equal(json_type(schema), 'array', f'Array type drift: {trail}')
        return check_dto(spec, dtos, dto[2:], schema['items'], trail + '[]')
    fields = {f['Name']: f for f in dtos[dto]}
    properties = schema.get('properties', {})
    required = set(schema.get('required', []))
    assert required <= set(fields), f'Required wire field missing in Go DTO {trail}: {sorted(required - set(fields))}'
    assert set(fields) <= set(properties), f'Go DTO field undocumented in the baseline {trail}: {sorted(set(fields) - set(properties))}'
    for name, field in fields.items():
        assert not (name in required and field['Optional']), f'Required field marked omitempty: {trail}.{name}'
        target = resolve(spec, properties[name])
        go_type = field['Type'].lstrip('*')
        kind = json_type(target)
        if go_type.startswith('[]'):
            assert_equal(kind, 'array', f'Array type drift: {trail}.{name}')
            target, go_type = resolve(spec, target['items']), go_type[2:].lstrip('*')
            kind = json_type(target)
        base = go_base(go_type)
        if base == 'ReportFilterValues':
            variants = [resolve(spec, item) for item in target.get('oneOf', [])]
            assert_equal([json_type(item) for item in variants], ['string', 'array'], f'Report filter union drift: {trail}.{name}')
            assert_equal(json_type(resolve(spec, variants[1]['items'])), 'string', f'Report filter array item drift: {trail}.{name}')
        elif base in dtos:
            assert_equal(kind, 'object', f'Nested DTO type drift: {trail}.{name}')
            check_dto(spec, dtos, base, target, f'{trail}.{name}')
        elif go_type.startswith('map['):
            assert_equal(kind, 'object', f'Map type drift: {trail}.{name}')
        else:
            allowed = {'string': {'string'}, 'time.Time': {'string'}, 'bool': {'boolean'}, 'float64': {'number'}}.get(base, {'integer'} if re.fullmatch(r'u?int(8|16|32|64)?', base) else set())
            assert kind in allowed, f'Primitive type drift: {trail}.{name} is {field["Type"]} but documented {kind}'
            if base == 'time.Time':
                assert_equal(target.get('format'), 'date-time', f'time.Time must be a documented date-time: {trail}.{name}')


def body_schema(spec, op, request):
    if request:
        content = op.get('requestBody', {}).get('content', {})
        return content.get('application/json', {}).get('schema')
    for status, response in op['responses'].items():
        if status.startswith('2'):
            content = resolve(spec, response).get('content', {})
            if 'application/json' in content:
                return content['application/json']['schema']
    return None


def expected_served(spec):
    """What `make swagger` serves, apart from its generated info.description."""
    return {**spec, 'info': {'title': 'VMS API', 'version': spec['info']['version']}}


def validate_source(manifest, spec, go, served):
    routes = inspect_routes(go)
    implemented = {operation_key(e): e for e in manifest['operations'] if e['status'] == 'implemented'}
    assert_equal(set(routes), set(implemented), 'Source routes differ from the implemented manifest operations: ' + ', '.join(f'{m} /v1{p}' for m, p in sorted(set(routes) ^ set(implemented))))
    dtos = {key.split(':')[-1]: fields for key, fields in go['dtos'].items()}
    for key, entry in implemented.items():
        method = entry['handler'].rsplit('.', 1)[-1]
        assert_equal(routes[key]['Handler'].rsplit('.', 1)[-1], method, f'Route registers a different handler: {key}')
        assert entry['handler'] in go['functions'], 'Handler function missing: ' + entry['handler']
        for call in entry['calls']:
            assert call in go['calls'][entry['handler']], f'Handler no longer calls {call}: ' + entry['handler']
        interface, usecase_method = entry['usecase'].split('.')
        assert any(name.endswith(':' + interface) and usecase_method in methods for name, methods in go['interfaces'].items()), 'Usecase method missing: ' + entry['usecase']
        op = spec['paths'][key[1]][key[0].lower()]
        for request, dto in [(True, entry['requestDto']), (False, entry['responseDto'])]:
            schema = body_schema(spec, op, request)
            if dto is None:
                assert schema is None or resolve(spec, schema) == spec['components']['schemas']['Empty'], f'Undeclared {"request" if request else "response"} DTO: {entry["id"]}'
                continue
            assert dto.removeprefix('[]') in dtos and schema is not None, f'DTO or body schema missing: {entry["id"]} {dto}'
            check_dto(spec, dtos, dto, schema, f'{entry["id"]}:{dto}')
    assert_equal(go['interfaces'], manifest['sourceInterfaces'], 'Interface inventory drift (added, removed or changed signature); review, then --write-inventory.')
    assert_equal(go['dtos'], manifest['sourceDTOs'], 'DTO field/type/omitempty drift; review, then --write-inventory.')
    catalogue = spec['x-error-catalogue']
    assert_equal(go['errorCatalogue'], {code: entry['message'] for code, entry in catalogue.items()}, 'Error-code catalogue drift between pkg/helper/failure/catalogue.go and x-error-catalogue.')
    for status, code in go['statusCodes'].items():
        assert code in catalogue and catalogue[code]['status'] == int(status), f'Default code for status {status} disagrees with the catalogue: {code}'
    served_info = served.get('info', {})
    total = len(manifest['operations'])
    assert f'{len(implemented)} of {total}' in served_info.get('description', ''), 'Served OpenAPI states a stale implemented count; run make swagger.'
    actual = {**served, 'info': {k: v for k, v in served_info.items() if k != 'description'}}
    assert_equal(actual, expected_served(spec), 'Served docs/openapi.json is stale; run make swagger.')
    return len(routes)


def go_inventory(backend, env):
    return json.loads(run(['go', 'run', str(HERE / 'inspect-go.go'), str(backend)], cwd=ROOT, env=env))


def migration_files(backend):
    return [(file.name, file.read_text(encoding='utf-8')) for file in sorted((backend / 'migration/files').glob('*.sql'))]


# ----------------------------------------------------------------------------- runtime

def validate_runtime(manifest, baseline, backend, env):
    spec = baseline['spec']
    catalogue = spec['x-error-catalogue']
    by_number = {e['number']: e for e in manifest['operations']}
    records = []
    with tempfile.TemporaryDirectory(prefix='vms-contract-') as temp:
        for index, capture in enumerate(manifest['runtimeCaptures']):
            target = Path(temp) / f'{index}.json'
            run(['go', 'test', capture['package'], '-run', '^' + capture['test'] + '$', '-count=1'], cwd=backend, env={**env, 'VMS_CONTRACT_CAPTURE': str(target)})
            records += read(target)
    expectations = baseline['runtimeExpectations']
    names = [record['name'] for record in records]
    assert_equal(len(set(names)), len(names), 'Duplicate contract capture names.')
    assert_equal(set(names), set(expectations), 'Contract captures and runtimeExpectations differ: ' + ', '.join(sorted(set(names) ^ set(expectations))))
    known, accepted = {}, 0
    for record in records:
        expected = expectations[record['name']]
        entry = by_number[expected['operation']]
        path_pattern = '/v1' + '/'.join(
            '[^/]+' if re.fullmatch(r'\{[^/{}]+\}', part) else re.escape(part)
            for part in entry['path'].split('/'))
        assert record['method'] == entry['method'] and re.fullmatch(path_pattern, record['path']), (
            'Capture does not exercise its expected operation: ' + record['name'])
        assert_equal(entry['status'], 'implemented', 'Capture exercises an operation not marked implemented: ' + record['name'])
        op = spec['paths'][entry['path']][entry['method'].lower()]
        status = str(record['status'])
        issues = set()
        if record['status'] != expected['status']:
            issues.add('status')
        response = op['responses'].get(status)
        if response is None:
            issues.add('undocumented-status')
            response = spec['components']['responses'].get('Error' + status)
            assert response is not None, 'Unrecognized HTTP status: ' + status
        content = resolve(spec, response).get('content')
        headers = record['headers']
        if not content:
            assert_equal(record['bytes'], 0, 'A response without documented content must have no body: ' + record['name'])
        elif 'application/json' in content:
            assert headers['Content-Type'].startswith('application/json'), 'JSON response without a JSON content type: ' + record['name']
            try:
                validate(spec, content['application/json']['schema'], record['body'])
            except ValidationError:
                issues.add('schema')
        else:
            assert any(headers['Content-Type'].startswith(media) for media in content), 'Undocumented content type: ' + record['name']
        body = record['body'] if isinstance(record['body'], dict) else {}
        if record['status'] >= 400:
            entry_code = catalogue.get(body.get('code'))
            if entry_code is None or entry_code['status'] != record['status']:
                issues.add('code')
            elif body.get('message') not in [entry_code['message'], *entry_code.get('variants', [])]:
                issues.add('message')
            assert body.get('requestId') == headers['X-Request-ID'], 'Error requestId must equal X-Request-ID: ' + record['name']
            assert 'no-store' in headers['Cache-Control'], 'Error responses must not be cached: ' + record['name']
        if 'code' in expected and body.get('code') != expected['code']:
            issues.add('code')
        if not set(expected.get('fieldPaths', [])) <= set(body.get('fieldErrors', {})):
            issues.add('fieldErrors')
        if expected.get('fieldErrors') == 'absent' and 'fieldErrors' in body:
            issues.add('fieldErrors')
        if entry['path'].startswith('/staff'):
            assert 'no-store' in headers['Cache-Control'], 'Private cache policy drift: ' + record['name']
        assert headers['X-Content-Type-Options'] == 'nosniff', 'nosniff missing: ' + record['name']
        assert headers['X-Request-ID'], 'X-Request-ID missing: ' + record['name']
        if issues:
            known[record['name']] = {'gap': baseline['runtimeKnownGaps'].get(record['name'], {}).get('gap'), 'issues': sorted(issues)}
        else:
            accepted += 1
    assert_equal(known, baseline['runtimeKnownGaps'], 'Runtime drift from the source baseline: ' + json.dumps(known))
    succeeded = {expected['operation'] for expected in expectations.values() if 200 <= expected['status'] < 300}
    uncovered = sorted(e['number'] for e in manifest['operations'] if e['status'] == 'implemented' and e['number'] not in succeeded)
    assert not uncovered, f'Implemented operations without a successful contract capture: {uncovered}'
    return {'executed': True, 'captures': len(records), 'accepted': accepted, 'knownGaps': known, 'operationsCovered': sorted({expectations[n]['operation'] for n in names})}


# ----------------------------------------------------------------------------- main

def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('--backend', type=Path, default=DEFAULT_BACKEND)
    parser.add_argument('--contracts', type=Path, default=HERE)
    parser.add_argument('--runtime', action='store_true')
    parser.add_argument('--consumers', action='store_true')
    parser.add_argument('--strict', action='store_true')
    parser.add_argument('--write-inventory', action='store_true')
    parser.add_argument('--output', type=Path)
    args = parser.parse_args()
    backend = args.backend.resolve()
    assert (backend / 'go.mod').is_file() and (backend / 'migration/files').is_dir(), f'Not a backend checkout: {backend}'
    contracts = args.contracts.resolve()
    defaults = backend == DEFAULT_BACKEND.resolve() and contracts == HERE
    output = args.output or (HERE / 'validation-results.json' if defaults else None)
    env = os.environ.copy()
    env.setdefault('GOCACHE', str(DEFAULT_BACKEND / '.runtime/go-cache'))
    env.setdefault('GOTMPDIR', str(DEFAULT_BACKEND / '.runtime/go-tmp'))
    Path(env['GOCACHE']).mkdir(parents=True, exist_ok=True)
    Path(env['GOTMPDIR']).mkdir(parents=True, exist_ok=True)

    manifest_path = contracts / 'manifest.json'
    baseline_path = contracts / 'source-baseline.json'
    manifest, baseline = read(manifest_path), read(baseline_path)
    assert_equal(manifest['sourceBaseline'], 'source-baseline.json', 'Unsupported source baseline file.')
    go = go_inventory(backend, env)
    tables = inventory_tables(migration_files(backend))
    if args.write_inventory:
        updated = {'sourceInterfaces': dict(sorted(go['interfaces'].items())), 'sourceDTOs': dict(sorted(go['dtos'].items())), 'migrationTables': tables}
        changed = [key for key, value in updated.items() if manifest[key] != value]
        manifest.update(updated)
        write_json(manifest_path, manifest)
        print('Inventory rewritten from ' + str(backend) + ': ' + (', '.join(changed) or 'no change') + '. Review the manifest diff.')
    checks = []

    implemented = validate_manifest(manifest)
    checks.append(f'{OPERATION_COUNT} manifest operations match the numbered operation reference; {len(implemented)} implemented')
    documented = validate_baseline(baseline, manifest)
    checks.append(f'Source baseline: OpenAPI 3.1 meta-schema, local refs, {len(documented)} implemented operations, examples and {len(baseline["spec"]["x-error-catalogue"])}-code error catalogue')
    served = read(backend / 'docs/openapi.json')
    registered = validate_source(manifest, baseline['spec'], go, served)
    checks.append(f'Go AST: {registered} routes, handlers and delegation, {len(go["interfaces"])} interfaces, {len(go["dtos"])} DTOs against schemas, error catalogue, served OpenAPI')
    assert_equal(tables, manifest['migrationTables'], 'Unreviewed table addition, removal or rename; review, then --write-inventory.')
    checks.append(f'{len(tables)} migration tables and Up/Down conventions; no DDL executed')
    link_count = validate_local_links(HERE)
    checks.append(f'{link_count} local documentation links in docs/contracts')

    # Deliberately perturb the inputs to show each guard rejects drift.
    mutations = []

    def rejects(name, guard):
        try:
            guard()
        except (AssertionError, ValidationError, KeyError):
            mutations.append(name)
        else:
            raise AssertionError('Guard accepted deliberate drift: ' + name)

    def source_with(mutate):
        changed = copy.deepcopy(go)
        mutate(changed)
        return lambda: validate_source(manifest, baseline['spec'], changed, served)

    first_interface = next(iter(go['interfaces']))
    first_dto = next(key for key, fields in go['dtos'].items() if fields)
    rejects('unregistered route', source_with(lambda g: g['routes'].append({'Method': 'GET', 'Path': '/v1/staff/undocumented', 'Handler': 'h.Unknown', 'File': 'fake.go'})))
    rejects('removed route', source_with(lambda g: g['routes'].pop()))
    rejects('changed interface', source_with(lambda g: g['interfaces'][first_interface].__setitem__('Unexpected', 'func()')))
    rejects('new interface', source_with(lambda g: g['interfaces'].__setitem__('internal/usecase/x/x_usecase.go:XUsecase', {})))
    rejects('renamed DTO field', source_with(lambda g: g['dtos'][first_dto][0].__setitem__('Name', 'unexpected')))
    rejects('changed error message', source_with(lambda g: g['errorCatalogue'].__setitem__('not_found', 'changed')))
    rejects('stale served OpenAPI', lambda: validate_source(manifest, baseline['spec'], go, {**served, 'paths': {}}))
    renumbered = copy.deepcopy(manifest)
    renumbered['operations'][0]['number'], renumbered['operations'][1]['number'] = 2, 1
    rejects('renumbered operation', lambda: validate_manifest(renumbered))
    unmarked = copy.deepcopy(manifest)
    next(e for e in unmarked['operations'] if e['status'] == 'implemented')['status'] = 'missing'
    rejects('implemented operation marked missing', lambda: validate_baseline(baseline, unmarked))
    hidden_table = migration_files(backend) + [('synthetic.sql', '-- +migrate Up\nCREATE TABLE IF NOT EXISTS support.unreviewed (id integer);\n-- +migrate Down\n')]
    assert inventory_tables(hidden_table) != tables, 'IF NOT EXISTS table escaped inventory'
    mutations.append('unreviewed IF NOT EXISTS table')
    checks.append(f'{len(mutations)} deliberate drift mutations rejected')

    runtime = {'executed': False}
    if args.runtime:
        runtime = validate_runtime(manifest, baseline, backend, env)
        checks.append(f"{runtime['captures']} real Fiber responses validated against the baseline; every implemented operation has a successful capture")
    consumers = {'executed': False}
    if args.consumers:
        result = json.loads(run(['node', '--experimental-strip-types', str(HERE / 'check-consumers.mjs'), str(baseline_path)], cwd=ROOT))
        assert_equal(result['knownGaps'], sorted(manifest['consumerKnownGaps']), 'New or stale consumer gap: ' + json.dumps(result['knownGaps']))
        assert_equal(result['requestCalls'], [{'file': c['file'], 'call': c['call']} for c in manifest['frontendRequestCalls']], 'Frontend request calls changed; map them to operations in manifest frontendRequestCalls.')
        consumers = {'executed': True, 'checks': len(result['cases']), 'knownGaps': result['knownGaps'], 'requestCalls': len(result['requestCalls'])}
        checks.append(f"{len(result['cases'])} frontend parser checks and {len(result['requestCalls'])} mapped request call sites")

    missing = [f"{e['number']:02d} {e['id']}" for e in manifest['operations'] if e['status'] == 'missing']
    gaps = sorted(baseline['runtimeKnownGaps']) + sorted(manifest['consumerKnownGaps'])
    incomplete = bool(missing or gaps)
    status = 'passed-with-missing-operations' if missing else 'passed-with-known-gaps' if gaps else 'conformance-passed'
    result = {
        'status': status,
        'strictConformance': 'failed' if incomplete else 'passed',
        'strictModeRequested': args.strict,
        'reviewedAt': manifest['reviewedAt'],
        'backend': os.path.relpath(backend, ROOT).replace('\\', '/'),
        'contracts': 'docs/contracts' if contracts == HERE else 'draft',
        'operations': len(manifest['operations']),
        'implemented': len(implemented),
        'sourceRoutes': registered,
        'missingOperations': missing,
        'knownGaps': gaps,
        'checks': checks,
        'negativeDriftChecks': mutations,
        'runtime': runtime,
        'consumers': consumers,
        'limitations': ['No live database, migrations, containers, server startup or deployed API tested.', 'Runtime captures use injected usecases: they prove HTTP decoding, status, error body and serialization, not authorization or PostgreSQL behaviour.', 'Missing operations are tracked by number only; their shapes enter the baseline with the slice that implements them.'],
    }
    if output:
        output.parent.mkdir(parents=True, exist_ok=True)
        write_json(output, result)
    print(json.dumps({k: result[k] for k in ['status', 'backend', 'contracts', 'operations', 'implemented', 'sourceRoutes']}))
    print(f"Missing operations: {len(missing)}; known gaps: {len(gaps)}; HTTP captures: {runtime.get('captures', 'not run')}; consumer checks: {consumers.get('checks', 'not run')}.")
    if args.strict and incomplete:
        raise SystemExit(1)


if __name__ == '__main__':
    try:
        main()
    except (AssertionError, ValidationError, KeyError) as exc:
        print('CONTRACT CHECK FAILED: ' + str(exc), file=sys.stderr)
        raise SystemExit(1)
