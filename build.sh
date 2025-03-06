#!/bin/bash

go fmt ./...        # Formatting code
go vet ./...        # Analyzing code for errors
go get ./...        # Download all dependencies from go.mod
go mod tidy         # Removal of unused and installing missing dependencies
go mod verify       # Checking dependencies
go build -v ./...   # Checking code compilation
go get -u ./...     # Update dependencies

golangci=$(echo $(go env GOPATH)/bin/golangci-lint)
gocritic=$(echo $(go env GOPATH)/bin/gocritic)
gosec=$(echo $(go env GOPATH)/bin/gosec)

golangci_version=$($golangci --version 2> /dev/null)
if [ -z "$golangci_version" ]; then
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi
gocritic_version=$($gocritic --version 2> /dev/null)
if [ -z "$gocritic_version" ]; then
    go install github.com/go-critic/go-critic/cmd/gocritic@latest
fi
gosec_version=$($gosec --version 2> /dev/null)
if [ -z "$gosec_version" ]; then
    go install github.com/securego/gosec/v2/cmd/gosec@latest
fi

golangci_check=$($golangci run ./main.go)
if [ "$?" -ne "0" ]; then
    echo -e "\033[31m❌ Golangci linter errors\033[0m"
    echo "$golangci_check"
else
    echo -e "✔  Golangci linter checks passed \033[32msuccessfully\033[0m"
fi
gocritic_check=$($gocritic check -enableAll ./main.go)
if [ "$?" -ne "0" ]; then
    echo -e "\033[31m❌ Critic linter errors\033[0m"
    echo "$gocritic_check"
else
    echo -e "✔  Critic linter checks passed \033[32msuccessfully\033[0m"
fi
gosec_check=$($gosec -severity=high ./...)
if [ "$?" -ne "0" ]; then
    echo -e "\033[31m❌ Security linter errors\033[0m"
    echo "$gosec_check"
else
    echo -e "✔  Security linter checks passed \033[32msuccessfully\033[0m"
fi

if [ "$1" != "false" ]; then
    version=$(cat main.go | grep Version: | awk -F '"' '{print $4}')
    echo -e "Build of the version: \033[33m$version\033[0m"
    mkdir -p bin
    rm -rf bin/*
    architectures=(amd64 arm64)
    for arch in "${architectures[@]}"; do
        GOOS=linux GOARCH=$arch go build -o bin/lazyjournal-$version-linux-$arch
        GOOS=darwin GOARCH=$arch go build -o bin/lazyjournal-$version-darwin-$arch
        GOOS=openbsd GOARCH=$arch go build -o bin/lazyjournal-$version-openbsd-$arch
        GOOS=freebsd GOARCH=$arch go build -o bin/lazyjournal-$version-freebsd-$arch
        GOOS=windows GOARCH=$arch go build -o bin/lazyjournal-$version-windows-$arch.exe
    done
    ls -lh bin
else
    echo -e "Build \033[33mskipped\033[0m"
fi
