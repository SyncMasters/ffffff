# AgentSearch

AgentSearch is a Go command-line tool with an optional operator HTTP API for authorized account discovery and exposure checks. It combines configurable website profile searches with authenticated email breach lookups and key-free Pwned Passwords range checks through [Have I Been Pwned (HIBP)](https://haveibeenpwned.com/), using a common result model and output pipeline.

It is intended for defensive investigations, personal exposure checks, and security research where you have permission to query the target. A match is evidence to review, not proof of identity or of a currently compromised account.

## Capabilities

| Status | Capability |
| --- | --- |
| Implemented | Website searches using native YAML/JSON, Sherlock, and Maigret site databases |
| Implemented | HIBP email breach lookup using an API subscription key |
| Implemented | Single-password Pwned Passwords API lookup with local SHA-1 and a no-echo prompt |
| Implemented | Fixed website worker pool, cancellable host delays, request timeouts, direct-request retries, proxy rotation, and User-Agent rotation |
| Implemented | Status, message, regex, redirect, header, and WAF detection rules |
| Implemented | Shared JSON, CSV, and TXT output; CLI and HTML reports |
| License-dependent | DOCX reports through the existing UniOffice dependency |
| Experimental | Website uTLS support; transport combinations have known limitations |
| Implemented | Explicit offline Pwned Passwords backend with immutable snapshots and separate corpus maintenance |
| Implemented | Authenticated, bounded HTTP searches over the same username/email/password engine |
| Planned | AI analysis and additional external sources |

**For password checks, use `-password-prompt`. Never put passwords in website targets, target files, email input, or service YAML.** The `-d` flag is retained for compatibility but remains a no-op stub.

## Installation

Requirements: Go 1.24 or later, and network access to the sources you query. DOCX generation additionally requires a valid UniOffice license.

```sh
git clone https://github.com/johan-larp/AgentSearch.git
cd AgentSearch
go build -o agentsearch ./cmd/agentsearch
go build -o convert ./cmd/convert
```

The examples below run from the repository root so relative configuration paths resolve. On Windows, build/use `agentsearch.exe` and `convert.exe` as appropriate.

## Choose a search mode

### Website search

```sh
./agentsearch -u johndoe
./agentsearch -f targets.txt -s configs/sites.yaml
./agentsearch -u johndoe -f additional-targets.txt -w 30 -rl 1s
```

`-u` and `-f` keep their original behavior: they search the configured websites. If both are supplied, their targets are appended. Target files contain one value per line; blank lines and lines beginning with `#` are ignored.

An email-like value passed through `-u` or `-f` is still substituted into website templates. It **does not** trigger HIBP, even if HIBP is enabled in the services file.

### HIBP email breach lookup

```sh
./agentsearch -email user@example.com
```

This mode registers only the HIBP source; it does not load the website database, create website workers, or query websites. It requires the configuration steps below. There is no email subcommand or batch email option in this release: use the explicit `-email` flag.

## Configure HIBP email lookup

1. Obtain an appropriate subscription key from the [official HIBP API key page](https://haveibeenpwned.com/API/Key).
2. Enable HIBP in `configs/services.yaml`:

   ```yaml
   services:
     hibp:
       enabled: true
       api_url: "https://haveibeenpwned.com/api/v3"
       api_key_env: "HIBP_API_KEY"
   ```

   The supplied example is disabled by default. `api_key_env` is an environment variable **name**, never a key value. If omitted, it defaults to `HIBP_API_KEY`; an omitted `api_url` uses the official endpoint. Unknown YAML fields, including a literal `api_key` field, are rejected.

3. Provide the key through your environment. These are placeholders, not credentials:

   ```powershell
   # PowerShell
   $env:HIBP_API_KEY="YOUR_API_KEY"
   .\agentsearch.exe -email user@example.com
   Remove-Item Env:HIBP_API_KEY
   ```

   ```sh
   # POSIX shell
   export HIBP_API_KEY='YOUR_API_KEY'
   ./agentsearch -email user@example.com
   unset HIBP_API_KEY
   ```

   For real use, prefer a protected session or your existing secret-injection mechanism rather than typing a key into a recorded shell. Do not publish environment dumps, screenshots, shell history, or CI logs containing credentials. There is no API-key CLI argument and no automatic `.env` loader.

4. Optionally select a different services file and output options:

   ```sh
   ./agentsearch -email user@example.com -services my-services.yaml -rt 15s -of json,csv,txt -rf cli,html -o email-results
   ```

The services file is loaded automatically for `-email`, or when explicitly passed with password mode. Legacy website modes never load it. Website-specific flags such as `-u`, `-f`, `-s`, `-p`, `-w`, `-rl`, `-ua`, `-utls`, and `-retries` cannot be combined with `-email`. HIBP uses a standard TLS client and an identifying AgentSearch User-Agent, not browser impersonation or proxy rotation. It does not use environment proxy settings.

### HIBP protocol and result semantics

AgentSearch calls the documented REST endpoint:

```text
GET /api/v3/breachedaccount/{URL-encoded email}?truncateResponse=false
```

It sends the required `hibp-api-key` header, an identifying User-Agent, and `Accept: application/json`. It does not scrape the HIBP website, bypass authentication, or call undocumented endpoints.

- Surrounding address whitespace is trimmed; case and plus tags are preserved. Only a single bare email address is accepted. Display names, internal whitespace, and obviously malformed addresses are rejected before a request. This is not a deliverability check.
- One API request returns the breach metadata. AgentSearch emits **one `found` result per breach**.
- HTTP 404 or HTTP 200 with an empty array emits one `not_found` result. This means no breaches were returned by this endpoint, not that the address has never been exposed.
- Authentication, rate-limit, server, network, timeout, cancellation, and invalid-response failures are represented as `error`, not `not_found`. Email mode exits nonzero on a lookup failure after writing the error result. Missing configuration or a missing key fails during initialization, before a lookup.
- For a match, `confidence: 100` means the API returned an exact association. It is **not** a probability of account compromise and does not override `verified` or `fabricated` metadata.
- Allowed metadata includes breach name/title, domain, breach/added/modified dates, verification/fabrication flags, and the number of breaches returned. Compromised data classes are separate evidence items. Descriptions, logos, raw bodies, and response headers are not retained.
- The source is `hibp`, its provenance type is `api`, and its target type is `email`. The legacy `site_name` field displays `Have I Been Pwned / <breach name>` so older writers remain useful.

Failure metadata uses `error_kind` values such as `unauthorized`, `forbidden`, `rate_limited`, `bad_request`, `service_unavailable`, `network_failure`, `timeout`, `cancelled`, or `invalid_response`, plus `http_status` where available.

### Rate limits, privacy, and coverage

HIBP limits depend on the subscription. AgentSearch does not assume a requests-per-minute limit and **does not automatically retry HIBP requests**, including 401, 403, 429, and 5xx responses. A valid `Retry-After` on 429/503 is preserved as `metadata.retry_after_seconds`; wait at least that long before another lookup. No automatic cooldown or cross-process quota coordinator is implemented.

The complete email address is sent to the configured API endpoint. This stage does not implement HIBP's email hash-range API. The public email endpoint excludes certain records, including sensitive and retired breaches, and follows HIBP's opt-out and access policies. Unverified results are not filtered out by AgentSearch. Read the [official API documentation](https://haveibeenpwned.com/API/v3) for current coverage, subscription, and acceptable-use requirements.

A custom `api_url` is useful for local testing but receives the key and target. Treat this configuration as trusted. HTTPS is required, except for HTTP endpoints on literal loopback IP addresses used in local tests. URLs containing credentials, query strings, or fragments are rejected. Redirects are not followed, preventing the custom API-key header from being forwarded to another endpoint.

HIBP breach data is attributed to **[Have I Been Pwned](https://haveibeenpwned.com/)** and is subject to its [Creative Commons Attribution 4.0 license](https://creativecommons.org/licenses/by/4.0/). Results link to HIBP and retain its name. Keep that attribution when sharing derived reports. Pwned Passwords is a separate, key-free integration described below, not part of this email lookup. Email subscription and attribution terms should not be assumed to describe the password service.

## Pwned Passwords: check one password

### Recommended: interactive, non-echoing input

```sh
./agentsearch -password-prompt
./agentsearch -password-prompt -rt 15s -tt 2m -o password-results -rf cli,html
```

```powershell
.\agentsearch.exe -password-prompt
```

Enter the complete password and press Enter. **No HIBP API key, subscription configuration, or services file is required.** An explicit password flag opts into this lookup; `services.hibp.enabled` controls email lookup only. Neither `HIBP_API_KEY` nor the configured email key is resolved or transmitted in password mode.

The prompt uses `golang.org/x/term` for normal Unix terminals and Windows consoles. It disables terminal echo synchronously, suppresses all line-editor output, and restores the original terminal settings before returning on completion, input failure, Ctrl+C, SIGINT/SIGTERM, or the total timeout. It refuses redirected stdin/pipes: there is no password-file, batch, wordlist, or stdin scanning mode. The prompt callback is injectable for automated tests.

Input is a non-empty single line; spaces, case, and Unicode are not trimmed or normalized before hashing. Enter terminates input and editing keys perform terminal editing. The prompt has a 4096-byte **input budget**, including Enter and editing/escape bytes (at most 4095 unedited password bytes). Exceeding it fails instead of silently accepting a truncated password. A canceled, blocked console reader may remain until this single-shot CLI exits; it cannot alter terminal settings or transmit a late result. Forced termination such as SIGKILL, console failure, or a process crash cannot guarantee terminal restoration.

### Argument form: supported, not recommended for real secrets

```sh
# Public demonstration fixture only; never use this phrase as a real password.
./agentsearch -password "correct horse battery staple"
```

**`-password` exposes its argument to shell history, process listings, terminal logs, and host auditing.** AgentSearch emits a warning but cannot erase those external records or reliably erase the immutable process argument. Prefer `-password-prompt`. Application headings, logs, filenames, results, and completion messages never use the password as their target label.

Choose exactly one password flag. Do not combine it with `-u`, `-f`, `-email`, website networking options, or positional arguments. Allowed options are `-services`, `-rt`, `-tt`, `-o`, `-of`, and `-rf`; timeouts must be positive. Password-mode invalid flags, repeated argument input, and malformed flag values produce safe diagnostics without echoing supplied values. Unknown password output/report formats are configuration errors.

### Local hashing and k-anonymity

1. CLI input enters a shared, redacted `security.Secret` handle, never `AppConfig.Targets` or a generic result field.
2. The application runner dispatches to the separate `PasswordSearcher` capability.
3. The source computes SHA-1 locally with Go's standard `crypto/sha1`, encodes uppercase hexadecimal, and clears its application-owned plaintext buffer **before networking**. SHA-1 is required for this protocol, not recommended for storing passwords.
4. The client sends only the first **five hexadecimal characters**:

   ```text
   GET https://api.pwnedpasswords.com/range/{five-character-SHA1-prefix}
   Add-Padding: true
   ```

   There is no password, full hash, suffix, query string, request body, API-key header, authorization, or cookie in the lookup request. An identifying User-Agent and `Accept: text/plain` are sent. It makes one request after complete input, never incremental queries while typing.
5. The remaining **35-character suffix** is compared locally against the returned range. Prefix, suffix, and full hash are not included in normalized results, diagnostics, filenames, or reports.

This is **k-anonymity**, not zero disclosure or protection from a compromised endpoint/device. The service still sees a hash prefix and connection metadata such as the client IP. Padding requests reduce response-size disclosure, but do not prove anonymity. See the [official range API documentation](https://haveibeenpwned.com/API/v3#PwnedPasswords).

The parser validates the entire bounded response, including records after a match. Each non-blank line must contain exactly a 35-hex-character suffix, a colon, and a non-negative decimal count that fits `uint64`. Suffix matching is case-insensitive. LF and CRLF are accepted; empty lines are ignored, but whitespace-only lines and empty/blank-only bodies are errors. Malformed lines, extra separators, signs/whitespace in counts, numeric overflow, duplicate suffixes, oversized lines, or responses over 2 MiB fail the lookup. Zero-count padding records never count as pwned. No fixed number of range rows is assumed.

### Outcomes and output

| Outcome | Normalized status | Safe metadata | Exit |
| --- | --- | --- | --- |
| **PWNED** | `found` | `pwned: "true"`, positive `occurrences` | `0` |
| **NOT PWNED** | `not_found` | `pwned: "false"`, `occurrences: "0"` | `0` |
| **ERROR** | `error` | Safe classification; no clean-password assertion | nonzero |

HTTP 200 alone does not mean pwned. A valid range with no matching suffix, or a matching zero-count padding record, is a successful negative result. Unlike the email API, a range HTTP 404 is an **error**, not a negative lookup. Configuration, input, network, timeout, cancellation, API, and parse failures fail the command. Existing output-manager error-propagation limitations still apply.

Human output includes the outcome, source, method `sha1-k-anonymity`, and occurrence count when available. JSON retains safe structured metadata; metadata values are strings for compatibility with the common model. The source is `pwned-passwords`, source type is `api`, target type is `password`, and target label is `[REDACTED]`. A positive exact corpus match uses the existing confidence value `100`; this is **not a probability of account compromise**. No `Result.Password` field is introduced.

Common TXT, CLI, HTML, and DOCX paths render semantic details; CSV preserves its original nine columns and therefore does not carry the count/method metadata. Output names use `[REDACTED]`, never the input or its hash. Multiple password runs in the same directory overwrite that safe label; choose separate output/working directories to preserve results.

**Corpus membership does not prove that any account was compromised.** NOT PWNED does not establish password strength, uniqueness, or safety; it only means no positive exact match was returned by the current corpus lookup. An API error means unknown, not safe.

### Networking and optional configuration

The password client is structurally separate from the authenticated email client, while sharing endpoint validation, safe error handling, and standard service transport. No website workers, proxies (including environment proxies), uTLS, cookie jar, or automatic retries are used. All redirects, including same-origin redirects, are rejected. Valid `Retry-After` on 429/503 is retained as seconds; no quota or aggressive retry schedule is invented. Respect the provider's current terms and wait before another lookup after a rate-limit response.

Only if explicitly supplying `-services`, the following optional field selects an endpoint:

```yaml
services:
  hibp:
    enabled: false # Email setting; does not disable explicit password mode.
    passwords_api_url: "https://api.pwnedpasswords.com/range"
```

```sh
./agentsearch -password-prompt -services configs/services.yaml
```

Use the official URL in production. A custom endpoint is trusted configuration and receives the prefix. HTTPS is required except literal-loopback HTTP for local tests. The path must be `/range`; credentials, query strings, and fragments are rejected. There are no default mirrors, scraping, automatic corpus downloads, or query caches. Stage 4 adds the separately selected offline backend below.

## Offline Pwned Passwords (Stage 4)

The API remains the default for `-password-prompt` and `-password VALUE`.
Offline lookup requires explicit selection and an operator-prepared database:

```sh
./agentsearch -password-prompt -password-backend local -password-db /srv/pwned-db
```

On Windows, use `agentsearch.exe -password-prompt -password-backend local -password-db C:\Data\Pwned`.
The path names a managed root, not an export file. `-password-db` alone is an error.
Local mode never downloads data or falls back to the API; unavailable or corrupt
local data returns an error. Prompt mode prepares the snapshot before asking for input.

Use the separate `agentsearch-corpus` command to import a completed, operator-owned
SHA-1 hash/count artifact, verify it, explicitly activate it, roll back, or prune
unreferenced/unleased generations. No corpus is bundled or committed. No password
lists or bulk checking are supported. See the **[offline setup and operations guide](docs/offline-passwords.md)**
for acquisition attestation, exact formats, commands, disk sizing and recovery.

Local results use source `pwned-passwords-local`, type `local`, method `sha1-offline`
and the existing redacted target/status/metadata system. Both a match and verified
absence use confidence **100** relative to the selected supplied dataset; the
existing remote negative confidence remains **0**. Technical errors never mean
not pwned. Membership is not proof of account compromise, and absence is not proof
of safety. Native Windows runtime, real-corpus compatibility and power-loss behavior
remain operationally unverified; no legal or production-readiness certification is implied.

## CLI reference

Run `./agentsearch -h` for current help. Durations use Go syntax such as `250ms`, `15s`, or `10m`.

| Flag | Default | Purpose / scope |
| --- | --- | --- |
| `-u` | unset | One website-search target |
| `-f` | unset | File of website-search targets |
| `-s` | `configs/sites.yaml` | Website database |
| `-p` | unset | Website proxy list |
| `-w` | `50` | Website worker count; must be positive |
| `-rl` | `500ms` | Website per-host delay; `0s` disables waiting |
| `-rt` | `15s` | HTTP request timeout; positive in email/password modes |
| `-tt` | `10m` | Deadline across the entire run, not per target |
| `-o` | `output` | Result directory; see the report-directory limitation below |
| `-of` | `json,csv,txt` | Comma-separated output formats |
| `-rf` | `cli,html` | Comma-separated report formats: `cli,html,docx`; empty disables reports |
| `-ua` | unset | Extra website User-Agent values, one per line |
| `-mc` | `500` | Website idle connection pool limit |
| `-mch` | `100` | Website per-host connection limit |
| `-utls` | `false` | Experimental website TLS fingerprint support |
| `-retries` | `2` | Website retry limit; see existing retry limitations below |
| `-d` | `false` | Retained website deep-search stub; no implemented action |
| `-email` | unset | Single HIBP email lookup, separate from website mode |
| `-password` | unset | One password range lookup; argument exposure warning; prefer prompt |
| `-password-prompt` | `false` | Interactive non-echoing single-password input |
| `-password-backend` | `api` | Explicit `api` or `local`; no automatic fallback |
| `-password-db` | unset | Managed local database root; requires local backend |
| `-services` | `configs/services.yaml` for email | Loaded for email; optional explicit password settings |

## Website databases and detection

### Native configuration

`configs/sites.yaml` contains 20 example sites, not a comprehensive or continuously verified database. Native databases use an array of `SiteConfig` records:

```yaml
- name: ExampleForum
  url: https://forum.example.com/users/{username}
  check_type: message
  absence_strs:
    - "User not found"
  presence_regexes:
    - "Member since [0-9]{4}"
  weight: 15
  tags: [forum]
  waf_indicators:
    status_codes: [403]
    body_substrings: ["verification required"]
```

The same native array can be written as JSON. The loader also supports native JSON maps and wrapped JSON site collections.

| Fields | Interpretation |
| --- | --- |
| `name`, `url` | Display name and target URL; entries without a URL are skipped |
| `url_probe` | Alternate check endpoint; takes precedence over `url` |
| `request_method`, `request_payload` | HTTP method and optional payload; default method is GET |
| `request_head_only` | Changes GET to HEAD |
| `headers` | Custom request headers; never put secrets in committed configuration |
| `check_type` | `status_code`, `message`, `response_url`, or `header` |
| `error_code` | Status indicating absence |
| `absence_strs`, `absence_regexes` | Body absence indicators; checked before presence indicators |
| `presence_strs`, `presence_regexes` | Body presence indicators |
| `error_url`, `redirect_failure_patterns` | Final-URL absence patterns |
| `follow_redirects` | Redirect policy; the native bool defaults to false, with a known retry-wrapper interaction |
| `required_headers` | Expected response headers; the existing detector accepts any matching pair |
| `waf_indicators` | `status_codes`, `header_keys`, and `body_substrings` |
| `protection` | Explicit protection markers used by the detector |
| `disabled` | Excludes the site from execution |
| `weight` | Base confidence contribution |
| `tags` | Retained site categories |
| `regex_check` | Retained username pattern; not currently enforced |
| `url_main`, `engine`, `username_claimed`, `username_unclaimed`, `alexa_rank`, `source` | Reference/conversion metadata; no full upstream engine emulation |

`{username}` and `{target}` are substituted in URLs and request payloads. Header values substitute `{username}` only. Detection strings and regular expressions are currently used literally, without target-template expansion. The processor limits response bodies to 128 KiB.

WAF/protection rules run first, followed by redirect failure patterns and the selected check type. Existing heuristics flag Cloudflare headers, HTTP 429, and known protection-page strings. Broad heuristics can yield false positives. Message checks without a clear presence/absence indicator can still produce a low-confidence match. Review results rather than treating `found` as a certainty.

### Sherlock and Maigret

```sh
./agentsearch -u johndoe -s sherlock_data.json
./agentsearch -u johndoe -s maigret_data.json
./convert -o merged_sites.yaml sherlock_data.json maigret_data.json
./agentsearch -u johndoe -s merged_sites.yaml
```

The existing converters translate external field names, normalize Sherlock `{}` placeholders, support scalar/array `errorMsg`, merge Maigret `presenceStrs` and its legacy `presenseStrs` spelling, and preserve supported request settings. Disabled records are excluded. The conversion command merges databases and distinguishes conflicting names with different URLs.

Compatibility covers the mapped site schema, not every upstream feature: engine inheritance and some engine-specific URL placeholders are not fully supported. Native fields explicitly present in external records override converted aliases.

## Website networking

### Proxies

Use only proxies you control or trust:

```sh
./agentsearch -u johndoe -p proxies.txt -w 30 -rl 1s
```

Supported input schemes are HTTP, HTTPS, SOCKS5, and SOCKS5h; an omitted scheme defaults to HTTP. Blank lines and comments are ignored. Example syntax:

```text
http://proxy.example.com:3128
https://proxy.example.com:443
socks5://proxy.example.com:1080
socks5h://proxy.example.com:1080
http://USER:PASSWORD@proxy.example.com:3128
```

The last line illustrates placeholders only. Keep credential-bearing proxy files outside version control and restrict their permissions. Rotation is round-robin. The existing proxy transport clones its transport per request; it does not share the direct retry path. A legacy programmatic ProxyScrape helper exists but is not automatically called by the CLI.

### Limits, retries, and cancellation

- Website workers apply the per-host delay before a check. Pending waits and submissions are cancellable.
- Direct website requests use the existing retryablehttp policy for eligible 429, 5xx, and network failures. There are no retries for invalid HIBP credentials because HIBP does not use this wrapper.
- The legacy website retry function treats a nonpositive `-retries` value as the default of two, not as “disable retries.”
- SIGINT/SIGTERM cancels active work. Already-delivered results are retained and streaming JSON is closed. Some in-flight website results may be discarded during cancellation.
- The legacy website CLI can log a target failure without a failing final exit code. Email-mode lookup failures return nonzero. Do not rely solely on the legacy website exit status when validating a report.

## Outputs and reports

All sources use the same `models.Result` and common writers. No HIBP-specific serializer or report generator is required.

- **JSON:** the full normalized result array. Existing fields (`site_name`, `target`, `url`, `found`, `confidence`, `status`, `duration`, `error`, `final_url`) remain, alongside `source`, `source_type`, `target_type`, `evidence`, and `metadata`. Duration is in nanoseconds; optional fields may be absent.
- **CSV:** the original nine-column projection; duration is in milliseconds:

  ```text
  site_name,target,url,found,confidence,status,duration_ms,error,final_url
  ```

- **TXT:** per-result lines with status/source and normalized evidence/metadata. CSV retains the legacy projection; use JSON for complete structured detail.
- **CLI:** an English terminal summary and plain-text report, including safe semantic details.
- **HTML:** self-contained report with filters, status cards, evidence, and metadata. Remote text is escaped rather than interpreted as HTML.
- **DOCX:** retained Word report generation with normalized semantic details, subject to UniOffice licensing. The license-dependent test skips explicitly if the license is unavailable.
- **Summary JSON:** counts of emitted results, generated when report formats are requested. With HIBP, two returned breaches count as two found results, not two HTTP requests.

For example, a HIBP match includes:

```json
{
  "source": "hibp",
  "source_type": "api",
  "target": "user@example.com",
  "target_type": "email",
  "site_name": "Have I Been Pwned / ExampleBreach",
  "url": "https://haveibeenpwned.com/",
  "status": "found",
  "found": true,
  "confidence": 100,
  "duration": 120000000,
  "metadata": {
    "breach_name": "ExampleBreach",
    "verified": "true",
    "fabricated": "false",
    "breach_count": "1"
  },
  "evidence": [
    {"kind": "compromised_data_class", "value": "Email addresses"}
  ]
}
```

Website/email filenames are based on the target with unsafe filename characters replaced. Password outputs instead use the fixed `[REDACTED]` label. **Known limitation:** result files and summary JSON honor `-o`, but CLI/HTML/DOCX generators still write into `output/`. Runs using the same target can overwrite previous output; use separate working/output directories when preserving investigations. Existing output/report managers log some write failures rather than propagating them to the process exit status.

## Security considerations

- Query only targets you are authorized to investigate. Respect applicable law, source terms, and HIBP's acceptable-use policy.
- Email addresses and breach associations are personal information. They appear in local output and filenames. Protect and retain that output appropriately; it is not encrypted by AgentSearch.
- Only environment variable names belong in service YAML. Never commit keys or credential-bearing proxy files.
- `security.Secret`, sensitive-key filtering, known-secret redaction, and URL-userinfo redaction protect normal diagnostic and result paths. HIBP errors contain safe classifications, not raw transport errors, bodies, or headers. The HIBP client does not log requests or responses.
- Redaction is a safety net, not a general detector for every possible secret representation. Do not add HTTP request dumps or place secrets into evidence. Go strings also cannot be reliably erased from memory.
- Password input is kept in a separate shared secret handle; consuming the handle clears the application-owned byte buffer before the range request and on normal cleanup paths. Do not call `Reveal` for password dispatch or add plaintext to results. API-key callers retain explicit reusable access.
- This is best-effort lifetime control, **not locked/secure memory**. Go runtime copies, prompt-library strings, original CLI arguments, swap, crash dumps, and host-level instrumentation cannot be guaranteed erased. Prompt mode avoids passing the password as an argument but does not secure a compromised host.
- Password results contain a fixed redacted target and allowlisted semantic fields only. Password-hash CLI input and bulk scanning are not implemented. Offline lookup is explicit and uses operator-owned data. Protect occurrence metadata and any reports you keep.

## Architecture

```text
CLI flags
  -> application composition and target parsing
  -> Runner -> source registry / capability dispatcher
       |-- websites -> workers -> host limiter -> existing HTTP client -> detector
       |-- hibp -> authenticated email breach client -> standard service HTTP transport
       `-- password backend (exactly one) -> local SHA-1 / secret consumption
              |-- api -> key-free range client -> local suffix match
              `-- local -> immutable corpus -> verified range -> binary search
  -> normalized, redacted results
  -> common storage and reports
```

The website source supports username and legacy email-template searches. HIBP has separate email (`EmailSearcher`) and password (`PasswordSearcher`) source types; neither implements the other capability. Source registration is mode-specific, preventing accidental HIBP calls from legacy commands. Sources own their concurrency; one email or password lookup does not create a worker pool. The password source consumes its shared input once; the CLI registers exactly one password provider, not multiple plaintext consumers.

`app.NewRunner(registry).Search(ctx, target, emit)` is the source-neutral execution boundary. The dispatcher derives capabilities from small interfaces. It preserves partial results, stops on consumer errors, and does not import storage or reports. Models depend only on the standard library and the small security package. The network layer does not depend on concrete sources. HTTP clients and secrets are supplied at the application composition boundary.

## Unified HTTP API (Stage 5)

The separate `agentsearch-server` command reuses app/runner, capability dispatch, and the existing
website, HIBP email, Pwned Passwords API and local sources. The CLI and its flags are unchanged.
The server does not run CLI output/report orchestration or persist submitted searches.

### Start and configure

Run from the repository root, or provide absolute operator-owned configuration paths:

```sh
go build ./cmd/agentsearch-server
# Generate an operator token; keep it out of Git, tickets and shared shell history.
export AGENTSEARCH_API_TOKEN="$(openssl rand -hex 32)"
./agentsearch-server -listen 127.0.0.1:8080 -s configs/sites.yaml
```

`GET /health` is public and returns `{"status":"ok"}`. It is liveness only: no network searches,
corpus re-verification or readiness/coverage certification. Search requires a single
`Authorization: Bearer <token>` header. The token must contain 32–4096 printable non-space ASCII
bytes; generate it randomly, not from a memorable phrase. It is read from the named environment
variable, never a literal command-line option. Only a SHA-256 comparison digest is retained by
the HTTP handler; comparisons are constant-time. The process environment still contains the
operator-supplied token. Restart to rotate it.

| Server option | Default | Meaning |
| --- | --- | --- |
| `-listen` | `127.0.0.1:8080` | TCP listen address; explicit opt-in to other interfaces |
| `-token-env` | `AGENTSEARCH_API_TOKEN` | Environment variable holding the API bearer token |
| `-read-timeout` | `15s` | HTTP read limit; header reads additionally capped at 5s |
| `-write-timeout` | `75s` | HTTP write limit; must exceed request timeout |
| `-idle-timeout` | `60s` | Keep-alive idle limit |
| `-request-timeout` | `60s` | Request-derived search deadline, including body parsing |
| `-max-concurrent` | `8` | Admitted searches, including body reads; range 1–128 |
| `-s` | `configs/sites.yaml` | Trusted site database; `-s ''` disables username search |
| `-services` | empty | Explicit existing services YAML; enables email only when HIBP is enabled |
| `-password-backend` | `api` | Exactly one password provider: `api` or `local` |
| `-password-db` | empty | Local managed database root, or resolve from explicit services YAML |
| `-w`, `-rt`, `-rl`, `-retries` | `10`, `15s`, `500ms`, `2` | Existing website workers, provider HTTP timeout, website per-host delay and website retries |
| `-p`, `-ua`, `-utls` | empty, empty, false | Existing website proxy/User-Agent files and experimental uTLS |

Email uses the existing `services.hibp.enabled`, `api_url` and `api_key_env` settings described
above. Missing/disabled email returns 503; explicitly enabled email with missing credentials or
invalid configuration fails startup. Password API requests remain key-free and prefix-only;
email keys are not sent to the password endpoint. Custom upstream URLs are operator-only and
retain the existing HTTPS/loopback and no-redirect restrictions. Clients cannot choose URLs,
files, providers or backends.

For an already imported and activated offline snapshot:

```sh
./agentsearch-server -s '' -password-backend local -password-db ./pwned-db
```

Local password lookup is offline, integrity checked and has **no API fallback**. Missing or
invalid selected data fails startup; a lookup-time integrity/capacity failure returns 503,
never `not_found`. The server owns one immutable snapshot, uses the existing bounded concurrent
readers and closes it after in-flight searches drain. Activating a new generation does not
refresh an already running server: restart to select it. No corpus download or maintenance
endpoint is exposed. See [offline operations](docs/offline-passwords.md).

### Search requests and responses

Only `POST /api/v1/search` accepts searches. Send one UTF-8 JSON object with exactly `type` and
`target`, or exactly `type` and `password`. Unknown, duplicate, mixed and null fields are rejected.
All query parameters are rejected. GET searches and automatic path redirects are not supported.

```sh
curl http://127.0.0.1:8080/health

curl http://127.0.0.1:8080/api/v1/search \
  -H "Authorization: Bearer $AGENTSEARCH_API_TOKEN" \
  -H 'Content-Type: application/json' \
  --data-binary '{"type":"username","target":"example"}'

curl http://127.0.0.1:8080/api/v1/search \
  -H "Authorization: Bearer $AGENTSEARCH_API_TOKEN" \
  -H 'Content-Type: application/json' \
  --data-binary '{"type":"email","target":"user@example.com"}'

# Synthetic demonstration only, NOT a real password.
curl http://127.0.0.1:8080/api/v1/search \
  -H "Authorization: Bearer $AGENTSEARCH_API_TOKEN" \
  -H 'Content-Type: application/json' \
  --data-binary '{"type":"password","password":"example-password"}'
```

These shell examples are demonstrative: literal bodies and expanded headers can be visible in
shell history/process arguments. For real secrets, use a client that reads input without echo
and submits the body and authorization header in memory, without debug dumps or persistence.
Never place a password in a URL, username/email target, filename or request-ID header.

Bodies are limited to **32 KiB**, headers to the server's **16 KiB** setting, and compressed request
bodies are unsupported. Usernames are 1–256 UTF-8 bytes and permit letters, digits, `_`, `-`, `.`
(excluding `.` and `..` alone), not arbitrary URLs or template syntax. Email input is at most
254 UTF-8 bytes and uses the existing single bare-address validator. Passwords are 1–4096 UTF-8
bytes after JSON decoding: no trimming, case changes or normalization. Unpaired Unicode
surrogate escapes are rejected instead of silently changing a password.

HTTP 200 returns `{"results":[...]}` containing the existing normalized `models.Result`:
`source`, `source_type`, `target_type`, `target`, `site_name`, `url`, `status`, `found`, `confidence`,
`duration`, and optional `evidence`, `metadata`, `error`, `final_url`. Duration is integer
nanoseconds, not milliseconds. A successful empty website selection returns `{"results":[]}`.
A successful negative password observation has a redacted target, `status: "not_found"`,
`found: false` and `metadata.occurrences: "0"`. Confidence remains 0 for an API negative and
100 for a verified local negative; neither proves that a password is safe.

Errors have the predictable form:

```json
{"error":{"code":"unauthorized","message":"valid bearer token required"}}
```

Operational failures can include `results` with partial observations; callers must inspect
both the HTTP status/error and individual result statuses. A failing result uses generic error
text with diagnostic URLs/evidence/metadata removed. Raw provider errors, filesystem paths,
upstream response bodies and credential details are not serialized as error messages.
A `blocked` website observation remains a normalized observation, not a caller-auth failure.
Responses are buffered, not streamed, with a 10,000-result cap (503 on exceeding it).

| HTTP status | Meaning |
| --- | --- |
| 200 | Health or completed search, including found/not-found/blocked observations |
| 400 | Invalid JSON, shape, type, target, forbidden query parameters or unreadable body |
| 401 | Missing/invalid API bearer token; not an upstream HIBP credential error |
| 404 / 405 | Unknown endpoint / unsupported method (with `Allow`) |
| 408 | Client cancellation if still writable, or body-read timeout |
| 413 / 415 | Oversized body / unsupported content type or content encoding |
| 429 | Admission capacity exhausted or upstream rate limiting; safe `Retry-After` when available |
| 500 | Unexpected internal failure; no panic values or stack traces returned |
| 502 | Upstream network, response or other lookup failure |
| 503 | Disabled search, upstream credential/access/unavailability, local failure or result cap |
| 504 | Search/provider deadline exceeded |

A disconnected client may receive no response. Context cancellation reaches the existing engine;
there is no detached background search. Shutdown cancels requests and allows a 10s HTTP grace
period before closing connections, then drains app-owned searches before closing the snapshot.
Ordinary kernel file reads are not guaranteed context-preemptible.

### Deployment and privacy limits

This is a **controlled-deployment API**, not a public multi-tenant service. It has one shared
operator token, not accounts, RBAC, quotas or a complete abuse-prevention system. There is no
built-in TLS: retain loopback binding or deploy behind an access-controlled TLS reverse proxy.
If exposing another interface, configure firewall/access controls and HTTPS before use. Do not
configure proxies, load balancers, clients or observability tools to log request bodies,
authorization headers or query strings; invalid incoming URLs can contain secrets even though
this server never echoes them. Query rejection cannot erase copies made by an earlier proxy.

The transport logs only server-generated request ID, fixed route label, status and elapsed time;
it ignores incoming request IDs and never logs arbitrary paths, bodies or headers. Existing
website-engine logs may identify username targets, so protect those logs as personal data.
Passwords stay on the password-only dispatch path and are never website targets. Request buffers
and consumable `security.Secret` handles are cleared/destroyed on success, error, cancellation
and consumer failure. JSON, Go runtime, HTTP/TLS and client buffers can create copies: this is
best-effort lifecycle management, **not guaranteed secure-memory erasure**.

No wildcard CORS, cookies/sessions, debug/pprof/config/filesystem endpoints, persistent search
storage, auto-reload, AI or bulk password requests are implemented. The existing website
proxy/retry/uTLS/limiter limitations still apply. Tests use local provider fixtures and synthetic
corpus data, not live credentials or a production-scale corpus. Native Windows execution,
production load/durability and legal suitability remain **UNVERIFIED**. Review the current
[HIBP Terms](https://haveibeenpwned.com/TermsOfUse) for actual public/commercial use, redistribution
or re-serving; this endpoint and operator-owned offline data do not certify legal permission.

## Development and verification

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/agentsearch
go build ./cmd/convert
go build ./cmd/agentsearch-corpus
go build ./cmd/agentsearch-server
```

Tests use local HTTP servers, runtime-generated email mock credentials, and obvious public password fixtures, not real HIBP keys, private passwords, or external API calls. They cover HIBP response/error semantics, headers, encoding, redirect protection, cancellation, timeouts, redaction, normalized results, and CLI execution, alongside the website regression suite. Password tests additionally cover independent SHA-1 vectors, exact request prefix/suffix boundaries, padding and strict range parsing, shared-buffer destruction, prompt injection/restoration, both password flags, and output/report leakage. Stage 3 added `golang.org/x/term` for terminal input. Stage 4 promotes the already-present `golang.org/x/sys` to a direct dependency for platform locking/space checks, without a version upgrade or new database, mmap, crypto or API framework. Synthetic offline tests protect range integrity, activation/leases, concurrency and secret/backend isolation.

A repository-wide test checks Unicode Cyrillic characters in current text files, including comments, documentation, configurations, and fixtures. The maintained codebase and primary documentation are English-only. Git history and externally supplied target/database/response data are not rewritten or translated.

### Remaining limitations

In addition to output-directory and error-propagation limitations noted above:

- Website proxy/retry and retry/redirect composition needs further testing and hardening.
- Experimental uTLS has unresolved server-name and transport-composition concerns; its presence is not a guarantee of successful TLS connections or WAF access.
- The website processor retains explicit compressed-response headers without explicit gzip/deflate/Brotli decoding.
- The host limiter still holds a shared mutex while waiting, which can serialize different hosts.
- Reports retain results in memory; summary tag/latency aggregation remains limited.
- HIBP responses are bounded to 2 MiB. No pagination, local cache, email hash-range lookup, pastes, domain enumeration, or separate stealer-log API is implemented.
- HIBP API behavior is tested against local contract fixtures, not certified against a live subscription during CI.

See [ROADMAP.md](ROADMAP.md) for the next stages. Stages 3–5 (password API, offline database/maintenance, unified HTTP API) are implemented. Acquisition stays external; a downloader, AI analysis and additional external sources are not implemented. Review current HIBP Terms before public/commercial deployment or corpus redistribution; software availability is not legal certification.

## License

AgentSearch is licensed under the [MIT License](LICENSE). Dependencies and external data retain their own license and service requirements, including HIBP attribution and UniOffice licensing.
