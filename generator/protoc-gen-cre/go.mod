module github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre

go 1.24.4

replace github.com/smartcontractkit/cre-sdk-go => ../..

require (
	github.com/smartcontractkit/chainlink-common v0.7.1-0.20250626003221-185d379f7afc
	github.com/smartcontractkit/chainlink-common/pkg/values v0.0.0-20250626050139-95e779d9eac6
	github.com/smartcontractkit/cre-sdk-go v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/shopspring/decimal v1.4.0 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/tools v0.26.0 // indirect
)
