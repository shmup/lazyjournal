<p align="center">
    <img src="/img/logo.jpg">
</p>

<p align="center">
    <a href="https://pkg.go.dev/github.com/Lifailon/lazyjournal"><img src="https://pkg.go.dev/badge/github.com/Lifailon/lazyjournal.svg" alt="Go Reference"></a>
    <a href="https://goreportcard.com/report/github.com/Lifailon/lazyjournal"><img src="https://goreportcard.com/badge/github.com/Lifailon/lazyjournal" alt="Go Report"></a>
    <a href="https://github.com/Lifailon/lazyjournal/actions/workflows/build.yml"><img title="Actions Build"src="https://img.shields.io/github/actions/workflow/status/Lifailon/lazyjournal/build.yml?logo=GitHub-Actions"></a>
    <a href="https://aur.archlinux.org/packages/lazyjournal"><img title="Arch Linux"src="https://img.shields.io/aur/version/lazyjournal?logo=arch-linux"></a>
    <a href="https://github.com/Lifailon/Kinozal-Bot/blob/rsa/LICENSE"><img title="License"src="https://img.shields.io/github/license/Lifailon/Kinozal-Bot?logo=readme&color=white"></a>
</p>

Terminal user interface for `journalctl`, file system logs, as well **Docker** and **Podman** containers for quick viewing and filtering with fuzzy find, regex support (like `fzf` and `grep`) and coloring the output, written in Go with the [awesome-gocui](https://github.com/awesome-gocui/gocui) (fork [gocui](https://github.com/jroimartin/gocui)) library.

This tool is inspired by and with love for [LazyDocker](https://github.com/jesseduffield/lazydocker) and [LazyGit](https://github.com/jesseduffield/lazygit), as well as is listed as [Awesome-TUIs](https://github.com/rothgar/awesome-tuis), check out the other useful projects in the repository page.

![interface](/img/fuzzy.jpg)

## Functional

- Simple installation, to run just download one executable file without any dependencies.
- List of all units (services, sockets, etc.) via `systemctl` with current running status.
- View all system and user journals via `journalctl` (tool for reading logs from [systemd-journald](https://github.com/systemd/systemd/tree/main/src/journal)).
- List of all system boots for kernel log output.
- File system logs (example, for `Apache` or `Nginx`), as well as `syslog` or `messages`, `dmesg` (kernel), etc.
- List of all log files of descriptors used by processes, as well as all log files in the home directories of users.
- Reading archived logs (`gz`, `xz` or `bz2` format), packet capture (`pcap` format) and Apple System Log (`asl` format).
- Docker containers, Podman pods and Swarm services logs (including offline).
- Filtering lists to find the desired journal.
- Displays the currently selected log and filters output in real-time.

Supports 3 filtering modes:

- **Default** - case sensitive exact search.
- **Fuzzy** - custom inexact case-insensitive search (searches for all phrases separated by a space anywhere on a line).
- **Regex** - search with regular expression support (based on `regexp` library), case insensitive by default (in case a regular expression syntax error occurs, the input field will be highlighted in red).

Supported coloring groups for output:

- **Green** - keywords indicating success.
- **Red** - keywords indicating an error.
- **Blue** - statuses, (info, debug, etc), actions (install, update, etc) and HTTP methods (GET, POST, etc).
- **Light Blue** - numbers (date, time, bytes, ip and mac-addresses).
- **Yellow** - known names (host name and system users) and warnings.
- **Purple** - url and full paths in the file system.
- **Custom** - unix processes.

## Install

Binaries are available for download on the [releases](https://github.com/Lifailon/lazyjournal/releases) page.

List of supported systems and architectures in which I was able to check the functionality:

| OS        | amd64 | arm64 | Systems                                                                          |
| -         | -     | -     | -                                                                                |
| Linux     | ✔     |  ✔   | Raspberry Pi, Oracle Linux (RHEL-based in WSL), Ubuntu Server 20.04.6 and above  |
| Darwin    | ✔     |       | macOS Sequoia 15.2                                                               |
| BSD-based | ✔     |       | OpenBSD 7.6 and FreeBSD 14.2                                                     |
| Windows   | ✔     |       | Windows 10 and 11                                                                |

### Unix-based

Run the command in the console to quickly install or update the stable version for Linux, macOS or the BSD-based system:

```shell
curl -sS https://raw.githubusercontent.com/Lifailon/lazyjournal/main/install.sh | bash
```

This command will run a script that will download the latest executable from the GitHub repository into your current user's home directory along with other executables (or create a directory) and grant execution permission.

### Arch Linux

If you an Arch Linux user you can also install from the [AUR](https://aur.archlinux.org/packages/lazyjournal):

```shell
paru -S lazyjournal
```

Thank you [Matteo Giordano](https://github.com/malteo) for upload and update the package in AUR.

### Windows

Use the following command to quickly install in your PowerShell console:

```PowerShell
Invoke-RestMethod https://raw.githubusercontent.com/Lifailon/lazyjournal/main/install.ps1 | Invoke-Expression
```

Supports reading containers logs as well as searching for logs in the following directories:

- `Program Files`
- `Program Files (x86)`
- `AppData\Local` for current user
- `AppData\Roamin` for current user

To read logs, automatic detection of the following encodings is supported:

- `UTF-8`
- `UTF-16 with BOM`
- `UTF-16 without BOM`
- `Windows-1251` by default

### Go Package

You can also use Go for install the dev version ([Go](https://go.dev/doc/install) must be installed in the system):

```shell
go install github.com/Lifailon/lazyjournal@latest
```

### Others

If you use other packag manager and want this package to be present there as well, open an issue or load it yourself and make [Pull requests](https://github.com/Lifailon/lazyjournal/pulls).

## Usage

You can run the interface from anywhere:

```shell
lazyjournal                # Run interface
lazyjournal --help, -h     # Show help
lazyjournal --version, -v  # Show version
```

Access to all system logs and containers may require elevated privileges for the current user.

## Build

Clone the repository and run the project:

```shell
git clone https://github.com/Lifailon/lazyjournal
cd lazyjournal
go run main.go
```

Check the source code on the linters using [golangci-lint](https://github.com/golangci/golangci-lint) and build binaries for different platforms and systems:

```shell
bash build.sh
```

## Hotkeys

- `Tab` - switch between windows.
- `Shift+Tab` - return to previous window.
- `Left/Right` - switch between journal lists in the selected window.
- `Enter` - selection a journal from the list to display log output.
- `<Up/PgUp>/<Down/PgDown>` - move up or down through all journal lists and log output, as well as changing the filtering mode in the filter window.
- `<Shift/Alt>+<Up/Down>` - quickly move up or down through all journal lists and log output every `10` or `100/500` lines.
- `Ctrl+E/Home` - go to top of log.
- `Ctrl+D/End` - go to the end of the log.
- `Ctrl+W` - clear text input field for filter to quickly update current log output without filtering.
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
