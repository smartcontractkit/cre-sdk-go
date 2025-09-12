package http_test

import (
	"context"
	"encoding/binary"
	"log/slog"
	"reflect"
	"testing"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/networking/http"
	httpmock "github.com/smartcontractkit/cre-sdk-go/capabilities/networking/http/mock"
	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/testutils"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var anyResponse = &http.Response{
	StatusCode: 200,
	Headers:    map[string]string{"Content-Type": "application/json"},
	Body:       []byte(`{"message": "success"}`),
}

var anyContext = []byte{4, 5, 6}
var anyReport = `prepended metadata: {"data": "example"}`
var anySigs = []*sdk.AttributedSignature{
	{
		Signature: []byte{7, 8, 9},
		SignerId:  1,
	},
	{
		Signature: []byte{10, 11, 12},
		SignerId:  2,
	},
}
var anyReportResponse = &sdk.ReportResponse{
	ConfigDigest:  []byte{1, 2, 3},
	SeqNr:         112,
	ReportContext: anyContext,
	RawReport:     []byte(anyReport),
	Sigs:          anySigs,
}

func TestClient_SendReport(t *testing.T) {
	testSendReport(t, func(rt cre.Runtime, report *cre.Report) (*http.Response, error) {
		return cre.RunInNodeMode("", rt, func(_ string, nrt cre.NodeRuntime) (*http.Response, error) {
			client := &http.Client{}
			return client.SendReport(nrt, makeRequest(report)).Await()
		}, cre.ConsensusIdenticalAggregation[*http.Response]()).Await()
	})
}

func TestSendRequester_SendReport(t *testing.T) {
	testSendReport(t, func(rt cre.Runtime, report *cre.Report) (*http.Response, error) {
		client := &http.Client{}
		return http.SendRequest("", rt, client, func(_ string, _ *slog.Logger, sendRequester *http.SendRequester) (*http.Response, error) {
			return sendRequester.SendReport(makeRequest(report)).Await()
		}, cre.ConsensusIdenticalAggregation[*http.Response]()).Await()
	})
}

func testSendReport(t *testing.T, sendReport func(rt cre.Runtime, report *cre.Report) (*http.Response, error)) {
	report, err := cre.X_GeneratedCodeOnly_WrapReport(anyReportResponse)
	require.NoError(t, err)

	c, err := httpmock.NewClientCapability(t)
	require.NoError(t, err)

	c.SendRequest = func(_ context.Context, input *http.Request) (*http.Response, error) {
		return assertReport(t, input)
	}

	rt := testutils.NewRuntime(t, map[string]string{})

	response, err := sendReport(rt, report)
	require.NoError(t, err)
	require.True(t, proto.Equal(anyResponse, response))
}

func assertReport(t *testing.T, input *http.Request) (*http.Response, error) {
	t.Helper()

	// NOTE: Using direct t directly instead of assert/require functions
	// because assert/require don't work reliably when called from another goroutine
	if input == nil {
		t.Fatal("Input request is nil")
		return anyResponse, nil
	}
	if input.Url != "https://example.com/api/report" {
		t.Errorf("Expected URL %q, got %q", "https://example.com/api/report", input.Url)
	}
	if input.Method != "POST" {
		t.Errorf("Expected method %q, got %q", "POST", input.Method)
	}
	expectedHeaders := map[string]string{"Content-Type": "application/json"}
	if !reflect.DeepEqual(input.Headers, expectedHeaders) {
		t.Errorf("Expected headers %v, got %v", expectedHeaders, input.Headers)
	}

	var expectedBody []byte
	expectedBody = append(expectedBody, anyReport...)
	expectedBody = append(expectedBody, anyContext...)
	for _, sig := range anySigs {
		expectedBody = append(expectedBody, sig.Signature...)
		expectedBody = binary.LittleEndian.AppendUint32(expectedBody, sig.SignerId)
	}
	if !reflect.DeepEqual(input.Body, expectedBody) {
		t.Errorf("Expected body %v, got %v", expectedBody, input.Body)
	}

	if input.TimeoutMs != int32(54321) {
		t.Errorf("Expected timeout %d, got %d", int32(54321), input.TimeoutMs)
	}
	expectedCacheSettings := &http.CacheSettings{MaxAgeMs: 60000, ReadFromCache: true}
	if !reflect.DeepEqual(input.CacheSettings, expectedCacheSettings) {
		t.Errorf("Expected cache settings %v, got %v", expectedCacheSettings, input.CacheSettings)
	}
	return anyResponse, nil
}

func makeRequest(report *cre.Report) *http.ReportRequest {
	return &http.ReportRequest{
		Url:           "https://example.com/api/report",
		Method:        "POST",
		Headers:       map[string]string{"Content-Type": "application/json"},
		Report:        report,
		TimeoutMs:     54321,
		CacheSettings: &http.CacheSettings{MaxAgeMs: 60000, ReadFromCache: true},
	}
}
