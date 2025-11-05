# Spot Instance Advisor

[![Go Version](https://img.shields.io/badge/Go-1.25.2-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Go Modules](https://img.shields.io/badge/Go%20Modules-Enabled-blue.svg)](go.mod)

A command-line tool for analyzing Alibaba Cloud Spot instance prices and availability. This tool helps you find the most cost-effective Spot instances based on historical price data and availability patterns.

## Features

- 🔍 **Instance Filtering**: Filter instances by CPU, memory, and instance family
- 📊 **Price Analysis**: Analyze historical Spot prices with customizable time windows
- 💰 **Cost Optimization**: Find instances with the best price-to-performance ratios
- 📈 **Availability Insights**: Get insights into instance availability patterns
- 🎯 **Multiple Output Formats**: Support for both human-readable tables and JSON output
- ⚡ **Fast & Efficient**: Optimized for quick analysis of large instance catalogs

## Quick Start

### Prerequisites

- Go 1.25.2 or later
- Alibaba Cloud account with ECS API access

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd spot-instance-advisor

# Build the binary (outputs to dist/spot-instance-advisor-OS-ARCH)
make build

# Or build directly with Go (not recommended for releases)
go build -o dist/spot-instance-advisor-$(go env GOOS)-$(go env GOARCH) .
```

### Basic Usage

```bash
# Basic usage with table output
./spot-instance-advisor \
  --accessKeyId YOUR_ACCESS_KEY_ID \
  --accessKeySecret YOUR_ACCESS_KEY_SECRET \
  --region cn-hangzhou

# JSON output for programmatic use
./spot-instance-advisor \
  --accessKeyId YOUR_ACCESS_KEY_ID \
  --accessKeySecret YOUR_ACCESS_KEY_SECRET \
  --region cn-hangzhou \
  --json
```

## Command Line Options

### Authentication

- `--accessKeyId`: Your Alibaba Cloud Access Key ID
- `--accessKeySecret`: Your Alibaba Cloud Access Key Secret
- `--region`: Target region (default: cn-hangzhou)

### Instance Filtering

- `--mincpu`: Minimum CPU cores (default: 1)
- `--maxcpu`: Maximum CPU cores (default: 32)
- `--minmem`: Minimum memory in GB (default: 2)
- `--maxmem`: Maximum memory in GB (default: 64)
- `--family`: Instance family filter (e.g., ecs.n1,ecs.n2)
- `--arch`: CPU architecture filter: x86_64 or arm64

### Analysis Parameters

- `--cutoff`: Discount threshold (default: 2)
- `--limit`: Maximum number of results (default: 20)
- `--resolution`: Price history analysis window in days (default: 7)

### Output Format

- `--json`: Output results in JSON format

## Development

### Project Structure

```tree
spot-instance-advisor/
├── main.go          # Main application entry point
├── meta.go          # Metadata and instance management
├── sort.go          # Price analysis and sorting logic
├── go.mod           # Go module dependencies
├── go.sum           # Dependency checksums
├── Makefile         # Build automation
└── README.md        # This file
```

### Building and Testing

```bash
# Install dependencies
make deps

# Run tests
make test

# Build the application
make build

# Build for Linux
make build-linux

# Run with coverage
make test-coverage

# Clean build artifacts
make clean

# Update dependencies
make deps-update
```

### Dependencies

This project uses modern Go modules for dependency management:

- **github.com/aliyun/alibaba-cloud-sdk-go**: Alibaba Cloud SDK for Go
- **github.com/fatih/color**: Terminal color output
- **github.com/sirupsen/logrus**: Structured logging

## JSON Output Format

When using the `--json` flag, the tool outputs structured JSON data:

```json
[
  {
    "instanceTypeId": "ecs.n1.small",
    "zoneId": "cn-hangzhou-a",
    "pricePerCore": 0.1234,
    "discount": 2.5,
    "possibility": 0.8,
    "cpuCoreCount": 1,
    "memorySize": 2.0,
    "instanceFamily": "ecs.n1",
    "arch": "x86_64"
  }
]
```

### JSON Field Descriptions

- `instanceTypeId`: Instance type identifier
- `zoneId`: Availability zone identifier
- `pricePerCore`: Price per CPU core
- `discount`: Discount multiplier compared to on-demand pricing
- `possibility`: Price stability indicator
- `cpuCoreCount`: Number of CPU cores
- `memorySize`: Memory size in GB
- `instanceFamily`: Instance family identifier
- `arch`: CPU architecture (x86_64 or arm64)

## Examples

### Find Small Instances with Good Discounts

```bash
./spot-instance-advisor \
  --accessKeyId YOUR_KEY \
  --accessKeySecret YOUR_SECRET \
  --mincpu 1 \
  --maxcpu 4 \
  --minmem 2 \
  --maxmem 8 \
  --cutoff 3 \
  --json
```

### Analyze Specific Instance Family

```bash
./spot-instance-advisor \
  --accessKeyId YOUR_KEY \
  --accessKeySecret YOUR_SECRET \
  --family ecs.n1,ecs.n2 \
  --arch x86_64 \
  --limit 10 \
  --json
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Alibaba Cloud for providing the ECS API
- The Go community for excellent tooling and libraries
