#!/bin/bash

version=${1:-"0.5.0"}
mkdir -p bin
rm -rf bin/*

architectures=(amd64 arm64)
for arch in "${architectures[@]}"; do
    GOOS=linux GOARCH=$arch go build -o bin/lazyjournal-$version-linux-$arch
    GOOS=darwin GOARCH=$arch go build -o bin/lazyjournal-$version-macos-$arch
    GOOS=windows GOARCH=$arch go build -o bin/lazyjournal-$version-windows-$arch.exe
done

ls -lh bin
