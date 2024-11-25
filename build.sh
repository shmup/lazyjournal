#!/bin/bash

version=${1:-"0.2.0"}
mkdir -p bin
rm -rf bin/*

architectures=(amd64 arm64)
for arch in "${architectures[@]}"; do
    GOOS=linux GOARCH=$arch go build -o bin/lazyjournal-$version-linux-$arch
done

if [[ -n "$2" ]]; then
    snapcraft --destructive-mode
    mv "$(ls *.snap)" "bin/$(ls *.snap | sed "s/_/-/g")"
fi
