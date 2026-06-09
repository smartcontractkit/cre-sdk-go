package main

import (
	"fmt"
	"log/slog"

	caperrors "github.com/smartcontractkit/cre-sdk-go/capabilities/errors"
	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/wasm"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
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
			checkCapabilityErrors,
		),
	}, nil
}

func checkCapabilityErrors(_ []byte, rt cre.Runtime, _ *basictrigger.Outputs) (string, error) {
	action := basicaction.BasicAction{}
	input := &basicaction.Inputs{InputThing: true}

	for {
		output, err := action.PerformAction(rt, input).Await()
		if err == nil {
			if output.AdaptedThing != "Done" {
				return "", fmt.Errorf("expected Done response, got %s", output.AdaptedThing)
			}
			return "Done", nil
		}

		capErr, ok := err.(caperrors.Error)
		if !ok {
			return "", fmt.Errorf("expected capability error, got %T: %v", err, err)
		}
		if capErr.Code() == caperrors.UnrecognisedErrorCode {
			return "", fmt.Errorf("expected recognised error code, got UnrecognisedErrorCode")
		}
	}
}
