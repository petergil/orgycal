#!/bin/sh

docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.42.1 golangci-lint run
