package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	ecsService "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/fatih/color"
)

const (
	TimeLayout = "2006-01-02T15:04:05Z"
)

// ecsAPI is the subset of the Aliyun ECS client surface this tool uses.
// Declaring it as an interface (rather than embedding *ecsService.Client
// directly) lets tests inject a fake client to exercise the API-dependent
// paths: Initialize (instance-type + availability fetch) and FetchSpotPrices
// (paginated spot price history). The concrete *ecsService.Client satisfies
// ecsAPI via Go's structural typing.
type ecsAPI interface {
	DescribeInstanceTypes(*ecsService.DescribeInstanceTypesRequest) (*ecsService.DescribeInstanceTypesResponse, error)
	DescribeAvailableResource(*ecsService.DescribeAvailableResourceRequest) (*ecsService.DescribeAvailableResourceResponse, error)
	DescribeSpotPriceHistory(*ecsService.DescribeSpotPriceHistoryRequest) (*ecsService.DescribeSpotPriceHistoryResponse, error)
}

type MetaStore struct {
	ecsAPI
	InstanceFamilyCache map[string]ecsService.InstanceType
}

// Initialize the instance type
func (ms *MetaStore) Initialize(region string, jsonOutput bool) error {
	var instanceTypes []ecsService.InstanceType
	nextToken := ""

	for {
		req := ecsService.CreateDescribeInstanceTypesRequest()
		req.RegionId = region
		req.MaxResults = requests.NewInteger(100)
		if nextToken != "" {
			req.NextToken = nextToken
		}
		resp, err := ms.DescribeInstanceTypes(req)
		if err != nil {
			return fmt.Errorf("failed to DescribeInstanceTypes: %v", err)
		}
		instanceTypes = append(instanceTypes, resp.InstanceTypes.InstanceType...)
		nextToken = resp.NextToken
		if nextToken == "" {
			break
		}
	}

	for _, instanceType := range instanceTypes {
		ms.InstanceFamilyCache[instanceType.InstanceTypeId] = instanceType
	}

	d_req := ecsService.CreateDescribeAvailableResourceRequest()
	d_req.RegionId = region
	d_req.DestinationResource = "InstanceType"
	d_req.InstanceChargeType = "PostPaid"
	d_req.SpotStrategy = "SpotWithPriceLimit"
	d_resp, err := ms.DescribeAvailableResource(d_req)
	if err != nil {
		return fmt.Errorf("failed to get available resource: %v", err)
	}

	// Build a set of instance type ids that have spot availability in at least
	// one zone. Guards each AvailableResource slice (a sold-out / empty zone
	// returns an empty slice and the old code indexed [0] unconditionally,
	// panicking) and turns the O(N*Z*R) nested scan into an O(N+Z*R) set
	// membership check.
	zoneStocks := d_resp.AvailableZones.AvailableZone
	available := make(map[string]struct{})
	for _, zoneStock := range zoneStocks {
		if len(zoneStock.AvailableResources.AvailableResource) == 0 {
			continue
		}
		for _, resource := range zoneStock.AvailableResources.AvailableResource[0].SupportedResources.SupportedResource {
			if resource.Value != "" {
				available[resource.Value] = struct{}{}
			}
		}
	}
	for instanceTypeId := range ms.InstanceFamilyCache {
		if _, ok := available[instanceTypeId]; !ok {
			delete(ms.InstanceFamilyCache, instanceTypeId)
		}
	}

	if !jsonOutput {
		fmt.Printf("Initialize cache ready with %d kinds of instanceTypes\n", len(ms.InstanceFamilyCache))
	}
	return nil
}

// Get the instanceType with in the range.
func (ms *MetaStore) FilterInstances(cpu, memory, maxCpu, maxMemory int, family string, instanceType string, arch string, jsonOutput bool) (instanceTypes []string) {
	instanceTypes = make([]string, 0)

	// If instanceType is specified, use it directly (skip all other filters)
	if strings.TrimSpace(instanceType) != "" {
		instanceTypeList := strings.Split(instanceType, ",")
		for _, it := range instanceTypeList {
			it = strings.TrimSpace(it)
			if it == "" {
				continue
			}
			// Verify the instance type exists in cache
			if _, ok := ms.InstanceFamilyCache[it]; ok {
				// Skip all filters (CPU, memory, architecture) when instanceType is specified
				instanceTypes = append(instanceTypes, it)
			}
		}

		if !jsonOutput {
			fmt.Printf("Filter %d of %d kinds of instanceTypes.\n", len(instanceTypes), len(ms.InstanceFamilyCache))
		}

		return instanceTypes
	}

	// Otherwise, use family-based filtering (existing logic)
	instancesFamily := strings.Split(family, ",")

	for key, instanceType := range ms.InstanceFamilyCache {
		if instanceType.CpuCoreCount >= cpu && instanceType.CpuCoreCount <= maxCpu &&
			instanceType.MemorySize >= float64(memory) && instanceType.MemorySize <= float64(maxMemory) {
			// architecture filter when provided
			if strings.TrimSpace(arch) != "" {
				if normalizeArch(getInstanceArch(instanceType)) != normalizeArch(arch) {
					continue
				}
			}
			for _, instanceFamily := range instancesFamily {
				if strings.Contains(key, instanceFamily) {
					instanceTypes = append(instanceTypes, key)
					break
				}
			}

		}
	}

	if !jsonOutput {
		fmt.Printf("Filter %d of %d kinds of instanceTypes.\n", len(instanceTypes), len(ms.InstanceFamilyCache))
	}

	return instanceTypes
}

// normalizeArch converts various aliases to linux-style names
// accepted inputs: x86_64, amd64, x86, X86, ARM, arm64
func normalizeArch(a string) string {
	aa := strings.ToLower(strings.TrimSpace(a))
	switch aa {
	case "amd64", "x86_64", "x86", "x86-64", "x64":
		return "x86_64"
	case "arm64", "aarch64", "arm" /* some apis may return ARM */ :
		return "arm64"
	default:
		return aa
	}
}

// getInstanceArch extracts the architecture from the instance type metadata.
// Prefer the SDK-provided CpuArchitecture when available; otherwise try to infer from the id/family.
func getInstanceArch(it ecsService.InstanceType) string {
	// Try field CpuArchitecture if populated by SDK
	if strings.TrimSpace(it.CpuArchitecture) != "" {
		return it.CpuArchitecture
	}
	// Fallback heuristic using instance type id/family naming conventions
	id := strings.ToLower(it.InstanceTypeId)
	fam := strings.ToLower(it.InstanceTypeFamily)
	// Common ARM families on Alibaba Cloud often contain a trailing 'g' (e.g., c6g) or y-series
	if strings.Contains(id, ".c6g") || strings.Contains(id, ".g6g") || strings.Contains(id, ".r6g") ||
		strings.Contains(id, ".c8y") || strings.Contains(id, ".g8y") || strings.Contains(id, ".r8y") ||
		strings.Contains(fam, "c6g") || strings.Contains(fam, "g6g") || strings.Contains(fam, "r6g") ||
		strings.Contains(fam, "c8y") || strings.Contains(fam, "g8y") || strings.Contains(fam, "r8y") {
		return "arm64"
	}
	return "x86_64"
}

// FetchSpotPrices retrieves spot price history for each instance type over the
// last `resolution` days, paginating the DescribeSpotPriceHistory API fully
// (it returns only one page at a time keyed by Offset/NextOffset).
//
// StartTime/EndTime are set BEFORE the call so the resolution window is
// actually transmitted (previously StartTime was assigned after the call, making
// --resolution a no-op). Per-instance fetch failures are reported on stderr
// (table mode); the returned error is non-nil only when every instance type
// failed, so a single transient error no longer silently empties the output.
func (ms *MetaStore) FetchSpotPrices(instanceTypes []string, resolution int, jsonOutput bool) (map[string][]ecsService.SpotPriceType, error) {
	if resolution <= 0 {
		return nil, fmt.Errorf("resolution must be a positive number of days, got %d", resolution)
	}

	// EndTime must be strictly less than "now" — the API rejects EndTime >= now
	// with InvalidParams.EndTime. Use UTC so the 'Z' layout suffix is truthful:
	// formatting local time with a Z label mislabels the zone and, east of UTC,
	// submits a future EndTime. The 1-minute buffer absorbs clock skew + latency.
	endTime := time.Now().UTC().Add(-time.Minute)
	startTime := endTime.AddDate(0, 0, -resolution)

	historyPrices := make(map[string][]ecsService.SpotPriceType)

	for _, instanceType := range instanceTypes {
		req := ecsService.CreateDescribeSpotPriceHistoryRequest()
		req.NetworkType = "vpc"
		req.InstanceType = instanceType
		req.IoOptimized = "optimized"
		req.StartTime = startTime.Format(TimeLayout)
		req.EndTime = endTime.Format(TimeLayout)

		// Paginate: the API returns a page plus NextOffset (0 == no more pages).
		// Also break on an empty page to avoid an infinite loop if the API ever
		// returns a non-zero NextOffset with no data.
		var collected []ecsService.SpotPriceType
		offset := 0
		for {
			req.Offset = requests.NewInteger(offset)
			resp, err := ms.DescribeSpotPriceHistory(req)
			if err != nil {
				// Warnings go to stderr so JSON output (stdout) stays clean yet
				// the signal is visible in both table and JSON modes.
				fmt.Fprintf(os.Stderr, "Warning: failed to fetch spot prices for %s: %v\n", instanceType, err)
				break
			}
			collected = append(collected, resp.SpotPrices.SpotPriceType...)
			if resp.NextOffset == 0 || len(resp.SpotPrices.SpotPriceType) == 0 {
				break
			}
			if resp.NextOffset <= offset {
				break // offset not advancing; guard against a stuck NextOffset response
			}
			offset = resp.NextOffset
		}
		if len(collected) > 0 {
			historyPrices[instanceType] = collected
		}
	}

	if !jsonOutput {
		fmt.Printf("Fetched spot prices for %d of %d instanceTypes.\n", len(historyPrices), len(instanceTypes))
	}

	if len(instanceTypes) > 0 && len(historyPrices) == 0 {
		return historyPrices, fmt.Errorf("failed to fetch spot prices for all %d instance types", len(instanceTypes))
	}
	return historyPrices, nil
}

// Print spot history sort and rank
func (ms *MetaStore) SpotPricesAnalysis(historyPrices map[string][]ecsService.SpotPriceType, jsonOutput bool) SortedInstancePrices {
	sp := make(SortedInstancePrices, 0)
	for instanceTypeId, prices := range historyPrices {
		var meta ecsService.InstanceType
		if m, ok := ms.InstanceFamilyCache[instanceTypeId]; !ok {
			continue
		} else {
			meta = m
		}

		priceAZMap := make(map[string][]ecsService.SpotPriceType)
		for _, price := range prices {
			if priceAZMap[price.ZoneId] == nil {
				priceAZMap[price.ZoneId] = make([]ecsService.SpotPriceType, 0)
			}
			priceAZMap[price.ZoneId] = append(priceAZMap[price.ZoneId], price)
		}

		for zoneId, price := range priceAZMap {
			ip, ok := CreateInstancePrice(meta, zoneId, price)
			if !ok {
				continue
			}
			sp = append(sp, ip)
		}
	}

	if !jsonOutput {
		fmt.Printf("Successfully compare %d kinds of instanceTypes\n", len(sp))
	}
	return sp
}

func (ms *MetaStore) PrintPriceRank(prices SortedInstancePrices, cutoff int, limit int, jsonOutput bool) {
	sort.Sort(prices)

	if jsonOutput {
		ms.printJSONOutput(prices, limit)
		return
	}

	color.Green("%30s %20s %15s %15s %15s\n", "InstanceTypeId", "ZoneId", "Price(Core)", "Discount", "ratio")

	for index, price := range prices {
		if index >= limit {
			break
		}
		if price.Discount <= float64(cutoff) {
			color.Green("%30s %20s %15.4f %15.1f %15.1f\n", price.InstanceTypeId, price.ZoneId, price.PricePerCore, price.Discount, price.Possibility)
		} else {
			color.Blue("%30s %20s %15.4f %15.1f %15.1f\n", price.InstanceTypeId, price.ZoneId, price.PricePerCore, price.Discount, price.Possibility)
		}
	}
}

func (ms *MetaStore) printJSONOutput(prices SortedInstancePrices, limit int) {
	var jsonResults []JSONOutput

	for index, price := range prices {
		if index >= limit {
			break
		}

		jsonResult := JSONOutput{
			InstanceTypeId: price.InstanceTypeId,
			ZoneId:         price.ZoneId,
			PricePerCore:   price.PricePerCore,
			Discount:       price.Discount,
			Possibility:    price.Possibility,
			CpuCoreCount:   price.CpuCoreCount,
			MemorySize:     price.MemorySize,
			InstanceFamily: price.InstanceTypeFamily,
			Arch:           normalizeArch(getInstanceArch(price.InstanceType)),
		}
		jsonResults = append(jsonResults, jsonResult)
	}

	jsonData, err := json.MarshalIndent(jsonResults, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
}

func NewMetaStore(client ecsAPI) *MetaStore {
	return &MetaStore{
		ecsAPI:              client,
		InstanceFamilyCache: make(map[string]ecsService.InstanceType),
	}
}
