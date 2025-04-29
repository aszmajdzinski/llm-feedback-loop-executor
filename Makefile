GO_BIN := $(shell which go)

lint:
	golangci-lint run --config=golangci.yml ./...

format:
	gofumpt -l -w .
	gci write --skip-generated -s standard -s default .
	golines -m 100 -w .
	
build:
	go build .

run:
	$(GO_BIN) run .

test:
	go test ./...

env:
	which go