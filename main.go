package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	ecsService "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

var (
	accessKeyId     = flag.String("accessKeyId", "", "Your accessKeyId of cloud account")
	accessKeySecret = flag.String("accessKeySecret", "", "Your accessKeySecret of cloud account")
	region          = flag.String("region", "cn-hangzhou", "The region of spot instances")
	cpu             = flag.Int("mincpu", 1, "Min cores of spot instances")
	memory          = flag.Int("minmem", 2, "Min memory of spot instances")
	maxCpu          = flag.Int("maxcpu", 32, "Max cores of spot instances ")
	maxMemory       = flag.Int("maxmem", 64, "Max memory of spot instances")
	family          = flag.String("family", "", "The spot instance family you want (e.g. ecs.n1,ecs.n2)")
    arch            = flag.String("arch", "", "CPU architecture filter: x86_64 or arm64")
	cutoff          = flag.Int("cutoff", 2, "Discount of the spot instance prices")
	limit           = flag.Int("limit", 20, "Limit of the spot instances")
	resolution      = flag.Int("resolution", 7, "The window of price history analysis")
	jsonOutput      = flag.Bool("json", false, "Output results in JSON format")
)

func main() {
	flag.Parse()

	client, err := ecsService.NewClientWithAccessKey(*region, *accessKeyId, *accessKeySecret)
	if err != nil {
		if *jsonOutput {
			outputJSONError("Failed to create ecs client", err.Error())
		} else {
			panic(fmt.Sprintf("Failed to create ecs client,because of %v", err))
		}
		return
	}

	metastore := NewMetaStore(client)

	err = metastore.Initialize(*region, *jsonOutput)
	if err != nil {
		if *jsonOutput {
			outputJSONError("Failed to initialize metastore", err.Error())
		} else {
			panic(fmt.Sprintf("Failed to initialize metastore,because of %v", err))
		}
		return
	}

    instanceTypes := metastore.FilterInstances(*cpu, *memory, *maxCpu, *maxMemory, *family, *arch, *jsonOutput)

	historyPrices := metastore.FetchSpotPrices(instanceTypes, *resolution, *jsonOutput)

	sortedInstancePrices := metastore.SpotPricesAnalysis(historyPrices, *jsonOutput)

	metastore.PrintPriceRank(sortedInstancePrices, *cutoff, *limit, *jsonOutput)
}

// JSON 错误输出结构
type JSONError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// 输出 JSON 格式的错误信息
func outputJSONError(message, details string) {
	errorResponse := JSONError{
		Error:   message,
		Message: details,
	}

	jsonData, err := json.MarshalIndent(errorResponse, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "{\"error\":\"Failed to marshal error\",\"message\":\"%s\"}\n", err.Error())
		return
	}

	fmt.Println(string(jsonData))
	os.Exit(1)
}
