package cre

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// reportHeader holds the decoded contents of the 109-byte metadata header prepended to every CRE
// report by the consensus plugin, plus the body that follows it.
//
// Layout (matches chainlink-common pkg/capabilities/consensus/ocr3/types.Metadata):
//
//	offset   0, size  1  – version
//	offset   1, size 32  – execution ID (hex)
//	offset  33, size  4  – timestamp (big-endian uint32)
//	offset  37, size  4  – DON ID (big-endian uint32)
//	offset  41, size  4  – DON config version (big-endian uint32)
//	offset  45, size 32  – workflow ID (hex)
//	offset  77, size 10  – workflow name (hex)
//	offset  87, size 20  – workflow owner (hex)
//	offset 107, size  2  – report ID (hex)
//	offset 109, …        – body (workflow payload)
type reportHeader struct {
	version          uint32
	executionID      string
	timestamp        uint32
	donID            uint32
	donConfigVersion uint32
	workflowID       string
	workflowName     string
	workflowOwner    string
	reportID         string
	body             []byte
}

func parseRawReport(raw []byte) (*reportHeader, error) {
	if raw == nil {
		return nil, ErrNilReport
	}

	if len(raw) < ReportMetadataHeaderLength {
		return nil, fmt.Errorf("%w: need %d bytes, got %d",
			ErrRawReportTooShort, ReportMetadataHeaderLength, len(raw))
	}

	return &reportHeader{
		version:          uint32(raw[0]),
		executionID:      hex.EncodeToString(raw[1:33]),
		timestamp:        binary.BigEndian.Uint32(raw[33:37]),
		donID:            binary.BigEndian.Uint32(raw[37:41]),
		donConfigVersion: binary.BigEndian.Uint32(raw[41:45]),
		workflowID:       hex.EncodeToString(raw[45:77]),
		workflowName:     string(raw[77:87]),
		workflowOwner:    hex.EncodeToString(raw[87:107]),
		reportID:         hex.EncodeToString(raw[107:109]),
		body:             raw[ReportMetadataHeaderLength:],
	}, nil
}

func (r *Report) parseHeader() (*reportHeader, error) {
	r.headerLock.Lock()
	defer r.headerLock.Unlock()

	if r.cachedHeader != nil {
		return r.cachedHeader, nil
	}

	var err error
	r.cachedHeader, err = parseRawReport(r.report.RawReport)

	return r.cachedHeader, err
}

// ── ReportResponse fields ──────────────────────────────────────────────────

// SeqNr returns the OCR sequence number from the ReportResponse.
func (r *Report) SeqNr() uint64 {
	if r.report == nil {
		return 0
	}
	return r.report.SeqNr
}

// ConfigDigest returns the OCR config digest from the ReportResponse.
func (r *Report) ConfigDigest() []byte {
	if r.report == nil {
		return nil
	}
	return r.report.GetConfigDigest()
}

// ReportContext returns the OCR report context bytes from the ReportResponse.
func (r *Report) ReportContext() []byte {
	if r.report == nil {
		return nil
	}
	return r.report.GetReportContext()
}

// RawReport returns the full raw report bytes (metadata header + body).
func (r *Report) RawReport() []byte {
	if r.report == nil {
		return nil
	}
	return r.report.RawReport
}

// ── Metadata header fields (parsed from RawReport) ────────────────────────

// Version returns the single-byte version field from the report metadata header.
func (r *Report) Version() uint32 {
	return r.cachedHeader.version
}

// ExecutionID returns the 32-byte workflow execution ID as a hex string.
func (r *Report) ExecutionID() string {
	return r.cachedHeader.executionID
}

// Timestamp returns the Unix timestamp (seconds) embedded in the report metadata.
func (r *Report) Timestamp() uint32 {
	return r.cachedHeader.timestamp
}

// DONID returns the DON identifier from the report metadata.
func (r *Report) DONID() uint32 {
	return r.cachedHeader.donID
}

// DONConfigVersion returns the DON configuration version from the report metadata.
func (r *Report) DONConfigVersion() uint32 {
	return r.cachedHeader.donConfigVersion
}

// WorkflowID returns the 32-byte workflow ID as a hex string.
func (r *Report) WorkflowID() string {
	return r.cachedHeader.workflowID
}

// WorkflowName returns the 10-byte workflow name as a hex string.
func (r *Report) WorkflowName() string {
	return r.cachedHeader.workflowName
}

// WorkflowOwner returns the 20-byte workflow owner address as a hex string.
func (r *Report) WorkflowOwner() string {
	return r.cachedHeader.workflowOwner
}

// ReportID returns the 2-byte report ID as a hex string.
func (r *Report) ReportID() string {
	return r.cachedHeader.reportID
}

// Body returns the workflow payload that follows the 109-byte metadata header.
func (r *Report) Body() []byte {
	return r.cachedHeader.body
}
