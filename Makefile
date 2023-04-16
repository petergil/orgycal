
DOCKER=podman

# root directory for project
root_dir := $(dir $(abspath $(MAKEFILE_LIST)))

.PHONY: lint

lint:
	@${DOCKER} run --rm -v ${root_dir}:/app -w /app golangci/golangci-lint:v1.50.1 golangci-lint run
