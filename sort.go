package main

import (
	"fmt"
	"math"
	"time"

	ecsService "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

// data structure of instance prices
type InstancePrice struct {
	ecsService.InstanceType
	ZoneId       string
	PricePerCore float64
	Price        string
	Discount     float64
	Possibility  float64
}

// JSON output structure
type JSONOutput struct {
	InstanceTypeId string  `json:"instanceTypeId"`
	ZoneId         string  `json:"zoneId"`
	PricePerCore   float64 `json:"pricePerCore"`
	Discount       float64 `json:"discount"`
	Possibility    float64 `json:"possibility"`
	CpuCoreCount   int     `json:"cpuCoreCount"`
	MemorySize     float64 `json:"memorySize"`
	InstanceFamily string  `json:"instanceFamily"`
	Arch           string  `json:"arch"`
}

// sorted structure of
type SortedInstancePrices []InstancePrice

func (sp SortedInstancePrices) Len() int {
	return len(sp)
}

func (sp SortedInstancePrices) Less(i, j int) bool {
	return sp[i].PricePerCore < sp[j].PricePerCore
}

func (sp SortedInstancePrices) Swap(i, j int) {
	sp[i], sp[j] = sp[j], sp[i]
}

// CreateInstancePrice builds a ranked price row for one (instance, zone). It
// returns ok=false when the ranking inputs are unknown or invalid (missing CPU
// count, missing/zero on-demand price, or no parseable timestamp) so the caller
// drops the row entirely. A zero-valued PricePerCore/Discount from missing data
// would otherwise sort to the top and pass the discount cutoff as a "best deal",
// mis-ranking the result; rejecting such rows also keeps NaN/Inf out of the
// sort comparator and the JSON output.
func CreateInstancePrice(meta ecsService.InstanceType, zoneId string, prices []ecsService.SpotPriceType) (InstancePrice, bool) {
	latestPrice := FindLatestPrice(prices)
	if meta.CpuCoreCount <= 0 || latestPrice.OriginPrice <= 0 || latestPrice.Timestamp == "" {
		return InstancePrice{}, false
	}
	return InstancePrice{
		InstanceType: meta,
		ZoneId:       zoneId,
		PricePerCore: latestPrice.SpotPrice / float64(meta.CpuCoreCount),
		Price:        fmt.Sprintf("%f", latestPrice.SpotPrice),
		Discount:     10 * latestPrice.SpotPrice / latestPrice.OriginPrice,
		Possibility:  GetPossibility(prices),
	}, true
}

func FindLatestPrice(prices []ecsService.SpotPriceType) ecsService.SpotPriceType {
	var latestPrice ecsService.SpotPriceType
	var latestDate time.Time
	haveLatest := false

	for _, price := range prices {
		// Skip entries with an unparseable timestamp instead of panicking — a
		// single malformed value in the API response should not terminate the
		// whole CLI. The zero value is returned when every entry is malformed.
		currentDate, err := time.Parse(time.RFC3339, price.Timestamp)
		if err != nil {
			continue
		}
		if !haveLatest || currentDate.After(latestDate) {
			latestPrice = price
			latestDate = currentDate
			haveLatest = true
		}
	}

	return latestPrice
}

func GetPossibility(prices []ecsService.SpotPriceType) float64 {
	if len(prices) == 0 {
		return 0.0
	}
	variance := 0.0
	sigma := 0.0

	for _, price := range prices {
		deviation := price.SpotPrice - 0.1*price.OriginPrice
		variance += deviation * deviation
	}

	sigma = math.Sqrt(variance / float64(len(prices)))

	return sigma
}
