package cre

import (
	"errors"
	"sync"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
)

type ReportRequest = sdk.ReportRequest

var (
	// ErrNilReport is returned when the report is nil.
	ErrNilReport = errors.New("report is nil")
	// ErrWrongSignatureCount is returned when fewer than f+1 signatures are provided.
	ErrWrongSignatureCount = errors.New("wrong number of signatures")
	// ErrParseSignature is returned when a signature cannot be parsed.
	ErrParseSignature = errors.New("failed to parse signature")
	// ErrRecoverSigner is returned when the signer address cannot be recovered from a signature.
	ErrRecoverSigner = errors.New("failed to recover signer address from signature")
	// ErrUnknownSigner is returned when a recovered signer address is not in the valid signers set.
	ErrUnknownSigner = errors.New("invalid signature")
	// ErrDuplicateSigner is returned when the same signer appears more than once.
	ErrDuplicateSigner = errors.New("duplicate signer")
	// ErrRawReportTooShort is returned when the raw report is shorter than the 109-byte metadata header.
	ErrRawReportTooShort = errors.New("raw report too short to contain metadata header")
)

// Report contains a signed report from the CRE workflow DON.
// Reports contain metadata, including the workflow ID, workflow owner, and execution ID, alongside the encoded payload,
// and signatures from F+1 nodes in the workflow DON. They can be used to prove that data came from a specific workflow,
// or author. Blockchains integrated with the CRE have forwarder contracts that can verify a report's integrity.
type Report struct {
	report *sdk.ReportResponse

	// headerOnce guards lazy parsing of the raw report metadata header.
	// cachedHeader and cachedHeaderErr hold the result after the first call.
	headerLock   sync.Mutex
	cachedHeader *reportHeader
}

// X_GeneratedCodeOnly_Unwrap is meant to be used by the code generator only.
func (r *Report) X_GeneratedCodeOnly_Unwrap() *sdk.ReportResponse { //nolint
	return r.report
}

// X_GeneratedCodeOnly_WrapReport is meant to be used by the code generator only.
func X_GeneratedCodeOnly_WrapReport(report *sdk.ReportResponse) (*Report, error) { //nolint
	creReport := &Report{report: report}
	if _, err := creReport.parseHeader(); err != nil {
		return nil, err
	}
	return creReport, nil
}
