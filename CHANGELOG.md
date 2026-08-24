# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2026-08-24

### Added

- **Automatic retry with exponential backoff for transient Aliyun API failures**: throttling (HTTP 429, or `Throttling*`/`FlowControl` error codes — which the API can also return with HTTP 400), server-side 5xx, and request timeouts are retried up to 3 times with exponential backoff (1s → 2s → 4s, capped at 10s). Each retry is logged to stderr; JSON output on stdout stays clean. A single transient failure no longer silently drops an instance type from the results — increasingly relevant now that full pagination raises per-run API call volume (upstream issue #9).
  - Timeouts match both shapes the SDK can produce: the `SDK.TimeoutError` client error (AutoRetry-enabled configs) and the raw `*url.Error`/`net.Error` transport timeout the default configuration returns.
  - Non-transient errors (invalid parameters, auth failures, endpoint misconfiguration) fail fast with a single attempt.
  - Retry applies per API call, including mid-pagination: a throttle while fetching page N refetches only that page; already-fetched pages survive.

---

## [1.1.0] - 2026-08-23

### Added

- **Environment variable credentials**: Access keys can be provided via `ALIYUN_ACCESS_KEY_ID` / `ALIYUN_ACCESS_KEY_SECRET` (or `ALIBABA_CLOUD_ACCESS_KEY_ID` / `ALIBABA_CLOUD_ACCESS_KEY_SECRET`) instead of command-line flags, keeping the secret out of argv / `ps` / shell history / CI logs. Explicit flags take precedence; whitespace-only values fall through.
- **Flag validation**: Invalid numeric ranges (e.g. `--limit -5`, `--mincpu 32 --maxcpu 1`, `--resolution 0`) now produce a clear error instead of silently empty output. `--cutoff 0` is allowed ("highlight only free instances").

### Fixed

- **Spot price history pagination**: `DescribeSpotPriceHistory` results are now fully paginated via `Offset`/`NextOffset`. Previously only the first page was fetched — and the API returns the oldest entries first — so the "latest" spot price and the stability score were computed on a truncated, stale sample.
- **`--resolution` takes effect**: the analysis window is now transmitted to the API (`StartTime`/`EndTime` set before the call, in UTC with a past buffer to satisfy the API's `EndTime < now` rule). Previously the window was assigned after the request was sent and silently ignored.
- **No more panics on sold-out zones**: an empty `AvailableResource` list (sold-out / out-of-stock zone) no longer crashes during initialization.
- **No more panics on malformed timestamps**: a single unparseable timestamp in the API response is skipped instead of terminating the CLI.
- **Invalid instance data no longer mis-ranks**: rows with missing ranking inputs (`CpuCoreCount <= 0`, `OriginPrice <= 0`, or no valid timestamp) are excluded from the ranking instead of sorting to the top as bogus "best deals"; `NaN`/`Infinity` can no longer reach the sort comparator or the JSON output.
- **Fetch failures are surfaced**: per-instance failures warn on stderr (table and JSON modes); a hard error is returned only when every instance type fails. Progress messages report actual fetched/filtered counts.

### Changed

- Runtime errors print a clean message and exit 1 (JSON mode emits a JSON error object) instead of dumping a Go stack trace via `panic()`.
- Availability filtering rewritten as an O(N+Z·R) set-membership check (was O(N·Z·R) nested scan).
- Removed the unused `logrus` dependency.

### Documentation

- Corrected the license badge and notice from MIT to Apache-2.0, matching the actual LICENSE file.

---

## [1.0.3] - 2026-04-11

### Fixed

- **DescribeInstanceTypes pagination**: The API defaults to returning only 100 records per call, causing newer instance types (e.g. `ecs.u2a-c1m2.xlarge`) to be missing from results. Added NextToken-based pagination loop with `MaxResults=100` to fetch all instance types.

### Documentation

- Added CHANGELOG and release notes for v1.0.0–v1.0.2

---

## [1.0.2] - 2025-12-12

### Added

- **New `--instanceType` parameter**: Specify exact instance types (comma-separated, e.g., `ecs.n1.small,ecs.n2.large`)
  - Takes precedence over `--family` parameter
  - When specified, all other filters (CPU, memory, architecture) are skipped
  - Invalid or non-existent instance types are automatically skipped
- **Parameter validation**: Added validation for required parameters (`accessKeyId` and `accessKeySecret`)
  - Help information is displayed when required parameters are missing
  - JSON format error output for missing parameters when using `--json` flag
- **Comprehensive test suite**: Added extensive test coverage
  - `meta_test.go`: Tests for architecture normalization, instance filtering, and price analysis
  - `sort_test.go`: Tests for price sorting, latest price finding, and possibility calculation
  - Edge case handling for empty price arrays

### Fixed

- Fixed `GetPossibility` function to properly handle empty price arrays (returns 0.0 instead of causing errors)
- Fixed code formatting and indentation issues

### Changed

- Updated `golang.org/x/sys` dependency from v0.37.0 to v0.39.0

### Documentation

- Updated `README.md` with `--instanceType` usage examples and behavior documentation
- Updated `USAGE.md` with detailed `--instanceType` parameter documentation
- Documented that `--instanceType` skips all other filters when specified

---

## [1.0.1] - 2025-11-05

### Added

- **New `--arch` filter parameter**: Filter instances by CPU architecture
  - Supported values: `x86_64` or `arm64`
  - Allows users to find instances matching their specific architecture requirements
- **Architecture field in JSON output**: Added `arch` field to JSON output format
  - Enables programmatic filtering and analysis by architecture

### Documentation

- Updated `README.md` with `--arch` parameter documentation and examples
- Updated `USAGE.md` with architecture filtering usage examples

---

## [1.0.0] - 2025-10-22

### Added

- **Initial release**: First stable version of Spot Instance Advisor
- **Multi-platform support**: Pre-built binaries for multiple platforms
  - Linux (amd64, arm64)
  - macOS/Darwin (amd64, arm64)
  - Windows (amd64)
- **JSON output support**: Added `--json` flag for machine-readable output
  - Structured JSON format for programmatic use
  - JSON error output format for better error handling in automation
- **Instance filtering**: Filter instances by CPU, memory, and instance family
  - `--mincpu` / `--maxcpu`: CPU core range filtering
  - `--minmem` / `--maxmem`: Memory range filtering
  - `--family`: Instance family filtering (comma-separated)
- **Price analysis**: Historical Spot price analysis with customizable time windows
  - `--cutoff`: Discount threshold filtering
  - `--resolution`: Price history analysis window (days)
  - `--limit`: Maximum number of results
- **Cost optimization**: Find instances with best price-to-performance ratios
- **Availability insights**: Price stability indicators (possibility scores)
- **Multiple output formats**: Human-readable tables and JSON output
- **GitHub Releases automation**: Automated build and release workflow
  - Cross-platform binary builds
  - Optimized production binaries (statically linked, stripped)
- **Go Modules migration**: Modern dependency management

### Infrastructure

- GitHub Actions CI/CD pipeline
- Automated release workflow
- Cross-platform build support
- Comprehensive build system with Makefile

[1.0.3]: https://github.com/maskshell/spot-instance-advisor/releases/tag/v1.0.3
[1.0.2]: https://github.com/maskshell/spot-instance-advisor/releases/tag/v1.0.2
[1.0.1]: https://github.com/maskshell/spot-instance-advisor/releases/tag/v1.0.1
[1.0.0]: https://github.com/maskshell/spot-instance-advisor/releases/tag/v1.0.0
