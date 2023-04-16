
DOCKER=podman
GOLANGCI_IMAGE=golangci/golangci-lint
GOLANGCI_TAG=v1.50.1

# root directory for project
root_dir := $(dir $(abspath $(MAKEFILE_LIST)))

.PHONY: lint

test:
	@${DOCKER} run --rm -v ${root_dir}:/app -w /app ${GOLANGCI_IMAGE}:${GOLANGCI_TAG} go test -coverprofile=coverage.out

lint:
	@${DOCKER} run --rm -v ${root_dir}:/app -w /app ${GOLANGCI_IMAGE}:${GOLANGCI_TAG} golangci-lint run
