<p align="center">
    <img src="/img/logo.jpg">
</p>

<p align="center">
    <a href="https://github.com/Lifailon/lazyjournal"><img title="Go Version"src="https://img.shields.io/github/go-mod/go-version/Lifailon/lazyjournal?logo=go"></a>
    <a href="https://github.com/Lifailon/lazyjournal/releases/latest"><img title="GitHub Release"src="https://img.shields.io/github/v/release/Lifailon/lazyjournal
?label=Latest release&logo=git&color=coral"></a>
    <a href="https://github.com/Lifailon/lazyjournal/releases"><img title="GitHub Downloads"src="https://img.shields.io/github/downloads/Lifailon/lazyjournal/total
?label=Downloads&logo=github&color=green"></a>
    <a href="https://github.com/Lifailon/Kinozal-Bot/blob/rsa/LICENSE"><img title="License"src="https://img.shields.io/github/license/Lifailon/Kinozal-Bot?logo=readme&color=white"></a>
</p>

Terminal user interface for `journalctl` (tool for reading logs from [systemd](https://github.com/systemd/systemd)), logs in the file system (including syslog and archival logs, for example, apache or nginx), Docker and Podman containers for quick viewing and filtering with fuzzy find and regex support (like `fzf` and `grep`), written in Go with the [awesome-gocui](https://github.com/awesome-gocui/gocui) (fork [gocui](https://github.com/jroimartin/gocui)) library.

This tool is inspired by and with love for [lazydocker](https://github.com/jesseduffield/lazydocker) and [lazygit](https://github.com/jesseduffield/lazygit).

![interface](/img/interface.png)

## Filter

Supported 3 filtering modes:

- **[Default]** - case sensitive exact search.
- **[Fuzzy]** - imprecise case-insensitive search (searches for all phrases separated by a space anywhere in the string).
- **[Regex]** - search with regular expression support, case insensitive by default (in case a regular expression syntax error occurs, the input field will be highlighted in red).

There is currently a 5000 line limit for outputting any log from the end.

## Roadmap

- [X] Support fuzzy find and regular expression to filter output.
- [X] Highlighting of found words and phrases during filtering.
- [X] Sorting logs by modification date and support archived logs from file system.
- [X] Add switch to load a list of user units and system loads for kernel logs
- [X] Add support for syslog, dmesg, authorization logs and downloading logs from user directories.
- [X] Podman log support.
- [ ] Swarm log support.
- [ ] Background update of selected log.
- [ ] Filter for log lists and change the number of lines for log output.
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
grep -F 'export PATH=$PATH:$HOME/go/bin' $HOME/.bashrc || echo 'export PATH=$PATH:$HOME/go/bin' >> $HOME/.bashrc && source $HOME/.bashrc
go install github.com/Lifailon/lazyjournal@latest
```

You can launch the interface anywhere:

```shell
lazyjournal
```

If the current user does not have rights to read logs in the `/var/log` directory or access to Docker containers (or the containerization system is not installed), then these windows will be empty.

Windows is not currently supported, but you can still run a project to access Docker and Podman container logs.

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
arch="amd64" # or "arm64"
GOOS=linux GOARCH=$arch go build -o bin/lazyjournal-$arch
```

<!--
Building the `snap` and `deb` packages:

```shell
bash build.sh "0.1.0" true true
```
-->

## Hotkeys

- `Tab` - Switch between windows.
- `Left/Right` - Switch between log lists in the selected window.
- `Enter` - Select a journal from the list to display log.
- `Ctrl+R` - Refresh current log to show changes.
- `Up/Down` - Move up or down through all journal lists and log output.
- `Shift+<Up/Down>` - Quickly move up or down (every 10 lines) through all journal lists and log output.
- `<Shift/Alt>+<Left/Right>` - Changing the mode in the filtering window.
- `Ctrl+<D/W>` - Clearing the text input field for the filter (available while focused on any window to quickly update the current log output without filtering).
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