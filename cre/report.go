package cre

import "github.com/smartcontractkit/chainlink-protos/cre/go/sdk"

type ReportRequest = sdk.ReportRequest

// Report contains a signed report from the CRE workflow DON.
// Reports contain metadata, including the workflow ID, workflow owner, and execution ID, alongside the encoded payload,
// and signatures from F+1 nodes in the workflow DON. They can be used to prove that data came from a specific workflow,
// or author. Blockchains integrated with the CRE have forwarder contracts that can verify a report's integrity.
// Chainlink will provide helpers to verify reports offline in at a later date.
type Report struct {
	report *sdk.ReportResponse
}

// X_GeneratedCodeOnly_Unwrap is meant to be used by the code generator only.
func (r *Report) X_GeneratedCodeOnly_Unwrap() *sdk.ReportResponse { //nolint
	return r.report
}

// X_GeneratedCodeOnly_WrapReport is meant to be used by the code generator only.
func X_GeneratedCodeOnly_WrapReport(report *sdk.ReportResponse) (*Report, error) { //nolint
	return &Report{report: report}, nil
}
