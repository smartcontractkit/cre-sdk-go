package cre

import (
	"errors"
	"log/slog"
	"math/rand"
	"reflect"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/chainlink-protos/cre/go/values"
)

type SecretRequest = sdk.SecretRequest
type Secret = sdk.Secret

// RuntimeBase is not thread safe and must not be used concurrently.
type RuntimeBase interface {
	// CallCapability is meant to be called by generated code
	CallCapability(request *sdk.CapabilityRequest) Promise[*sdk.CapabilityResponse]
	Rand() (*rand.Rand, error)
	Logger() *slog.Logger
}

type SecretsProvider interface {
	GetSecret(*SecretRequest) Promise[*Secret]
}

// NodeRuntime is not thread safe and must not be used concurrently.
type NodeRuntime interface {
	RuntimeBase
	IsNodeRuntime()
}

// Runtime is not thread safe and must not be used concurrently.
type Runtime interface {
	RuntimeBase

	// RunInNodeMode is meant to be used by the helper method RunInNodeMode
	RunInNodeMode(fn func(nodeRuntime NodeRuntime) *sdk.SimpleConsensusInputs) Promise[values.Value]
	GenerateReport(*ReportRequest) Promise[*Report]
	SecretsProvider
}

type ConsensusAggregation[T any] interface {
	Descriptor() *sdk.ConsensusDescriptor
	Default() *T
	Err() error
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
