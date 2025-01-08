#!/bin/bash

version=${1:-"0.7.0"}
mkdir -p bin
rm -rf bin/*

go mod tidy

golangci_version=$(golangci-lint --version 2> /dev/null)
if [ -z "$golangci_version" ]; then
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

gocritic_version=$(gocritic --version 2> /dev/null)
if [ -z "$gocritic_version" ]; then
    go install -v github.com/go-critic/go-critic/cmd/gocritic@latest
fi

golangci_check_1=$(golangci-lint run ./... --build-tags=buildvcs=false)
golangci_check_2=$(golangci-lint run ./... --config ./.golangci-enable-all.yml --build-tags=buildvcs=false)
gocritic_check=$(gocritic check -enableAll ./...)

if [ -n "$golangci_check_1" ]; then
    echo "$golangci_check_1"
elif [ -n "$golangci_check_2" ]; then
    echo "$golangci_check_2"
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
