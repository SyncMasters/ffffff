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

## Stage 8 — HTTP API operational hardening: implemented

- [x] Stop accepting before draining; preserve active requests until completion or grace expiry.
- [x] Positive `-shutdown-timeout` setting, retaining the existing 10s default grace.
- [x] Force context cancellation and connection closure on expiry; avoid unbounded command cleanup waits.
- [x] Barrier-based shutdown and bounded-return tests; body/admission/slot-release regressions.

Existing 32 KiB bodies, fail-fast bounded search admission, transport/request timeouts and safe
HTTP errors remain. No queue, provider, target/result change, password change or dependency was added.
Native Windows runtime, live-provider and production load verification remain outside this stage.

## Stage 9 — Safe operational diagnostics: implemented

- [x] Existing slog search/source summaries, monotonic elapsed time and bounded safe labels/categories.
- [x] HTTP request correlation, admission/rejection snapshots and debug-only healthy health checks.
- [x] Remove raw targets, URLs, paths and errors from search diagnostics, including password detail.
- [x] Local outcome/redaction, timing-barrier, request-correlation and admission regression tests.

No public schema, result/report contents, provider behavior, CLI flags or Stage 8 resource/lifecycle
policy changed. No telemetry dependency, metrics endpoint or production monitoring claim is added.
Live providers, native Windows runtime and production load remain unverified.

## Stage 10 — Search orchestration and result aggregation: implemented

- [x] Preserve deterministic registry selection and synchronous single-source/sensitive paths.
- [x] Multi-source non-sensitive invocations in bounded pairs, with ordered streaming backpressure.
- [x] Join launched work on cancellation, consumer stop and caller-side panic propagation.
- [x] Preserve partial observations and registry-ordered failures; no guessed deduplication.
- [x] Barrier-based concurrency/cleanup, context, evidence and CLI/HTTP regression tests.

No provider, model/schema, credential, password, writer or HTTP-policy change. Existing website
emission order is retained; this stage makes cross-source aggregation deterministic, not website
worker completion order. Standard composition remains single-source. No throughput claim or live
provider/native Windows runtime verification is inferred from local tests and cross-builds.

## Stage 11 — Evidence and result provenance: implemented

- [x] Reuse existing source/type/status/evidence metadata; clarify legacy website attribution.
- [x] Internal, non-serialized evidence interpretation: observation, scoped absence, error, unknown.
- [x] Distinguish detector inference from corpus negatives and domain-profile observations.
- [x] Local provider, concurrent partial-evidence, output compatibility and secrecy regressions.

No public schema, timestamp, score, password evidence, provider behavior or output change.
No deduplication without canonical identity; no acquisition time inferred from normalization or
render time. Stage 9 diagnostics and Stage 10 orchestration remain intact. Live-provider, native
Windows, production and forensic-certainty claims are not made.

## Stage 12 — Operational reliability and failure semantics: implemented

- [x] Audit lifecycle/error ownership against actual dispatcher, providers, outputs and HTTP code.
- [x] Join website-owned workers/submission before consumer-panic propagation and HTTP recovery.
- [x] Preserve already-expired request deadline identity before admission (504, not 408).
- [x] Close acquired output writers on setup/finalization failure; check CSV header-flush errors.
- [x] Barrier regressions for active deadline, provider-local cancellation, consumer diagnostics,
  concurrent panic precedence, admission cleanup and shutdown during ordered source delivery.

Bounded-pair orchestration and Stage 8 forced-shutdown ownership remain unchanged. No new public
error taxonomy, schema, CLI flag, dependency, password/corpus behavior or provider lookup policy.
Cooperative cleanup remains required; no live-provider/native Windows/production guarantee.

## Later integrations — planned, not implemented
- [ ] Optional AI analysis of normalized results.
- [ ] Further external providers through capability-specific interfaces.

Existing CLI/HTML reports, license-dependent DOCX generation, website retries, proxy rotation, and WAF detection remain present. Their inherited limitations, including output paths and error propagation, are documented in the [README](README.md), not presented as new integrations.
