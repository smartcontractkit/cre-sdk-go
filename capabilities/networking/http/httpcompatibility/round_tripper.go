package httpcompatibility

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	chttp "github.com/smartcontractkit/cre-sdk-go/capabilities/networking/http"
	"github.com/smartcontractkit/cre-sdk-go/cre"
)

type roundTripper struct {
	NodeRuntime cre.NodeRuntime
}

func NewRoundTripper(nodeRuntime cre.NodeRuntime) http.RoundTripper {
	return &roundTripper{NodeRuntime: nodeRuntime}
}

func NewCompatibilityWithConsensus[C, T any](
	config C,
	runtime cre.Runtime,
	fn func(config C, logger *slog.Logger, roundTripper http.RoundTripper) (T, error),
	ca cre.ConsensusAggregation[T]) cre.Promise[T] {
	wrapped := func(config C, nodeRuntime cre.NodeRuntime) (T, error) {
		rt := NewRoundTripper(nodeRuntime)
		return fn(config, runtime.Logger(), rt)
	}

	return cre.RunInNodeMode(config, runtime, wrapped, ca)
}

func (r *roundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	client := &chttp.Client{}

	headers := map[string]*chttp.HeaderValues{}
	for name, values := range request.Header {
		headers[name] = &chttp.HeaderValues{Values: values}
	}

	var body []byte
	if request.Body != nil {
		var err error
		body, err = io.ReadAll(request.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	response, err := client.SendRequest(r.NodeRuntime, &chttp.Request{
		Url:          request.URL.String(),
		Method:       request.Method,
		MultiHeaders: headers,
		Body:         body,
	}).Await()

	if err != nil {
		return nil, err
	}

	responseHeaders := http.Header{}
	for name, value := range response.MultiHeaders {
		responseHeaders[name] = make([]string, len(value.Values))
		for i, v := range value.Values {
			responseHeaders[name][i] = v
		}
	}

	return &http.Response{
		Status:     http.StatusText(int(response.StatusCode)),
		StatusCode: int(response.StatusCode),
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
		Header:     responseHeaders,
		Body:       io.NopCloser(bytes.NewReader(response.Body)),
		// TODO verify I should be setting this given the other field's values.
		ContentLength: int64(len(response.Body)),
		// TransferEncoding: nil,
		// Close:            false,
		// Uncompressed:     false,
	}, nil
}
