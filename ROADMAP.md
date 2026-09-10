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

## Stage 3 — Pwned Passwords API integration: planned

- [ ] Separate password-search capability using the official range API.
- [ ] Local hashing and suffix matching; no plaintext password transmission.
- [ ] Safe input handling, response padding, normalized evidence, and local contract tests.
- [ ] Preserve strict separation from email breach lookup.

## Later stages — planned, not implemented

- [ ] Optional offline Pwned Passwords database and a separately reviewed maintenance workflow.
- [ ] HTTP API over the existing runner and normalized results.
- [ ] Optional AI analysis of normalized results, not arbitrary network orchestration.

## Maintenance priorities

- [ ] Correct website retry/proxy/redirect composition and compressed-response handling.
- [ ] Validate and harden experimental uTLS combinations.
- [ ] Improve limiter isolation between hosts.
- [ ] Make report paths consistently honor the output directory and propagate write failures.
- [ ] Improve summary aggregation and large-result report memory use.

Existing CLI/HTML reports, license-dependent DOCX generation, website retries, proxy rotation, and WAF detection are already present. Their known limitations are documented in the [README](README.md); they are not new planned integrations.
