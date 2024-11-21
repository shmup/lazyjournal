#!/bin/bash

version=${1:-"0.1.0"}
rm -r bin/*
architectures=(amd64 arm64)
for arch in "${architectures[@]}"; do
    GOOS=linux GOARCH=$arch go build -o bin/lazyjournal-$version-linux-$arch
done