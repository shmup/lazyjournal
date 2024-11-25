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
    mv "$(ls *.snap)" "$(ls *.snap | sed "s/_/-/g")"
fi

if [[ -n "$3" ]]; then
    mkdir -p DEBIAN usr/local/bin
    cd ..
    for arch in "${architectures[@]}"; do
        rm -f lazyjournal/usr/local/bin/lazyjournal
        cp lazyjournal/bin/lazyjournal-$version-linux-$arch lazyjournal/usr/local/bin/lazyjournal
        echo "Package: lazyjournal
Version: $version
Architecture: $arch
Maintainer: https://github.com/Lifailon
Description: TUI for journalctl, logs in the file system and docker containers for quick viewing and filtering with fuzzy find and regex support.
" > lazyjournal/DEBIAN/control
        dpkg-deb --build lazyjournal "lazyjournal/bin/lazyjournal-$version-$arch.deb"
    done
fi
