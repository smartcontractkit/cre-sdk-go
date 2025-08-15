package cre

import (
	"errors"
	"io"
	"log/slog"
	"math/rand"
	"testing"
	"time"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/chainlink-protos/cre/go/values"
	valuespb "github.com/smartcontractkit/chainlink-protos/cre/go/values/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInNodeMode_SimpleConsensusType(t *testing.T) {
	runtime := &mockRuntime{}

	p := RunInNodeMode("", runtime, func(_ string, nr NodeRuntime) (int, error) {
		return 42, nil
	}, ConsensusMedianAggregation[int]())

	val, err := p.Await()
	require.NoError(t, err)
	assert.Equal(t, 42, val)
}

func TestRunInNodeMode_PrimitiveConsensusWithUnusedDefault(t *testing.T) {
	runtime := &mockRuntime{}

	p := RunInNodeMode("", runtime, func(_ string, nr NodeRuntime) (int, error) {
		return 99, nil
	}, ConsensusMedianAggregation[int]().WithDefault(100))

	val, err := p.Await()
	require.NoError(t, err)
	assert.Equal(t, 99, val)
}

func TestRunInNodeMode_PrimitiveConsensusWithUsedDefault(t *testing.T) {
	runtime := &mockRuntime{}

	p := RunInNodeMode("", runtime, func(_ string, nr NodeRuntime) (int, error) {
		return 0, errors.New("error")
	}, ConsensusMedianAggregation[int]().WithDefault(100))

	val, err := p.Await()
	require.NoError(t, err)
	assert.Equal(t, 100, val)
}

func TestRunInNodeMode_ErrorFromFunction(t *testing.T) {
	runtime := &mockRuntime{}

	p := RunInNodeMode("", runtime, func(_ string, nr NodeRuntime) (int, error) {
		return 0, errors.New("some error")
	}, ConsensusMedianAggregation[int]())

	_, err := p.Await()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "some error")
}

func TestRunInNodeMode_ErrorWrappingDefault(t *testing.T) {
	runtime := &mockRuntime{}

	type unsupported struct {
		Test chan int
	}

	p := RunInNodeMode("", runtime, func(_ string, nr NodeRuntime) (*unsupported, error) {
		return nil, errors.New("some error")
	}, &medianTestFieldDescription[*unsupported]{T: &unsupported{Test: make(chan int)}})

	_, err := p.Await()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not wrap into value:")
}

// mockNodeRuntime implements NodeRuntime for testing.
type mockNodeRuntime struct{}

func (m mockNodeRuntime) Rand() (*rand.Rand, error) {
	panic("unused in tests")
}

func (m mockNodeRuntime) CallCapability(_ *sdk.CapabilityRequest) Promise[*sdk.CapabilityResponse] {
	panic("unused in tests")
}

func (m mockNodeRuntime) Now() time.Time {
	panic("unused in tests")
}

func (m mockNodeRuntime) Config() []byte {
	panic("unused in tests")
}

func (m mockNodeRuntime) LogWriter() io.Writer {
	panic("unused in tests")
}

func (m mockNodeRuntime) Logger() *slog.Logger {
	panic("unused in tests")
}

func (m mockNodeRuntime) IsNodeRuntime() {}

type mockRuntime struct{}

func (m *mockRuntime) GetSecret(_ *SecretRequest) Promise[*Secret] {
	panic("unused in tests")
}

func (m *mockRuntime) Rand() (*rand.Rand, error) {
	panic("unused in tests")
}

func (m *mockRuntime) GenerateReport(_ *sdk.ReportRequest) Promise[*Report] {
	panic("unused in tests")
}

func (m *mockRuntime) Now() time.Time {
	panic("unused in tests")
}

func (m *mockRuntime) RunInNodeMode(fn func(nodeRuntime NodeRuntime) *sdk.SimpleConsensusInputs) Promise[values.Value] {
	req := fn(mockNodeRuntime{})

	if errObs, ok := req.Observation.(*sdk.SimpleConsensusInputs_Error); ok {
		if req.Default != nil && req.Default.Value != nil {
			return PromiseFromResult[values.Value](values.FromProto(reportFromValue(req.Default)))
		}

		return PromiseFromResult[values.Value](nil, errors.New(errObs.Error))
	}
	val, _ := values.FromProto(reportFromValue(req.Observation.(*sdk.SimpleConsensusInputs_Value).Value))
	return PromiseFromResult(val, nil)
}

func (m *mockRuntime) CallCapability(*sdk.CapabilityRequest) Promise[*sdk.CapabilityResponse] {
	panic("not used in test")
}
func (m *mockRuntime) Config() []byte       { return nil }
func (m *mockRuntime) LogWriter() io.Writer { return nil }
func (m *mockRuntime) Logger() *slog.Logger { return nil }

type medianTestFieldDescription[T any] struct {
	T T
}

func (h *medianTestFieldDescription[T]) Descriptor() *sdk.ConsensusDescriptor {
	return &sdk.ConsensusDescriptor{
		Descriptor_: &sdk.ConsensusDescriptor_FieldsMap{
			FieldsMap: &sdk.FieldsMap{
				Fields: map[string]*sdk.ConsensusDescriptor{
					"Test": {Descriptor_: &sdk.ConsensusDescriptor_Aggregation{Aggregation: sdk.AggregationType_AGGREGATION_TYPE_MEDIAN}},
				},
			},
		},
	}
}

func (h *medianTestFieldDescription[T]) Default() *T {
	return &h.T
}

func (h *medianTestFieldDescription[T]) Err() error {
	return nil
}

func (h *medianTestFieldDescription[T]) WithDefault(t T) ConsensusAggregation[T] {
	return &medianTestFieldDescription[T]{T: t}
}

func reportFromValue(result *valuespb.Value) *valuespb.Value {
	return &valuespb.Value{
		Value: result.Value,
	}
}
