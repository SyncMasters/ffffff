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

## Stage 4 — Offline Pwned Passwords database: planned, not implemented

- [ ] Local database lookup through the source architecture.
- [ ] Separately reviewed storage, indexing, and database maintenance workflow.

No corpus downloader, database, index, password cache, or batch scanner is part of Stage 3.

## Later integrations — planned, not implemented

- [ ] HTTP API over the existing runner and normalized results.
- [ ] Optional AI analysis of normalized results.
- [ ] Additional external sources through capability-specific interfaces.

Existing CLI/HTML reports, license-dependent DOCX generation, website retries, proxy rotation, and WAF detection remain present. Their inherited limitations, including output paths and error propagation, are documented in the [README](README.md), not presented as new integrations.
