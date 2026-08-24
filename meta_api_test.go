package main

import (
	"fmt"
	"testing"
	"time"

	sdkErrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	ecsService "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

// fakeEcsClient is a stand-in for *ecsService.Client that satisfies ecsAPI.
// It lets Initialize / FetchSpotPrices be exercised without a real Aliyun
// endpoint, which is where the worst pre-fix bugs lived (pagination, the
// empty-AvailableResource panic, and the no-op StartTime).
type fakeEcsClient struct {
	instanceTypes []ecsService.InstanceType
	zones         []ecsService.AvailableZone

	spotPages         []fakePage
	spotErr           error
	failTypes         map[string]bool // per-instance-type forced errors (partial-failure test)
	transientFailures int             // first N spot calls fail with a retryable throttling error
	throttlePages     map[int]bool    // page indices that fail ONCE with a retryable throttle (mid-pagination test)
	spotCalls         int
	spotPageIdx       int // next page to serve; advanced only by a successful page read (retried failures do not consume pages)
	lastStartTime     string
	lastEndTime       string
}

type fakePage struct {
	prices     []ecsService.SpotPriceType
	nextOffset int
}

func (f *fakeEcsClient) DescribeInstanceTypes(*ecsService.DescribeInstanceTypesRequest) (*ecsService.DescribeInstanceTypesResponse, error) {
	resp := &ecsService.DescribeInstanceTypesResponse{}
	resp.InstanceTypes.InstanceType = f.instanceTypes
	return resp, nil
}

func (f *fakeEcsClient) DescribeAvailableResource(*ecsService.DescribeAvailableResourceRequest) (*ecsService.DescribeAvailableResourceResponse, error) {
	resp := &ecsService.DescribeAvailableResourceResponse{}
	resp.AvailableZones.AvailableZone = f.zones
	return resp, nil
}

func (f *fakeEcsClient) DescribeSpotPriceHistory(req *ecsService.DescribeSpotPriceHistoryRequest) (*ecsService.DescribeSpotPriceHistoryResponse, error) {
	f.spotCalls++
	f.lastStartTime = req.StartTime
	f.lastEndTime = req.EndTime
	if f.transientFailures > 0 {
		f.transientFailures--
		return nil, sdkErrors.NewServerError(429, `{"Code":"Throttling","Message":"Requests too frequent"}`, "")
	}
	if f.spotErr != nil {
		return nil, f.spotErr
	}
	if f.failTypes[req.InstanceType] {
		return nil, fmt.Errorf("throttled: %s", req.InstanceType)
	}
	if f.throttlePages[f.spotPageIdx] {
		delete(f.throttlePages, f.spotPageIdx)
		return nil, sdkErrors.NewServerError(429, `{"Code":"Throttling","Message":"Requests too frequent"}`, "")
	}
	if f.spotPageIdx >= len(f.spotPages) {
		return &ecsService.DescribeSpotPriceHistoryResponse{}, nil
	}
	p := f.spotPages[f.spotPageIdx]
	f.spotPageIdx++
	resp := &ecsService.DescribeSpotPriceHistoryResponse{NextOffset: p.nextOffset}
	resp.SpotPrices.SpotPriceType = p.prices
	return resp, nil
}

// zoneWithAvailable builds a zone that advertises the given instance type ids.
func zoneWithAvailable(types ...string) ecsService.AvailableZone {
	supported := make([]ecsService.SupportedResource, 0, len(types))
	for _, ty := range types {
		supported = append(supported, ecsService.SupportedResource{Value: ty})
	}
	return ecsService.AvailableZone{
		ZoneId: "cn-hangzhou-a",
		AvailableResources: ecsService.AvailableResourcesInDescribeResourcesModification{
			AvailableResource: []ecsService.AvailableResource{
				{
					SupportedResources: ecsService.SupportedResourcesInDescribeAvailableResource{
						SupportedResource: supported,
					},
				},
			},
		},
	}
}

// emptyZone builds a zone whose AvailableResource slice is empty — the input
// that used to panic at AvailableResources.AvailableResource[0].
func emptyZone() ecsService.AvailableZone {
	return ecsService.AvailableZone{
		AvailableResources: ecsService.AvailableResourcesInDescribeResourcesModification{
			AvailableResource: []ecsService.AvailableResource{},
		},
	}
}

// TestInitialize_EmptyAvailableResourceNoPanic locks in the H1 fix: an empty
// AvailableResource slice (sold-out zone) must not panic.
func TestInitialize_EmptyAvailableResourceNoPanic(t *testing.T) {
	fake := &fakeEcsClient{
		instanceTypes: []ecsService.InstanceType{{InstanceTypeId: "ecs.n1.small", CpuCoreCount: 2}},
		zones:         []ecsService.AvailableZone{emptyZone()},
	}
	ms := NewMetaStore(fake)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Initialize panicked on empty AvailableResource: %v", r)
		}
	}()

	if err := ms.Initialize("cn-hangzhou", true); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	// Empty zone -> nothing available -> cache fully pruned, no panic.
	if len(ms.InstanceFamilyCache) != 0 {
		t.Errorf("expected empty cache after pruning against an empty zone, got %d", len(ms.InstanceFamilyCache))
	}
}

// TestInitialize_PrunesUnavailable locks in the L4/membership filter: only
// instance types advertised by some zone survive.
func TestInitialize_PrunesUnavailable(t *testing.T) {
	fake := &fakeEcsClient{
		instanceTypes: []ecsService.InstanceType{
			{InstanceTypeId: "ecs.n1.small", CpuCoreCount: 2},
			{InstanceTypeId: "ecs.n2.large", CpuCoreCount: 4},
		},
		zones: []ecsService.AvailableZone{zoneWithAvailable("ecs.n1.small")},
	}
	ms := NewMetaStore(fake)

	if err := ms.Initialize("cn-hangzhou", true); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	if _, ok := ms.InstanceFamilyCache["ecs.n1.small"]; !ok {
		t.Error("ecs.n1.small should remain in cache (it is available)")
	}
	if _, ok := ms.InstanceFamilyCache["ecs.n2.large"]; ok {
		t.Error("ecs.n2.large should be pruned (not available in any zone)")
	}
}

// TestFetchSpotPrices_PaginatesAndSetsStartTime locks in C1 + C2: the API is
// called once per page (3 pages -> 3 calls), all pages are accumulated, and
// StartTime is actually transmitted on the request (previously assigned after
// the call, making --resolution a no-op).
func TestFetchSpotPrices_PaginatesAndSetsStartTime(t *testing.T) {
	fake := &fakeEcsClient{
		spotPages: []fakePage{
			{prices: []ecsService.SpotPriceType{{Timestamp: "2024-01-01T10:00:00Z", SpotPrice: 0.1, OriginPrice: 1.0, ZoneId: "cn-hangzhou-a"}}, nextOffset: 10},
			{prices: []ecsService.SpotPriceType{{Timestamp: "2024-01-02T10:00:00Z", SpotPrice: 0.2, OriginPrice: 1.0, ZoneId: "cn-hangzhou-a"}}, nextOffset: 20},
			{prices: []ecsService.SpotPriceType{{Timestamp: "2024-01-03T10:00:00Z", SpotPrice: 0.3, OriginPrice: 1.0, ZoneId: "cn-hangzhou-a"}}, nextOffset: 0},
		},
	}
	ms := NewMetaStore(fake)

	hist, err := ms.FetchSpotPrices([]string{"ecs.n1.small"}, 7, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fake.spotCalls != 3 {
		t.Errorf("expected 3 paginated DescribeSpotPriceHistory calls, got %d", fake.spotCalls)
	}
	if fake.lastStartTime == "" {
		t.Error("StartTime must be set on the request before the call (was a no-op before the fix)")
	}
	// EndTime must be set AND strictly in the past (UTC): the real Aliyun API
	// rejects EndTime >= now with InvalidParams.EndTime, and formatting local
	// time with a 'Z' suffix mislabels the zone (a future EndTime east of UTC).
	if fake.lastEndTime == "" {
		t.Fatal("EndTime must be set on the request")
	}
	end, err := time.Parse(time.RFC3339, fake.lastEndTime)
	if err != nil {
		t.Fatalf("EndTime must be a parseable RFC3339 timestamp, got %q: %v", fake.lastEndTime, err)
	}
	if !end.Before(time.Now().UTC()) {
		t.Errorf("EndTime must be strictly in the past (UTC), got %v", end)
	}

	got := hist["ecs.n1.small"]
	if len(got) != 3 {
		t.Errorf("expected 3 prices accumulated across pages, got %d", len(got))
	}
}

// TestFetchSpotPrices_AllFailReturnsError locks in M1: when every instance type
// fails to fetch, the caller gets a real error instead of silent empty output.
func TestFetchSpotPrices_AllFailReturnsError(t *testing.T) {
	fake := &fakeEcsClient{spotErr: fmt.Errorf("throttled")}
	ms := NewMetaStore(fake)

	if _, err := ms.FetchSpotPrices([]string{"ecs.n1.small"}, 7, true); err == nil {
		t.Error("expected an error when all instance types fail to fetch")
	}
}

func TestFetchSpotPrices_InvalidResolution(t *testing.T) {
	ms := NewMetaStore(&fakeEcsClient{})
	if _, err := ms.FetchSpotPrices([]string{"ecs.n1.small"}, 0, true); err == nil {
		t.Error("expected an error for resolution <= 0")
	}
}

// TestFetchSpotPrices_PartialFailure locks in the subtler half of M1: when SOME
// (not all) instance types fail, the successful ones are still returned and the
// error is nil. This is the common real-world path (one type throttles) that the
// all-fail test does not cover.
func TestFetchSpotPrices_PartialFailure(t *testing.T) {
	fake := &fakeEcsClient{
		spotPages: []fakePage{
			{prices: []ecsService.SpotPriceType{{Timestamp: "2024-01-01T10:00:00Z", SpotPrice: 0.1, OriginPrice: 1.0, ZoneId: "cn-hangzhou-a"}}, nextOffset: 0},
		},
		failTypes: map[string]bool{"ecs.fail": true},
	}
	ms := NewMetaStore(fake)

	hist, err := ms.FetchSpotPrices([]string{"ecs.ok", "ecs.fail"}, 7, true)
	if err != nil {
		t.Fatalf("partial failure must NOT return an error, got %v", err)
	}
	if len(hist) != 1 {
		t.Fatalf("expected 1 successful entry, got %d: %v", len(hist), hist)
	}
	if _, ok := hist["ecs.ok"]; !ok {
		t.Error("the successful instance type ecs.ok should be in the results")
	}
	if _, ok := hist["ecs.fail"]; ok {
		t.Error("the failed instance type ecs.fail should not be in the results")
	}
}

// TestFetchSpotPrices_RetriesTransientThrottling locks in N4: a transient
// throttling response is retried with backoff instead of dropping the instance
// type for the rest of the run.
func TestFetchSpotPrices_RetriesTransientThrottling(t *testing.T) {
	origSleep := retrySleep
	retrySleep = func(time.Duration) {} // keep the test instant
	defer func() { retrySleep = origSleep }()

	fake := &fakeEcsClient{
		transientFailures: 2, // throttled twice, then the call succeeds
		spotPages: []fakePage{
			{prices: []ecsService.SpotPriceType{{Timestamp: "2024-01-01T10:00:00Z", SpotPrice: 0.1, OriginPrice: 1.0, ZoneId: "cn-hangzhou-a"}}, nextOffset: 0},
		},
	}
	ms := NewMetaStore(fake)

	hist, err := ms.FetchSpotPrices([]string{"ecs.ok"}, 7, true)
	if err != nil {
		t.Fatalf("transient throttling should be retried, not fatal: %v", err)
	}
	if len(hist["ecs.ok"]) != 1 {
		t.Fatalf("expected the fetched price after retries, got %v", hist)
	}
	if fake.spotCalls != 3 { // 2 throttled attempts + 1 success
		t.Errorf("expected 3 calls (initial + 2 retries), got %d", fake.spotCalls)
	}
}

// TestFetchSpotPrices_NonRetryableFailsFast locks in the other half of N4: a
// non-transient error (e.g. invalid parameters) is NOT retried.
func TestFetchSpotPrices_NonRetryableFailsFast(t *testing.T) {
	origSleep := retrySleep
	retrySleep = func(time.Duration) {}
	defer func() { retrySleep = origSleep }()

	fake := &fakeEcsClient{spotErr: serverErr(400, "InvalidParams.EndTime")}
	ms := NewMetaStore(fake)

	if _, err := ms.FetchSpotPrices([]string{"ecs.bad"}, 7, true); err == nil {
		t.Fatal("expected an error when the only instance type fails")
	}
	if fake.spotCalls != 1 {
		t.Errorf("a non-retryable error must fail fast (1 call), got %d", fake.spotCalls)
	}
}

// TestFetchSpotPrices_RetriesMidPaginationThrottle proves the retry wraps EACH
// page call: a throttle striking while fetching page 2 is retried, page 1's
// data survives, and page 3 is still fetched — no page is lost or refetched
// from scratch.
func TestFetchSpotPrices_RetriesMidPaginationThrottle(t *testing.T) {
	origSleep := retrySleep
	retrySleep = func(time.Duration) {}
	defer func() { retrySleep = origSleep }()

	fake := &fakeEcsClient{
		spotPages: []fakePage{
			{prices: []ecsService.SpotPriceType{{Timestamp: "2024-01-01T10:00:00Z", SpotPrice: 0.1, OriginPrice: 1.0, ZoneId: "cn-hangzhou-a"}}, nextOffset: 10},
			{prices: []ecsService.SpotPriceType{{Timestamp: "2024-01-02T10:00:00Z", SpotPrice: 0.2, OriginPrice: 1.0, ZoneId: "cn-hangzhou-a"}}, nextOffset: 20},
			{prices: []ecsService.SpotPriceType{{Timestamp: "2024-01-03T10:00:00Z", SpotPrice: 0.3, OriginPrice: 1.0, ZoneId: "cn-hangzhou-a"}}, nextOffset: 0},
		},
		throttlePages: map[int]bool{1: true}, // page 2 (index 1) throttles exactly once
	}
	ms := NewMetaStore(fake)

	hist, err := ms.FetchSpotPrices([]string{"ecs.n1.small"}, 7, true)
	if err != nil {
		t.Fatalf("a mid-pagination throttle should recover: %v", err)
	}
	got := hist["ecs.n1.small"]
	if len(got) != 3 {
		t.Fatalf("expected all 3 pages' prices after the retry, got %d: %v", len(got), got)
	}
	if fake.spotCalls != 4 { // 3 page fetches + 1 retried page-2 fetch
		t.Errorf("expected 4 calls (3 pages + 1 retry), got %d", fake.spotCalls)
	}
}
