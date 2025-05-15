GO_BIN := $(shell which go)

lint:
	golangci-lint run --config=golangci.yml ./...

format:
	gofumpt -l -w .
	gci write --skip-generated -s standard -s default .
	golines -m 100 -w .
	
build:
	$(GO_BIN) build .

run:
	$(GO_BIN) run . -config example-configuration.yaml

test:
	$(GO_BIN) test ./...

env:
	which go