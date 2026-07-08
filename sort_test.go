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
			switch tt.name {
			case "empty prices", "single price", "multiple prices - stable":
				// Variance is exactly 0 for these inputs.
				if result != tt.expected {
					t.Errorf("Expected %f, got %f", tt.expected, result)
				}
			case "multiple prices - varying":
				// deviations from 0.1*OriginPrice: 0.0, 0.1, 0.2
				// sigma = sqrt((0 + 0.01 + 0.04) / 3) ≈ 0.1291. Tight tolerance
				// catches a formula regression (previously only 0 < x < 1).
				if result < 0.128 || result > 0.130 {
					t.Errorf("Expected sigma ≈ 0.129, got %f", result)
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

	result, ok := CreateInstancePrice(meta, "cn-hangzhou-a", prices)
	if !ok {
		t.Fatal("expected CreateInstancePrice to accept valid inputs")
	}

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

// TestFindLatestPrice_MalformedTimestamp locks in H2: a single bad timestamp
// is skipped (not fatal) and the latest valid entry is still returned.
func TestFindLatestPrice_MalformedTimestamp(t *testing.T) {
	prices := []ecsService.SpotPriceType{
		{Timestamp: "not-a-date", SpotPrice: 0.9, OriginPrice: 1.0},
		{Timestamp: "2024-01-01T10:00:00Z", SpotPrice: 0.1, OriginPrice: 1.0},
		{Timestamp: "2024-01-03T10:00:00Z", SpotPrice: 0.2, OriginPrice: 1.0},
		{Timestamp: "", SpotPrice: 0.9, OriginPrice: 1.0},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("FindLatestPrice panicked on a malformed timestamp: %v", r)
		}
	}()

	result := FindLatestPrice(prices)
	if result.Timestamp != "2024-01-03T10:00:00Z" {
		t.Errorf("expected the latest valid timestamp, got %q", result.Timestamp)
	}
}

// TestFindLatestPrice_AllMalformed returns a zero value (no panic) when no
// timestamp parses.
func TestFindLatestPrice_AllMalformed(t *testing.T) {
	prices := []ecsService.SpotPriceType{
		{Timestamp: "bad", SpotPrice: 0.1},
		{Timestamp: "also-bad", SpotPrice: 0.2},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("FindLatestPrice panicked: %v", r)
		}
	}()

	if result := FindLatestPrice(prices); result.Timestamp != "" {
		t.Errorf("expected zero value when all timestamps are malformed, got %q", result.Timestamp)
	}
}

// TestCreateInstancePrice_RejectsInvalidInputs locks in the H3 follow-up:
// rows whose ranking inputs are missing/invalid are rejected (ok=false) so they
// are dropped instead of mis-ranking via a zero sentinel. Each guard is exercised
// INDEPENDENTLY — only one dimension is invalid per case, the rest valid — so a
// removed or broken guard actually fails its case (no conflation, per the hetero
// review).
func TestCreateInstancePrice_RejectsInvalidInputs(t *testing.T) {
	validMeta := ecsService.InstanceType{InstanceTypeId: "ecs.valid", InstanceTypeFamily: "ecs.valid", CpuCoreCount: 2}
	validPrice := ecsService.SpotPriceType{Timestamp: "2024-01-01T10:00:00Z", SpotPrice: 0.15, OriginPrice: 1.0}

	// Guard 1: CpuCoreCount == 0 (metadata otherwise valid, price valid).
	if _, ok := CreateInstancePrice(
		ecsService.InstanceType{InstanceTypeId: "ecs.weird", CpuCoreCount: 0},
		"cn-hangzhou-a", []ecsService.SpotPriceType{validPrice}); ok {
		t.Error("CpuCoreCount=0 must be rejected")
	}

	// Guard 2: OriginPrice == 0 (metadata + timestamp valid).
	if _, ok := CreateInstancePrice(validMeta, "cn-hangzhou-a",
		[]ecsService.SpotPriceType{{Timestamp: "2024-01-01T10:00:00Z", SpotPrice: 0.15, OriginPrice: 0}}); ok {
		t.Error("OriginPrice=0 must be rejected")
	}

	// Guard 3: no parseable timestamp (metadata + price fields valid).
	if _, ok := CreateInstancePrice(validMeta, "cn-hangzhou-a",
		[]ecsService.SpotPriceType{{Timestamp: "not-a-date", SpotPrice: 0.15, OriginPrice: 1.0}}); ok {
		t.Error("an unparseable-timestamp-only input must be rejected")
	}

	// Negative control: a genuinely free spot (SpotPrice==0, otherwise valid) is
	// ACCEPTED and legitimately ranks first — only MISSING data is rejected.
	ip, ok := CreateInstancePrice(validMeta, "cn-hangzhou-a",
		[]ecsService.SpotPriceType{{Timestamp: "2024-01-01T10:00:00Z", SpotPrice: 0, OriginPrice: 1.0}})
	if !ok {
		t.Fatal("a free-but-valid instance must be accepted")
	}
	if ip.PricePerCore != 0 || ip.Discount != 0 {
		t.Errorf("free instance should have PricePerCore=0 and Discount=0, got %f/%f", ip.PricePerCore, ip.Discount)
	}
}
