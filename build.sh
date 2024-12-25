#!/bin/bash

version=${1:-"0.6.0"}
mkdir -p bin
rm -rf bin/*

go get -u golang.org/x/sys

if ! command -v golangci-lint &> /dev/null; then
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
else
    golangci-lint --version
fi

output=$(golangci-lint run ./... --build-tags=buildvcs=false)

if [ -z "$output" ]; then
    architectures=(amd64 arm64)
    for arch in "${architectures[@]}"; do
        GOOS=linux GOARCH=$arch go build -o bin/lazyjournal-$version-linux-$arch
        GOOS=darwin GOARCH=$arch go build -o bin/lazyjournal-$version-darwin-$arch
        GOOS=windows GOARCH=$arch go build -o bin/lazyjournal-$version-windows-$arch.exe
    done
    ls -lh bin
else
    echo "$output"
fi
