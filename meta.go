package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	ecsService "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/fatih/color"
)

const (
	TimeLayout = "2006-01-02T15:04:05Z"
)

type MetaStore struct {
	*ecsService.Client
	InstanceFamilyCache map[string]ecsService.InstanceType
}

// Initialize the instance type
func (ms *MetaStore) Initialize(region string, jsonOutput bool) error {
	req := ecsService.CreateDescribeInstanceTypesRequest()
	req.RegionId = region
	resp, err := ms.DescribeInstanceTypes(req)
	if err != nil {
		return fmt.Errorf("failed to DescribeInstanceTypes: %v", err)
	}
	instanceTypes := resp.InstanceTypes.InstanceType

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

	zoneStocks := d_resp.AvailableZones.AvailableZone

	for instanceTypeId := range ms.InstanceFamilyCache {
		found := 0
		for _, zoneStock := range zoneStocks {
			for _, resource := range zoneStock.AvailableResources.AvailableResource[0].SupportedResources.SupportedResource {
				if resource.Value == instanceTypeId {
					found = 1
					break
				}
			}
			if found == 1 {
				break
			}
		}
		if found == 0 {
			delete(ms.InstanceFamilyCache, instanceTypeId)
		}
	}

	if !jsonOutput {
		fmt.Printf("Initialize cache ready with %d kinds of instanceTypes\n", len(instanceTypes))
	}
	return nil
}

// Get the instanceType with in the range.
func (ms *MetaStore) FilterInstances(cpu, memory, maxCpu, maxMemory int, family string, arch string, jsonOutput bool) (instanceTypes []string) {
	instanceTypes = make([]string, 0)

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

// Fetch spot price history
func (ms *MetaStore) FetchSpotPrices(instanceTypes []string, resolution int, jsonOutput bool) (historyPrices map[string][]ecsService.SpotPriceType) {

	historyPrices = make(map[string][]ecsService.SpotPriceType)

	for _, instanceType := range instanceTypes {
		req := ecsService.CreateDescribeSpotPriceHistoryRequest()
		req.NetworkType = "vpc"
		req.InstanceType = instanceType
		req.IoOptimized = "optimized"
		resp, err := ms.DescribeSpotPriceHistory(req)

		resolutionDuration := time.Duration(resolution*-1*24) * time.Hour
		req.StartTime = time.Now().Add(resolutionDuration).Format(TimeLayout)
		if err != nil {
			continue
		}

		historyPrices[instanceType] = resp.SpotPrices.SpotPriceType
	}

	if !jsonOutput {
		fmt.Printf("Fetch %d kinds of InstanceTypes prices successfully.\n", len(instanceTypes))
	}

	return historyPrices
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
			ip := CreateInstancePrice(meta, zoneId, price)
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
			InstanceFamily: price.InstanceType.InstanceTypeFamily,
			Arch:           normalizeArch(getInstanceArch(price.InstanceType)),
		}
		jsonResults = append(jsonResults, jsonResult)
	}

	jsonData, err := json.MarshalIndent(jsonResults, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
}

func NewMetaStore(client *ecsService.Client) *MetaStore {
	return &MetaStore{
		Client:              client,
		InstanceFamilyCache: make(map[string]ecsService.InstanceType),
	}
}
