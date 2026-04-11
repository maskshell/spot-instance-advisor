# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
