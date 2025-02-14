#!/bin/bash

# version=${1:-"0.7.3"}
version=$(cat main.go | grep Version: | awk -F '"' '{print $4}')
mkdir -p bin
rm -rf bin/*

go mod tidy

golangci=$(echo $(go env GOPATH)/bin/golangci-lint)
gocritic=$(echo $(go env GOPATH)/bin/gocritic)

golangci_version=$($golangci --version 2> /dev/null)
if [ -z "$golangci_version" ]; then
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

gocritic_version=$($gocritic --version 2> /dev/null)
if [ -z "$gocritic_version" ]; then
    go install -v github.com/go-critic/go-critic/cmd/gocritic@latest
fi

golangci_check=$($golangci run ./... --build-tags=buildvcs=false)
gocritic_check=$($gocritic check -enableAll ./...)

if [ -n "$golangci_check" ]; then
    echo "$golangci_check"
elif [ -n "$gocritic_check" ]; then
    echo "$gocritic_check"
else
    architectures=(amd64 arm64)
    for arch in "${architectures[@]}"; do
        GOOS=linux GOARCH=$arch go build -o bin/lazyjournal-$version-linux-$arch
        GOOS=darwin GOARCH=$arch go build -o bin/lazyjournal-$version-darwin-$arch
        GOOS=openbsd GOARCH=$arch go build -o bin/lazyjournal-$version-openbsd-$arch
        GOOS=freebsd GOARCH=$arch go build -o bin/lazyjournal-$version-freebsd-$arch
        GOOS=windows GOARCH=$arch go build -o bin/lazyjournal-$version-windows-$arch.exe
    done
    ls -lh bin
fi
