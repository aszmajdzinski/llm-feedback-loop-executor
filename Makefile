lint:
	golangci-lint run --config=golangci.yml ./...

format:
	gofumpt -l -w .
	gci write --skip-generated -s standard -s default .
	golines -m 100 -w .
	
build:
	go build .

run:
	go run .

test:
	go test ./...
