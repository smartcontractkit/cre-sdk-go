package cre

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecutionMetadataReturnsCopy(t *testing.T) {
	want := WorkflowExecutionMetadata{
		WorkflowID:    "0020350fb4f89856a031268826d6e9b9340996637aa27859baa7b07db58bb020",
		ExecutionID:   "9f38bdece8892b3452c401a7a2d5a170f078ab9b2c98c591e24028ce94b33ed2",
		WorkflowOwner: "5366890c26e8737b89d9084b4c632f7de73d1674",
		DonID:         17,
		DonF:          3,
	}
	zeroF := want
	zeroF.DonF = 0
	subscription := want
	subscription.ExecutionID = ""

	for _, tc := range []struct {
		name     string
		metadata WorkflowExecutionMetadata
	}{
		{name: "trigger", metadata: want},
		{name: "zero fault tolerance", metadata: zeroF},
		{name: "subscription", metadata: subscription},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runtime := &metadataRuntime{metadata: tc.metadata}

			got := ExecutionMetadata(runtime)

			require.Equal(t, tc.metadata, got)
			got.WorkflowID = "0052457bc477e58cad894372b8a80d1f0b03322c8a070d93bcf9bbd1f21cbd09"
			require.Equal(t, tc.metadata, ExecutionMetadata(runtime))
		})
	}
}

func TestExecutionMetadataReturnsZero(t *testing.T) {
	var don Runtime = &mockRuntime{}
	var node NodeRuntime = mockNodeRuntime{}

	for _, tc := range []struct {
		name    string
		runtime RuntimeBase
	}{
		{name: "DON without provider", runtime: don},
		{name: "node without provider", runtime: node},
		{name: "nil runtime"},
		{name: "empty provider", runtime: &metadataRuntime{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Zero(t, ExecutionMetadata(tc.runtime))
		})
	}
}

// Embedded runtime methods panic if called; metadata access requires none of them.
type metadataRuntime struct {
	mockRuntime
	metadata WorkflowExecutionMetadata
}

func (r *metadataRuntime) ExecutionMetadata() WorkflowExecutionMetadata {
	return r.metadata
}
