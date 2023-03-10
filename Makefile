.PHONY: lint test clean all clean

BINARY_NAME=kced

all: build test lint

build:
	go build -o ${BINARY_NAME} main.go

lint:
	golangci-lint run

test:
	go test -v ./...

clean:
	$(RM) ./convertoas3/oas3_testfiles/*.generated.json
	$(RM) ./${BINARY_NAME}
	$(RM) ./go-apiops
	go mod tidy
	go clean
