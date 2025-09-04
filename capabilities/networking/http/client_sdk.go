package http

import (
	"encoding/binary"

	"github.com/smartcontractkit/cre-sdk-go/cre"
)

type ReportRequest struct {
	Url       string
	Method    string
	Headers   map[string]string
	Report    *cre.Report
	TimeoutMs int32

	// CacheSettings is more limited than on a Request, as reports may contain different sets of signatures on different nodes, leading to a cache miss.
	CacheSettings *CacheSettings
}

// SendReport functions the same as SendRequest, but takes a [cre.Report] as the body.
// Note that caching is limited as reports may contain different sets of signatures on different nodes, leading to a cache miss.
func (c *SendRequester) SendReport(input *ReportRequest) cre.Promise[*Response] {
	return c.SendRequest(reportRequestToRequest(input))
}

// SendReport functions the same as SendRequest, but takes a [cre.Report] as the body.
// Note that caching is limited as reports may contain different sets of signatures on different nodes, leading to a cache miss.
func (c *Client) SendReport(runtime cre.NodeRuntime, input *ReportRequest) cre.Promise[*Response] {
	return c.SendRequest(runtime, reportRequestToRequest(input))
}

func reportRequestToRequest(in *ReportRequest) *Request {
	report := in.Report.X_GeneratedCodeOnly_Unwrap()
	sigLen := 0
	if len(report.Sigs) != 0 {
		sigLen = len(report.Sigs) * (len(report.Sigs[0].Signature) + 4)
	}
	body := make([]byte, len(report.RawReport)+len(report.ReportContext)+sigLen)
	copy(body, report.RawReport)
	copy(body[len(report.RawReport):], report.ReportContext)
	pos := len(report.RawReport) + len(report.ReportContext)
	for _, sig := range report.Sigs {
		copy(body[pos:], sig.Signature)
		pos += len(sig.Signature)
		binary.LittleEndian.PutUint32(body[pos:], sig.SignerId)
		pos += 4
	}
	return &Request{
		Url:           in.Url,
		Method:        in.Method,
		Headers:       in.Headers,
		Body:          body,
		TimeoutMs:     in.TimeoutMs,
		CacheSettings: in.CacheSettings,
	}
}
