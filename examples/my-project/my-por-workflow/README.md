# Trying out Developer PoR example

This template provides an e2e PoR example (including compiled smart contracts). It aims to give an overview of CRE capabilities and to get started with local simulation.

Steps to run the example

## 1. Update .env file

You need to add a private key to env file. This is specifically required if you want to simulate chain writes. For that to work the key should be valid and funded.
If your workflow does not do any chain write then you can just put any dummy key as a private key. e.g.
```
CRE_ETH_PRIVATE_KEY=0000000000000000000000000000000000000000000000000000000000000001
```

## 2. Generate contract bindings

Contract binding generator is not available yet. For now mock bindings are already provided in the template so skip this step.

Please take a look at the `balance_reader.go` and `message_emitter.go` files in the `contracts/bindings` directory to see how to create a contract binding for a simple contract. The example contracts are a BalanceReader that reads native balances of a list of addresses, and a MessageEmitter that emits custom messages to the blockchain.

## 3. Deploy contracts

Deploy the BalanceReader, MessageEmitter, ReserveManager and SimpleERC20 contracts. You can either do this on a local chain or on a testnet using tools like cast/foundry.

For quickstart, you can use the existing deployed contracts on eth sepolia:
- chainID: `16015286601757825753`
- ReserveManager contract address: `0x073671aE6EAa2468c203fDE3a79dEe0836adF032`
- SimpleERC20 contract address: `0x4700A50d858Cb281847ca4Ee0938F80DEfB3F1dd`
- BalanceReader contract address: `0x4b0739c94C1389B55481cb7506c62430cA7211Cf`
- MessageEmitter contract address: `0x1d598672486ecB50685Da5497390571Ac4E93FDc`

## 3. Configure workflow

Configure config.yaml for the workflow
- `tokenAddress` should be the SimpleERC20 contract address
- `porAddress` should be the ReserveManager contract address
- `chainSelector` should be chainID of selected chain (refer to https://github.com/smartcontractkit/chain-selectors/blob/main/selectors.yml)

The config is already populated with deployed contracts in template.

## 4. Configure RPC endpoints

For local simulation to interact with a chain, you must specify RPC endpoints for the chains you interact with in the `project.yaml` file. This is required for submitting transactions and reading blockchain state.

Note: Only eth sepolia (chain selector `16015286601757825753`) is supported in local simulation

Add your preferred RPCs under the `rpcs` section. For chain selectors refer to https://github.com/smartcontractkit/chain-selectors/blob/main/selectors.yml

```yaml
rpcs:
  - chain-selector: 16015286601757825753
    url: <Your RPC endpoint to ETH Sepolia>
```
Ensure the provided URLs point to valid RPC endpoints for the specified chains. You may use public RPC providers or set up your own node.

## 5. Simulate the workflow
Run the command from <b>workflow root directory</b> (Run `cd workflowName` if you are in project root directory)

```bash
cre workflow simulate --target local-simulation --config config.json main.go
```

After this you will get a set of options similar to:

```
🚀 Workflow simulation ready. Please select a trigger:
1. cron-trigger@1.0.0 Trigger
2. evm@1.0.0 LogTrigger
3. http-trigger@1.0.0-alpha Trigger

Enter your choice (1-3):
```

You can simulate each of the following triggers types as follows

### 5a. Simulating Cron Trigger Workflows

Select option 1, and the workflow should immediately execute.

### 5b. Simulating Log Trigger Workflows

Select option 2, and then two additional prompts will come up and you can pass in the example inputs:

Transaction Hash: 0x420721d7d00130a03c5b525b2dbfd42550906ddb3075e8377f9bb5d1a5992f8e
Log Event Index: 0

The output will look like:
```
🔗 EVM Trigger Configuration:
Please provide the transaction hash and event index for the EVM log event.
Enter transaction hash (0x...): 0x420721d7d00130a03c5b525b2dbfd42550906ddb3075e8377f9bb5d1a5992f8e
Enter event index (0-based): 0
Fetching transaction receipt for transaction 0x420721d7d00130a03c5b525b2dbfd42550906ddb3075e8377f9bb5d1a5992f8e...
Found log event at index 0: contract=0x1d598672486ecB50685Da5497390571Ac4E93FDc, topics=3
Created EVM trigger log for transaction 0x420721d7d00130a03c5b525b2dbfd42550906ddb3075e8377f9bb5d1a5992f8e, event 0
```

### 5c. Simulating HTTP Trigger Workflows

Select option 3, and then an additional prompt will come up where you can pass in:

File Path: ./http_trigger_payload.json

The output will look like:
```
🔍 HTTP Trigger Configuration:
Please provide JSON input for the HTTP trigger.
You can either:
1. Enter a file path to a JSON file
2. Enter JSON directly

Enter your input: ./http_trigger_payload.json   
Loaded JSON from file: ./http_trigger_payload.json
Created HTTP trigger payload with 1 fields
```
