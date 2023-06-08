.PHONY: all check-main-dependencies check-test-dependencies check-lint-dependencies build lint test coverage clean

BINARY_NAME=kced

echo_fail = printf "\e[31m✘ \033\e[0m$(1)\n"
echo_pass = printf "\e[32m✔ \033\e[0m$(1)\n"

check-dependency = $(if $(shell command -v $(1)),$(call echo_pass,found $(1)),$(call echo_fail,$(1) not installed);exit 1)

all: check-test-dependencies check-lint-dependencies build test lint

check-main-dependencies:
	@$(call check-dependency,go)

check-test-dependencies: check-main-dependencies
	@$(call check-dependency,ginkgo)

check-lint-dependencies:
	@$(call check-dependency,golangci-lint)

build: check-main-dependencies
	go build -o ${BINARY_NAME} main.go

lint: check-lint-dependencies
	golangci-lint run

test: check-test-dependencies
	ginkgo -r

coverage: check-test-dependencies
	ginkgo -r --race --coverprofile coverage.out

clean: check-main-dependencies
	$(RM) ./openapi2kong/oas3_testfiles/*.generated.json
	$(RM) ./${BINARY_NAME}
	$(RM) ./go-apiops
	go mod tidy
	go clean
