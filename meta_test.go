package main

import (
	"testing"

	ecsService "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

func TestNormalizeArch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"x86_64 lowercase", "x86_64", "x86_64"},
		{"x86_64 uppercase", "X86_64", "x86_64"},
		{"amd64", "amd64", "x86_64"},
		{"x86", "x86", "x86_64"},
		{"x86-64", "x86-64", "x86_64"},
		{"x64", "x64", "x86_64"},
		{"arm64 lowercase", "arm64", "arm64"},
		{"arm64 uppercase", "ARM64", "arm64"},
		{"aarch64", "aarch64", "arm64"},
		{"arm", "arm", "arm64"},
		{"ARM", "ARM", "arm64"},
		{"with spaces", "  x86_64  ", "x86_64"},
		{"unknown arch", "riscv", "riscv"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeArch(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeArch(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetInstanceArch(t *testing.T) {
	tests := []struct {
		name     string
		instance ecsService.InstanceType
		expected string
	}{
		{
			name: "with CpuArchitecture field",
			instance: ecsService.InstanceType{
				InstanceTypeId:  "ecs.n1.small",
				CpuArchitecture: "x86_64",
			},
			expected: "x86_64",
		},
		{
			name: "with CpuArchitecture arm64",
			instance: ecsService.InstanceType{
				InstanceTypeId:  "ecs.c6g.large",
				CpuArchitecture: "arm64",
			},
			expected: "arm64",
		},
		{
			name: "ARM instance type id c6g",
			instance: ecsService.InstanceType{
				InstanceTypeId:     "ecs.c6g.large",
				InstanceTypeFamily: "ecs.c6g",
				CpuArchitecture:    "",
			},
			expected: "arm64",
		},
		{
			name: "ARM instance type id g6g",
			instance: ecsService.InstanceType{
				InstanceTypeId:     "ecs.g6g.large",
				InstanceTypeFamily: "ecs.g6g",
				CpuArchitecture:    "",
			},
			expected: "arm64",
		},
		{
			name: "ARM instance type id r6g",
			instance: ecsService.InstanceType{
				InstanceTypeId:     "ecs.r6g.large",
				InstanceTypeFamily: "ecs.r6g",
				CpuArchitecture:    "",
			},
			expected: "arm64",
		},
		{
			name: "ARM instance type id c8y",
			instance: ecsService.InstanceType{
				InstanceTypeId:     "ecs.c8y.large",
				InstanceTypeFamily: "ecs.c8y",
				CpuArchitecture:    "",
			},
			expected: "arm64",
		},
		{
			name: "x86 instance type",
			instance: ecsService.InstanceType{
				InstanceTypeId:     "ecs.n1.small",
				InstanceTypeFamily: "ecs.n1",
				CpuArchitecture:    "",
			},
			expected: "x86_64",
		},
		{
			name: "ARM family name",
			instance: ecsService.InstanceType{
				InstanceTypeId:     "ecs.some.type",
				InstanceTypeFamily: "ecs.c6g",
				CpuArchitecture:    "",
			},
			expected: "arm64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInstanceArch(tt.instance)
			if result != tt.expected {
				t.Errorf("getInstanceArch() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestFilterInstances_WithInstanceType(t *testing.T) {
	ms := &MetaStore{
		InstanceFamilyCache: map[string]ecsService.InstanceType{
			"ecs.n1.small": {
				InstanceTypeId:     "ecs.n1.small",
				InstanceTypeFamily: "ecs.n1",
				CpuCoreCount:       2,
				MemorySize:         4.0,
				CpuArchitecture:    "x86_64", // Explicitly set to x86_64
			},
			"ecs.n1.large": {
				InstanceTypeId:     "ecs.n1.large",
				InstanceTypeFamily: "ecs.n1",
				CpuCoreCount:       4,
				MemorySize:         8.0,
				CpuArchitecture:    "x86_64", // Explicitly set to x86_64
			},
			"ecs.c6g.large": {
				InstanceTypeId:     "ecs.c6g.large",
				InstanceTypeFamily: "ecs.c6g",
				CpuCoreCount:       4,
				MemorySize:         8.0,
				CpuArchitecture:    "", // Empty, will be inferred as arm64 by getInstanceArch
			},
		},
	}

	tests := []struct {
		name         string
		cpu          int
		memory       int
		maxCpu       int
		maxMemory    int
		family       string
		instanceType string
		arch         string
		expected     []string
	}{
		{
			name:         "instanceType takes precedence over family",
			cpu:          1,
			memory:       2,
			maxCpu:       32,
			maxMemory:    64,
			family:       "ecs.n1",
			instanceType: "ecs.n1.small",
			arch:         "",
			expected:     []string{"ecs.n1.small"},
		},
		{
			name:         "multiple instance types",
			cpu:          1,
			memory:       2,
			maxCpu:       32,
			maxMemory:    64,
			family:       "",
			instanceType: "ecs.n1.small,ecs.n1.large",
			arch:         "",
			expected:     []string{"ecs.n1.small", "ecs.n1.large"},
		},
		{
			name:         "instanceType with arch filter - filters are skipped",
			cpu:          1,
			memory:       2,
			maxCpu:       32,
			maxMemory:    64,
			family:       "",
			instanceType: "ecs.n1.small,ecs.c6g.large",
			arch:         "arm64",
			expected:     []string{"ecs.n1.small", "ecs.c6g.large"}, // all specified types, arch filter is skipped
		},
		{
			name:         "instanceType with CPU filter - filters are skipped",
			cpu:          4,
			memory:       2,
			maxCpu:       32,
			maxMemory:    64,
			family:       "",
			instanceType: "ecs.n1.small,ecs.n1.large",
			arch:         "",
			expected:     []string{"ecs.n1.small", "ecs.n1.large"}, // all specified types, CPU filter is skipped
		},
		{
			name:         "instanceType with memory filter - filters are skipped",
			cpu:          1,
			memory:       8,
			maxCpu:       32,
			maxMemory:    64,
			family:       "",
			instanceType: "ecs.n1.small,ecs.n1.large",
			arch:         "",
			expected:     []string{"ecs.n1.small", "ecs.n1.large"}, // all specified types, memory filter is skipped
		},
		{
			name:         "invalid instanceType",
			cpu:          1,
			memory:       2,
			maxCpu:       32,
			maxMemory:    64,
			family:       "",
			instanceType: "ecs.invalid.type",
			arch:         "",
			expected:     []string{},
		},
		{
			name:         "instanceType with spaces",
			cpu:          1,
			memory:       2,
			maxCpu:       32,
			maxMemory:    64,
			family:       "",
			instanceType: " ecs.n1.small , ecs.n1.large ",
			arch:         "",
			expected:     []string{"ecs.n1.small", "ecs.n1.large"},
		},
		{
			name:         "empty instanceType falls back to family",
			cpu:          1,
			memory:       2,
			maxCpu:       32,
			maxMemory:    64,
			family:       "ecs.n1",
			instanceType: "",
			arch:         "",
			expected:     []string{"ecs.n1.small", "ecs.n1.large"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ms.FilterInstances(tt.cpu, tt.memory, tt.maxCpu, tt.maxMemory, tt.family, tt.instanceType, tt.arch, true)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d results, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			// Check that all expected items are in result (order may differ)
			resultMap := make(map[string]bool)
			for _, r := range result {
				resultMap[r] = true
			}
			for _, expected := range tt.expected {
				if !resultMap[expected] {
					t.Errorf("Expected %q in results, but not found. Got: %v", expected, result)
				}
			}
		})
	}
}

func TestFilterInstances_WithFamily(t *testing.T) {
	ms := &MetaStore{
		InstanceFamilyCache: map[string]ecsService.InstanceType{
			"ecs.n1.small": {
				InstanceTypeId:     "ecs.n1.small",
				InstanceTypeFamily: "ecs.n1",
				CpuCoreCount:       2,
				MemorySize:         4.0,
				CpuArchitecture:    "x86_64",
			},
			"ecs.n1.large": {
				InstanceTypeId:     "ecs.n1.large",
				InstanceTypeFamily: "ecs.n1",
				CpuCoreCount:       4,
				MemorySize:         8.0,
				CpuArchitecture:    "x86_64",
			},
			"ecs.n2.medium": {
				InstanceTypeId:     "ecs.n2.medium",
				InstanceTypeFamily: "ecs.n2",
				CpuCoreCount:       2,
				MemorySize:         4.0,
				CpuArchitecture:    "x86_64",
			},
		},
	}

	tests := []struct {
		name      string
		cpu       int
		memory    int
		maxCpu    int
		maxMemory int
		family    string
		arch      string
		expected  []string
	}{
		{
			name:      "single family",
			cpu:       1,
			memory:    2,
			maxCpu:    32,
			maxMemory: 64,
			family:    "ecs.n1",
			arch:      "",
			expected:  []string{"ecs.n1.small", "ecs.n1.large"},
		},
		{
			name:      "multiple families",
			cpu:       1,
			memory:    2,
			maxCpu:    32,
			maxMemory: 64,
			family:    "ecs.n1,ecs.n2",
			arch:      "",
			expected:  []string{"ecs.n1.small", "ecs.n1.large", "ecs.n2.medium"},
		},
		{
			name:      "with arch filter",
			cpu:       1,
			memory:    2,
			maxCpu:    32,
			maxMemory: 64,
			family:    "ecs.n1",
			arch:      "x86_64",
			expected:  []string{"ecs.n1.small", "ecs.n1.large"},
		},
		{
			name:      "with CPU filter",
			cpu:       4,
			memory:    2,
			maxCpu:    32,
			maxMemory: 64,
			family:    "ecs.n1",
			arch:      "",
			expected:  []string{"ecs.n1.large"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ms.FilterInstances(tt.cpu, tt.memory, tt.maxCpu, tt.maxMemory, tt.family, "", tt.arch, true)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d results, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			resultMap := make(map[string]bool)
			for _, r := range result {
				resultMap[r] = true
			}
			for _, expected := range tt.expected {
				if !resultMap[expected] {
					t.Errorf("Expected %q in results, but not found. Got: %v", expected, result)
				}
			}
		})
	}
}

// Integration tests - testing functions that work with cache
func TestSpotPricesAnalysis(t *testing.T) {
	ms := &MetaStore{
		InstanceFamilyCache: map[string]ecsService.InstanceType{
			"ecs.n1.small": {
				InstanceTypeId:     "ecs.n1.small",
				InstanceTypeFamily: "ecs.n1",
				CpuCoreCount:       2,
				MemorySize:         4.0,
			},
			"ecs.n1.large": {
				InstanceTypeId:     "ecs.n1.large",
				InstanceTypeFamily: "ecs.n1",
				CpuCoreCount:       4,
				MemorySize:         8.0,
			},
		},
	}

	historyPrices := map[string][]ecsService.SpotPriceType{
		"ecs.n1.small": {
			{
				Timestamp:   "2024-01-01T10:00:00Z",
				SpotPrice:   0.1,
				OriginPrice: 1.0,
				ZoneId:      "cn-hangzhou-a",
			},
			{
				Timestamp:   "2024-01-02T10:00:00Z",
				SpotPrice:   0.15,
				OriginPrice: 1.0,
				ZoneId:      "cn-hangzhou-a",
			},
		},
		"ecs.n1.large": {
			{
				Timestamp:   "2024-01-01T10:00:00Z",
				SpotPrice:   0.2,
				OriginPrice: 2.0,
				ZoneId:      "cn-hangzhou-b",
			},
		},
	}

	result := ms.SpotPricesAnalysis(historyPrices, true)

	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}

	// Check that results are properly created
	foundSmall := false
	foundLarge := false
	for _, price := range result {
		if price.InstanceTypeId == "ecs.n1.small" && price.ZoneId == "cn-hangzhou-a" {
			foundSmall = true
			if price.PricePerCore <= 0 {
				t.Error("Expected positive PricePerCore for ecs.n1.small")
			}
		}
		if price.InstanceTypeId == "ecs.n1.large" && price.ZoneId == "cn-hangzhou-b" {
			foundLarge = true
			if price.PricePerCore <= 0 {
				t.Error("Expected positive PricePerCore for ecs.n1.large")
			}
		}
	}

	if !foundSmall {
		t.Error("Expected ecs.n1.small result")
	}
	if !foundLarge {
		t.Error("Expected ecs.n1.large result")
	}
}

func TestSpotPricesAnalysis_WithMissingInstanceType(t *testing.T) {
	ms := &MetaStore{
		InstanceFamilyCache: map[string]ecsService.InstanceType{
			"ecs.n1.small": {
				InstanceTypeId:     "ecs.n1.small",
				InstanceTypeFamily: "ecs.n1",
				CpuCoreCount:       2,
				MemorySize:         4.0,
			},
		},
	}

	historyPrices := map[string][]ecsService.SpotPriceType{
		"ecs.n1.small": {
			{
				Timestamp:   "2024-01-01T10:00:00Z",
				SpotPrice:   0.1,
				OriginPrice: 1.0,
				ZoneId:      "cn-hangzhou-a",
			},
		},
		"ecs.n1.invalid": { // This instance type is not in cache
			{
				Timestamp:   "2024-01-01T10:00:00Z",
				SpotPrice:   0.1,
				OriginPrice: 1.0,
				ZoneId:      "cn-hangzhou-a",
			},
		},
	}

	result := ms.SpotPricesAnalysis(historyPrices, true)

	// Should only have 1 result (ecs.n1.small), invalid one should be skipped
	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}
	if result[0].InstanceTypeId != "ecs.n1.small" {
		t.Errorf("Expected ecs.n1.small, got %s", result[0].InstanceTypeId)
	}
}

func TestSpotPricesAnalysis_MultipleZones(t *testing.T) {
	ms := &MetaStore{
		InstanceFamilyCache: map[string]ecsService.InstanceType{
			"ecs.n1.small": {
				InstanceTypeId:     "ecs.n1.small",
				InstanceTypeFamily: "ecs.n1",
				CpuCoreCount:       2,
				MemorySize:         4.0,
			},
		},
	}

	historyPrices := map[string][]ecsService.SpotPriceType{
		"ecs.n1.small": {
			{
				Timestamp:   "2024-01-01T10:00:00Z",
				SpotPrice:   0.1,
				OriginPrice: 1.0,
				ZoneId:      "cn-hangzhou-a",
			},
			{
				Timestamp:   "2024-01-02T10:00:00Z",
				SpotPrice:   0.15,
				OriginPrice: 1.0,
				ZoneId:      "cn-hangzhou-b",
			},
		},
	}

	result := ms.SpotPricesAnalysis(historyPrices, true)

	// Should have 2 results (one for each zone)
	if len(result) != 2 {
		t.Errorf("Expected 2 results (one per zone), got %d", len(result))
	}

	zones := make(map[string]bool)
	for _, price := range result {
		zones[price.ZoneId] = true
	}
	if !zones["cn-hangzhou-a"] {
		t.Error("Expected result for cn-hangzhou-a")
	}
	if !zones["cn-hangzhou-b"] {
		t.Error("Expected result for cn-hangzhou-b")
	}
}

// TestSpotPricesAnalysis_SkipsInvalidMetadata locks in the H3 follow-up: a
// cached instance type with missing ranking inputs (CpuCoreCount == 0) is
// dropped from the results instead of sorting to the top as the "cheapest".
func TestSpotPricesAnalysis_SkipsInvalidMetadata(t *testing.T) {
	ms := &MetaStore{
		InstanceFamilyCache: map[string]ecsService.InstanceType{
			"ecs.valid":  {InstanceTypeId: "ecs.valid", InstanceTypeFamily: "ecs.valid", CpuCoreCount: 2, MemorySize: 4.0},
			"ecs.broken": {InstanceTypeId: "ecs.broken", InstanceTypeFamily: "ecs.broken", CpuCoreCount: 0, MemorySize: 4.0},
		},
	}

	historyPrices := map[string][]ecsService.SpotPriceType{
		"ecs.valid": {
			{Timestamp: "2024-01-01T10:00:00Z", SpotPrice: 0.1, OriginPrice: 1.0, ZoneId: "cn-hangzhou-a"},
		},
		"ecs.broken": {
			{Timestamp: "2024-01-01T10:00:00Z", SpotPrice: 0.1, OriginPrice: 1.0, ZoneId: "cn-hangzhou-a"},
		},
	}

	result := ms.SpotPricesAnalysis(historyPrices, true)

	if len(result) != 1 {
		t.Fatalf("expected 1 result (invalid-metadata row excluded), got %d: %+v", len(result), result)
	}
	if result[0].InstanceTypeId != "ecs.valid" {
		t.Errorf("expected only ecs.valid to remain, got %s", result[0].InstanceTypeId)
	}
}
