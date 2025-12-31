# bumpversion

A CLI tool for computing and creating version tags for the CRE SDK releases.

## Overview

This tool automates the versioning process by:
1. Reading existing git tags to determine the latest versions
2. Computing the next version numbers based on the versioning scheme
3. Creating and optionally pushing the new tags
4. Verifying tags were successfully created

## Version Schemes

| Component | Tag Pattern | Increment |
|-----------|-------------|-----------|
| EVM Capability | `capabilities/blockchain/evm/vX.Y.Z-beta.N` | beta number (+1) |
| SDK | `vX.Y.Z` | patch version (+1) |

## Usage

```bash
# Preview what tags would be created (no changes made)
go run ./cre/tools/bumpversion --dry-run

# Create tags locally
go run ./cre/tools/bumpversion

# Create and push tags to origin
go run ./cre/tools/bumpversion --push

# Output as JSON for CI integration
go run ./cre/tools/bumpversion --dry-run --output-json
```

## Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Print the tags that would be created without creating them |
| `--push` | Push the created tags to origin after creating them |
| `--output-json` | Output results as JSON for CI integration |

## Example Output

### Standard Output

```
$ go run ./cre/tools/bumpversion --dry-run
Next EVM tag: capabilities/blockchain/evm/v1.1.2-beta.2
Next SDK tag: v1.1.3
Dry run mode - no tags created
```

### JSON Output

```
$ go run ./cre/tools/bumpversion --dry-run --output-json
{"evm_tag":"capabilities/blockchain/evm/v1.1.2-beta.2","sdk_tag":"v1.1.3"}
```

### With Push and Verification

```
$ go run ./cre/tools/bumpversion --push --output-json
{"evm_tag":"capabilities/blockchain/evm/v1.1.2-beta.2","sdk_tag":"v1.1.3","evm_pushed":true,"sdk_pushed":true}
```

## JSON Schema

When using `--output-json`, the output follows this schema:

| Field | Type | Description |
|-------|------|-------------|
| `evm_tag` | string | The EVM capability tag (created or planned) |
| `sdk_tag` | string | The SDK tag (created or planned) |
| `evm_pushed` | boolean | Whether the EVM tag was successfully pushed (only with `--push`) |
| `sdk_pushed` | boolean | Whether the SDK tag was successfully pushed (only with `--push`) |

## GitHub Workflow Integration

This tool is used by the `bump-protos.yml` workflow when the `cut_tags` input is set to `true`. The workflow:

1. Updates the `chainlink-protos` dependency
2. Regenerates protobuf files
3. Validates the build
4. Commits changes (creates PR or pushes directly)
5. Runs this tool with `--push --output-json` to create and publish new version tags
6. Outputs the created tags for downstream workflows

### Workflow Inputs

| Input | Description |
|-------|-------------|
| `protos_commit_hash` | Git commit hash from chainlink-protos repository |
| `chain_name` | Name of the chain being added (for commit context) |
| `cut_tags` | Whether to create new version tags |
| `create_pr` | Create a PR instead of pushing directly |

### Workflow Outputs

| Output | Description |
|--------|-------------|
| `evm_tag` | The created EVM capability tag |
| `sdk_tag` | The created SDK tag |
| `branch_name` | The feature branch name (if `create_pr` is true) |
