package main

import (
	"sort"
	"testing"

	ecsService "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

func TestFindLatestPrice(t *testing.T) {
	tests := []struct {
		name     string
		prices   []ecsService.SpotPriceType
		expected string // expected timestamp
	}{
		{
			name:     "empty prices",
			prices:   []ecsService.SpotPriceType{},
			expected: "",
		},
		{
			name: "single price",
			prices: []ecsService.SpotPriceType{
				{
					Timestamp:   "2024-01-01T10:00:00Z",
					SpotPrice:   0.1,
					OriginPrice: 1.0,
				},
			},
			expected: "2024-01-01T10:00:00Z",
		},
		{
			name: "multiple prices - latest first",
			prices: []ecsService.SpotPriceType{
				{
					Timestamp:   "2024-01-03T10:00:00Z",
					SpotPrice:   0.1,
					OriginPrice: 1.0,
				},
				{
					Timestamp:   "2024-01-02T10:00:00Z",
					SpotPrice:   0.2,
					OriginPrice: 1.0,
				},
				{
					Timestamp:   "2024-01-01T10:00:00Z",
					SpotPrice:   0.3,
					OriginPrice: 1.0,
				},
			},
			expected: "2024-01-03T10:00:00Z",
		},
		{
			name: "multiple prices - latest last",
			prices: []ecsService.SpotPriceType{
				{
					Timestamp:   "2024-01-01T10:00:00Z",
					SpotPrice:   0.3,
					OriginPrice: 1.0,
				},
				{
					Timestamp:   "2024-01-02T10:00:00Z",
					SpotPrice:   0.2,
					OriginPrice: 1.0,
				},
				{
					Timestamp:   "2024-01-03T10:00:00Z",
					SpotPrice:   0.1,
					OriginPrice: 1.0,
				},
			},
			expected: "2024-01-03T10:00:00Z",
		},
		{
			name: "multiple prices - latest in middle",
			prices: []ecsService.SpotPriceType{
				{
					Timestamp:   "2024-01-01T10:00:00Z",
					SpotPrice:   0.3,
					OriginPrice: 1.0,
				},
				{
					Timestamp:   "2024-01-03T10:00:00Z",
					SpotPrice:   0.1,
					OriginPrice: 1.0,
				},
				{
					Timestamp:   "2024-01-02T10:00:00Z",
					SpotPrice:   0.2,
					OriginPrice: 1.0,
				},
			},
			expected: "2024-01-03T10:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindLatestPrice(tt.prices)
			if tt.expected == "" {
				if result.Timestamp != "" {
					t.Errorf("Expected empty timestamp, got %s", result.Timestamp)
				}
			} else {
				if result.Timestamp != tt.expected {
					t.Errorf("Expected timestamp %s, got %s", tt.expected, result.Timestamp)
				}
			}
		})
	}
}

func TestGetPossibility(t *testing.T) {
	tests := []struct {
		name     string
		prices   []ecsService.SpotPriceType
		expected float64
	}{
		{
			name:     "empty prices",
			prices:   []ecsService.SpotPriceType{},
			expected: 0.0,
		},
		{
			name: "single price",
			prices: []ecsService.SpotPriceType{
				{
					SpotPrice:   0.1,
					OriginPrice: 1.0,
				},
			},
			expected: 0.0, // variance is 0 when there's only one price
		},
		{
			name: "multiple prices - stable",
			prices: []ecsService.SpotPriceType{
				{
					SpotPrice:   0.1,
					OriginPrice: 1.0,
				},
				{
					SpotPrice:   0.1,
					OriginPrice: 1.0,
				},
				{
					SpotPrice:   0.1,
					OriginPrice: 1.0,
				},
			},
			expected: 0.0, // all prices are 0.1, variance is 0
		},
		{
			name: "multiple prices - varying",
			prices: []ecsService.SpotPriceType{
				{
					SpotPrice:   0.1,
					OriginPrice: 1.0,
				},
				{
					SpotPrice:   0.2,
					OriginPrice: 1.0,
				},
				{
					SpotPrice:   0.3,
					OriginPrice: 1.0,
				},
			},
			// Expected: sqrt(((0.1-0.1)^2 + (0.2-0.1)^2 + (0.3-0.1)^2) / 3)
			// = sqrt((0 + 0.01 + 0.04) / 3) = sqrt(0.01667) ≈ 0.129
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPossibility(tt.prices)
			if tt.name == "empty prices" {
				if result != tt.expected {
					t.Errorf("Expected %f, got %f", tt.expected, result)
				}
			} else if tt.name == "single price" {
				if result != tt.expected {
					t.Errorf("Expected %f, got %f", tt.expected, result)
				}
			} else if tt.name == "multiple prices - stable" {
				if result != tt.expected {
					t.Errorf("Expected %f, got %f", tt.expected, result)
				}
			} else if tt.name == "multiple prices - varying" {
				// For varying prices, just check it's positive and reasonable
				if result <= 0 {
					t.Errorf("Expected positive value, got %f", result)
				}
				if result > 1.0 {
					t.Errorf("Expected reasonable value (< 1.0), got %f", result)
				}
			}
		})
	}
}

func TestCreateInstancePrice(t *testing.T) {
	meta := ecsService.InstanceType{
		InstanceTypeId:     "ecs.n1.small",
		InstanceTypeFamily: "ecs.n1",
		CpuCoreCount:       2,
		MemorySize:         4.0,
	}

	prices := []ecsService.SpotPriceType{
		{
			Timestamp:   "2024-01-01T10:00:00Z",
			SpotPrice:   0.1,
			OriginPrice: 1.0,
			ZoneId:      "cn-hangzhou-a",
		},
		{
			Timestamp:   "2024-01-02T10:00:00Z",
			SpotPrice:   0.2,
			OriginPrice: 1.0,
			ZoneId:      "cn-hangzhou-a",
		},
		{
			Timestamp:   "2024-01-03T10:00:00Z",
			SpotPrice:   0.15,
			OriginPrice: 1.0,
			ZoneId:      "cn-hangzhou-a",
		},
	}

	result := CreateInstancePrice(meta, "cn-hangzhou-a", prices)

	if result.InstanceTypeId != "ecs.n1.small" {
		t.Errorf("Expected InstanceTypeId ecs.n1.small, got %s", result.InstanceTypeId)
	}
	if result.ZoneId != "cn-hangzhou-a" {
		t.Errorf("Expected ZoneId cn-hangzhou-a, got %s", result.ZoneId)
	}
	// Latest price is 0.15, so PricePerCore should be 0.15 / 2 = 0.075
	expectedPricePerCore := 0.15 / 2.0
	if result.PricePerCore != expectedPricePerCore {
		t.Errorf("Expected PricePerCore %f, got %f", expectedPricePerCore, result.PricePerCore)
	}
	// Discount = 10 * 0.15 / 1.0 = 1.5
	expectedDiscount := 10.0 * 0.15 / 1.0
	if result.Discount != expectedDiscount {
		t.Errorf("Expected Discount %f, got %f", expectedDiscount, result.Discount)
	}
	if result.Possibility <= 0 {
		t.Errorf("Expected positive Possibility, got %f", result.Possibility)
	}
}

func TestSortedInstancePrices(t *testing.T) {
	prices := SortedInstancePrices{
		{
			InstanceType: ecsService.InstanceType{
				InstanceTypeId: "ecs.n1.large",
			},
			PricePerCore: 0.2,
		},
		{
			InstanceType: ecsService.InstanceType{
				InstanceTypeId: "ecs.n1.small",
			},
			PricePerCore: 0.1,
		},
		{
			InstanceType: ecsService.InstanceType{
				InstanceTypeId: "ecs.n1.medium",
			},
			PricePerCore: 0.15,
		},
	}

	// Test Len
	if prices.Len() != 3 {
		t.Errorf("Expected length 3, got %d", prices.Len())
	}

	// Test Less
	if !prices.Less(1, 0) { // 0.1 < 0.2
		t.Error("Expected prices[1] < prices[0]")
	}
	if prices.Less(0, 1) { // 0.2 < 0.1 should be false
		t.Error("Expected prices[0] >= prices[1]")
	}

	// Test Swap
	original0 := prices[0]
	original1 := prices[1]
	prices.Swap(0, 1)
	if prices[0].InstanceTypeId != original1.InstanceTypeId {
		t.Errorf("After swap, prices[0] should be %s, got %s", original1.InstanceTypeId, prices[0].InstanceTypeId)
	}
	if prices[1].InstanceTypeId != original0.InstanceTypeId {
		t.Errorf("After swap, prices[1] should be %s, got %s", original0.InstanceTypeId, prices[1].InstanceTypeId)
	}

	// Test sorting
	sort.Sort(prices)
	expectedOrder := []string{"ecs.n1.small", "ecs.n1.medium", "ecs.n1.large"}
	for i, expected := range expectedOrder {
		if prices[i].InstanceTypeId != expected {
			t.Errorf("After sorting, prices[%d] should be %s, got %s", i, expected, prices[i].InstanceTypeId)
		}
	}
}
