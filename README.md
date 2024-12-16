<p align="center">
    <img src="/img/logo.jpg">
</p>

<p align="center">
    <a href="https://pkg.go.dev/github.com/Lifailon/lazyjournal"><img src="https://pkg.go.dev/badge/github.com/Lifailon/lazyjournal.svg" alt="Go Reference"></a>
    <a href="https://goreportcard.com/report/github.com/Lifailon/lazyjournal"><img src="https://goreportcard.com/badge/github.com/Lifailon/lazyjournal" alt="Go Report"></a>
    <a href="https://github.com/Lifailon/lazyjournal"><img title="Go Version"src="https://img.shields.io/github/go-mod/go-version/Lifailon/lazyjournal?logo=go"></a>
    <a href="https://github.com/Lifailon/Kinozal-Bot/blob/rsa/LICENSE"><img title="License"src="https://img.shields.io/github/license/Lifailon/Kinozal-Bot?logo=readme&color=white"></a>
</p>

Terminal user interface for `journalctl`, file system logs, as well Docker and Podman containers for quick viewing and filtering with fuzzy find, regex support (like `fzf` and `grep`) and coloring the output (like `tailspin`), written in Go with the [awesome-gocui](https://github.com/awesome-gocui/gocui) (fork [gocui](https://github.com/jroimartin/gocui)) library.

This tool is inspired by and with love for [lazydocker](https://github.com/jesseduffield/lazydocker) and [lazygit](https://github.com/jesseduffield/lazygit).

![interface](/img/fuzzy.png)

![interface](/img/regex.png)

## Functional

- Simple installation, to run it, just download a single executable file without dependencies.
- List of all units (services, sockets, etc.) via `systemctl` with current running status.
- View all system and user journals via `journalctl` (tool for reading logs from [systemd-journald](https://github.com/systemd/systemd/tree/main/src/journal)).
- List of all system boots for kernel log output.
- File system logs (example, for Apache or Nginx), as well as `syslog` or `messages`, `dmesg` (kernel), etc.
- List of all log files of descriptors used by processes, as well as all log files in the home directories of users.
- Reading archived logs (`gz` format).
- Podman pods, Docker containers and Swarm services logs for all systems.
- Displays the currently selected log and filters output in real-time.

Supports 3 filtering modes:

- **Default** - case sensitive exact search.
- **Fuzzy** - custom inexact case-insensitive search (searches for all phrases separated by a space anywhere on a line).
- **Regex** - search with regular expression support (based on `regexp` library), case insensitive by default (in case a regular expression syntax error occurs, the input field will be highlighted in red).

## Roadmap

This is an up-to-date roadmap in addition to the functionality described above.

- [X] File system support for MacOS and the RHEL based systems.
- [X] Syntax coloring for logging output (this functionality will be improved in major versions as part of minor version `0.6`).
- [ ] Interface for scrolling and mouse support.
- [ ] Windows support (`Windows Events` via `PowerShell` and log files from `Program Files` and others directories).
- [ ] Support remote machines via `ssh` protocol.

## Install

### Releases

Binaries for the Linux operating system are available on the [releases](https://github.com/Lifailon/lazyjournal/releases) page.

Development is carried out on the Ubuntu Server system, and is also tested on the Raspberry Pi (`aarch64` platform), the WSL environment on the Oracle Linux system and the Darwin system (`x64` platform).

Run the command in the console to quickly install or update the stable version on Linux or MacOS:

```shell
curl https://raw.githubusercontent.com/Lifailon/lazyjournal/main/install.sh | bash
```

This command will run a script that will download the latest executable from the GitHub repository into your current user's home directory along with other executables (or create a directory) and grant execution permission.

### Latest

You can also use Go for install the dev version. To do this, the Go interpreter must be installed on the system, for example, in Ubuntu you can use the SnapCraft package manager:

```shell
sudo snap install go --classic
grep -F 'export PATH=$PATH:$HOME/go/bin' $HOME/.bashrc || echo 'export PATH=$PATH:$HOME/go/bin' >> $HOME/.bashrc && source $HOME/.bashrc

go install github.com/Lifailon/lazyjournal@latest
```

### Arch Linux

If you're an Arch Linux user you can also install from the [AUR](https://aur.archlinux.org/packages/lazyjournal) with your favorite [AUR helper](https://wiki.archlinux.org/title/AUR_helpers), e.g.:

```shell
paru -S lazyjournal
```

Thank you [Matteo Giordano](https://github.com/malteo) for upload and update the package in AUR.

### Others

If you use other packag manager and want this package to be present there as well, open an issue or load it yourself and make `Pull requests`.

## Usage

You can launch the interface anywhere (no parameters are used):

```shell
lazyjournal
```

Access to all system logs and containers may require elevated privileges for the current user.

Windows is not currently supported, but the project will work to access Docker and Podman containers logs.

## Build

Clone the repository, install dependencies from `go.mod` and run the project:

```shell
git clone https://github.com/Lifailon/lazyjournal
cd lazyjournal
go mod tidy
go run main.go
```

Build executable files for different platforms and all systems:

```shell
bash build.sh
```

## Hotkeys

- `Tab` - switch between windows.
- `Shift+Tab` - return to previous window.
- `Left/Right` - switch between log lists in the selected window.
- `Enter` - selection a journal from the list to display log.
- `Up/Down` - move up or down through all journal lists and log output, as well as changing the filtering mode in the filter window.
- `<Shift/Alt>+<Up/Down>` - quickly move up or down (every `10` or `500` lines) through all journal lists and log output.
- `Ctrl+R` - refresh the current log manually and go to the bottom of the output.
- `Ctrl+<D/W>` - clear text input field for filter to quickly update current log output without filtering.
- `Ctrl+C` - exit.

## Contributing

Any contribution is welcome. If you want to implement a feature or fix something, please [open an issue](https://github.com/Lifailon/lazyjournal/issues) first.

## Alternatives

- [lnav](https://github.com/tstack/lnav) - The Logfile Navigator is a **log file** viewer for the terminal.
- [Dozzle](https://github.com/amir20/dozzle) - is a small lightweight application with a web based interface to monitor **Docker logs**.

If you like using TUI tools, try [multranslate](https://github.com/Lifailon/multranslate) for translating text in multiple translators simultaneously, with support for translation history and automatic language detection.

## License

This project is licensed under the **MIT License**. See the [LICENSE](LICENSE) file for details.

Copyright (C) 2024 Lifailon (Alex Kup)


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
