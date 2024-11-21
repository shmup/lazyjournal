#!/bin/bash

ARCH=$(uname -m)
case $ARCH in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *)
        echo -e "Processor architecture not supported:\033[31m $ARCH \033[0m"
        echo -e "Create a request with a \033[31mproblem\033[0m: https://github.com/Lifailon/lazyjournal/issues"
        exit 1
        ;;
esac

mkdir -p $HOME/.local/bin
grep -F 'export PATH=$PATH:$HOME/.local/bin' $HOME/.bashrc || echo 'export PATH=$PATH:$HOME/.local/bin' >> $HOME/.bashrc && source $HOME/.bashrc

GITHUB_LATEST_VERSION=$(curl -L -s -H 'Accept: application/json' https://github.com/Lifailon/lazyjournal/releases/latest | sed -e 's/.*"tag_name":"\([^"]*\)".*/\1/')
curl -L -s https://github.com/Lifailon/lazyjournal/releases/download/$GITHUB_LATEST_VERSION/lazyjournal-$GITHUB_LATEST_VERSION-linux-$ARCH -o $HOME/.local/bin/lazyjournal
chmod +x $HOME/.local/bin/lazyjournal

echo -e "✔ \033[32m Installation completed successfully \033[0m"
exit 0
