"""Render review documents from the single OpenAPI and audited interface manifest.

Run with --write after a deliberate contract edit; default mode checks generated files.
Does not modify application behavior or the source OpenAPI.
"""
from pathlib import Path
import argparse
import json
import sys
from urllib.parse import urlencode

sys.dont_write_bytecode = True

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[1]
SPEC = ROOT / 'docs/BACKEND-OPENAPI-2026-11-12.json'


def anchor(name):
    return name.lower()


def js(value):
    return json.dumps(value, indent=2, ensure_ascii=False)


def schema_label(value):
    if '$ref' in value:
        n = value['$ref'].split('/')[-1]
        return f'[{n}](SCHEMAS.md#{anchor(n)})'
    if value.get('type') == 'array':
        return 'array of ' + schema_label(value['items'])
    return '`' + json.dumps(value, ensure_ascii=False, separators=(',', ':')).replace('|', '\\|') + '`'


GROUPS = [
    ('Authentication', ['staffLogin', 'staffMe']),
    ('Visitor registration', ['publicCatalogs', 'publicCreateBooking', 'resolveSubmission']),
    ('Visitor status and ticket', ['publicLookup', 'publicCancel', 'publicAdmission']),
    ('Staff catalogs', ['staffCatalogs']),
    ('Staff bookings and dashboard', ['staffBookings', 'staffBooking', 'staffCreateBooking', 'staffApprove', 'staffReject', 'staffReschedule', 'staffRestrictedUpdate', 'staffCancel']),
    ('Reception and attendance', ['staffResolve', 'staffCheckin', 'staffCheckout']),
    ('Private attachments', ['publicUpload', 'staffUpload', 'staffAttachment']),
    ('Employee directory (read-only)', ['staffEmployees', 'staffEmployee']),
    ('Integrations', ['staffIntegrations', 'patchIntegration']),
    ('Reports', ['reportSummary', 'reportExport']),
]


def cell(value):
    return str(value).replace('|', '\\|').replace('\n', ' ')


def resolved(spec, schema):
    if '$ref' in schema:
        return spec['components']['schemas'][schema['$ref'].split('/')[-1]]
    return schema


def type_label(spec, schema):
    if '$ref' in schema:
        return schema_label(schema)
    if 'oneOf' in schema:
        return 'one of ' + ', '.join(type_label(spec, v) for v in schema['oneOf'])
    if schema.get('type') == 'array':
        return 'array of ' + type_label(spec, schema['items'])
    if 'const' in schema:
        return '`' + json.dumps(schema['const'], ensure_ascii=False) + '`'
    return '`' + schema.get('type', 'object') + '`'


def constraints(spec, schema):
    value = resolved(spec, schema)
    parts = []
    if 'description' in value:
        parts.append(value['description'])
    if 'enum' in value:
        parts.append('Allowed: ' + ', '.join('`' + json.dumps(v, ensure_ascii=False) + '`' for v in value['enum']))
    if 'const' in value:
        parts.append('Must equal `' + json.dumps(value['const'], ensure_ascii=False) + '`')
    for key, label in [('format', 'Format'), ('minLength', 'Min length'), ('maxLength', 'Max length'), ('minimum', 'Min'), ('maximum', 'Max'), ('minItems', 'Min items'), ('maxItems', 'Max items'), ('uniqueItems', 'Unique items'), ('pattern', 'Pattern'), ('additionalProperties', 'Additional keys'), ('x-length-unit', 'Length unit'), ('x-normalization', 'Normalization'), ('x-min-utf8-bytes', 'Min UTF-8 bytes'), ('x-max-utf8-bytes', 'Max UTF-8 bytes'), ('x-max-bytes', 'Max bytes')]:
        if key in value:
            parts.append(f'{label}: `{json.dumps(value[key], ensure_ascii=False)}`')
    return '; '.join(parts) or 'See linked model and business rules.'


def object_fields(spec, schema):
    value = resolved(spec, schema)
    properties = dict(value.get('properties', {}))
    required = set(value.get('required', []))
    for part in value.get('allOf', []):
        # if/then/else remains conditional, never unconditional field requirements.
        p, r = object_fields(spec, part)
        properties.update(p)
        required.update(r)
    return properties, required


def field_tables(spec, schema):
    value = resolved(spec, schema)
    if 'oneOf' in value:
        lines = ['Exactly one response/request variant must match; fields from different privacy or creation variants cannot be combined.', '']
        for index, branch in enumerate(value['oneOf'], 1):
            lines += [f'**Variant {index}: {type_label(spec, branch)}**', '']
            lines += field_tables(spec, branch)
        return lines
    if value.get('type') == 'array':
        return ['Array items: ' + type_label(spec, value['items']) + '. ' + constraints(spec, value), '']
    properties, required = object_fields(spec, value)
    if not properties:
        return [type_label(spec, schema) + '. ' + constraints(spec, schema), '']
    lines = ['| Field | Type / model | Required | Default | Validation / meaning |', '| --- | --- | --- | --- | --- |']
    for name, prop in properties.items():
        p = resolved(spec, prop)
        default = '`' + json.dumps(p['default'], ensure_ascii=False) + '`' if 'default' in p else 'No wire default'
        lines.append(f"| `{name}` | {cell(type_label(spec, prop))} | {'Yes' if name in required else 'No; omit when absent'} | {cell(default)} | {cell(constraints(spec, prop))} |")
        # Inline objects have no separate model anchor; list their actual child keys.
        if p.get('type') == 'object' and '$ref' not in prop:
            for child, child_schema in p.get('properties', {}).items():
                lines.append(f"| `{name}.{child}` | {cell(type_label(spec, child_schema))} | {'Yes, when parent present' if child in p.get('required', []) else 'No'} | - | {cell(constraints(spec, child_schema))} |")
    lines += ['', 'Required refers to the selected variant. Conditional constraints and nested model definitions remain normative in [SCHEMAS](SCHEMAS.md); omitted, empty and null have the distinct meanings described below.', '']
    return lines


def parameter_table(spec, parameters):
    lines = ['| Parameter | Type / model | Required | Default | Validation / meaning |', '| --- | --- | --- | --- | --- |']
    for p in parameters:
        value = resolved(spec, p['schema'])
        default = '`' + json.dumps(value['default'], ensure_ascii=False) + '`' if 'default' in value else 'None declared'
        meaning = constraints(spec, p['schema'])
        if p.get('description'):
            meaning += '; ' + p['description']
        lines += [f"| `{p['name']}` | {cell(type_label(spec, p['schema']))} | {'Yes' if p.get('required', False) else 'No'} | {cell(default)} | {cell(meaning)} |"]
    return lines + ['']


def response_value(spec, response):
    return spec['components']['responses'][response['$ref'].split('/')[-1]] if '$ref' in response else response


def auth_label(op):
    if op.get('operationId') == 'staffLogin':
        return 'No existing login. Email/password in JSON; no authentication cookies, CSRF token or initialization call.'
    security = op.get('security', [])
    if not security:
        return 'No staff login; endpoint-specific public proof still applies where stated.'
    if any('BearerAccess' in branch for branch in security):
        return 'Yes — Bearer access JWT plus current active employee, single database role and resource authority.'
    if any('AnonymousBinding' in branch for branch in security):
        return 'No existing staff session. Initialized anonymous binding, signed CSRF cookie/header and exact Origin are required.'
    return 'Refresh-cookie lifecycle proof, signed CSRF cookie/header and exact Origin; see credential table and any unresolved boundary below.'


def render_reference(spec, manifest, examples):
    entries = {entry['id']: entry for entry in manifest['operations']}
    features = {f['id']: f for f in manifest['features']}
    grouped = [id for _, ids in GROUPS for id in ids]
    assert len(grouped) == len(set(grouped)) and set(grouped) == set(entries), 'Every endpoint must appear once in the SIHP-style groups.'
    lines = [f"All {len(manifest['operations'])} Phase 1 target business operations are listed once by feature below. Each group starts with **Daftar Endpoint**, followed by the full specification. Responses are VMS direct records, pages or bytes. A source-registered route can still have the implementation gaps shown in its section.", '', 'Jump to a feature group:', '']
    for group, _ in GROUPS:
        lines += [f'- [{group}](#{group.lower().replace('(', '').replace(')', '').replace(' ', '-')})']
    lines += ['']
    for group, ids in GROUPS:
        lines += [f'### {group}', '', '**Daftar Endpoint**', '', '| Endpoint | Method | Frontend feature / control | Success response | Error statuses |', '| --- | --- | --- | --- | --- |']
        for id in ids:
            entry = entries[id]
            op = spec['paths'][entry['path']][entry['method'].lower()]
            success = []
            for code, raw in op['responses'].items():
                if code.startswith('2'):
                    response = response_value(spec, raw)
                    models = ', '.join(type_label(spec, v['schema']) for v in response.get('content', {}).values()) or 'No body'
                    success += [f'`{code}` {models}']
            errors = ', '.join(code for code in op['responses'] if not code.startswith('2'))
            lines += [f"| [`/v1{entry['path']}`](#{id.lower()}) | `{entry['method']}` | {cell(entry['frontendSummary'])} | {cell('; '.join(success))} | {errors} |"]
        lines += ['', '---', '']
        for id in ids:
            entry = entries[id]
            op = spec['paths'][entry['path']][entry['method'].lower()]
            lines += [f'#### {id}', '', f"**{entry['method']} /v1{entry['path']} — {op['summary']}**", '', '**Auth Required:** ' + auth_label(op), '', '**Authorization / resource scope:** ' + entry['authorization'], '', '**Implementation Status:** ' + ('Source registered; the gaps below still apply.' if entry['implemented'] else 'Required by frontend/selected decisions; backend route missing.'), '']
            if op.get('description'):
                lines += ['**Description:** ' + op['description'], '']
            lines += ['**Frontend Usage**', '', '| Feature / route | UI element and action | Request / returned-data use | Source / current integration |', '| --- | --- | --- | --- |']
            for f_id in entry['features']:
                f = features[f_id]
                lines += [f"| [{f_id}](FRONTEND-TRACEABILITY.md#{f_id.lower()}); {cell(f['route'])} | `{cell(f['component'])}` — {cell(f['uiElement'])} | {cell(f['wireUsage'][id])} | [{f['file']}](../../{f['file']}); {cell(f['current'])} |"]
            lines += ['', '**Query Parameters**', '']
            query = [p for p in op.get('parameters', []) if p['in'] == 'query']
            lines += parameter_table(spec, query) if query else ['None. Query keys are not accepted on this operation.', '']
            if 'x-query-schema' in op:
                lines += ['The following view variants are mutually exclusive. Their required keys/defaults override the broad parameter inventory above; other-view keys are rejected.', '']
                for variant in op['x-query-schema']['oneOf']:
                    view = variant['properties']['view']['const']
                    lines += [f'**`view={view}` query variant**', ''] + field_tables(spec, variant)
                lines += ['Reception `mode=checkout` rejects `date` and includes still-inside visits from prior dates. Dashboard requires `date` and `page=1`. Check-in date behavior and exact sorting boundaries remain as specified by the selected rules and OPEN-READ-ORDER.', '']
            if query:
                samples = [case['query'] for case in examples['queryCases'] if case['operation'] == id and case['valid']]
                for sample in samples[:2]:
                    lines += ['```http', entry['method'] + ' /v1' + entry['path'] + '?' + urlencode(sample), '```', '']
            lines += ['**Path Parameters**', '']
            path_params = [p for p in op.get('parameters', []) if p['in'] == 'path']
            lines += parameter_table(spec, path_params) if path_params else ['None.', '']
            lines += ['**Headers and Cookies**', '', 'JSON bodies require `Content-Type: application/json`; uploads use browser-generated `multipart/form-data` with a boundary. Protected staff requests use the Authorization header. Phase 1 uses no authentication cookies or signed-CSRF initialization.', '']
            headers = [p for p in op.get('parameters', []) if p['in'] in ('header', 'cookie')]
            if headers:
                lines += parameter_table(spec, headers)
            security_rows = []
            for branch_index, branch in enumerate(op.get('security', []), 1):
                for name in branch:
                    sec = spec['components']['securitySchemes'][name]
                    security_rows += [f"| {branch_index} | `{sec.get('name', 'Authorization: Bearer <accessToken>')}` | {sec.get('in', 'header')} | {cell(sec.get('description', 'Current verified Bearer JWT plus database account/single-role/assignment authority; no session lookup.'))} |"]
            if security_rows:
                lines += ['| Security branch | Credential / protection | Location | Validation |', '| --- | --- | --- | --- |'] + security_rows + ['']
            else:
                lines += ['No staff authentication header/cookie is required. Public pair/reference proof, when applicable, belongs in the request body.', '']
            lines += ['**Request Body**', '']
            body = op.get('requestBody')
            if body:
                for media, content in body['content'].items():
                    schema = content['schema']
                    lines += [f"Media type: `{media}`; body required: **{'yes' if body.get('required') else 'no'}**; model: {type_label(spec, schema)}.", '']
                    lines += field_tables(spec, schema)
                    value = resolved(spec, schema)
                    if media == 'multipart/form-data':
                        lines += ['Send actual file bytes in the `file` part and a UUID string in `reference`. The file is not a JSON/base64 string. One file only; file limit 5,242,880 bytes; multipart limit 6,291,456 bytes; verified PDF/JPEG/PNG.', '']
                    else:
                        variants = value.get('oneOf', [])
                        if variants:
                            for branch in variants:
                                name = branch['$ref'].split('/')[-1]
                                lines += ['Example: ' + type_label(spec, branch), '', '```json', js(examples['schemas'][name]), '```', '']
                        else:
                            lines += ['```json', js(examples['operations'][id]['request']), '```', '']
            else:
                lines += ['None. No request body is defined; do not add command fields to the URL.', '']
            lines += ['**Business Logic and Side Effects**', '', entry['rules'], '']
            if body:
                for content in body['content'].values():
                    model = resolved(spec, content['schema'])
                    for rule in model.get('x-server-invariants', []):
                        lines += ['- ' + rule]
                lines += ['']
            affected = [issue for issue in manifest['contractIssues'] if id in issue['operations']]
            for issue in affected:
                lines += [f"Unresolved boundary **{issue['id']}**: {issue['finding']} See [CONTRACT-ISSUES](CONTRACT-ISSUES.md).", '']
            lines += ['**Success Responses**', '', '| HTTP status | Body / model | Headers | Meaning |', '| --- | --- | --- | --- |']
            response_models = []
            for code, raw in op['responses'].items():
                if not code.startswith('2'):
                    continue
                response = response_value(spec, raw)
                content = response.get('content', {})
                models = '; '.join(media + ': ' + type_label(spec, value['schema']) for media, value in content.items()) or 'No body'
                lines += [f"| `{code}` | {cell(models)} | {cell(', '.join(response.get('headers', {})) or 'Common response headers')} | {cell(response['description'])} |"]
                for media, content_value in content.items():
                    if media == 'application/json' and content_value['schema'] not in response_models:
                        response_models.append(content_value['schema'])
            lines += ['', '**Response Fields**', '']
            for model in response_models:
                lines += [type_label(spec, model), ''] + field_tables(spec, model)
            if not response_models:
                detail = 'Private file success returns verified bytes with Content-Disposition, Cache-Control: no-store and X-Content-Type-Options: nosniff.' if id == 'staffAttachment' else 'HTTP 204 returns no body.'
                lines += ['No JSON success fields. ' + detail, '']
            seen_examples = set()
            for code, value in examples['operations'][id]['success'].items():
                key = js(value)
                if key in seen_examples:
                    lines += [f'HTTP `{code}` returns the same body shape/example above; status meaning is defined in the success table.', '']
                    continue
                seen_examples.add(key)
                lines += [f'**Response ({code}) — synthetic example**', '']
                if value is None:
                    lines += ['No body.', '']
                elif not response_models:
                    lines += ['Binary bytes of the verified MIME type; not a JSON object.', '']
                else:
                    lines += ['```json', js(value), '```', '']
            lines += ['**Error Responses**', '', 'JSON errors use [ApiError](SCHEMAS.md#apierror) and the [common error examples](#error-examples). These are target branches, not evidence that all branches are implemented.', '', '| HTTP status | Condition |', '| --- | --- |']
            for code, raw in op['responses'].items():
                if not code.startswith('2'):
                    response = response_value(spec, raw)
                    lines += [f"| `{code}` | {cell(op['x-error-conditions'].get(code, response.get('description', '')))} |"]
            lines += ['', '**Backend Interface Flow and Evidence**', '', f"Handler: `internal/handler/{entry['module']}_handler.go:{entry['handler']}` → usecase `{entry['usecase']}` → repository operations " + ', '.join('`' + r + '`' for r in entry['repositories']) + f". Serializer: `{entry['serializer'] or '(no JSON projection)'}`. DTOs: `{entry['requestDto'] or '(no request DTO)'}` → `{entry['responseDto'] or '(no response body)'}`. Exact existing/proposed signatures: [INTERFACES](INTERFACES.md).", '', 'Known gaps: ' + ', '.join(entry['gaps']) + '. Requirement and source evidence: ' + entry['evidence']['status'] + '.', '']
            if entry['evidence']['source']:
                source = entry['evidence']['source']
                lines += [f'Current handler: [{source}](../../{source}). Route registration does not establish complete behavior.', '']
            lines += ['---', '']
    return '\n'.join(lines)


def render_main_sections(spec, manifest, examples):
    enums = ['The exact wire schemas, including all nested models, discriminated projections, conditional constraints and server invariants, are in [SCHEMAS](SCHEMAS.md). Field tables and examples appear with each endpoint above. The canonical [OpenAPI](../BACKEND-OPENAPI-2026-11-12.json) remains the machine-readable authority.', '', '### Enums', '', '| Model | Allowed values |', '| --- | --- |']
    for name, schema in spec['components']['schemas'].items():
        if 'enum' in schema:
            enums += [f"| [{name}](SCHEMAS.md#{name.lower()}) | {cell(', '.join('`' + str(v) + '`' for v in schema['enum']))} |"]
    enums += ['', '`missed` is a derived display/report status; the persisted Status enum has six values. Restricted projection variants are privacy boundaries, not optional field bundles.', '', '### Model directory', '', '| Model | Shape |', '| --- | --- |']
    for name, schema in spec['components']['schemas'].items():
        shape = ', '.join(v['$ref'].split('/')[-1] for v in schema.get('oneOf', []) if '$ref' in v) or schema.get('type', 'composed object')
        enums += [f'| [{name}](SCHEMAS.md#{name.lower()}) | {cell(shape)} |']
    errors = ['The HTTP status is authoritative; `code` is the machine-readable error identifier. Only the statuses listed for an operation apply to it. No stacktrace, SQL, secret, provider detail or private conflicting-record data is returned.', '', '| HTTP status | Meaning |', '| --- | --- |']
    for code, raw in spec['components']['responses'].items():
        errors += [f"| `{code.removeprefix('Error')}` | {cell(raw['description'])} |"]
    errors += ['', '### Error examples', '', 'Field errors use actual form keys and arrays of messages. `429` includes an integer-seconds `Retry-After` header.', '']
    for code, value in examples['errors'].items():
        errors += [f'HTTP `{code}`', '', '```json', js(value), '```', '']
    implemented = sum(e['implemented'] for e in manifest['operations'])
    status = [f"**{len(manifest['operations'])} Phase 1 target operations; {implemented} matching source-registered route; {len(manifest['operations']) - implemented} missing target routes.** The unchanged backend also has seven superseded identity routes, for eight source routes total. New login/me is not implemented. Source registration does not certify frontend, lifecycle, database or security conformance. See [GAPS](GAPS.md), [validation evidence](VALIDATION.md) and the [implementation plan](IMPLEMENTATION-PLAN.md).", '', '| Feature group | Source registered | Backend missing |', '| --- | --- | --- |']
    by_id = {e['id']: e for e in manifest['operations']}
    for group, ids in GROUPS:
        registered = [f'[{id}](#{id.lower()})' for id in ids if by_id[id]['implemented']]
        missing = [f'[{id}](#{id.lower()})' for id in ids if not by_id[id]['implemented']]
        status += [f"| {group} | {', '.join(registered) or 'None'} | {', '.join(missing) or 'None'} |"]
    status += ['', '### Unresolved decisions', '', '| Issue | Affected operations | Decision needed |', '| --- | --- | --- |']
    for issue in manifest['contractIssues']:
        status += [f"| [{issue['id']}](CONTRACT-ISSUES.md#{issue['id'].lower()}) | {', '.join(issue['operations'])} | {cell(issue['decisionNeeded'])} |"]
    mapping = ['These are the actual pages, dialogs, cards, selectors and shared consumers inspected in VMS. The detailed endpoint sections show which request/response fields each control uses and disclose existing integration gaps. Full callsite evidence remains in [FRONTEND-TRACEABILITY](FRONTEND-TRACEABILITY.md).', '', '| Feature | Route / component | UI feature or control | Endpoint method and path |', '| --- | --- | --- | --- |']
    for f in manifest['features']:
        operations = '; '.join(f"[{by_id[id]['method']} /v1{by_id[id]['path']}](#{id.lower()})" for id in f['operations']) or 'Deferred in Phase 1; no API call'
        mapping += [f"| [{f['id']}](FRONTEND-TRACEABILITY.md#{f['id'].lower()}) | {cell(f['route'])}; `{f['component']}` | {cell(f['uiElement'])} ({f['phase']}) | {operations} |"]
    mapping += ['', 'Static landing content, ticket printing, camera capture, local filter state, identity-confirmation checkboxes and draft/preferences storage do not introduce separate business endpoints. Notes are submitted with their owning booking command; selectors share the audience-appropriate catalogs call.', '']
    deferred = ['These operations are outside the Phase 1 target. Existing session routes remain source evidence until the auth slice is replaced; their current presence is not Phase 1 support. Account/role editors must be hidden or disabled. Local Sign out has no HTTP operation.', '', '| Previous operation | Method / path | Disposition / replacement | Reason |', '| --- | --- | --- | --- |']
    for item in manifest['deferredOperations']:
        replacement = ('; [' + item['replacement'] + '](#' + item['replacement'].lower() + ')') if item['replacement'] else ''
        deferred += [f"| {item['id']} | {item['method']} /v1{item['path']} | {item['disposition']}{replacement}; {'currently source registered' if item['sourceRegistered'] else 'backend absent'} | {cell(item['reason'])} |"]
    authority = ['| Database role | Fixed permission keys |', '| --- | --- |']
    for role, keys in manifest['phase1']['rolePermissions'].items():
        authority += [f"| `{role}` | {', '.join('`' + key + '`' for key in keys)} |"]
    return {'AUTHORITY': '\n'.join(authority) + '\n', 'DEFERRED': '\n'.join(deferred) + '\n', 'ENDPOINTS': render_reference(spec, manifest, examples), 'MODELS': '\n'.join(enums) + '\n', 'ERRORS': '\n'.join(errors) + '\n', 'STATUS': '\n'.join(status) + '\n', 'FRONTEND': '\n'.join(mapping) + '\n'}


def replace_sections(document, sections):
    for name, content in sections.items():
        start, end = f'<!-- BEGIN GENERATED {name} -->', f'<!-- END GENERATED {name} -->'
        assert document.count(start) == document.count(end) == 1, f'Missing/duplicate contract section marker: {name}'
        before, rest = document.split(start, 1)
        _, after = rest.split(end, 1)
        document = before + start + '\n\n' + content.rstrip() + '\n\n' + end + after
    return document


def render(spec, manifest):
    schemas = spec['components']['schemas']
    examples = json.loads((HERE / 'examples.json').read_text(encoding='utf-8'))
    schema_doc = ['# Wire schemas', '', 'Generated from the canonical [OpenAPI](../BACKEND-OPENAPI-2026-11-12.json). Required lists, additionalProperties, oneOf/allOf, defaults and validation extensions are normative. No nullable types are declared: optional fields are omitted, not null. `x-*` state/byte constraints require application checks as described in the main contract. Examples are synthetic.', '']
    for name, schema in schemas.items():
        schema_doc += [f'## {name}', '', '```json', js(schema), '```', '']
    main_doc = replace_sections((HERE / 'API-CONTRACT.md').read_text(encoding='utf-8'), render_main_sections(spec, manifest, examples))
    endpoints = ['# Endpoint specifications', '', 'The complete SIHP-structured endpoint reference is now in [API-CONTRACT](API-CONTRACT.md#api-endpoints). This file preserves existing operation links as a navigation index. Each linked specification includes frontend controls/field usage, method/path, authentication, query/path/header/cookie parameters, request fields/examples, business logic, response fields/examples and operation-specific errors.', '']
    for entry in manifest['operations']:
        endpoints += [f"## {entry['id']}", '', f"[{entry['method']} /v1{entry['path']} — {entry['frontendSummary']}](API-CONTRACT.md#{entry['id'].lower()})", '']
    endpoints += ['## Error examples', '', 'See [common error codes and examples](API-CONTRACT.md#error-codes).', '']
    trace = ['# Frontend data traceability', '', 'Every application feature/page and shared data adapter was inspected. `confirmed requirement` refers to source or retained decisions; `current` describes actual consumer code, never a claim that the backend exists. All source-registered versus missing backend status comes from [manifest](manifest.json). Proposed interfaces are explicitly marked in [INTERFACES](INTERFACES.md). Preview fixtures and local UI preferences are not server persistence.', '', '| ID / source / component | Route | UI element / action / required data | Existing integration | Endpoint operation IDs | Evidence / gaps |', '| --- | --- | --- | --- | --- | --- |']
    for f in manifest['features']:
        trace += [f"| [{f['id']}](#{f['id'].lower()}); [{f['file']}](../../{f['file']}); `{f['component']}` | {f['route'].replace('|','/')} | {f['uiElement']}; {f['action']}; {f['data']} | {f['current']} | {', '.join(f['operations'])} | {f['evidence']}; {', '.join(f['gaps'])} |"]
    for f in manifest['features']:
        trace += ['', f"## {f['id']}", '', f"`{f['file']}` — {f['component']}. {f['action']}. {f['data']}. {f['current']}. Phase: {f['phase']}. Phase 1 operations: {', '.join(f['operations']) or 'none; deferred UI'}. Gaps: {', '.join(f['gaps'])}."]
        if f.get('phaseNote'):
            trace += ['', f['phaseNote']]
        for action in f.get('localActions', []):
            trace += ['', 'Local action: ' + action]
        trace += ['', '**UI element:** ' + f['uiElement'], '', '| Operation | Request / returned-data use |', '| --- | --- |']
        for id, usage in f['wireUsage'].items():
            trace += [f"| [{id}](API-CONTRACT.md#{id.lower()}) | {cell(usage)} |"]
    trace += ['', '## Source review boundary', '', 'The checker inventories request() calls with the existing TypeScript parser and compares their normalized AST text to the reviewed snapshot. New/changed requests require a deliberate manifest update and endpoint mapping. Every application source file and shared data/transport module is classified below; data-bearing consumers are mapped above. Decorative UI, camera capture and local preferences create no additional API requirement.', '', '| File | Classification | Reason |', '| --- | --- | --- |']
    for row in manifest['frontendFiles']:
        trace += [f"| {row['file']} | {row['kind']} | {row['reason']} |"]
    gaps = ['# Implementation gaps', '', 'Review date: ' + manifest['reviewedAt'] + ' WITA. Phase 1 has 29 target operations: one matching catalog route and 28 missing target routes. Seven old identity routes remain in source and must be replaced/removed for Phase 1. Session lifecycle and account/role writes are deferred, not target gaps. A baseline check can pass with known gaps; strict target conformance fails until target-blocking gaps are resolved. DB-RELEASE is a production-only gate and does not block a disabled-capability contract. See [validation results](validation-results.json) for actual executed evidence and limitations.', '', '| Gap | Status / gate | Actual finding | Required change |', '| --- | --- | --- | --- |']
    for g in manifest['gaps']:
        gaps += [f"| {g['id']} | {g['status']}; {'target' if g.get('blocksTarget', True) else 'production only'} | {g['finding']} | {g['resolution']} |"]
    gaps += ['', '## Operations requiring backend implementation', '', '| Operation | Target route | Missing layer owner |', '| --- | --- | --- |']
    for op in manifest['operations']:
        if not op['implemented']: gaps += [f"| {op['id']} | {op['method']} /v1{op['path']} | {op['module']}; handler/use-case/repository/serializer/DTO |"]
    gaps += ['', '## Deferred findings (outside the Phase 1 gate)', '', '| Previous finding | Phase 1 disposition |', '| --- | --- |']
    for gap in manifest['deferredGaps']:
        gaps += [f"| {gap['id']} | {cell(gap['phase1Disposition'])} |"]
    for correction in manifest['sourceCorrections']:
        gaps += ['', 'Source correction **' + correction['id'] + '**: ' + correction['finding']]
    gaps += ['', '## What already matches', '', '- Existing catalog and identity code use interfaces in module packages, concrete implementation subpackages, constructor injection and DTO serialization.', '- Catalog reads active officials joined to Jabatan for VIP and stable integer room IDs in one repeatable-read snapshot; notice comes from configuration and attachmentsEnabled is false.', '- Current source uses a stored-session authority reader; this is explicitly superseded by the Phase 1 token/account/single-role reader and is not target conformance.', '- Existing SQL uses sql-migrate Up/Down files and retained directory/assignment rows. Session/refresh/policy structures remain source history; Phase 1 does not require them or authorize deleting deployed tables.', '- Frontend production adapters already use direct camelCase records, typed paginated views, versioned commands, public credential POST bodies and explicit preview isolation.', '', 'Source matches are not database deployment evidence. No migrations or live database actions were performed for this audit.', '']
    interfaces = ['# Backend interface contract', '', 'Current source signatures below are inspected, not assumed. Proposed signatures are documentation only: no interface stubs, second backend or missing-handler placeholders were created. Constructors return the module interface; implementations belong in the matching implementation subpackage. Handler constructors receive the usecase interface; routers are the composition root; repositories own GORM; serializers map explicit allowed fields into DTOs. These are the actual SIHP conventions retained by the current VMS modules.', '', 'Phase 1 retains Fiber3, direct DTO/error returns and INTEGER IDs/string wire IDs. It uses a signed access JWT plus a current database account/single-role/PA-assignment read, without stored login sessions. Existing source session signatures below are legacy inventory; the proposed Phase 1 signatures replace their active use, within the same modules. SIHP Req/Res DTO prefixes are reference naming, while existing VMS DTOs use semantic names; preserve existing names and use consistent semantic names for new modules.', '', '## Existing source inventory (auth is superseded)', '']
    for name, methods in manifest['sourceInterfaces'].items():
        interfaces += [f'### {name}', '', '```go']
        for method, signature in methods.items():
            interfaces += [signature if method.startswith('@') else method + signature.removeprefix('func')]
        interfaces += ['```', '']
    interfaces += ['## Development sequence and type boundaries', '', 'Follow the inspected SIHP composition and layer boundaries: confirm the frontend action and selected contract; define the typed request/response and usecase/repository/serializer interfaces; implement them behind injected interfaces; wire the router explicitly; then verify contract and real behavior. Source code establishes these boundaries, not the historical order in which SIHP was written.', '', 'Interface implementation slices must include a compile-time assertion, an interface-returning constructor, typed handler input, and explicit serializer projection. The repository owns database access and its transaction-bound connection; usecases coordinate validation and policy. Existing SessionRepository callback signatures are source evidence only; Phase 1 must not depend on session or refresh state. Visit/integration transactions still belong to repositories.', '', 'Omitted, empty and null are distinct wire cases. Future Go DTOs must preserve presence for fields whose omission preserves and whose empty value clears; a plain string with a zero-value fallback cannot implement that distinction. No request schema accepts null. Discriminated standard/restricted inputs and projections must retain their variant validation instead of one permissive bag of optional fields.', '', 'Aliases: current dto.PublicCatalog maps to wire Catalogs; integration reads are arrays. Phase 1 Login returns dto.AuthResult directly, containing an access token, expiry and dto.CurrentUser. No CredentialResult cookie wrapper is part of the target. PrivateFile is a bounded private stream, not a JSON DTO. Query DTOs describe parsed query parameters, not GET bodies.', '', 'Proposed entity/DTO names below are interface design obligations. Their Go representation must be defined and compiled in the owning slice; resolving a documentation symbol does not prove the future type exists. OPEN-READ-ORDER remains an active decision. The old cookie/logout-proof issue is deferred and does not apply to local-only Phase 1 logout.', '']
    interfaces += ['## Proposed additions (not implemented)', '', 'Signatures specify reviewable boundaries within existing layers. Domain entity structs are to be introduced with their owning slices; repository entities are not HTTP DTO aliases. All resource usecases must derive current authority from proof and recheck it transactionally for writes. Constructors follow NewVisitUsecase/NewVisitRepository/NewVisitSerializer etc.; router dependencies are injected explicitly. Do not pass a Fiber context to a repository or hide a GORM transaction in context values.', '', 'CommandResult[T] is an internal result containing Value T and Replayed bool, letting handlers select201 versus200 without duplicate usecase methods. SubmissionReference includes audience, principal/email binding, UUID, normalized digest and resolve/close intent; ResolveSubmission/CreateVisit coordinate outcome linkage in the same transaction. VisitActor identifies authenticated proof or public pair/reference without granting authority from an actor label alone. ScheduleChange carries complete old/new resource interval data. ResourceKey order is stable by type/ID. AttendanceCommand carries visitID/version/credential/action and no fabricated identityVerified field.', '', 'PrivateFile contains an io.ReadCloser plus verified MIME/safe display filename/size; it never serializes a private path into JSON. Multipart DTO Upload is decoded into a bounded reader plus reference/declared metadata before StageUpload; field/token ownership and content verification remain required. AttachmentQuery contains disposition=inline|attachment default inline. ReportQuery is from/to/status and ReportExportQuery adds revision. BookingQuery is the typed view union; EmployeeQuery contains q/page/pageSize. These query structs are not extra request-body schemas.', '', 'Repositories read current authorized aggregates, but serializers still enforce per-field projection. Phase 1 permissions come from the fixed backend role mapping; no editable policy graph or generic updateMap is exposed. Transaction methods use one transaction-bound connection; AppendAudit/EnqueueNotifications are atomic parts of the command, not unrelated external calls. StageUpload cannot satisfy visit linkage by itself; CreateVisit coordinates owned verified linkage and cleanup exclusion within its aggregate transaction.', '']
    for name, methods in manifest['proposedInterfaces'].items():
        interfaces += [f'### {name}', '', '```go']
        for method, signature in methods.items():
            interfaces += [method + signature.removeprefix('func')]
        interfaces += ['```', '']
    interfaces += ['## Per-operation mappings', '', '| HTTP operation | Handler | Usecase | Request -> response DTO | Serializer | Repository operations | State |', '| --- | --- | --- | --- | --- | --- | --- |']
    for e in manifest['operations']:
        interfaces += [f"| {e['method']} /v1{e['path']} | {e['module']}_handler.go:{e['handler']} | {e['usecase']} | {e['requestDto'] or 'none'} -> {e['responseDto'] or 'none'} | {e['serializer'] or 'no JSON projection'} | {', '.join(e['repositories'])} | {'source registered' if e['implemented'] else 'proposed'} |"]
    interfaces += ['', 'Implementations live at internal/usecase/{module}/implementation, internal/repository/{module}/implementation and internal/serializer/{module}/implementation; DTOs stay in pkg/dto and domain aggregates in internal/entity. Public catalog and staff catalog intentionally reuse the same wire shape. Phase 1 AuthUsecase.Login returns dto.AuthResult; AuthUsecase.GetCurrentUser returns dto.CurrentUser. AccessProof contains the verified employee subject and token issue/expiry/issuer/audience context, never a session ID or role grant. CurrentAuthority contains account active/name, exactly one role and current PA assignments; serializers derive the fixed permission list. Existing SessionUsecase/GetSession signatures above describe the superseded binary only, not required Phase 1 work.', '']
    issues = ['# Unresolved contract boundaries', '', 'These are explicit decisions still needed within the selected target. They are distinct from missing implementation and production configuration. The review does not supply invented defaults; strict conformance remains blocked while they are open.', '']
    for issue in manifest['contractIssues']:
        issues += ['## ' + issue['id'], '', 'Affected operations: ' + ', '.join(issue['operations']) + '.', '', issue['finding'], '', 'Decision needed: ' + issue['decisionNeeded'], '', 'Blocks: ' + issue['blocks'], '', 'Evidence: ' + ', '.join('[' + name + '](../../' + name + ')' for name in issue['evidence']) + '.', '']
    issues += ['## Deferred boundary (not a Phase 1 decision)', '']
    for issue in manifest['deferredContractIssues']:
        issues += ['**' + issue['id'] + '**: ' + issue['phase1Disposition'], '']
    return {'API-CONTRACT.md': main_doc, 'ENDPOINTS.md': '\n'.join(endpoints) + '\n', 'SCHEMAS.md': '\n'.join(schema_doc) + '\n', 'FRONTEND-TRACEABILITY.md': '\n'.join(trace) + '\n', 'GAPS.md': '\n'.join(gaps) + '\n', 'INTERFACES.md': '\n'.join(interfaces) + '\n', 'CONTRACT-ISSUES.md': '\n'.join(issues) + '\n'}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--write', action='store_true')
    args = parser.parse_args()
    spec = json.loads(SPEC.read_text(encoding='utf-8'))
    manifest = json.loads((HERE / 'manifest.json').read_text(encoding='utf-8'))
    stale = []
    for name, content in render(spec, manifest).items():
        path = HERE / name
        if args.write:
            path.write_text(content, encoding='utf-8')
        elif not path.exists() or path.read_text(encoding='utf-8') != content:
            stale.append(name)
    if stale:
        raise SystemExit('Generated contract docs differ: ' + ', '.join(stale) + '; review inputs then run render.py --write')
    print('Contract documents ' + ('rendered.' if args.write else 'match source.'))


if __name__ == '__main__':
    main()
