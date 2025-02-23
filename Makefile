lint:
	golangci-lint run --config=golangci.yml ./...

format:
	gofumpt -l -w .
	gci write --skip-generated -s standard -s default .

build:
	go build .

run:
	go run .

test:
	go test ./...
