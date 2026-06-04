package http_test

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/networking/http"
	httpmock "github.com/smartcontractkit/cre-sdk-go/capabilities/networking/http/mock"
	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/testutils"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

var anyResponse = &http.Response{
	StatusCode: 200,
	Headers:    map[string]string{"Content-Type": "application/json"},
	Body:       []byte(`{"message": "success"}`),
}

var anyContext = mustDecodeHex("000e8ce31db48e5e44619d24d9dadfc5f22a34db8205b2b25cd831eab02244c500000000000000000000000000000000000000000000000000000001612bcb000000000000000000000000000000000000000000000000000000000000000000")
var anyReport = mustDecodeHex("010d46e9e05b28a322cd96c94f1f60c236a1b52bd446d98396f1a84afae388836a69c96faa000000010000000100a3bb3b17a1053fe409147c2b170af4866c182070905e5a0f314be04c8296a336613263393832646464b0f2d38245dd6d397ebbdb5a814b753d56c30715000148656c6c6f2c20576f726c6421")
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
	testSendReport(t, func(rt cre.Runtime, report cre.Report) (*http.Response, error) {
		return cre.RunInNodeMode("", rt, func(_ string, nrt cre.NodeRuntime) (*http.Response, error) {
			client := &http.Client{}
			return client.SendReport(nrt, report, reportToRequest).Await()
		}, cre.ConsensusIdenticalAggregation[*http.Response]()).Await()
	})
}

func TestSendRequester_SendReport(t *testing.T) {
	testSendReport(t, func(rt cre.Runtime, report cre.Report) (*http.Response, error) {
		client := &http.Client{}
		return http.SendRequest("", rt, client, func(_ string, _ *slog.Logger, sendRequester *http.SendRequester) (*http.Response, error) {
			return sendRequester.SendReport(report, reportToRequest).Await()
		}, cre.ConsensusIdenticalAggregation[*http.Response]()).Await()
	})
}

func testSendReport(t *testing.T, sendReport func(rt cre.Runtime, report cre.Report) (*http.Response, error)) {
	report, err := cre.X_GeneratedCodeOnly_WrapReport(anyReportResponse)
	require.NoError(t, err)

	c, err := httpmock.NewClientCapability(t)
	require.NoError(t, err)

	c.SendRequest = func(_ context.Context, input *http.Request) (*http.Response, error) {
		return assertReport(t, input)
	}

	rt := testutils.NewRuntime(t, testutils.Secrets{})

	response, err := sendReport(rt, *report)
	require.NoError(t, err)
	require.True(t, proto.Equal(anyResponse, response))
}

func reportToRequest(reportResponse *sdk.ReportResponse) (*http.Request, error) {
	var body []byte
	body = append(body, reportResponse.RawReport...)
	body = append(body, reportResponse.ReportContext...)
	for _, sig := range reportResponse.Sigs {
		body = append(body, sig.Signature...)
		body = binary.LittleEndian.AppendUint32(body, sig.SignerId)
	}

	return &http.Request{
		Url:           "https://example.com/api/report",
		Method:        "POST",
		Headers:       map[string]string{"Content-Type": "application/json"},
		Body:          body,
		Timeout:       durationpb.New(time.Duration(54321)),
		CacheSettings: &http.CacheSettings{MaxAge: durationpb.New(time.Duration(600000)), Store: true},
	}, nil
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

	if input.Timeout.AsDuration() != time.Duration(54321) {
		t.Errorf("Expected timeout %v, got %v", durationpb.New(time.Duration(54321)), input.Timeout)
	}
	expectedCacheSettings := &http.CacheSettings{MaxAge: durationpb.New(time.Duration(600000)), Store: true}
	if !proto.Equal(input.CacheSettings, expectedCacheSettings) {
		t.Errorf("Expected cache settings %v, got %v", expectedCacheSettings, input.CacheSettings)
	}
	return anyResponse, nil
}

func mustDecodeHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
