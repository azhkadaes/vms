"""Offline contract regression gate. No server, database, containers or network.

Default: no unreviewed drift. --strict: also require zero recorded target gaps.
--runtime: real Fiber HTTP captures with injected usecases, not real DB authority.
--consumers: actual frontend validators/HTTP client plus TypeScript request ASTs.
"""
from pathlib import Path
import argparse
import copy
import hashlib
import json
import os
import re
import subprocess
import sys
import tempfile

sys.dont_write_bytecode = True
from jsonschema import Draft202012Validator, FormatChecker, ValidationError, validators
from render import render
from invariants import inventory_tables, validate_correlations, validate_local_links, validate_mappings

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[1]
SPEC_PATH = ROOT / 'docs/BACKEND-OPENAPI-2026-11-12.json'
METHODS = {'get', 'post', 'put', 'patch', 'delete'}


def read(path):
    return json.loads(path.read_text(encoding='utf-8-sig'))


def utf8_limit(validator, maximum, instance, schema):
    if isinstance(instance, str) and len(instance.encode('utf-8')) > maximum:
        yield ValidationError('UTF-8 byte limit exceeded')


Validator = validators.extend(Draft202012Validator, {'x-max-utf8-bytes': utf8_limit})


def validate(spec, schema, value):
    Validator({'components': spec['components'], **schema}, format_checker=FormatChecker()).validate(value)


def run(args, cwd=ROOT, env=None):
    result = subprocess.run(args, cwd=cwd, env=env, capture_output=True, text=True, encoding='utf-8', errors='replace')
    if result.returncode:
        raise AssertionError('Command failed: ' + ' '.join(map(str, args)) + '\n' + result.stdout[-4000:] + result.stderr[-4000:])
    return result.stdout


def assert_equal(actual, expected, message):
    if actual != expected:
        raise AssertionError(message)


def inspect_routes(go):
    assert not go.get('unresolvedRoutes'), 'Dynamic/unresolved route registration requires an explicit inspector adapter and review.'
    routes = {}
    for route in go['routes']:
        key = (route['Method'], re.sub(r':([A-Za-z]+)', r'{\1}', route['Path'][3:]))
        assert key not in routes, f'Duplicate route: {key}'
        routes[key] = route
    return routes


def validate_source_baseline(spec, manifest, go, served):
    ops = {(e['method'], e['path']): e for e in manifest['operations']}
    implemented = {k: v for k, v in ops.items() if v['implemented']}
    actual = inspect_routes(go)
    assert_equal(set(actual), set(implemented), 'Source routes differ from implemented manifest (new, missing or unsupported dynamic registration).')
    for key, entry in implemented.items():
        assert_equal(actual[key]['Handler'], 'h.' + entry['handler'], f'Handler route drift: {key}')
        handler = f"internal/handler/{entry['module']}_handler.go:{entry['module']}Handler.{entry['handler']}"
        assert handler in go['functions'], f'Handler missing: {handler}'
        method = entry['usecase'].split('.')[-1]
        if entry['id'] != 'getSession':
            assert 'h.usecase.' + method in go['calls'][handler], f'Handler no longer delegates to expected interface: {handler}'
        else:
            assert 'restmiddleware.Session' in go['calls'][handler], 'Current Session handler no longer uses authenticated middleware projection.'
        for symbol in entry['repositories']:
            interface, method = symbol.split('.')
            methods = [methods for name, methods in go['interfaces'].items() if name.endswith(':' + interface)]
            found = any(method in methods for methods in methods)
            if not found and interface == 'SessionRepository':
                found = any(method in methods for name, methods in go['interfaces'].items() if name.endswith(':AuthRepository'))
            assert found, f'Repository operation disappeared: {symbol}'
    for key, methods in manifest['sourceInterfaces'].items():
        assert_equal(go['interfaces'].get(key), methods, 'Interface signature drift: ' + key)
    assert_equal(go['dtos'], manifest['sourceDTOs'], 'DTO field/type/omitempty drift.')
    projected = {(method.upper(), path): operation for path, item in served['paths'].items() for method, operation in item.items() if method in METHODS}
    assert_equal(set(projected), set(implemented), 'Served OpenAPI does not match source-registered operations; regenerate via vms-dev swagger.')
    for key, operation in projected.items():
        assert_equal(operation, spec['paths'][key[1]][key[0].lower()], f'Served operation drift: {key}')
    assert_equal(served['components'], spec['components'], 'Served schemas/security schemes are stale.')
    assert_equal(served['info']['version'], spec['info']['version'], 'Served contract version is stale.')
    for symbol, module in [('AuthSerializer.ToSession', 'auth'), ('CatalogSerializer.ToPublicCatalog', 'catalog')]:
        method = symbol.split('.')[-1]
        assert any('u.serializer.' + method in calls for function, calls in go['calls'].items() if function.startswith('internal/usecase/' + module + '/implementation/')), 'Inspected serializer delegation disappeared: ' + symbol
    # Check Go DTO primitive types and nested wire projections against OpenAPI.
    dto_map = {'CatalogOfficer': 'Officer', 'CatalogRoom': 'Room', 'PublishedNotice': None, 'PublicCatalog': 'Catalogs', 'Session': 'Session', 'SessionTiming': 'SessionTiming', 'TokenResult': 'TokenResult', 'Login': 'Login', 'PasswordChange': 'PasswordChange', 'RoleSelect': 'RoleSelect', 'Refresh': 'Refresh'}
    schema_names = {k.split(':')[-1]: v for k, v in go['dtos'].items()}
    def resolved(s):
        return resolved(spec['components']['schemas'][s['$ref'].split('/')[-1]]) if '$ref' in s else s
    for dto_name, schema_name in dto_map.items():
        if schema_name is None:
            target = resolved(spec['components']['schemas']['Catalogs']['properties']['privacyNotice'])
        else:
            target = spec['components']['schemas'][schema_name]
        fields = {f['Name']: f for f in schema_names[dto_name]}
        assert set(target.get('required', [])) <= set(fields), f'Required wire field missing in Go: {dto_name}'
        assert set(fields) <= set(target['properties']), f'Undocumented Go field: {dto_name}'
        for name, field in fields.items():
            assert not (name in target.get('required', []) and field['Optional']), f'Required field omitempty: {dto_name}.{name}'
            s = resolved(target['properties'][name]); got = field['Type']
            typ = s.get('type')
            if typ == 'array':
                assert got.startswith('[]'), f'Array type drift: {dto_name}.{name}'
            elif typ in ['integer', 'number']:
                assert got in ['int', 'int32', 'int64', 'float64'], f'Numeric type drift: {dto_name}.{name}'
            elif typ == 'string':
                assert got in ['string', 'time.Time'], f'String type drift: {dto_name}.{name}'
    return len(actual)


def validate_source(spec, manifest, go, served):
    baseline = read(HERE / manifest['sourceBaseline'])
    source_manifest = {**manifest, 'operations': baseline['operations']}
    validate_source_baseline(baseline['spec'], source_manifest, go, served)
    actual = inspect_routes(go)
    implemented = [entry for entry in manifest['operations'] if entry['implemented']]
    for entry in implemented:
        key = (entry['method'], entry['path'])
        assert key in actual, 'Target marked implemented without a source route: ' + entry['id']
        target = spec['paths'][entry['path']][entry['method'].lower()]
        previous = baseline['spec']['paths'][entry['path']][entry['method'].lower()]
        for field in ['parameters', 'security', 'requestBody', 'responses']:
            assert_equal(target.get(field), previous.get(field), 'Target transport differs from current-source baseline: ' + entry['id'])
    return len(implemented)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--runtime', action='store_true')
    parser.add_argument('--consumers', action='store_true')
    parser.add_argument('--strict', action='store_true')
    parser.add_argument('--output', type=Path, default=HERE / 'validation-results.json')
    args = parser.parse_args()
    spec, manifest, examples = read(SPEC_PATH), read(HERE / 'manifest.json'), read(HERE / 'examples.json')
    checks = []
    official = read(ROOT / 'docs/planning-evidence/openapi-3.1-schema-2025-09-15.json')
    Draft202012Validator(official).validate(spec)
    for schema in spec['components']['schemas'].values():
        Draft202012Validator.check_schema(schema)
    def refs(value):
        if isinstance(value, dict):
            if '$ref' in value:
                assert value['$ref'].startswith('#/'), 'External ref requires explicit offline dependency.'
                target = spec
                for segment in value['$ref'][2:].split('/'):
                    target = target[segment.replace('~1', '/').replace('~0', '~')]
            for item in value.values(): refs(item)
        elif isinstance(value, list):
            for item in value: refs(item)
    refs(spec)
    checks.append('OpenAPI meta-schema, component schemas and all local refs')
    actual_ops = {(method.upper(), path): op for path, item in spec['paths'].items() for method, op in item.items() if method in METHODS}
    mapped = {(e['method'], e['path']): e for e in manifest['operations']}
    assert len(actual_ops) == len(mapped) == 29
    assert_equal(set(actual_ops), set(mapped), 'OpenAPI operation mapping differs.')
    assert len({op['operationId'] for op in actual_ops.values()}) == 29
    assert sorted(op['x-inventory-number'] for op in actual_ops.values()) == list(range(1, 30)), 'Phase 1 inventory numbers drifted'
    assert spec['x-phase1'] == manifest['phase1'], 'Phase 1 policy drift'
    assert set(spec['components']['securitySchemes']) == {'BearerAccess'}, 'Deferred auth-cookie transport leaked into Phase 1'
    assert not {'Session', 'SessionTiming', 'TokenResult', 'Refresh', 'RoleSelect', 'PasswordChange'} & set(spec['components']['schemas']), 'Deferred lifecycle DTO leaked into Phase 1'
    assert all('/session' not in path for path in spec['paths']), 'Session endpoint leaked into Phase 1'
    feature_ids = {f['id'] for f in manifest['features']}
    gaps = {g['id'] for g in manifest['gaps']}
    for key, op in actual_ops.items():
        entry = mapped[key]
        assert_equal(op['operationId'], entry['id'], 'Operation ID drift')
        assert entry.get('frontendSummary', '').strip(), 'Endpoint is missing its frontend control summary: ' + entry['id']
        assert op.get('x-reject-unknown-query') and op.get('x-reject-duplicate-query-keys')
        assert set(entry['features']) <= feature_ids and entry['features'], 'Unmapped endpoint consumer'
        assert set(entry['gaps']) <= gaps
        assert_equal(op['x-vms-resource-authorization'], entry['authorization'], 'Authorization docs drift')
        assert_equal(op['x-vms-business-rules'], entry['rules'], 'Business-rule docs drift')
        for field, value in op['x-vms-interface'].items():
            assert_equal(entry[field], value, 'Interface mapping drift: ' + entry['id'])
        for response in op['responses'].values():
            for media, content in response.get('content', {}).items():
                if 'example' in content and 'json' in media:
                    validate(spec, content['schema'], content['example'])
        for media, content in op.get('requestBody', {}).get('content', {}).items():
            validate(spec, content['schema'], content['example'])
            validate(spec, content['schema'], examples['operations'][entry['id']]['request'])
        for status, value in examples['operations'][entry['id']]['success'].items():
            for media, content in op['responses'][status].get('content', {}).items():
                if 'json' in media:
                    validate(spec, content['schema'], value)
    for name, value in examples['schemas'].items():
        validate(spec, {'$ref': '#/components/schemas/' + name}, value)
    for role, permissions in manifest['phase1']['rolePermissions'].items():
        user = {**examples['schemas']['CurrentUser'], 'role': role, 'permissions': permissions, 'assignedOfficerIds': ['1'] if role == 'PA' else []}
        validate(spec, {'$ref': '#/components/schemas/CurrentUser'}, user)
        denied = next(key for key in spec['components']['schemas']['Permission']['enum'] if key not in permissions)
        invalid = {**user, 'permissions': [*permissions, denied]}
        try: validate(spec, {'$ref': '#/components/schemas/CurrentUser'}, invalid)
        except ValidationError: pass
        else: raise AssertionError('Fixed permission map accepted extra grant: ' + role)
    checks.append('29 Phase 1 operations, authorization/rules, auth-cookie/session exclusion and all request/response/schema examples')
    validate_mappings(ROOT, spec, manifest)
    checks.append('All Phase 1 usecase, repository and serializer mappings resolve; DTO/schema cardinality and evidence agree')
    for case in examples['queryCases']:
        op = next(op for op in actual_ops.values() if op['operationId'] == case['operation'])
        accepted = Validator({'components': spec['components'], **op['x-query-schema']}, format_checker=FormatChecker()).is_valid(case['query'])
        assert accepted == case['valid'], 'Query fixture: ' + case['name']
    for case in examples['correlationCases']:
        validate(spec, {'$ref': '#/components/schemas/ReadModel'}, case['body'])
        try:
            validate_correlations(case['query'], case['body'])
            accepted = True
        except AssertionError:
            accepted = False
        assert accepted == case['valid'], 'Response correlation fixture: ' + case['name']
    checks.append(f"{len(examples['queryCases'])} query variants and {len(examples['correlationCases'])} response-correlation fixtures")
    for feature in manifest['features']:
        assert (ROOT / feature['file']).is_file(), 'Traceability file missing: ' + feature['file']
        assert feature.get('uiElement', '').strip(), 'Missing concrete frontend UI element: ' + feature['id']
        assert_equal(set(feature.get('wireUsage', {})), set(feature['operations']), 'Frontend field-use mapping differs: ' + feature['id'])
        assert all(value.strip() for value in feature['wireUsage'].values()), 'Empty frontend field-use mapping: ' + feature['id']
        assert set(feature['operations']) <= {op['operationId'] for op in actual_ops.values()}
        assert set(feature['gaps']) <= gaps
        if not feature['operations']:
            assert feature['phase'] == 'later' and feature['phaseNote'], 'Unmapped feature must be explicitly deferred'
        for id in feature['operations']:
            assert feature['id'] in next(e for e in manifest['operations'] if e['id'] == id)['features']
    classified = {f['file'] for f in manifest['frontendFiles']}
    frontend_roots = ['frontendx/apps/admin/src', 'frontendx/apps/visitor/src', 'frontendx/packages/data/src']
    frontend = {p.relative_to(ROOT).as_posix() for directory in frontend_roots for p in (ROOT / directory).rglob('*') if p.suffix in ['.tsx', '.ts']}
    assert_equal(frontend, classified, 'New/removed frontend source needs classification and traceability review.')
    for name, content in render(spec, manifest).items():
        assert_equal((HERE / name).read_text(encoding='utf-8'), content, 'Generated documentation drift: ' + name)
    link_count = validate_local_links(HERE)
    checks.append(f'{link_count} local documentation file links and control-character checks')
    checks.append(f"{len(manifest['features'])} trace rows, {len(classified)} classified frontend files and generated documents")
    env = os.environ.copy()
    env['GOCACHE'] = str(ROOT / 'backend/.runtime/go-cache')
    env['GOTMPDIR'] = str(ROOT / 'backend/.runtime/go-tmp')
    Path(env['GOCACHE']).mkdir(parents=True, exist_ok=True)
    Path(env['GOTMPDIR']).mkdir(parents=True, exist_ok=True)
    go = json.loads(run(['go', 'run', str(HERE / 'inspect-go.go'), str(ROOT / 'backend')], env=env))
    served = read(ROOT / 'backend/docs/openapi.json')
    registered = validate_source(spec, manifest, go, served)
    checks.append('Go AST routes/handlers/delegation, interface signatures, DTO shapes/types, served OpenAPI')
    sql_files = [(file.name, file.read_text(encoding='utf-8')) for file in sorted((ROOT / 'backend/migration/files').glob('*.sql'))]
    tables = inventory_tables(sql_files)
    assert_equal(tables, manifest['migrationTables'], 'Unreviewed table addition/removal/rename; justify against a confirmed requirement.')
    checks.append(f'{len(tables)} SQL-source tables and Up/Down migration conventions; no DDL executed')
    # Deliberately perturb the artifacts to establish that the guard rejects drift.
    mutations = []
    for name, mutate in [
        ('unknown route', lambda g: g['routes'].append({'Method': 'GET', 'Path': '/v1/staff/undocumented', 'Handler': 'h.Unknown', 'File': 'fake.go'})),
        ('changed interface', lambda g: g['interfaces'][next(iter(manifest['sourceInterfaces']))].__setitem__('Unexpected', 'func()')),
        ('renamed DTO field', lambda g: next(iter(g['dtos'].values()))[0].__setitem__('Name', 'unexpected')),
    ]:
        changed = copy.deepcopy(go); mutate(changed)
        try: validate_source(spec, manifest, changed, served)
        except AssertionError: mutations.append(name)
        else: raise AssertionError('Guard accepted deliberate drift: ' + name)
    negatives = [('mass assignment', 'PublicCreate', {**examples['schemas']['PublicCreate'], 'status': 'approved'}), ('missing ack', 'PublicCreate', {k:v for k,v in examples['schemas']['PublicCreate'].items() if k != 'ack'}), ('restricted identity leak', 'RestrictedReception', {**examples['schemas']['RestrictedReception'], 'name':'Private visitor'}), ('password UTF8 overflow', 'Password', '\U0001f600' * 19)]
    for name, schema, value in negatives:
        try: validate(spec, {'$ref': '#/components/schemas/' + schema}, value)
        except ValidationError: mutations.append(name)
        else: raise AssertionError('Schema accepted prohibited input: ' + name)
    checks.append('Seven deliberate source/schema drift mutations rejected')
    for name, mutate in [
        ('unknown proposed serializer', lambda m: m['operations'][0].__setitem__('serializer', 'AuthSerializer.Unknown')),
        ('missing proposed usecase method', lambda m: m['operations'][7].__setitem__('usecase', 'VisitUsecase.Unknown')),
        ('wrong response array cardinality', lambda m: next(o for o in m['operations'] if o['id'] == 'staffIntegrations').__setitem__('responseDto', 'Integration')),
    ]:
        changed = copy.deepcopy(manifest); mutate(changed)
        try: validate_mappings(ROOT, spec, changed)
        except AssertionError: mutations.append(name)
        else: raise AssertionError('Mapping guard accepted: ' + name)
    hidden_table = sql_files + [('synthetic.sql', '-- +migrate Up\nCREATE TABLE IF NOT EXISTS support.unreviewed (id integer);\n-- +migrate Down\n')]
    assert inventory_tables(hidden_table) != tables, 'IF NOT EXISTS table escaped inventory'
    mutations.append('unreviewed IF NOT EXISTS table')
    invalid_users = [
        ('editable policy grants leaked into fixed role', {**examples['schemas']['CurrentUser'], 'permissions': ['people']}),
        ('deferred role picker in current user', {**examples['schemas']['CurrentUser'], 'availableRoles': ['PA', 'Super Admin']}),
    ]
    for name, value in invalid_users:
        try: validate(spec, {'$ref': '#/components/schemas/CurrentUser'}, value)
        except ValidationError: mutations.append(name)
        else: raise AssertionError('Phase 1 current-user schema accepted prohibited state: ' + name)
    checks.append('Additional interface/table/fixed-role/deferred-state drift mutations rejected')
    runtime = {'executed': False}
    if args.runtime:
        baseline = read(HERE / manifest['sourceBaseline'])
        runtime_spec = baseline['spec']
        source_ops = {(method.upper(), path): op for path, item in runtime_spec['paths'].items() for method, op in item.items() if method in METHODS}
        with tempfile.TemporaryDirectory(prefix='vms-contract-') as temp:
            capture = Path(temp) / 'http.json'; capture_env = {**env, 'VMS_CONTRACT_CAPTURE': str(capture)}
            run(['go', 'test', './internal/handler', '-run', '^TestContractHTTP$', '-count=1'], cwd=ROOT / 'backend', env=capture_env)
            records = read(capture); known = {}; valid = 0
            for record in records:
                path, method, code = record['path'][3:], record['method'], str(record['status'])
                op = source_ops[(method, path)]; response = op['responses'].get(code)
                issues = set()
                expected = baseline['examples']['runtimeExpectations'].get(record['name'], {})
                if 'status' in expected and record['status'] != expected['status']:
                    issues.add('status')
                if response is None:
                    issues.add('status')
                    response = runtime_spec['components']['responses'].get('Error' + code)
                    assert response is not None, 'Unrecognized HTTP status: ' + code
                if '$ref' in response:
                    response = runtime_spec['components']['responses'][response['$ref'].split('/')[-1]]
                if code != '204':
                    content = response['content']['application/json']
                    try: validate(runtime_spec, content['schema'], record['body'])
                    except ValidationError: issues.add('schema')
                    assert record['headers']['Content-Type'].startswith('application/json')
                    if 'code' in expected and record['body'].get('code') != expected['code']:
                        issues.add('code')
                    if not set(expected.get('fieldPaths', [])) <= set(record['body'].get('fieldErrors', {})):
                        issues.add('fieldErrors')
                    if int(code) >= 400:
                        assert record['body']['requestId'] == record['headers']['X-Request-ID'], 'Error correlation drift'
                else:
                    assert record['bytes'] == 0, '204 must have no body'
                if path.startswith('/staff'):
                    assert 'no-store' in record['headers']['Cache-Control'], f"Private cache policy drift: {record['name']}"
                assert record['headers']['X-Content-Type-Options'] == 'nosniff'
                assert record['headers']['X-Request-ID']
                if issues:
                    gap = baseline['runtimeKnownGaps'].get(record['name'], {}).get('gap')
                    known[record['name']] = {'gap': gap, 'issues': sorted(issues)}
                else:
                    valid += 1
            assert_equal(known, baseline['runtimeKnownGaps'], 'Current-source runtime drift; review baseline separately from the Phase 1 target.')
            covered = sorted({actual_ops[(r['method'], r['path'][3:])]['operationId'] for r in records if (r['method'], r['path'][3:]) in actual_ops})
            runtime = {'executed': True, 'realFiberResponses': len(records), 'sourceBaselineAccepted': valid, 'legacyKnownMismatches': known, 'phase1OperationsExercised': covered, 'boundary': 'Current-source HTTP regression only. Superseded session samples do not establish Phase 1 login/me or database authorization conformance.'}
        checks.append('Current-source Fiber regression against frozen source evidence; Phase 1 coverage reported separately')
    consumers = {'executed': False}
    if args.consumers:
        result = json.loads(run(['node', '--experimental-strip-types', str(HERE / 'check-consumers.mjs')]))
        assert_equal(result['knownGaps'], sorted(manifest['consumerKnownGaps']), 'New or stale consumer gap; explicit coordinated review required.')
        assert_equal(result['requestCalls'], manifest['frontendRequestCalls'], 'Frontend request AST changed; map/review it before updating snapshot.')
        consumers = {'executed': True, 'positiveNegativeChecks': len(result['cases']), 'knownMismatches': result['knownGaps'], 'reviewedRequestCallsites': len(result['requestCalls'])}
        checks.append(f"Actual frontend schemas/HTTP headers and {len(result['requestCalls'])} TypeScript request AST callsites")
    missing = [e['id'] for e in manifest['operations'] if not e['implemented']]
    blocking = [g['id'] for g in manifest['gaps'] if g.get('blocksTarget', True)]
    unresolved = [issue['id'] for issue in manifest['contractIssues']]
    source_only = sorted(f'{method} /v1{path}' for method, path in set(inspect_routes(go)) - set(actual_ops))
    incomplete = bool(missing or blocking or unresolved or source_only)
    result = {'status': 'baseline-passed-with-known-gaps' if incomplete else 'conformance-passed', 'strictConformance': 'failed' if incomplete else 'passed', 'strictModeRequested': args.strict, 'date': manifest['reviewedAt'], 'canonicalSpecSha256': hashlib.sha256(SPEC_PATH.read_bytes()).hexdigest(), 'operations': len(actual_ops), 'sourceRegistered': registered, 'currentSourceRoutes': len(go['routes']), 'sourceOnlyRoutes': len(source_only), 'sourceOnlyOperations': source_only, 'missingOperations': missing, 'knownGapIds': sorted(gaps), 'targetBlockingGapIds': blocking, 'unresolvedContractIssueIds': unresolved, 'checks': checks, 'negativeDriftChecks': mutations, 'runtime': runtime, 'consumers': consumers, 'limitations': ['No live database, migrations, containers, server startup or deployed API tested.', 'Legacy session-handler samples are source regression evidence only; Phase 1 login/me and database authority remain unimplemented.', 'Proposed interface mappings resolve in documentation; no claim that future DTO/entity types compile or that absent modules are implemented.', 'Query fixtures validate normalized values, not a future HTTP parser, database collation, filtering or pagination execution.', 'AST and examples do not establish scheduling locks, secret encryption, MIME verification, rate-limit correctness, SMTP delivery or all authorization branches.', 'Unimplemented routes/interfaces remain design requirements; strict completion also requires unresolved contract issues and source-only identity routes to be removed. Production-only gates do not block the disabled-capability contract.', 'CI must invoke this check; no enabled root CI workflow was present.']}
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(result, indent=2) + '\n', encoding='utf-8')
    print(json.dumps({k: result[k] for k in ['status', 'strictConformance', 'operations', 'sourceRegistered']}))
    print(f"Missing operations: {len(missing)}; known gap groups: {len(gaps)}; HTTP checks: {runtime.get('realFiberResponses', 'not run')}; consumer checks: {consumers.get('positiveNegativeChecks', 'not run')}.")
    if args.strict and incomplete:
        raise SystemExit(1)


if __name__ == '__main__':
    try:
        main()
    except (AssertionError, ValidationError, KeyError) as exc:
        print('CONTRACT CHECK FAILED: ' + str(exc), file=sys.stderr)
        raise SystemExit(1)
