package http

import (
	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/cre"
)

// SendReport functions the same as SendRequest, but takes a [cre.Report] and a function to convert the inner [sdk.ReportResponse] to a [Request].
// Note that caching is limited as reports may contain different sets of signatures on different nodes, leading to a cache miss.
func (c *SendRequester) SendReport(report cre.Report, fn func(*sdk.ReportResponse) (*Request, error)) cre.Promise[*Response] {
	return c.client.SendReport(c.nodeRuntime, report, fn)
}

// SendReport functions the same as SendRequest, but takes a [cre.Report] and a function to convert the inner [sdk.ReportResponse] to a [Request].
// Note that caching is limited as reports may contain different sets of signatures on different nodes, leading to a cache miss.
func (c *Client) SendReport(runtime cre.NodeRuntime, report cre.Report, fn func(*sdk.ReportResponse) (*Request, error)) cre.Promise[*Response] {
	rawReport := report.X_GeneratedCodeOnly_Unwrap()
	input, err := fn(rawReport)
	if err != nil {
		return cre.PromiseFromResult[*Response](nil, err)
	}
	return c.SendRequest(runtime, input)
}
