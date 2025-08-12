package cre

import "github.com/smartcontractkit/chainlink-protos/cre/go/sdk"

type ReportRequest = sdk.ReportRequest

type Report struct {
	report *sdk.ReportResponse
}

// X_GeneratedCodeOnly_Unwrap is meant to be used by the code generator only.
func (r *Report) X_GeneratedCodeOnly_Unwrap() *sdk.ReportResponse { //nolint
	return r.report
}

// X_GeneratedCodeOnly_Wrap is meant to be used by the code generator only.
func X_GeneratedCodeOnly_WrapReport(report *sdk.ReportResponse) (*Report, error) { //nolint
	return &Report{report: report}, nil
}
