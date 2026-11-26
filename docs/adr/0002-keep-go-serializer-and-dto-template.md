# ADR-0002: Keep the Go serializer interfaces and DTO-shaped usecases

Status: accepted (2026-11-26)

## Context

An architecture review (2026-11-26) found that every `serializer/*` package defines an interface that has exactly one stateless implementation, and that usecases take and return `pkg/dto` types. A single new field (multiple employee phone numbers, ca410db) touched about 20 files. The review proposed turning serializers into plain functions and having usecases work in entities.

## Decision

Keep the template from `docs/BACKEND-GO-CONVENTIONS.md`. Do not turn serializer interfaces into functions, and do not move DTO mapping out of usecases.

## Reasons

- Each single-implementation interface costs little: the extra code is boilerplate, not hidden behaviour.
- The change would touch every backend package, with little gain in testability. The real testing gap (visit usecases with no usecase tests) is fixed by adding fakes behind the repository interfaces, which already have a second adapter in tests.

## Consequences

Future architecture reviews should not suggest this again unless the cost of the template grows, for example if serializers start holding real logic that needs its own seam.
