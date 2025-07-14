package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"my-por-workflow/contracts/bindings"

	"github.com/shopspring/decimal"

	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/networking/http"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/cron"
	"github.com/smartcontractkit/cre-sdk-go/sdk"
)

type EvmConfig struct {
	TokenAddress          string `json:"tokenAddress"`
	PorAddress            string `json:"porAddress"`
	BalanceReaderAddress  string `json:"balanceReaderAddress"`
	MessageEmitterAddress string `json:"messageEmitterAddress"`
	ChainSelector         uint64 `json:"chainSelector"`
	GasLimit              uint64 `json:"gasLimit"`
}

type Config struct {
	Schedule string      `json:"schedule"`
	Url      string      `json:"url"`
	Evms     []EvmConfig `json:"evms"`
}

type HttpTriggerPayload struct {
	ExecutionTime time.Time `json:"executionTime"`
}

type ReserveInfo struct {
	LastUpdated  time.Time       `consensus_aggregation:"median" json:"lastUpdated"`
	TotalReserve decimal.Decimal `consensus_aggregation:"median" json:"totalReserve"`
}

type PorResponse struct {
	AccountName string    `json:"accountName"`
	TotalTrust  float64   `json:"totalTrust"`
	TotalToken  float64   `json:"totalToken"`
	Ripcord     bool      `json:"ripcord"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func InitWorkflow(wcx *sdk.Environment[*Config]) (sdk.Workflow[*Config], error) {
	cronTriggerCfg := &cron.Config{
		Schedule: wcx.Config.Schedule,
	}

	logTriggerCfg := &evm.FilterLogTriggerRequest{
		Addresses: make([][]byte, len(wcx.Config.Evms)),
	}

	for i, evmConfig := range wcx.Config.Evms {
		address, err := hex.DecodeString(evmConfig.MessageEmitterAddress[2:])
		if err != nil {
			return nil, fmt.Errorf("failed to decode MessageEmitter address %s: %w", evmConfig.MessageEmitterAddress, err)
		}
		logTriggerCfg.Addresses[i] = address
	}

	httpTriggerCfg := &http.Config{}

	return sdk.Workflow[*Config]{
		sdk.Handler(
			cron.Trigger(cronTriggerCfg),
			OnPorCronTrigger),
		sdk.Handler(
			evm.LogTrigger(logTriggerCfg),
			OnLogTrigger),
		sdk.Handler(
			http.Trigger(httpTriggerCfg),
			OnHttpTrigger),
	}, nil
}

func OnPorCronTrigger(env *sdk.Environment[*Config], runtime sdk.Runtime, outputs *cron.Payload) (string, error) {
	return DoPor(env, runtime, outputs.ScheduledExecutionTime.AsTime())
}

func OnLogTrigger(env *sdk.Environment[*Config], runtime sdk.Runtime, payload *evm.Log) (string, error) {
	messageEmitter, err := prepareMessageEmitter(env, env.Config.Evms[0])
	if err != nil {
		return "", fmt.Errorf("failed to prepare message emitter: %w", err)
	}

	topics := payload.GetTopics()
	if len(topics) < 3 {
		env.Logger.Error("Log payload does not contain enough topics", "topics", topics)
		return "", fmt.Errorf("log payload does not contain enough topics: %d", len(topics))
	}

	emitter := topics[1]

	messagePromise := messageEmitter.GetLastMessage(env.Logger, runtime, nil, emitter)
	message, err := messagePromise.Await()
	if err != nil {
		env.Logger.Error("Could not read from contract", "contract_chain", env.Config.Evms[0].ChainSelector, "err", err.Error())
		return "", err
	}

	env.Logger.Info("Message retrieved from the contract", "message", message)

	return messageEmitter.ReadEmittedMessage(env.Logger, topics, payload.GetData())
}

func OnHttpTrigger(env *sdk.Environment[*Config], runtime sdk.Runtime, payload *http.Payload) (string, error) {
	env.Logger.Info("Raw HTTP trigger received", "payload", payload)

	payloadMap := payload.Input.AsMap()
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		env.Logger.Error("failed to marshal http trigger payload", "err", err)
		return "", err
	}

	env.Logger.Info("Payload bytes", "payloadBytes", string(payloadBytes))
	httpTriggerPayload := &HttpTriggerPayload{}
	if err := json.Unmarshal(payloadBytes, httpTriggerPayload); err != nil {
		env.Logger.Error("failed to unmarshal http trigger payload", "err", err)
		return "", err
	}

	env.Logger.Info("Parsed HTTP trigger received", "payload", httpTriggerPayload)
	return DoPor(env, runtime, httpTriggerPayload.ExecutionTime)
}

func DoPor(env *sdk.Environment[*Config], runtime sdk.Runtime, executionTime time.Time) (string, error) {
	env.Logger = env.Logger.With("execution time", executionTime)
	// Fetch Por
	env.Logger.Info("fetching por", "url", env.Config.Url, "evms", env.Config.Evms)
	reserveInfo, err := sdk.RunInNodeMode(env, runtime, func(env *sdk.NodeEnvironment[*Config], nodeRuntime sdk.NodeRuntime) (*ReserveInfo, error) {
		reserveInfo, err := fetchPor(env.Config.Url, nodeRuntime)
		if err != nil {
			env.Logger.Error("error fetching por", "err", err)
			return nil, err
		}
		return reserveInfo, nil
	}, sdk.ConsensusAggregationFromTags[*ReserveInfo]()).Await()
	if err != nil {
		return "", err
	}

	env.Logger.Info("ReserveInfo", "reserveInfo", reserveInfo)

	if reserveInfo.LastUpdated.Sub(executionTime) > time.Hour {
		return "", errors.New("por data is too old, last updated: " + reserveInfo.LastUpdated.String() + ", execution time: " + executionTime.String())
	}

	totalSupply, err := getTotalSupply(env, runtime, env.Config.Evms)
	if err != nil {
		return "", err
	}

	env.Logger.Info("TotalSupply", "totalSupply", totalSupply)
	totalReserveScaled := reserveInfo.TotalReserve.Mul(decimal.NewFromUint64(1e18)).BigInt()
	env.Logger.Info("TotalReserveScaled", "totalReserveScaled", totalReserveScaled)

	nativeTokenBalance, err := fetchNativeTokenBalance(env, runtime, env.Config.Evms[0])
	if err != nil {
		return "", fmt.Errorf("failed to fetch native token balance: %w", err)
	}
	env.Logger.Info("Native token balance", "token", env.Config.Evms[0].TokenAddress, "balance", nativeTokenBalance)

	return reserveInfo.TotalReserve.String(), nil
}

func prepareMessageEmitter(env *sdk.Environment[*Config], evmConfig EvmConfig) (*bindings.MessageEmitter, error) {
	evmClient := &evm.Client{}

	address, err := hexToBytes(evmConfig.MessageEmitterAddress)
	if err != nil {
		env.Logger.Error("failed to decode message emitter address", "address", evmConfig.MessageEmitterAddress, "err", err)
		return nil, fmt.Errorf("failed to decode message emitter address %s: %w", evmConfig.MessageEmitterAddress, err)
	}

	messageEmitter, err := bindings.NewMessageEmitter(evmClient, address)
	if err != nil {
		env.Logger.Error("failed to create message emitter", "address", evmConfig.MessageEmitterAddress, "err", err)
		return nil, fmt.Errorf("failed to create message emitter for address %s: %w", evmConfig.MessageEmitterAddress, err)
	}

	return messageEmitter, nil
}

func fetchNativeTokenBalance(env *sdk.Environment[*Config], runtime sdk.Runtime, evmConfig EvmConfig) (*big.Int, error) {
	evmClient := &evm.Client{}

	balanceReaderAddress, err := hexToBytes(evmConfig.BalanceReaderAddress)
	if err != nil {
		env.Logger.Error("failed to decode balance reader address", "address", evmConfig.BalanceReaderAddress, "err", err)
		return nil, fmt.Errorf("failed to decode balance reader address %s: %w", evmConfig.BalanceReaderAddress, err)
	}
	balanceReader, err := bindings.NewBalanceReader(evmClient, balanceReaderAddress)
	if err != nil {
		env.Logger.Error("failed to create balance reader", "address", evmConfig.BalanceReaderAddress, "err", err)
		return nil, fmt.Errorf("failed to create balance reader for address %s: %w", evmConfig.BalanceReaderAddress, err)
	}
	tokenAddress, err := hexToBytes(evmConfig.TokenAddress)
	if err != nil {
		env.Logger.Error("failed to decode token address", "address", evmConfig.TokenAddress, "err", err)
		return nil, fmt.Errorf("failed to decode token address %s: %w", evmConfig.TokenAddress, err)
	}

	balancePromise := balanceReader.GetNativeBalances(env.Logger, runtime, nil, [][]byte{tokenAddress})
	balances, err := balancePromise.Await()
	if err != nil {
		env.Logger.Error("Could not read from contract", "contract_chain", evmConfig.ChainSelector, "err", err.Error())
		return nil, err
	}

	if len(balances) != 1 {
		env.Logger.Error("No balances returned from contract", "contract_chain", evmConfig.ChainSelector)
		return nil, fmt.Errorf("no balances returned from contract for chain %d", evmConfig.ChainSelector)
	}

	return balances[0], nil
}

func getTotalSupply(env *sdk.Environment[*Config], runtime sdk.Runtime, evms []EvmConfig) (*big.Int, error) {
	// Fetch supply from all EVMs in parallel
	supplyPromises := make([]sdk.Promise[*big.Int], len(evms))
	for i, evmConfig := range evms {
		evmClient := &evm.Client{}

		address, err := hexToBytes(evmConfig.TokenAddress)
		if err != nil {
			env.Logger.Error("failed to decode token address", "address", evmConfig.TokenAddress, "err", err)
			return nil, fmt.Errorf("failed to decode token address %s: %w", evmConfig.TokenAddress, err)
		}
		token := bindings.NewIERC20(bindings.ContractInputs{EVM: evmClient, Address: address})
		evmTotalSupplyPromise := token.Methods.TotalSupply.Call(runtime, nil)
		supplyPromises[i] = evmTotalSupplyPromise
	}

	// We can add sdk.AwaitAll that takes []sdk.Promise[T] and returns ([]T, error)
	totalSupply := big.NewInt(0)
	for i, promise := range supplyPromises {
		supply, err := promise.Await()
		if err != nil {
			selector := evms[i].ChainSelector
			env.Logger.Error("Could not read from contract", "contract_chain", selector, "err", err.Error())
			return nil, err
		}

		totalSupply = totalSupply.Add(totalSupply, supply)
	}

	return totalSupply, nil
}

func fetchPor(urlString string, runtime sdk.NodeRuntime) (*ReserveInfo, error) {
	httpAction := http.Client{}
	httpActionOut, err := httpAction.SendRequest(runtime, &http.Request{
		Method: "GET",
		Url:    urlString,
	}).Await()

	if err != nil {
		return nil, err
	}

	porResponse := &PorResponse{}
	if err = json.Unmarshal(httpActionOut.Body, porResponse); err != nil {
		return nil, err
	}

	if porResponse.Ripcord {
		return nil, errors.New("ripcord is true")
	}

	reserveInfo := &ReserveInfo{
		LastUpdated:  porResponse.UpdatedAt.UTC(),
		TotalReserve: decimal.NewFromFloat(porResponse.TotalToken),
	}

	return reserveInfo, nil
}

func hexToBytes(hexStr string) ([]byte, error) {
	if len(hexStr) < 2 || hexStr[:2] != "0x" {
		return nil, fmt.Errorf("invalid hex string: %s", hexStr)
	}
	return hex.DecodeString(hexStr[2:])
}
