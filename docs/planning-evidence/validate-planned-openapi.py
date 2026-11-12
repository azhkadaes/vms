"""Validate the planning artifact only; never starts the VMS application or a DB."""

import json
import hashlib
from pathlib import Path
import re

from jsonschema import Draft202012Validator, FormatChecker, ValidationError, validators


DOCS = Path(__file__).resolve().parents[1]
SPEC_PATH = DOCS / "BACKEND-OPENAPI-2026-11-12.json"
spec = json.loads(SPEC_PATH.read_text(encoding="utf-8"))
official = json.loads((DOCS / "planning-evidence/openapi-3.1-schema-2025-09-15.json").read_text(encoding="utf-8"))
Draft202012Validator(official).validate(spec)
schemas = spec["components"]["schemas"]
for schema in schemas.values():
    Draft202012Validator.check_schema(schema)

refs = []


def visit(value):
    if isinstance(value, dict):
        if "$ref" in value:
            refs.append(value["$ref"])
        for nested in value.values():
            visit(nested)
    elif isinstance(value, list):
        for nested in value:
            visit(nested)


visit(spec)
for pointer in refs:
    assert pointer.startswith("#/"), pointer
    value = spec
    for key in pointer[2:].split("/"):
        value = value[key.replace("~1", "/").replace("~0", "~")]

operations = [(method, path, operation) for path, item in spec["paths"].items() for method, operation in item.items()]
assert len(operations) == 38
assert len({operation["operationId"] for _, _, operation in operations}) == 38
assert sorted(operation["x-inventory-number"] for _, _, operation in operations) == list(range(1, 39))
expected = set()
for line in (DOCS / "BACKEND-API-CONTRACT-2026-11-11.md").read_text(encoding="utf-8").splitlines():
    cells = [cell.strip() for cell in line.split("|")]
    if len(cells) > 3 and cells[1].isdigit() and 1 <= int(cells[1]) <= 38:
        match = re.fullmatch(r"(GET|POST|PATCH|DELETE)\s+`([^`]+)`", cells[2])
        assert match, cells[2]
        expected.add((match[1].lower(), match[2]))
actual = {(method, path) for method, path, _ in operations}
assert expected == actual, (expected - actual, actual - expected)
plan_text = (DOCS / "BACKEND-PLAN-2026-11-11.md").read_text(encoding="utf-8")
decision_ids = [int(value) for value in re.findall(r"^\| D([0-9]+) \|", plan_text, flags=re.MULTILINE)]
assert decision_ids == list(range(1, 101)), decision_ids
schema_text = (DOCS / "BACKEND-SCHEMA-EVIDENCE-2026-11-11.md").read_text(encoding="utf-8")
diagram_section = schema_text.split("## Transcribed diagram fields and reconciliation", 1)[1].split("## Relationship evidence", 1)[0]
diagram_rows = [line for line in diagram_section.splitlines() if line.startswith("| ") and not line.startswith(("| Diagram entity", "| ---"))]
assert len(diagram_rows) == 20, len(diagram_rows)
for method, path, operation in operations:
    assert operation.get("x-reject-unknown-query") is True
    names = [(p["in"], p["name"]) for p in operation.get("parameters", [])]
    assert len(names) == len(set(names)), (method, path)
    if "{id}" in path:
        assert ("path", "id") in names
    if "requestBody" in operation:
        for media in operation["requestBody"]["content"].values():
            assert "$ref" in media["schema"]
    assert any(code.startswith("2") for code in operation["responses"])


def localize(value):
    if isinstance(value, dict):
        return {key: item.replace("#/components/schemas/", "#/$defs/") if key == "$ref" else localize(item) for key, item in value.items()}
    if isinstance(value, list):
        return [localize(item) for item in value]
    return value


definitions = localize(schemas)
cases = []


def utf8_limit(validator, maximum, instance, schema):
    if isinstance(instance, str) and len(instance.encode("utf-8")) > maximum:
        yield ValidationError("UTF-8 byte limit exceeded")


ExtendedValidator = validators.extend(Draft202012Validator, {"x-max-utf8-bytes": utf8_limit})


def check(name, schema_name, instance, valid=True):
    schema = {"$defs": definitions, "$ref": "#/$defs/" + schema_name}
    accepted = ExtendedValidator(schema, format_checker=FormatChecker()).is_valid(instance)
    assert accepted == valid, (name, schema_name, accepted)
    cases.append({"case": name, "passed": True})


for value in ["1", "2147483647", "999999999", "2000000000", "2147483640"]:
    check("valid business ID " + value, "BusinessId", value)
for value in ["0", "01", "-1", "+1", "1.0", "2147483648", "9999999999", "10000000000", 1]:
    check("reject business ID " + str(value), "BusinessId", value, False)
check("valid version maximum", "Version", 2147483647)
for value in [0, -1, 1.5, 2147483648, "1"]:
    check("reject version " + str(value), "Version", value, False)
public = {"visitKind": "standard", "idempotencyKey": "550e8400-e29b-41d4-a716-446655440000", "officerId": "1", "date": "2030-01-02", "time": "16:00", "name": "Visitor", "organization": "Office", "email": "user@example.com", "phone": "0812345678", "purpose": "Meeting", "identityType": "KTP", "visitors": ["Visitor"], "ack": True}
check("public privacyVersion optional D90", "PublicCreate", public)
check("public ack mandatory", "PublicCreate", {k: v for k, v in public.items() if k != "ack"}, False)
check("public ack cannot be false", "PublicCreate", {**public, "ack": False}, False)
check("public rejects mass-assigned status", "PublicCreate", {**public, "status": "approved"}, False)
check("16:00 start allowed", "StartTime", "16:00")
check("16:01 start rejected", "StartTime", "16:01", False)
check("invalid date rejected", "Date", "2030-02-30", False)
check("unicode mailbox rejected", "Email", "üser@example.com", False)
check("ASCII punycode mailbox accepted", "Email", "user@xn--bcher-kva.example")
check("new password 15 ASCII accepted", "Password", "a" * 15)
check("new password 14 ASCII rejected", "Password", "a" * 14, False)
check("new password 72 bytes accepted", "Password", "a" * 72)
check("new password 73 bytes rejected", "Password", "a" * 73, False)
check("new password 15 non-BMP code points accepted", "Password", "\U0001f600" * 15)
check("new password over 72 UTF8 bytes rejected", "Password", "\U0001f600" * 19, False)
employee = {"name": "Employee", "email": "staff@example.com", "roles": ["PA"], "active": True, "assignedOfficerIds": [str(i) for i in range(1, 101)], "version": 1}
check("100 assignments accepted", "EmployeePatch", employee)
check("101 assignments rejected", "EmployeePatch", {**employee, "assignedOfficerIds": [str(i) for i in range(1, 102)]}, False)
check("duplicate assignments rejected", "EmployeePatch", {**employee, "assignedOfficerIds": ["1", "1"]}, False)
check("duplicate roles rejected", "EmployeePatch", {**employee, "roles": ["PA", "PA"]}, False)
check("employee patch rejects initialPassword", "EmployeePatch", {**employee, "initialPassword": "not-for-an-edit"}, False)
check("restricted update requires schedule", "RestrictedUpdate", {"version": 1, "visitKind": "restricted"}, False)
check("restricted update supports explicit clears", "RestrictedUpdate", {"version": 1, "visitKind": "restricted", "date": "2030-01-02", "time": "09:00", "officerId": "", "name": ""})
base = {"id": "1", "visitKind": "restricted", "date": "2030-01-02", "time": "09:00", "duration": 60, "room": "", "status": "approved", "version": 1, "timing": {"serverTime": 1, "opensAt": 1, "endsAt": 2, "displayStatus": "approved", "canCheckIn": True, "canCheckOut": False, "canCancel": False}, "reference": "Restricted visit", "restrictedView": "reception"}
check("restricted reception minimal allowed", "StaffBooking", base)
for key, value in {"name": "Secret visitor", "code": "01234567", "organization": "Private", "coordinationNote": "Private", "serviceId": "001"}.items():
    check("restricted reception rejects " + key, "StaffBooking", {**base, key: value}, False)
check("role patch rejects unknown grant", "RolePatch", {"version": 1, "permissions": ["root"]}, False)
check("secret response cannot expose value", "IntegrationField", {"key": "password", "label": "Password", "kind": "secret", "required": True, "configured": True, "value": "secret"}, False)
check("SMTP supported values accepted", "SmtpValues", {"host": "smtp.example.org", "port": "465", "tlsMode": "implicit_tls", "username": "staff@example.org", "sender": "staff@example.org"})
for values in [{"endpoint": "https://example.org"}, {"port": "0"}, {"port": "65536"}, {"port": "0465"}, {"tlsMode": "none"}, {"host": "https://smtp.example.org"}, {"password": "\U0001f600" * 1025}]:
    check("SMTP invalid settings " + str(list(values)), "SmtpValues", values, False)
check("SMTP maximum port", "SmtpValues", {"port": "65535"})
check("SMTP secret clear allowlist", "IntegrationPatch", {"version": 1, "enabled": False, "values": {}, "clearSecrets": ["password"]})
check("SMTP unsupported clear key", "IntegrationPatch", {"version": 1, "enabled": False, "values": {}, "clearSecrets": ["credential"]}, False)

query_schema = localize(spec["paths"]["/staff/bookings"]["get"]["x-query-schema"])
for query, valid in [({"view": "dashboard", "date": "2030-01-02", "page": 1}, True), ({"view": "dashboard", "date": "2030-01-02", "q": "private"}, False), ({"view": "reception", "mode": "checkout"}, True), ({"view": "reception"}, False)]:
    assert Draft202012Validator({"$defs": definitions, **query_schema}, format_checker=FormatChecker()).is_valid(query) == valid
    cases.append({"case": "view query " + json.dumps(query), "passed": True})

result = {"artifact": SPEC_PATH.name, "status": "planning-document-checks-passed", "official_openapi_schema": official.get("$id"), "operations_match_contract": len(actual), "unique_operation_ids": 38, "schemas_meta_schema_valid": len(schemas), "resolved_internal_refs": len(refs), "schema_cases_passed": len(cases), "cases": cases, "limitations": ["No application build, backend test, database, container or live API operation.", "OpenAPI and JSON Schema cannot enforce PG authority, stateful replay, clocks, locks, byte-length extensions or delivery behavior.", "Deferred implementation/release gates remain listed in the artifact; document validation does not verify runtime behavior or authorize implementation."]}
result.update({"selected_decisions_sequential": len(decision_ids), "diagram_entity_rows": len(diagram_rows), "artifact_sha256": hashlib.sha256(SPEC_PATH.read_bytes()).hexdigest()})
(DOCS / "planning-evidence/openapi-validation-2026-11-12.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
print(json.dumps({k: v for k, v in result.items() if k not in ["cases", "limitations"]}))
