package cre

import (
	"errors"
	"log/slog"
	"math/rand"
	"reflect"
	"time"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/chainlink-protos/cre/go/values"
)

type SecretRequest = sdk.SecretRequest
type Secret = sdk.Secret

// RuntimeBase provides the basic functionality of a CRE runtime.
// It is not thread safe and must not be used concurrently.
type RuntimeBase interface {
	// CallCapability is meant to be called by generated code
	CallCapability(request *sdk.CapabilityRequest) Promise[*sdk.CapabilityResponse]

	// Rand provides access to a random number generator for the mode the runtime operates in.
	// Attempting to use the returned generator outside the correct scope will panic.
	Rand() (*rand.Rand, error)

	// Now provides the current time, with the mechanism for doing so based on the current mode.
	Now() time.Time

	// Logger provides a logger that can be used to log messages.
	Logger() *slog.Logger
}

// SecretsProvider provides access to secrets.
type SecretsProvider interface {
	// GetSecret retrieves a secret by its request.
	GetSecret(*SecretRequest) Promise[*Secret]
}

// NodeRuntime provides access to Node capabilities
// It is not thread safe and must not be used concurrently.
type NodeRuntime interface {
	RuntimeBase

	// IsNodeRuntime is a placeholder to differentiate NodeRuntime.
	IsNodeRuntime()
}

// Runtime provides access to DON capabilities and allows NodeRuntime use with consensus.
// It is not thread safe and must not be used concurrently.
type Runtime interface {
	RuntimeBase

	// RunInNodeMode is meant to be used by the helper method RunInNodeMode
	RunInNodeMode(fn func(nodeRuntime NodeRuntime) *sdk.SimpleConsensusInputs) Promise[values.Value]

	// GenerateReport is used to generate a report for a given ReportRequest.
	GenerateReport(*ReportRequest) Promise[*Report]
	SecretsProvider
}

// ConsensusAggregation is an interface that informs consensus how to aggregate values.
// Workflow author do not need to implement this interface directly; instead the helper functions
// below can be used to create instances of this interface:
// - ConsensusMedianAggregation
// - ConsensusIdenticalAggregation
// - ConsensusCommonPrefixAggregation
// - ConsensusCommonSuffixAggregation
// - ConsensusAggregationFromTags
// By using this interface with capability SDKs or RunInNodeMode, you are assured that all aggregated values are Byzantine fault-tolerant.
type ConsensusAggregation[T any] interface {
	// Descriptor is meant to be used by the Runtime
	Descriptor() *sdk.ConsensusDescriptor

	// Default returns the default value or nil when there is no default value
	Default() *T

	// Err is meant to be used by the Runtime
	Err() error

	// WithDefault returns a new ConsensusAggregation with the given default value
	// If consensus cannot be reached, the default value will be used if it is not nil instead of returning an error.
	WithDefault(t T) ConsensusAggregation[T]
}

type consensusDescriptor[T any] sdk.ConsensusDescriptor

func (c *consensusDescriptor[T]) Descriptor() *sdk.ConsensusDescriptor {
	return (*sdk.ConsensusDescriptor)(c)
}

func (c *consensusDescriptor[T]) Default() *T {
	return nil
}

func (c *consensusDescriptor[T]) Err() error {
	return nil
}

func (c *consensusDescriptor[T]) WithDefault(t T) ConsensusAggregation[T] {
	return &consensusWithDefault[T]{
		ConsensusDescriptor: c.Descriptor(),
		DefaultValue:        t,
	}
}

var _ ConsensusAggregation[int] = (*consensusDescriptor[int])(nil)

type consensusWithDefault[T any] struct {
	ConsensusDescriptor *sdk.ConsensusDescriptor
	DefaultValue        T
}

func (c *consensusWithDefault[T]) Descriptor() *sdk.ConsensusDescriptor {
	return c.ConsensusDescriptor
}

func (c *consensusWithDefault[T]) Default() *T {
	cpy := c.DefaultValue
	return &cpy
}

func (c *consensusWithDefault[T]) Err() error {
	return nil
}

func (c *consensusWithDefault[T]) WithDefault(t T) ConsensusAggregation[T] {
	return &consensusWithDefault[T]{
		ConsensusDescriptor: c.ConsensusDescriptor,
		DefaultValue:        t,
	}
}

type consensusDescriptorError[T any] struct {
	Error error
}

func (d *consensusDescriptorError[T]) Descriptor() *sdk.ConsensusDescriptor {
	return nil
}

func (d *consensusDescriptorError[T]) Default() *T {
	return nil
}

func (d *consensusDescriptorError[T]) Err() error {
	return d.Error
}

func (d *consensusDescriptorError[T]) WithDefault(_ T) ConsensusAggregation[T] {
	return d
}

var nodeModeCallInDonMode = errors.New("cannot use NodeRuntime outside RunInNodeMode")

func NodeModeCallInDonMode() error {
	return nodeModeCallInDonMode
}

var donModeCallInNodeMode = errors.New("cannot use Runtime inside RunInNodeMode")

func DonModeCallInNodeMode() error {
	return donModeCallInNodeMode
}

func RunInNodeMode[C, T any](
	config C,
	runtime Runtime,
	fn func(config C, nodeRuntime NodeRuntime) (T, error),
	ca ConsensusAggregation[T],
) Promise[T] {
	observationFn := func(nodeRuntime NodeRuntime) *sdk.SimpleConsensusInputs {
		if ca.Err() != nil {
			return &sdk.SimpleConsensusInputs{Observation: &sdk.SimpleConsensusInputs_Error{Error: ca.Err().Error()}}
		}

		var defaultValue values.Value
		descriptor := ca.Descriptor()
		var err error
		if d := ca.Default(); d != nil {
			defaultValue, err = values.Wrap(d)
			if err != nil {
				return &sdk.SimpleConsensusInputs{Observation: &sdk.SimpleConsensusInputs_Error{Error: err.Error()}}
			}
		}

		returnValue := &sdk.SimpleConsensusInputs{
			Descriptors: descriptor,
			Default:     values.Proto(defaultValue),
		}

		result, err := fn(config, nodeRuntime)
		if err != nil {
			returnValue.Observation = &sdk.SimpleConsensusInputs_Error{Error: err.Error()}
			return returnValue
		}

		wrapped, err := values.Wrap(result)
		if err != nil {
			returnValue.Observation = &sdk.SimpleConsensusInputs_Error{Error: err.Error()}
			return returnValue
		}

		returnValue.Observation = &sdk.SimpleConsensusInputs_Value{Value: values.Proto(wrapped)}
		return returnValue
	}

	return Then(runtime.RunInNodeMode(observationFn), func(v values.Value) (T, error) {
		var t T
		var err error

		typ := reflect.TypeOf(t)
		// If T is a pointer type, we need to allocate the underlying type and pass its pointer to UnwrapTo
		if typ.Kind() == reflect.Ptr {
			elem := reflect.New(typ.Elem())
			err = v.UnwrapTo(elem.Interface())
			t = elem.Interface().(T)
		} else {
			err = v.UnwrapTo(&t)
		}
		return t, err
	})
}
