<p align="center">
    <img src="/img/logo.jpg">
</p>

Terminal user interface for `journalctl` (tool for reading logs from [systemd](https://github.com/systemd/systemd)), logs in the file system (including archival, for example, apache or nginx) and docker containers for quick viewing and filtering with fuzzy find and regex support (like `fzf` and `grep`), written in Go with the [awesome-gocui](https://github.com/awesome-gocui/gocui) (fork [gocui](https://github.com/jroimartin/gocui)) library.

This tool is inspired by and with love for [lazydocker](https://github.com/jesseduffield/lazydocker) and [lazygit](https://github.com/jesseduffield/lazygit).

![interface](/img/interface.png)

## Filter

Supported 3 filtering modes:

- **[Default]** - case sensitive exact search.
- **[Fuzzy]** - imprecise case-insensitive search (searches for all phrases separated by a space anywhere in the string).
- **[Regex]** - search with regular expression support, case insensitive by default (in case a regular expression syntax error occurs, the input field will be highlighted in red).

There is currently a 5000 line limit for outputting any log from the end.

## Roadmap

- [X] Sorting logs by modification date and support archived logs from `/var/log` directory.
- [X] Support fuzzy find and regular expression to filter output.
- [X] Highlighting of found words and phrases during filtering..
- [ ] Filter for log lists and change the number of lines for log output.
- [ ] Add a switch to load other logs (for example, `USER_UNIT`) and other log paths in the file system.
- [ ] Podman log support.
- [ ] Background checking and updating the log when data changes.
- [ ] Windows support via PowerShell (events and logs from Program Files and others).
- [ ] Scrolling interface.
- [ ] Mouse support.
- [ ] Syntax coloring for logging output.
- [ ] Support remote machines via `ssh` protocol.

## Install

Binaries for the Linux operating system are available on the [releases](https://github.com/Lifailon/lazyjournal/releases) page.

> Development is done on the Ubuntu system, also tested in WSL environment on Debian system (`x64` platform) and Raspberry Pi (`aarch64` platform).

Run the command in your console to quickly install or update:

```shell
curl https://raw.githubusercontent.com/Lifailon/lazyjournal/main/install.sh | bash
```

This command will run a script that will download the latest executable from the GitHub repository into your current user's home directory along with other executables (or create a directory) and grant execution permission.

You can also use Go for installation. To do this, the Go interpreter must be installed on the system, for example, for Ubuntu you can use the SnapCraft package manager:

```shell
sudo snap install go --classic
go install github.com/Lifailon/lazyjournal@latest
```

You can launch the interface anywhere:

```shell
lazyjournal
```

If the current user does not have rights to read logs in the `/var/log` directory or access to Docker containers (or the containerization system is not installed), then these windows will be empty.

## Build

Clone the repository, install dependencies from `go.mod` and run the project:

```shell
git clone https://github.com/Lifailon/lazyjournal
cd lazyjournal
go mod tidy
go run main.go
```

Building the executable files:

```shell
version="0.1.0"
for arch in amd64 arm64; do
    GOOS=linux GOARCH=$arch go build -o bin/lazyjournal-$version-linux-$arch
done
```

<!-- 
### Build deb package

```shell
mkdir -p DEBIAN usr/local/bin
cp bin/lazyjournal-$version-linux-$arch lazyjournal/usr/local/bin/lazyjournal
```

`vim DEBIAN/control`

```
Package: lazyjournal
Version: 0.1.0
Architecture: amd64
Maintainer: https://github.com/Lifailon
Description: TUI for journalctl, logs in the file system and docker containers for quick viewing and filtering with fuzzy find and regex support.
```

`cd .. && dpkg-deb --build lazyjournal`
 -->

## Hotkeys

- `Tab` - Switch between windows.
- `Enter` - Select a journal from the list to display logs.
- `Up/Down` - Move up or down through all journal lists and log output.
- `Shift+<Up/Down>` - Quickly move up or down (every 10 lines) through all journal lists and log output.
- `<Shift/Alt>+<Left/Right>` - Changing the mode in the filtering window. Available: **Default**, **Fuzzy** and **Regex**.
- `Ctrl+C` - Exit.

## Alternatives

- [Dozzle](https://github.com/amir20/dozzle) - is a small lightweight application with a web based interface to monitor Docker logs. It doesn’t store any log files. It is for live monitoring of your container logs only.

If you like using TUI tools, try [multranslate](https://github.com/Lifailon/multranslate) for translating text in multiple translators simultaneously, with support for translation history and automatic language detection.

<!--
```j
 /$$                                                            
| $$                                                            
| $$        /$$$$$$  /$$$$$$$$ /$$   /$$                        
| $$       |____  $$|____ /$$/| $$  | $$                        
| $$        /$$$$$$$   /$$$$/ | $$  | $$                        
| $$       /$$__  $$  /$$__/  | $$  | $$                        
| $$$$$$$$|  $$$$$$$ /$$$$$$$$|  $$$$$$$                        
|________/ \_______/|________/ \____  $$                        
                               /$$  | $$                        
                              |  $$$$$$/                        
                               \______/                         
    /$$$$$                                                   /$$
   |__  $$                                                  | $$
      | $$  /$$$$$$  /$$   /$$  /$$$$$$  /$$$$$$$   /$$$$$$ | $$
      | $$ /$$__  $$| $$  | $$ /$$__  $$| $$__  $$ |____  $$| $$
 /$$  | $$| $$  \ $$| $$  | $$| $$  \__/| $$  \ $$  /$$$$$$$| $$
| $$  | $$| $$  | $$| $$  | $$| $$      | $$  | $$ /$$__  $$| $$
|  $$$$$$/|  $$$$$$/|  $$$$$$/| $$      | $$  | $$|  $$$$$$$| $$
 \______/  \______/  \______/ |__/      |__/  |__/ \_______/|__/
```
-->