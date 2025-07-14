package main

import (
	"fmt"
	"math/big"
	"strconv"

	"my-calculator-workflow/bindings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm"

	"github.com/smartcontractkit/cre-sdk-go/capabilities/networking/http"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/cron"
	"github.com/smartcontractkit/cre-sdk-go/sdk"
)

type Config struct {
	Schedule        string `json:"schedule"`
	ApiUrl          string `json:"apiUrl"`
	ContractAddress string `json:"contractAddress"`
}

type MyResult struct{}

func fetchMathResult(env *sdk.NodeEnvironment[*Config], nodeRuntime sdk.NodeRuntime) (float64, error) {
	client := &http.Client{}
	req := &http.Request{Url: env.Config.ApiUrl, Method: "GET"}
	resp, err := client.SendRequest(nodeRuntime, req).Await()
	if err != nil {
		return 0, fmt.Errorf("failed to get API response: %w", err)
	}
	return strconv.ParseFloat(string(resp.Body), 64)
}

func OnCronTrigger(env *sdk.Environment[*Config], runtime sdk.Runtime, trigger *cron.Payload) (*MyResult, error) {
	// Step 1: Fetch offchain data (from Part 2)
	mathPromise := sdk.RunInNodeMode(env, runtime, fetchMathResult, sdk.ConsensusIdenticalAggregation[float64]())
	offchainValue, err := mathPromise.Await()
	if err != nil {
		return nil, err
	}
	env.Logger.Info("Successfully fetched offchain value", "result", offchainValue)

	// Step 2: Read onchain data using the new binding
	evmClient := &evm.Client{}
	contractAddress := common.HexToAddress(env.Config.ContractAddress).Bytes()
	storageContract, err := bindings.NewStorage(contractAddress, evmClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract instance: %w", err)
	}

	onchainValuePromise := storageContract.Get(runtime)
	onchainValue, err := onchainValuePromise.Await()
	if err != nil {
		return nil, fmt.Errorf("failed to read onchain value: %w", err)
	}
	env.Logger.Info("Successfully read onchain value", "result", onchainValue)

	// Step 3: Combine the results
	finalResult := new(big.Float).SetInt(onchainValue)
	finalResult.Add(finalResult, big.NewFloat(offchainValue))
	env.Logger.Info("Final calculated result", "result", finalResult)

	return &MyResult{}, nil
}

func InitWorkflow(env *sdk.Environment[*Config]) (sdk.Workflow[*Config], error) {
	return sdk.Workflow[*Config]{
		sdk.Handler(cron.Trigger(&cron.Config{Schedule: env.Config.Schedule}), OnCronTrigger),
	}, nil
}
