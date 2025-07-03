gomods: ## Install gomods
	go install github.com/jmank88/gomods@v0.1.5

.PHONY: gomodtidy
gomodtidy: gomods
	gomods tidy

.PHONY: install-protoc
install-protoc:
	script/install-protoc.sh 29.3 /
	go install google.golang.org/protobuf/cmd/protoc-gen-go@`go list -m -json google.golang.org/protobuf | jq -r .Version`
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

.PHONY: generate
generate: install-protoc gomods
	export PATH="$(HOME)/.local/bin:$(PATH)"; gomods -go generate -x ./...

.PHONY: clean
clean:
	find . | grep -F .pb.go | grep -v proto_vendor | xargs rm -f
	find . | grep -F _gen.go | grep -v proto_vendor | xargs rm -f
	find . | grep -F .lock | grep -v proto_vendor | xargs rm -f

.PHONY: clean-generate
clean-generate: clean generate