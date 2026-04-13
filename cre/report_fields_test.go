package cre_test

import (
	"testing"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getReport(t *testing.T) *cre.Report {
	t.Helper()
	setupDonSettingRead(t, productionEnvironmentReplay)
	runtime := testutils.NewRuntime(t, testutils.Secrets{})
	sigsWithExtra := make([][]byte, 0, len(reportSigs)+1)
	sigsWithExtra = append(sigsWithExtra, reportSigs[0])
	sigsWithExtra = append(sigsWithExtra, notASignerSig)
	sigsWithExtra = append(sigsWithExtra, reportSigs[1:]...)
	sigsWithExtra = append(sigsWithExtra, extraValidSig)
	report, err := cre.ParseReport(runtime, rawReport, sigsWithExtra, reportContext)
	require.NoError(t, err)
	require.NotNil(t, report)
	return report
}

func TestReport_SeqNr(t *testing.T) {
	report := getReport(t)
	assert.Equal(t, reportSeqNr, report.SeqNr())
}

func TestReport_ConfigDigest(t *testing.T) {
	report := getReport(t)
	assert.Equal(t, reportDigest, report.ConfigDigest())
}

func TestReport_ReportContext(t *testing.T) {
	report := getReport(t)

	assert.Equal(t, reportContext, report.ReportContext())
}

func TestReport_RawReport(t *testing.T) {
	report := getReport(t)
	assert.ElementsMatch(t, rawReport, report.RawReport())
}

func TestReport_Version(t *testing.T) {
	report := getReport(t)
	assert.Equal(t, reportVersion, report.Version())
}

func TestReport_ExecutionID(t *testing.T) {
	report := getReport(t)
	assert.Equal(t, reportExecID, report.ExecutionID())
}

func TestReport_Timestamp(t *testing.T) {
	report := getReport(t)
	assert.Equal(t, reportTimestampUnix, report.Timestamp())
}

func TestReport_DONID(t *testing.T) {
	report := getReport(t)
	assert.Equal(t, reportDONID, report.DONID())
}

func TestReport_DONConfigVersion(t *testing.T) {
	report := getReport(t)
	assert.Equal(t, reportDONCfgVer, report.DONConfigVersion())
}

func TestReport_WorkflowID(t *testing.T) {
	report := getReport(t)
	assert.Equal(t, reportWfID, report.WorkflowID())
}

func TestReport_WorkflowName(t *testing.T) {
	report := getReport(t)
	assert.Equal(t, reportWfName, report.WorkflowName())
}

func TestReport_WorkflowOwner(t *testing.T) {
	report := getReport(t)
	assert.Equal(t, reportWfOwner, report.WorkflowOwner())
}

func TestReport_ReportID(t *testing.T) {
	report := getReport(t)
	assert.Equal(t, reportReportID, report.ReportID())
}

func TestReport_Body(t *testing.T) {
	report := getReport(t)
	assert.Equal(t, reportBody, string(report.Body()))
}

func TestReport_SignerIds(t *testing.T) {
	report := getReport(t)
	unwrapped := report.X_GeneratedCodeOnly_Unwrap()
	require.NotNil(t, unwrapped)

	// Only the f+1 accepted signatures must be present (extras are stripped).
	require.Len(t, unwrapped.Sigs, len(reportSigs), "expected exactly f+1 sigs after verification")
	for i, sig := range reportSigs {
		assert.ElementsMatch(t, sig, unwrapped.Sigs[i].Signature)
		assert.Equal(t, reportSigIds[i], unwrapped.Sigs[i].SignerId)
	}
}
