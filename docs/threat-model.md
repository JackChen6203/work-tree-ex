# Threat Model

## Scope
Travel planning PWA covering auth, trips, itinerary, budget, AI planning, notifications, sync, and provider integrations.

## Key Assets
- user session and profile data
- trip collaboration data
- itinerary and budget records
- AI provider configuration metadata
- notification state and sync versions
- idempotency keys and versioned mutations

## Main Threats
1. Unauthorized access to trip data
2. Privilege escalation via trip member role mutation
3. Replay or duplicate mutation submission
4. Lost update / stale write during sync and patch flows
5. Prompt injection / invalid AI draft adoption
6. Leakage of provider secrets to browser storage or logs
7. Abuse of invitation / notification / OAuth callback flows
8. Offline queue divergence leading to user-visible inconsistency

## Controls Implemented In Repo
- request validation and error envelopes in HTTP layer
- optimistic concurrency with version checks on mutable resources
- `Idempotency-Key` on critical create / reorder / flush operations
- AI draft adoption separated from draft generation
- notification and sync flows covered with route-level tests
- client-side queue state surfaced clearly in UI

## Controls Still To Harden
- auth/session hardening for production cookie settings
- CSRF and rate limiting verification in deployment environment
- persistent audit logging for role changes and sensitive mutations
- stronger provider secret storage and redaction guarantees
- outbox / DLQ and notification delivery failure persistence
- share / invite token expiry and abuse monitoring

## Abuse Cases To Test
- duplicate trip creation with same idempotency key
- stale version patch on trip / itinerary / expense mutation
- repeated sync flush with same idempotency key
- invalid notification cursor / limit abuse
- malformed AI draft adoption and validation bypass attempts
