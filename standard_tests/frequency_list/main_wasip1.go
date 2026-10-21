package main

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/wasm"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/nodeaction"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/chainlink-protos/cre/go/values"
)

func main() {
	runner := wasm.NewRunner(func(configBytes []byte) ([]byte, error) {
		return configBytes, nil
	})
	runner.Run(initFn)
}

func initFn([]byte, *slog.Logger, cre.SecretsProvider) (cre.Workflow[[]byte], error) {
	return cre.Workflow[[]byte]{
		cre.Handler(
			basictrigger.Trigger(&basictrigger.Config{}),
			frequencyListWorkflow,
		),
	}, nil
}

type nodeObservation struct {
	Payload    []byte `consensus_aggregation:"identical" mapstructure:"payload"`
	Signatures []byte `consensus_aggregation:"frequency_list" mapstructure:"signatures"`
}

type consensusResult struct {
	Payload    []byte                           `mapstructure:"payload"`
	Signatures []cre.FrequencyListEntry[[]byte] `mapstructure:"signatures"`
}

func frequencyListWorkflow(_ []byte, rt cre.Runtime, _ *basictrigger.Outputs) (string, error) {
	ca := cre.ConsensusAggregationFromTags[nodeObservation]()
	result, err := cre.Then(
		rt.RunInNodeMode(func(nrt cre.NodeRuntime) *sdk.SimpleConsensusInputs {
			if ca.Err() != nil {
				return &sdk.SimpleConsensusInputs{Observation: &sdk.SimpleConsensusInputs_Error{Error: ca.Err().Error()}}
			}

			returnValue := &sdk.SimpleConsensusInputs{
				Descriptors: ca.Descriptor(),
				Default:     values.Proto(nil),
			}

			// node-mode action
			actionResult, err := (&nodeaction.BasicAction{}).PerformAction(nrt, &nodeaction.NodeInputs{InputThing: true}).Await()
			if err != nil {
				returnValue.Observation = &sdk.SimpleConsensusInputs_Error{Error: err.Error()}
				return returnValue
			}

			obs := nodeObservation{
				Payload:    []byte(fmt.Sprintf("payload-%d", actionResult.OutputThing)),
				Signatures: []byte(fmt.Sprintf("sig-%d", actionResult.OutputThing)),
			}
			wrapped, err := values.Wrap(obs)
			if err != nil {
				returnValue.Observation = &sdk.SimpleConsensusInputs_Error{Error: err.Error()}
				return returnValue
			}

			returnValue.Observation = &sdk.SimpleConsensusInputs_Value{Value: values.Proto(wrapped)}
			return returnValue
		}),
		func(v values.Value) (consensusResult, error) {
			// manually unwrap the list type
			var result consensusResult
			if err := v.UnwrapTo(&result); err != nil {
				return consensusResult{}, err
			}
			return result, nil
		},
	).Await()
	if err != nil {
		return "", err
	}

	return formatResult(result), nil
}

func formatResult(result consensusResult) string {
	return fmt.Sprintf("%s|%s", hex.EncodeToString(result.Payload), formatFrequencyList(result.Signatures))
}

func formatFrequencyList(entries []cre.FrequencyListEntry[[]byte]) string {
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, fmt.Sprintf("%sx%d", hex.EncodeToString(entry.Value), entry.Count))
	}
	return strings.Join(parts, ",")
}
