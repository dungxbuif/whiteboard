# EXC-024: Coturn & Docker Setup

## Status

planned

## Lane

high-risk

## Product Contract

Add coturn service to docker-compose (instrumentisto/coturn, port 3478, auth-secret), update .env.example

## Relevant Product Docs

- `04-01-PLAN.md`

## Acceptance Criteria

- Coturn service configuration exists in docker-compose.yml.
- Auth secret config is added to env settings.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-04/EXC-024-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-024-design.md)

- **Modified Files**: `docker-compose.yml`, `.env.example`
- **Sprint Phase**: phase-04

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | docker compose config |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-024 in the harness database.

## Evidence

(No evidence yet)
