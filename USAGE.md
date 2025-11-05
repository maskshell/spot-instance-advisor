# Spot Instance Advisor 使用说明

## 功能概述

Spot Instance Advisor 是一个用于分析阿里云 Spot 实例价格和可用性的工具。现在支持两种输出格式：表格格式和 JSON 格式。

## 命令行参数

### 基本参数

- `--accessKeyId`: 阿里云访问密钥 ID
- `--accessKeySecret`: 阿里云访问密钥 Secret
- `--region`: 区域（默认：cn-hangzhou）

### 实例筛选参数

- `--mincpu`: 最小 CPU 核心数（默认：1）
- `--maxcpu`: 最大 CPU 核心数（默认：32）
- `--minmem`: 最小内存（GB）（默认：2）
- `--maxmem`: 最大内存（GB）（默认：64）
- `--family`: 实例族（例如：ecs.n1,ecs.n2）
- `--arch`: 架构过滤（x86_64 或 arm64）

### 分析参数

- `--cutoff`: 折扣阈值（默认：2）
- `--limit`: 结果数量限制（默认：20）
- `--resolution`: 价格历史分析窗口（天）（默认：7）

### 输出格式参数

- `--json`: 以 JSON 格式输出结果

## 使用示例

### 1. 基本使用（表格格式）

```bash
./spot-instance-advisor \
  --accessKeyId YOUR_ACCESS_KEY_ID \
  --accessKeySecret YOUR_ACCESS_KEY_SECRET \
  --region cn-hangzhou \
  --mincpu 2 \
  --maxcpu 8 \
  --minmem 4 \
  --maxmem 16 \
  --arch arm64
```

### 2. JSON 格式输出（纯 JSON，无摘要信息）

```bash
./spot-instance-advisor \
  --accessKeyId YOUR_ACCESS_KEY_ID \
  --accessKeySecret YOUR_ACCESS_KEY_SECRET \
  --region cn-hangzhou \
  --mincpu 2 \
  --maxcpu 8 \
  --minmem 4 \
  --maxmem 16 \
  --json
```

**注意**: 使用 `--json` 参数时，程序将：

- 只输出纯 JSON 结果，不显示任何摘要信息
- 错误时也以 JSON 格式输出错误信息
- 适合程序化处理和自动化脚本

### 3. 指定实例族

```bash
./spot-instance-advisor \
  --accessKeyId YOUR_ACCESS_KEY_ID \
  --accessKeySecret YOUR_ACCESS_KEY_SECRET \
  --family ecs.n1,ecs.n2 \
  --json
```

## JSON 输出格式

当使用 `--json` 参数时，输出将是格式化的 JSON 数组，每个元素包含以下字段：

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

### JSON 字段说明

- `instanceTypeId`: 实例类型 ID
- `zoneId`: 可用区 ID
- `pricePerCore`: 每核心价格
- `discount`: 折扣倍数
- `possibility`: 价格稳定性指标
- `cpuCoreCount`: CPU 核心数
- `memorySize`: 内存大小（GB）
- `instanceFamily`: 实例族
- `arch`: CPU 架构（x86_64 或 arm64）

## 构建

```bash
go build -o spot-instance-advisor .
```

## 依赖管理

项目使用 Go modules 进行依赖管理：

```bash
go mod tidy
go mod vendor
```

## JSON 错误输出格式

当使用 `--json` 参数且发生错误时，程序会输出 JSON 格式的错误信息：

```json
{
  "error": "错误类型",
  "message": "详细错误信息"
}
```

### 错误输出示例

```json
{
  "error": "Failed to initialize metastore",
  "message": "failed to DescribeInstanceTypes: SDK.ServerError\nErrorCode: InvalidAccessKeyId.NotFound\nMessage: Specified access key is not found."
}
```
