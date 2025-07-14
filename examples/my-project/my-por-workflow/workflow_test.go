package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm"
	evmmock "github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm/mock"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/networking/http"
	httpmock "github.com/smartcontractkit/cre-sdk-go/capabilities/networking/http/mock"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/cron"
	"github.com/smartcontractkit/cre-sdk-go/sdk/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"my-por-workflow/contracts/bindings/bindingsmock"
)

//go:embed config.json
var configStr string

var anyExecutionTime = time.Unix(1752514917, 0)

var anyPoRResponse = PorResponse{
	AccountName: "test",
	TotalTrust:  501933900.88,
	TotalToken:  494515082.75,
	Ripcord:     false,
	UpdatedAt:   anyExecutionTime.Add(-time.Minute * 20),
}

var anyTotalSupply = big.NewInt(200000000)
var anyBalance = big.NewInt(50000000000)

const anyEmittedMessage = "This is a test message"

func TestInitWorkflow(t *testing.T) {
	_, env := testutils.NewRuntimeAndEnv(t, makeTestConfig(t), nil)

	workflow, err := InitWorkflow(env)
	require.NoError(t, err)

	require.Len(t, workflow, 3)
	require.Equal(t, cron.Trigger(&cron.Config{}).CapabilityID(), workflow[0].CapabilityID())
	require.Equal(t, evm.LogTrigger(&evm.FilterLogTriggerRequest{}).CapabilityID(), workflow[1].CapabilityID())
	require.Equal(t, http.Trigger(&http.Config{}).CapabilityID(), workflow[2].CapabilityID())
}

func TestDoPorHappyPath(t *testing.T) {
	runtime, env := testutils.NewRuntimeAndEnv(t, makeTestConfig(t), nil)
	setupSuccessfulPoR(t, env.Config)

	reserveStr, err := DoPor(env, runtime, anyExecutionTime)
	require.NoError(t, err)

	assert.Equal(t, decimal.NewFromFloat(anyPoRResponse.TotalToken).String(), reserveStr)
	logs := runtime.GetLogs()
	assert.Len(t, logs, 6)
	assertLogContains(t, logs, `level=INFO msg="fetching por" "execution time"=2025-07-14T13:41:57.000-04:00 url=https://api.real-time-reserves.verinumus.io/v1/chainlink/proof-of-reserves/TrueUSD evms="[{TokenAddress:0x4700A50d858Cb281847ca4Ee0938F80DEfB3F1dd PorAddress:0x073671aE6EAa2468c203fDE3a79dEe0836adF032 BalanceReaderAddress:0x4b0739c94C1389B55481cb7506c62430cA7211Cf MessageEmitterAddress:0x1d598672486ecB50685Da5497390571Ac4E93FDc ChainSelector:16015286601757825753 GasLimit:1000000}]`)
	assertLogContains(t, logs, `level=INFO msg=ReserveInfo "execution time"=2025-07-14T13:41:57.000-04:00 reserveInfo="&{LastUpdated:2025-07-14 17:21:57 +0000 UTC TotalReserve:494515082.75}"`)
	assertLogContains(t, logs, `level=INFO msg=TotalSupply "execution time"=2025-07-14T13:41:57.000-04:00 totalSupply=200000000`)
	assertLogContains(t, logs, `level=INFO msg=TotalReserveScaled "execution time"=2025-07-14T13:41:57.000-04:00 totalReserveScaled=494515082750000000000000000`)
	assertLogContains(t, logs, `level=INFO msg="Packed data for GetNativeBalances" "execution time"=2025-07-14T13:41:57.000-04:00 data="L\x04\xbf\x99\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00 \x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00G\x00\xa5\r\x85\x8c\xb2\x81\x84|\xa4\xee\t8\xf8\r\xef\xb3\xf1\xdd"`)
	assertLogContains(t, logs, `level=INFO msg="Native token balance" "execution time"=2025-07-14T13:41:57.000-04:00 token=0x4700A50d858Cb281847ca4Ee0938F80DEfB3F1dd balance=50000000000`)
}

func TestOnPorCronTrigger(t *testing.T) {
	// Cron delegates to DoPor, so test that they produce the same result.
	runtime, env := testutils.NewRuntimeAndEnv(t, makeTestConfig(t), nil)
	setupSuccessfulPoR(t, env.Config)

	expected, err := DoPor(env, runtime, anyExecutionTime)
	require.NoError(t, err)

	trigger := cron.Payload{ScheduledExecutionTime: timestamppb.New(anyExecutionTime)}
	actual, err := OnPorCronTrigger(env, runtime, &trigger)
	require.NoError(t, err)

	assert.Equal(t, expected, actual)
}

func TestOnHttpTrigger(t *testing.T) {
	// Http delegates to DoPor, so test that they produce the same result.
	runtime, env := testutils.NewRuntimeAndEnv(t, makeTestConfig(t), nil)
	setupSuccessfulPoR(t, env.Config)

	expected, err := DoPor(env, runtime, anyExecutionTime)
	require.NoError(t, err)

	input, err := structpb.NewStruct(map[string]any{"executionTime": anyExecutionTime.Format(time.RFC3339)})
	require.NoError(t, err)
	trigger := http.Payload{Input: input}
	actual, err := OnHttpTrigger(env, runtime, &trigger)
	require.NoError(t, err)

	assert.Equal(t, expected, actual)
}

func TestOnLogTrigger(t *testing.T) {
	config := makeTestConfig(t)
	runtime, env := testutils.NewRuntimeAndEnv(t, config, nil)
	anyTestAddress := common.HexToAddress("0x1203400345600000000000000000000000000000000000")
	trigger := bindingsmock.MessageEmitterMessageEmittedTrigger(anyTestAddress, big.NewInt(anyExecutionTime.Unix()), anyEmittedMessage)

	evmMock, err := evmmock.NewClientCapability(t)
	require.NoError(t, err)
	emitterMock := bindingsmock.NewMessageEmitterMock(common.HexToAddress(config.Evms[0].MessageEmitterAddress), evmMock)
	anyTestMessage := "Hi there! This is a test message"
	emitterMock.GetLastMessage = func(actualAddress common.Address) (string, error) {
		assert.Equal(t, anyTestAddress, actualAddress)
		return anyTestMessage, nil
	}

	result, err := OnLogTrigger(env, runtime, trigger)
	require.NoError(t, err)
	require.Equal(t, anyEmittedMessage, result)
	logs := runtime.GetLogs()
	assertLogContains(t, logs, anyTestMessage)
}

func makeTestConfig(t *testing.T) *Config {
	config := &Config{}
	require.NoError(t, json.Unmarshal([]byte(configStr), config))
	return config
}

func setupSuccessfulPoR(t *testing.T, config *Config) {
	httpClient, err := httpmock.NewClientCapability(t)
	require.NoError(t, err)

	httpClient.SendRequest = func(ctx context.Context, req *http.Request) (*http.Response, error) {
		body, err := json.Marshal(anyPoRResponse)
		require.NoError(t, err)
		return &http.Response{Body: body}, nil
	}

	evmClient, err := evmmock.NewClientCapability(t)
	require.NoError(t, err)

	ierc20 := bindingsmock.NewIERC20Mock(common.HexToAddress(config.Evms[0].TokenAddress), evmClient)
	ierc20.TotalSupply = func() (*big.Int, error) { return anyTotalSupply, nil }

	balanceReader := bindingsmock.NewBalanceReaderMock(common.HexToAddress(config.Evms[0].BalanceReaderAddress), evmClient)
	balanceReader.GetNativeBalances = func(addresses []common.Address) ([]*big.Int, error) {
		require.Len(t, addresses, 1)
		require.ElementsMatch(t, common.HexToAddress(config.Evms[0].TokenAddress), addresses[0])
		return []*big.Int{anyBalance}, nil
	}
}

func assertLogContains(t *testing.T, logs [][]byte, substr string) {
	for _, line := range logs {
		if strings.Contains(string(line), substr) {
			return
		}
	}
	t.Fatalf("Expected logs to contain substring %q, but it was not found", substr)
}
