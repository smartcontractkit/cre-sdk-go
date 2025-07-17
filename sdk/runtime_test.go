package sdk

import (
	"errors"
	"io"
	"log/slog"
	"math/rand"
	"testing"

	"github.com/smartcontractkit/chainlink-common/pkg/values"
	valuespb "github.com/smartcontractkit/chainlink-common/pkg/values/pb"
	"github.com/smartcontractkit/chainlink-common/pkg/workflows/sdk/v2/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInNodeMode_SimpleConsensusType(t *testing.T) {
	runtime := &mockDonRuntime{}

	p := RunInNodeMode(&Environment[string]{}, runtime, func(_ *NodeEnvironment[string], nr NodeRuntime) (int, error) {
		return 42, nil
	}, ConsensusMedianAggregation[int]())

	val, err := p.Await()
	require.NoError(t, err)
	assert.Equal(t, 42, val)
}

func TestRunInNodeMode_PrimitiveConsensusWithUnusedDefault(t *testing.T) {
	runtime := &mockDonRuntime{}

	p := RunInNodeMode(&Environment[string]{}, runtime, func(_ *NodeEnvironment[string], nr NodeRuntime) (int, error) {
		return 99, nil
	}, ConsensusMedianAggregation[int]().WithDefault(100))

	val, err := p.Await()
	require.NoError(t, err)
	assert.Equal(t, 99, val)
}

func TestRunInNodeMode_PrimitiveConsensusWithUsedDefault(t *testing.T) {
	runtime := &mockDonRuntime{}

	p := RunInNodeMode(&Environment[string]{}, runtime, func(_ *NodeEnvironment[string], nr NodeRuntime) (int, error) {
		return 0, errors.New("error")
	}, ConsensusMedianAggregation[int]().WithDefault(100))

	val, err := p.Await()
	require.NoError(t, err)
	assert.Equal(t, 100, val)
}

func TestRunInNodeMode_ErrorFromFunction(t *testing.T) {
	runtime := &mockDonRuntime{}

	p := RunInNodeMode(&Environment[string]{}, runtime, func(_ *NodeEnvironment[string], nr NodeRuntime) (int, error) {
		return 0, errors.New("some error")
	}, ConsensusMedianAggregation[int]())

	_, err := p.Await()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "some error")
}

func TestRunInNodeMode_ErrorWrappingDefault(t *testing.T) {
	runtime := &mockDonRuntime{}

	type unsupported struct {
		Test chan int
	}

	p := RunInNodeMode(&Environment[string]{}, runtime, func(_ *NodeEnvironment[string], nr NodeRuntime) (*unsupported, error) {
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

func (m mockNodeRuntime) CallCapability(_ *pb.CapabilityRequest) Promise[*pb.CapabilityResponse] {
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

type mockDonRuntime struct{}

func (m *mockDonRuntime) Rand() (*rand.Rand, error) {
	panic("unused in tests")
}

func (m *mockDonRuntime) GenerateReport(_ *pb.ReportRequest) Promise[*pb.ReportResponse] {
	panic("unused in tests")
}

func (m *mockDonRuntime) RunInNodeMode(fn func(nodeRuntime NodeRuntime) *pb.SimpleConsensusInputs) Promise[values.Value] {
	req := fn(mockNodeRuntime{})

	if errObs, ok := req.Observation.(*pb.SimpleConsensusInputs_Error); ok {
		if req.Default != nil && req.Default.Value != nil {
			return PromiseFromResult[values.Value](values.FromProto(reportFromValue(req.Default)))
		}

		return PromiseFromResult[values.Value](nil, errors.New(errObs.Error))
	}
	val, _ := values.FromProto(reportFromValue(req.Observation.(*pb.SimpleConsensusInputs_Value).Value))
	return PromiseFromResult(val, nil)
}

func (m *mockDonRuntime) CallCapability(*pb.CapabilityRequest) Promise[*pb.CapabilityResponse] {
	panic("not used in test")
}
func (m *mockDonRuntime) Config() []byte       { return nil }
func (m *mockDonRuntime) LogWriter() io.Writer { return nil }
func (m *mockDonRuntime) Logger() *slog.Logger { return nil }

type medianTestFieldDescription[T any] struct {
	T T
}

func (h *medianTestFieldDescription[T]) Descriptor() *pb.ConsensusDescriptor {
	return &pb.ConsensusDescriptor{
		Descriptor_: &pb.ConsensusDescriptor_FieldsMap{
			FieldsMap: &pb.FieldsMap{
				Fields: map[string]*pb.ConsensusDescriptor{
					"Test": {Descriptor_: &pb.ConsensusDescriptor_Aggregation{Aggregation: pb.AggregationType_AGGREGATION_TYPE_MEDIAN}},
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
