package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	evmmock "github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm/mock"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/networking/http"
	httpmock "github.com/smartcontractkit/cre-sdk-go/capabilities/networking/http/mock"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/cron"
	"github.com/smartcontractkit/cre-sdk-go/sdk/testutils"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"my-calculator-workflow/bindings/bindingsmock"
)

var anyExecutionTime = time.Unix(1752514917, 0)

func TestInitWorkflow(t *testing.T) {
	_, env := testutils.NewRuntimeAndEnv(t, makeTestConfig(t), nil)

	workflow, err := InitWorkflow(env)
	require.NoError(t, err)

	require.Len(t, workflow, 1)
	require.Equal(t, cron.Trigger(&cron.Config{}).CapabilityID(), workflow[0].CapabilityID())
}

func TestOnLogTrigger(t *testing.T) {
	config := makeTestConfig(t)
	runtime, env := testutils.NewRuntimeAndEnv(t, config, nil)

	httpMock, err := httpmock.NewClientCapability(t)
	require.NoError(t, err)
	httpMock.SendRequest = func(ctx context.Context, input *http.Request) (*http.Response, error) {
		return &http.Response{Body: []byte("20")}, nil
	}

	evmMock, err := evmmock.NewClientCapability(t)
	require.NoError(t, err)

	storageMock := bindingsmock.NewStorageMock(common.HexToAddress(config.ContractAddress), evmMock)
	storageMock.Get = func() (*big.Int, error) {
		return big.NewInt(101), nil
	}

	result, err := OnCronTrigger(env, runtime, &cron.Payload{ScheduledExecutionTime: timestamppb.New(anyExecutionTime)})
	require.NoError(t, err)
	require.NotNil(t, result)

	fmt.Println(runtime.GetLogs())
	assertLogContains(t, runtime.GetLogs(), `msg="Final calculated result" result=121`)
}

//go:embed config.json
var configJson []byte

func makeTestConfig(t *testing.T) *Config {
	config := &Config{}
	require.NoError(t, json.Unmarshal(configJson, config))
	return config
}

func assertLogContains(t *testing.T, logs [][]byte, substr string) {
	for _, line := range logs {
		if strings.Contains(string(line), substr) {
			return
		}
	}
	t.Fatalf("Expected logs to contain substring %q, but it was not found", substr)
}
