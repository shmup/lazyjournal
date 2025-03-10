<p align="center">
    <img src="/img/logo.jpg">
</p>

<p align="center">
    <a href="https://github.com/Lifailon/lazyjournal/actions/workflows/build.yml"><img title="Actions Build"src="https://github.com/Lifailon/lazyjournal/actions/workflows/build.yml/badge.svg"></a>
    <a href="https://raw.githubusercontent.com/wiki/Lifailon/lazyjournal/coverage.html"><img title="Go coverage report"src="https://raw.githubusercontent.com/wiki/Lifailon/lazyjournal/coverage.svg"></a>
    <a href="https://goreportcard.com/report/github.com/Lifailon/lazyjournal"><img src="https://goreportcard.com/badge/github.com/Lifailon/lazyjournal" alt="Go Report"></a>
    <a href="https://pkg.go.dev/github.com/Lifailon/lazyjournal"><img src="https://pkg.go.dev/badge/github.com/Lifailon/lazyjournal.svg" alt="Go Reference"></a>
    <a href="https://aur.archlinux.org/packages/lazyjournal"><img title="Arch Linux" src="https://img.shields.io/aur/version/lazyjournal?logo=arch-linux"></a>
    <a href="https://anaconda.org/conda-forge/lazyjournal"><img title="conda-forge" src="https://img.shields.io/conda/vn/conda-forge/lazyjournal?logo=anaconda"></a>
    <a href="https://formulae.brew.sh/formula/lazyjournal"><img title="Homebrew" src="https://img.shields.io/homebrew/v/lazyjournal?logo=homebrew"></a>
    <a href="https://github.com/Lifailon/Kinozal-Bot/blob/rsa/LICENSE"><img title="License"src="https://img.shields.io/github/license/Lifailon/Kinozal-Bot?logo=readme&color=white"></a>
</p>

Terminal user interface for reading logs from `journalctl`, file system, Docker and Podman containers, as well Kubernetes pods for quick viewing and filtering with fuzzy find (like `fzf`), regex support (like `grep`) and coloring the output, written in Go with the [awesome-gocui](https://github.com/awesome-gocui/gocui) (fork [gocui](https://github.com/jroimartin/gocui)) library.

This tool is inspired by and with love for [LazyDocker](https://github.com/jesseduffield/lazydocker) and [LazyGit](https://github.com/jesseduffield/lazygit), as well as is included in [Awesome-TUIs](https://github.com/rothgar/awesome-tuis?tab=readme-ov-file#development) and [Awesome-Docker](https://github.com/veggiemonk/awesome-docker?tab=readme-ov-file#terminal-ui), check out other useful projects on the repository pages.

![interface](/img/fuzzy.jpg)

## Functional

- Simple installation, download one executable file without dependencies for starting.
- List of all units (`services`, `sockets`, etc.) via `systemctl` with current running status.
- View all system and user journals via `journalctl` (tool for reading logs from [systemd-journald](https://github.com/systemd/systemd/tree/main/src/journal)).
- List of all system boots for kernel log output.
- File system logs (example, for `Apache` or `Nginx`), as well as `syslog` or `messages`, `dmesg` for kernel logs, etc.
- List of all log files of descriptors used by processes, as well as all log files in the home directories of users.
- Reading archived logs (`gz`, `xz` or `bz2` format), packet capture (`pcap` format) and Apple System Log (`asl` format).
- Docker containers (including `timestamp` and `stderr`), Podman pods and the Docker Swarm services.
- Kubernetes pods via `kubectl`
- Windows Event Logs (in test mode via `powershell` and reading via `wevtutil`) and application logs from Windows file system.
- Filtering lists to find the desired journal.
- Displays the currently selected log output in real-time.

Supports 3 filtering modes:

- **Default** - case sensitive exact search.
- **Fuzzy** - custom inexact case-insensitive search (searches for all phrases separated by a space anywhere on a line).
- **Regex** - search with regular expression support (based on the built-in [regexp](https://pkg.go.dev/regexp) library), case insensitive by default (in case a regular expression syntax error occurs, the input field will be highlighted in red).

Supported coloring groups for output:

- **Green** - keywords indicating success.
- **Red** - keywords indicating an error.
- **Blue** - statuses (info, debug, etc), actions (install, update, etc) and HTTP methods (GET, POST, etc).
- **Light Blue** - numbers (date, time, bytes, ip and mac-addresses).
- **Yellow** - known names (host name and system users) and warnings.
- **Purple** - url and full paths in the file system.
- **Custom** - unix processes.

## Install

Binaries are available for download on the [releases](https://github.com/Lifailon/lazyjournal/releases) page.

List of supported systems and architectures in which functionality is checked: 

| OS        | amd64 | arm64 | Systems                                                                                                                                   |
| -         | -     | -     | -                                                                                                                                         |
| Linux     | ✔     |  ✔   | Raspberry Pi (`aarch64`), Oracle Linux (RHEL-based in WSL environment), Arch Linux, Rocky Linux, Ubuntu Server 20.04.6 and above         |
| Darwin    | ✔     |  ✔   | macOS Sequoia 15.2 `x64` on MacBook and the `arm64` in GitHub Actions                                                                    |
| BSD       | ✔     |       | OpenBSD 7.6 and FreeBSD 14.2                                                                                                             |
| Windows   | ✔     |       | Windows 10 and 11                                                                                                                        |

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

### conda / mamba / pixi (Linux / MacOS / Windows)

If you use package managers like conda or mamba, you can install lazyjournal from [conda-forge](https://conda-forge.org/).

```shell
conda install -c conda-forge lazyjournal
mamba install -c conda-forge lazyjournal
```

You can install lazyjournal user-globally using [pixi](https://prefix.dev/).

```shell
pixi global install lazyjournal
```

### Homebrew (MacOS / Linux)

Use the following command to install lazyjournal using [Homebrew](https://brew.sh/).

```shell
brew install lazyjournal
```

### Windows

Use the following command to quickly install in your PowerShell console:

```PowerShell
irm https://raw.githubusercontent.com/Lifailon/lazyjournal/main/install.ps1 | iex
```

The following directories are used to search for logs in the file system:

- `Program Files`
- `Program Files (x86)`
- `ProgramData`
- `AppData\Local` and `AppData\Roamin` for current user

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
lazyjournal --audit, -a    # Show audit information
```

Access to all system logs and containers may require elevated privileges for the current user.

## Build

Clone the repository and run the project:

```shell
git clone https://github.com/Lifailon/lazyjournal
cd lazyjournal
go run main.go
```

Use make or [go-task](https://github.com/go-task/task) to build binaries for different platforms and systems:

```shell
make build
# or
task build
```

Check the source code on the base linters using [golangci-lint](https://github.com/golangci/golangci-lint) (including critic and security):

```shell
make lint
```

## Testing

Run unit tests to check functions and their performance:

```shell
go test -v
```

The test coverage report using CI Actions for different systems is available on the [Wiki](https://github.com/Lifailon/lazyjournal/wiki) page.

## Hotkeys

- `Tab` - switch between windows.
- `Shift+Tab` - return to previous window.
- `Left/Right` - switch between journal lists in the selected window.
- `Enter` - selection a journal from the list to display log output.
- `<Up/PgUp>` and `<Down/PgDown>` - move up and down through all journal lists and log output, as well as changing the filtering mode in the filter window.
- `<Shift/Alt>+<Up/Down>` - quickly move up and down through all journal lists and log output every `10` or `100` lines (`500` for log output).
- `<Shift/Ctrl>+<U/D>` - quickly move up and down (alternative for macOS).
- `Ctrl+A` or `Home` - go to top of log.
- `Ctrl+E` or `End` - go to the end of the log.
- `Ctrl+W` - clear text input field for filter to quickly update current log output without filtering.
- `Ctrl+C` - exit.

## Contributing

Since this is my first Go project, there may be some bad practices, BUT I want to make LazyJournal better. Any contribution will be appreciated! If you want to implement any new feature or fix something, please [open an issue](https://github.com/Lifailon/lazyjournal/issues) first.

## Alternatives

- [lnav](https://github.com/tstack/lnav) - The Logfile Navigator is a **log file** viewer for the terminal.
- [Dozzle](https://github.com/amir20/dozzle) - is a small lightweight application with a web based interface to monitor **Docker logs**.

If you like using TUI tools, try [multranslate](https://github.com/Lifailon/multranslate) for translating text in multiple translators simultaneously and LLM, with support for translation history and automatic language detection.

## License

This project is licensed under the **MIT License**. See the [LICENSE](LICENSE) file for details.

Copyright (C) 2024 Lifailon (Alex Kup)
