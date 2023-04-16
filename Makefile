
DOCKER=podman
GOLANGCI_IMAGE=golangci/golangci-lint
GOLANGCI_TAG=v1.52.2

# root directory for project
root_dir := $(dir $(abspath $(MAKEFILE_LIST)))

.PHONY: lint test testlocal

lint:
	@${DOCKER} run --rm -v ${root_dir}:/app -w /app ${GOLANGCI_IMAGE}:${GOLANGCI_TAG} golangci-lint run

test:
	@${DOCKER} run --rm -v ${root_dir}:/app -w /app ${GOLANGCI_IMAGE}:${GOLANGCI_TAG} make testlocal

testlocal:
	@go test -coverprofile=coverage.out
