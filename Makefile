.PHONY: check-main-dependencies all build lint test clean

BINARY_NAME=kced

echo_fail = printf "\e[31m✘ \033\e[0m$(1)\n"
echo_pass = printf "\e[32m✔ \033\e[0m$(1)\n"

check-dependency = $(if $(shell command -v $(1)),$(call echo_pass,found $(1)),$(call echo_fail,$(1) not installed);exit 1)

check-main-dependencies:
	@$(call check-dependency,go)

all: build test lint

build: check-main-dependencies
	go build -o ${BINARY_NAME} main.go

lint:
	@$(call check-dependency,golangci-lint)
	golangci-lint run

test: check-main-dependencies
	go test -v ./...

clean: check-main-dependencies
	$(RM) ./convertoas3/oas3_testfiles/*.generated.json
	$(RM) ./${BINARY_NAME}
	$(RM) ./go-apiops
	go mod tidy
	go clean
