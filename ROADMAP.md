# AgentSearch roadmap

This roadmap describes implemented behavior and separately scoped future work. A checked item does not imply that every external service or network combination has been live-tested.

## Stage 1 — Source architecture: implemented

- [x] Typed targets and normalized results with legacy output adapters.
- [x] Capability-specific sources, registry, and source-neutral runner.
- [x] Existing website engine behind a source implementation.
- [x] Sensitive-input redaction and separate service configuration.
- [x] Website regression tests, cancellable submission/waits, and the User-Agent concurrency fix.

## Stage 2 — HIBP email lookup and English migration: implemented

- [x] Official authenticated HIBP email breach endpoint, with environment-based API keys.
- [x] Explicit `-email` mode; `-u` and `-f` remain website-only.
- [x] Normalized breach evidence and distinct no-result/error semantics.
- [x] Safe error classification, Retry-After metadata, bounded responses, and redirect protection.
- [x] Local API contract tests, redaction tests, and CLI integration tests.
- [x] English-only maintained documentation, comments, configuration guidance, and runtime text.
- [x] Repository-wide Unicode Cyrillic regression check.

HIBP email lookup does not include Pwned Passwords, email hash-range lookup, pastes, or domain/stealer-log enumeration. No real API subscription is required by the tests.

## Stage 3 — Pwned Passwords API integration: implemented

- [x] Separate password-search source using the official, key-free range API.
- [x] Local standard-library SHA-1; only five prefix characters transmitted, suffix matched locally.
- [x] Explicit `-password` and injectable, non-echoing `-password-prompt` input on Unix/Windows.
- [x] Shared consumable secret buffers, argument-risk warning, and safe output labels.
- [x] Response padding, strict bounded suffix/count parsing, and found/not-found/error semantics.
- [x] Timeout/cancellation handling, safe Retry-After metadata, no automatic retry or redirect following.
- [x] Normalized outcomes/counts through existing writers and reports; legacy CSV compatibility.
- [x] Local contract, request-boundary, hashing, prompt, leakage, and CLI regression tests.
- [x] Strict separation from email authentication and existing website behavior.

## Stage 4 — Offline Pwned Passwords database: implemented

- [x] Canonical 28-byte records, fixed headers, 20-bit offset/SHA-256 index and strict bounded manifests.
- [x] Immutable snapshots, bounded concurrent ReadAt, and checksum verification before either lookup outcome.
- [x] Strict bounded prefix-group importer with explicit artifact digest/completion attestation; no built-in acquisition.
- [x] OS updater/control locks, generation leases, redundant activation records, explicit rollback and safe pruning.
- [x] Separate corpus maintenance command; no corpus bundled, committed or automatically downloaded.
- [x] Consuming local password source through the existing capability/runner/results.
- [x] Explicit API/local CLI selection with API default and no fallback; snapshot preparation before local prompting.
- [x] Synthetic integrity, activation/concurrency, lifecycle, selection and output regressions.

Operational follow-ups: native Windows filesystem execution, actual full HIBP corpus compatibility,
power-loss testing and production-scale validation remain **UNVERIFIED**. Completion of Stage 4
implementation is not a production-readiness, secure-memory or legal certification.
See the [offline operations guide](docs/offline-passwords.md). No bulk password checking is implemented.

## Stage 5 — Unified HTTP API: implemented

- [x] Separate stdlib HTTP server over the existing app/runner and capability dispatch.
- [x] Public lightweight `GET /health` and authenticated `POST /api/v1/search` for username/email/password.
- [x] Existing normalized results, safe errors/partial observations, and upstream Retry-After mapping.
- [x] Strict bounded JSON input, password-body-only contract, secret ownership and cleanup.
- [x] Operator-selected API/local password backend, exactly one consumer and no local fallback.
- [x] Environment bearer token, loopback default, finite timeouts, request cancellation and admission bounds.
- [x] Safe request IDs/access logs, graceful shutdown and app-owned immutable snapshot lifetime.
- [x] Local transport/provider tests for routing, auth, validation, leakage, errors, cancellation and concurrency.

This is a controlled-deployment API, not a production/public multi-tenant service. TLS termination,
operator access controls and HIBP Terms review remain deployment responsibilities. No search
persistence, browser CORS, accounts, debug endpoints, bulk searches or corpus acquisition were added.
Native Windows execution, full-corpus/production-scale verification, power-loss durability and legal
suitability remain **UNVERIFIED**. See the HTTP startup, contract and limits in the [README](README.md).

## Stage 6 — Additional external intelligence: implemented

- [x] One additional provider: official read-only SecurityTrails Get Domain API.
- [x] Validated domain target and DomainSearcher capability through the existing registry/dispatcher.
- [x] Separate provider client/source with sorted normalized DNS observations and attributable metadata.
- [x] Opt-in environment credentials and centralized composition; unused providers require no key.
- [x] Existing service transport, bounded responses, no redirects/retries, cancellable pacing and safe errors.
- [x] CLI `-domain` and generic HTTP `POST /api/v1/search` with `type: "domain"`.
- [x] Focused local provider contracts and CLI/API integration tests; no live CI dependency.

Selection stopped at one useful provider. No IP/URL targets, scanning, scraping, HIBP duplication,
correlation engine, selector language, queues or AI were added. SecurityTrails domain profiles are
observations, not compromise probabilities; undocumented negative responses are never treated as
verified absence. API subscription, internal-use and redistribution restrictions apply.
Live provider verification: **NOT RUN**. Native Windows execution, production scale and legal
suitability: **UNVERIFIED**. See [external intelligence](docs/external-intelligence.md) and README
for official references, configuration, usage and limitations.

## Stage 7 — Core search contract stabilization: implemented

- [x] HTTP email uses the existing canonical validator, including its post-trim length limit.
- [x] Dispatcher preserves safe context-error identity without retaining raw provider error chains.
- [x] Focused CLI/HTTP target parity and runner/error-status regressions.

No new provider, schema, confidence rule, CLI flag, password workflow or dependency was added.
Legacy website-input compatibility and stricter HTTP input bounds remain intentional.

## Later integrations — planned, not implemented
- [ ] Optional AI analysis of normalized results.
- [ ] Further external providers through capability-specific interfaces.

Existing CLI/HTML reports, license-dependent DOCX generation, website retries, proxy rotation, and WAF detection remain present. Their inherited limitations, including output paths and error propagation, are documented in the [README](README.md), not presented as new integrations.
