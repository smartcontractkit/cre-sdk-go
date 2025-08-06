# cre-sdk-go

## Setup

Run `make clean-generate` to generate all necessary protos.

## Updating the SDK in all capabilities

`make update-capabilities` will update all capabilities to the latest commit of the SDK (defaulting to the `main` branch) and re-generate them.

To use a different branch, run:

`make update-capabilities CRE_BRANCH=<branch-name>`

