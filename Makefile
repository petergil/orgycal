
DOCKER=podman
GOLANGCI_IMAGE=golangci/golangci-lint
GOLANGCI_TAG=v1.52.2

# root directory for project
root_dir := $(dir $(abspath $(MAKEFILE_LIST)))

run_in_container := ${DOCKER} run --rm -v ${root_dir}:/app -w /app ${GOLANGCI_IMAGE}:${GOLANGCI_TAG}

.PHONY: lint lintlocal test testlocal

lint:
	@${run_in_container} make lintlocal

lintlocal:
	@golangci-lint run

test:
	@${run_in_container} make testlocal

testlocal:
	@go test -coverprofile=coverage.out
