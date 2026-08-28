golangci_version = v2.13.1
templ_version = v0.3.1020
goreleaser_version = v2.18.0

.PHONY: build
build:
	go run github.com/goreleaser/goreleaser/v2@${goreleaser_version} build --config ./.goreleaser-dev.yaml --single-target --snapshot --clean

.PHONY: build-tmp
build-tmp:
	go run github.com/goreleaser/goreleaser/v2@${goreleaser_version} build --config ./.goreleaser-tmp.yaml --snapshot --clean

.PHONY: format
format:
	go mod tidy
	./codegen/format.sh

.PHONY: format-extra
format-extra:
	gofumpt -w -extra .

.PHONY: format-all
format-all: format format-extra

.PHONY: lint
lint: format
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${golangci_version} run

.PHONY: test
test:
	go test -timeout 10s -race -short -v ./...

.PHONY: test-pretty
test-pretty:
	gotestsum --format dots-v2 -- -timeout 10s -race -short -v ./...

.PHONY: test-all
test-all:
	go test -timeout 5m -race -v ./...

.PHONY: generate
generate:
	./codegen/gen_code.sh
	go run github.com/a-h/templ/cmd/templ@${templ_version} generate

.PHONY: generate-docs
generate-docs:
	./codegen/gen_docs.sh
