gomods: ## Install gomods
	go install github.com/jmank88/gomods@v0.1.6

.PHONY: gomodtidy
gomodtidy: gomods
	gomods -s proto_vendor tidy

.PHONY: install-protoc
install-protoc:
	script/install-protoc.sh 29.3 /
	go install google.golang.org/protobuf/cmd/protoc-gen-go@`go list -m -json google.golang.org/protobuf | jq -r .Version`
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

.PHONY: generate
generate: install-protoc gomods modgraph
	export PATH="$(HOME)/.local/bin:$(PATH)"; gomods -s proto_vendor -go generate -x ./...

.PHONY: modgraph
modgraph: gomods
	go install github.com/jmank88/modgraph@v0.1.0
	./modgraph > go.md


.PHONY: clean
clean:
	find . | grep -F .pb.go | grep -v proto_vendor | xargs rm -f
	find . | grep -F _gen.go | grep -v proto_vendor | xargs rm -f
	find . | grep -F .lock | grep -v proto_vendor | xargs rm -f

.PHONY: clean-generate
clean-generate: clean generate

.PHONY: test
test: unit_test standard_tests

.PHONY: unit_test
unit_test: gomods
	gomods -s proto_vendor -go test ./...

CRE_BRANCH ?= main

update-capabilities:
	@echo "Finding go.mod files under capabilities/..."
	@dirs=$$(find capabilities -name go.mod -exec dirname {} \;); \
	echo "Found directories:" $$dirs; \
	failed=(); \
	for dir in $$dirs; do \
		( \
			echo "Updating SDK in $$dir..."; \
			cd $$dir && \
			go get github.com/smartcontractkit/cre-sdk-go@`git log $${CRE_BRANCH} -n 1 --pretty=format:"%H"` && \
			go mod tidy && \
			go generate . && \
			go test ./... && \
			echo "✅ $$dir success" \
		) || failed+=($$dir) & \
	done; \
	wait; \
	if [ $${#failed[@]} -ne 0 ]; then \
		echo ""; \
		echo "❌ SDK update failed in the following directories:"; \
		for f in $${failed[@]}; do echo " - $$f"; done; \
		exit 1; \
	else \
		echo ""; \
		echo "✅ All SDK updates completed successfully."; \
	fi


# Override if you wish to test against a branch.
# Alternatively, you can override the directory in chainlink-common to point to this repository
COMMON_VERSION ?= cre-std-tests@0.5.0
MODULE := github.com/smartcontractkit/chainlink-common

# Override on Windows if you aren't using a Unix-like shell:
#   make standard_tests MKDIR_P=mkdir
MKDIR_P ?= mkdir -p

.PHONY: standard_tests
standard_tests:
	@echo "Downloading standard test version $(COMMON_VERSION)"
	@$(MKDIR_P) .tools
	@go mod download -json $(MODULE)@$(COMMON_VERSION) >/dev/null
	@mod_dir=$$(go list -m -f '{{.Dir}}' $(MODULE)@$(COMMON_VERSION)); \
	abs_dir=$$(cd .tools && pwd); \
	echo "Building standard tests"; \
	( cd $$mod_dir/pkg/workflows/wasm/host && go test -c -o $$abs_dir/host.test . ); \
	echo "Running standard tests"; \
	$$abs_dir/host.test -test.v -test.run ^TestStandard -path=standard_tests
