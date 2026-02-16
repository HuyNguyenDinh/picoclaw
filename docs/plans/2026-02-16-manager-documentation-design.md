# Manager Documentation Design

**Date**: 2026-02-16
**Status**: Implemented

## Decision

Single comprehensive `docs/manager/README.md` covering What/Why/How/Where/When.

## Approach

Option A was chosen: one well-structured document over a multi-file docs structure.

**Rationale**: Phase 1 benefits from a single source of truth. Easy to read top-to-bottom for both technical and non-technical audiences. Can split into multi-file when the project grows (Phase 2+).

## Structure

1. **What** — Architecture diagram, component overview
2. **Why** — Democratizing AI agents for non-technical users (teams, educators, service providers)
3. **How** — Full API reference with curl examples, auth, error format
4. **Where** — Three deployment paths (local, Docker, K8s) + env vars table
5. **When** — Tenant lifecycle, monitoring tips, default resource limits

## Alternatives Considered

- **Multi-file docs**: Clean separation but overkill for Phase 1
- **README + OpenAPI**: Machine-readable spec useful later, verbose to maintain now
