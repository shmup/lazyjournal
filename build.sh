#!/bin/bash

version=${1:-"0.3.0"}
mkdir -p bin
rm -rf bin/*

architectures=(amd64 arm64)
for arch in "${architectures[@]}"; do
    GOOS=linux GOARCH=$arch go build -o bin/lazyjournal-$version-linux-$arch
    GOOS=darwin GOARCH=$arch go build -o bin/lazyjournal-$version-macos-$arch
    if [ $arch == "amd64" ]; then
        GOOS=windows GOARCH=$arch go build -o bin/lazyjournal-$version-windows-$arch
    fi
done

if [ -n "$2" ]; then
    snapcraft --destructive-mode
    mv "$(ls *.snap)" "bin/$(ls *.snap | sed "s/_/-/g")"
fi

ls -lh bin
