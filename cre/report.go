package cre

import "github.com/smartcontractkit/chainlink-common/pkg/workflows/sdk/v2/pb"

type Report struct {
	report *pb.ReportResponse
}

// X_GeneratedCodeOnly_Unwrap is meant to be used by the code generator only.
func (r *Report) X_GeneratedCodeOnly_Unwrap() *pb.ReportResponse { //nolint
	return r.report
}

// X_GeneratedCodeOnly_Wrap is meant to be used by the code generator only.
func X_GeneratedCodeOnly_WrapReport(report *pb.ReportResponse) (*Report, error) { //nolint
	return &Report{report: report}, nil
}
