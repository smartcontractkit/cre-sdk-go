package confidentialhttpcompatibility

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	chttp "github.com/smartcontractkit/cre-sdk-go/capabilities/networking/confidentialhttp"
	"github.com/smartcontractkit/cre-sdk-go/cre"
)

type roundTripper struct {
	runtime cre.Runtime
	options []chttp.RequestOption
}

// NewRoundTripper provides a compatibility shim for HTTP libraries that transforms HTTP requests to use the confidential HTTP capability.
// If injectionFn is nil, no secrets or template values will be added to the call before its made
func NewRoundTripper(runtime cre.Runtime, options ...chttp.RequestOption) http.RoundTripper {
	return &roundTripper{runtime: runtime, options: options}
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

	req := &chttp.HTTPRequest{
		Url:          request.URL.String(),
		Method:       request.Method,
		Body:         &chttp.HTTPRequest_BodyBytes{BodyBytes: body},
		MultiHeaders: headers,
	}

	response, err := client.Send(r.runtime, req, r.options...).Await()
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
