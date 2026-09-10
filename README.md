# AgentSearch

AgentSearch is a Go command-line tool for authorized account discovery and exposure checks. It combines configurable website profile searches with authenticated email breach lookups through [Have I Been Pwned (HIBP)](https://haveibeenpwned.com/), using a common result model and output pipeline.

It is intended for defensive investigations, personal exposure checks, and security research where you have permission to query the target. A match is evidence to review, not proof of identity or of a currently compromised account.

## Capabilities

| Status | Capability |
| --- | --- |
| Implemented | Website searches using native YAML/JSON, Sherlock, and Maigret site databases |
| Implemented | HIBP email breach lookup using an API subscription key |
| Implemented | Fixed website worker pool, cancellable host delays, request timeouts, direct-request retries, proxy rotation, and User-Agent rotation |
| Implemented | Status, message, regex, redirect, header, and WAF detection rules |
| Implemented | Shared JSON, CSV, and TXT output; CLI and HTML reports |
| License-dependent | DOCX reports through the existing UniOffice dependency |
| Experimental | Website uTLS support; transport combinations have known limitations |
| Planned | Pwned Passwords API, offline password database, HTTP API, and AI analysis |

**Password lookup is not implemented.** Do not pass passwords to any current CLI option. The `-d` flag is retained for compatibility but remains a no-op stub.

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

## Configure HIBP

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

The services file is loaded **only** for `-email`. Website-specific flags such as `-u`, `-f`, `-s`, `-p`, `-w`, `-rl`, `-ua`, `-utls`, and `-retries` cannot be combined with `-email`. HIBP uses a standard TLS client and an identifying AgentSearch User-Agent, not browser impersonation or proxy rotation. It does not use environment proxy settings.

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

HIBP breach data is attributed to **[Have I Been Pwned](https://haveibeenpwned.com/)** and is subject to its [Creative Commons Attribution 4.0 license](https://creativecommons.org/licenses/by/4.0/). Results link to HIBP and retain its name. Keep that attribution when sharing derived reports. Pwned Passwords is a separate planned integration, not part of this email lookup.

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
| `-rt` | `15s` | HTTP request timeout; positive in email mode |
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
| `-services` | `configs/services.yaml` | Service settings; valid only with `-email` |

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

- **TXT:** compact per-result lines, including status and source display name. CSV and TXT do not include all structured metadata; use JSON for complete breach detail.
- **CLI:** an English terminal summary and plain-text report.
- **HTML:** self-contained report with filters, status cards, evidence, and metadata. Remote text is escaped rather than interpreted as HTML.
- **DOCX:** retained Word report generation, subject to UniOffice licensing. The license-dependent test skips explicitly if the license is unavailable.
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

Filenames are based on the target with unsafe filename characters replaced. **Known limitation:** result files and summary JSON honor `-o`, but CLI/HTML/DOCX generators still write into `output/`. Runs using the same target can overwrite previous output; use separate working/output directories when preserving investigations. Existing output/report managers log some write failures rather than propagating them to the process exit status.

## Security considerations

- Query only targets you are authorized to investigate. Respect applicable law, source terms, and HIBP's acceptable-use policy.
- Email addresses and breach associations are personal information. They appear in local output and filenames. Protect and retain that output appropriately; it is not encrypted by AgentSearch.
- Only environment variable names belong in service YAML. Never commit keys or credential-bearing proxy files.
- `security.Secret`, sensitive-key filtering, known-secret redaction, and URL-userinfo redaction protect normal diagnostic and result paths. HIBP errors contain safe classifications, not raw transport errors, bodies, or headers. The HIBP client does not log requests or responses.
- Redaction is a safety net, not a general detector for every possible secret representation. Do not add HTTP request dumps or place secrets into evidence. Go strings also cannot be reliably erased from memory.
- The typed model has password/password-hash categories for future integrations, with redacted formatting. There is currently no password command or lookup implementation.

## Architecture

```text
CLI flags
  -> application composition and target parsing
  -> Runner -> source registry / capability dispatcher
       |-- websites -> workers -> host limiter -> existing HTTP client -> detector
       `-- hibp -> email breach client -> standard service HTTP transport
  -> normalized, redacted results
  -> common storage and reports
```

The website source supports username and legacy email-template searches. HIBP implements only `EmailSearcher`. Source registration is mode-specific, preventing accidental HIBP calls from legacy commands. Sources own their concurrency; one email request does not create a worker pool.

`app.NewRunner(registry).Search(ctx, target, emit)` is the source-neutral execution boundary. The dispatcher derives capabilities from small interfaces. It preserves partial results, stops on consumer errors, and does not import storage or reports. Models depend only on the standard library and the small security package. The network layer does not depend on concrete sources. HTTP clients and secrets are supplied at the application composition boundary.

## Development and verification

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/agentsearch
go build ./cmd/convert
```

Tests use local HTTP servers and runtime-generated mock credentials, not real HIBP keys or external API calls. They cover HIBP response/error semantics, headers, encoding, redirect protection, cancellation, timeouts, redaction, normalized results, and CLI execution, alongside the website regression suite.

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

See [ROADMAP.md](ROADMAP.md) for the next stages. Stage 3 is **Pwned Passwords API integration**, followed by separately scoped work on offline lookups, an HTTP API, and optional analysis. None of those planned features is implemented here.

## License

AgentSearch is licensed under the [MIT License](LICENSE). Dependencies and external data retain their own license and service requirements, including HIBP attribution and UniOffice licensing.
