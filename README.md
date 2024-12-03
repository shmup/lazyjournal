<p align="center">
    <img src="/img/logo.jpg">
</p>

<p align="center">
    <a href="https://pkg.go.dev/github.com/Lifailon/lazyjournal"><img src="https://pkg.go.dev/badge/github.com/Lifailon/lazyjournal.svg" alt="Go Reference"></a>
    <a href="https://goreportcard.com/report/github.com/Lifailon/lazyjournal"><img src="https://goreportcard.com/badge/github.com/Lifailon/lazyjournal" alt="Go Report"></a>
    <a href="https://github.com/Lifailon/lazyjournal"><img title="Go Version"src="https://img.shields.io/github/go-mod/go-version/Lifailon/lazyjournal?logo=go"></a>
    <a href="https://github.com/Lifailon/Kinozal-Bot/blob/rsa/LICENSE"><img title="License"src="https://img.shields.io/github/license/Lifailon/Kinozal-Bot?logo=readme&color=white"></a>
</p>

Terminal user interface for `journalctl`, file system logs, as well Docker and Podman containers for quick viewing and filtering with fuzzy find and regex support (like `fzf` and `grep`), written in Go with the [awesome-gocui](https://github.com/awesome-gocui/gocui) (fork [gocui](https://github.com/jroimartin/gocui)) library.

This tool is inspired by and with love for [lazydocker](https://github.com/jesseduffield/lazydocker) and [lazygit](https://github.com/jesseduffield/lazygit).

![interface](/img/fuzzy.png)

![interface](/img/regex.png)

## Functional

- Simple installation, to run it, just download a single executable file without dependencies.
- View all system and user units logs via `journalctl` (tool for reading logs using `journald` from [systemd](https://github.com/systemd/systemd)).
- List of all system boots for kernel log output.
- File system logs from `/var/log` (for example, for Apache or Nginx), as well as syslog, dmesg (kernel) and user authentication (`wtmp` and `btmp`) sorted by modification date.
- View all log files in the home directories of users and root.
- Reading archived logs in `gz` format.
- Podman, Docker containers and Swarm services logs.
- Displays the currently selected log (when loading a log, a delimiter with loading time is displayed) and filters output in real-time.

Supports 3 filtering modes:

- **Default** - case sensitive exact search.
- **Fuzzy** - imprecise case-insensitive search (searches for all phrases separated by a space anywhere in the string).
- **Regex** - search with regular expression support, case insensitive by default (in case a regular expression syntax error occurs, the input field will be highlighted in red).

Regex mode uses the built-in `regexp` library, which is resource-intensive when filtering several tens of thousands of lines, so when switching modes, a limit of 50k lines is applied (for other modes the default is 200k lines), but this number can be changed using the buttons `Left/Right` in the log output window (maximum 300k lines).

## Roadmap

This is an up-to-date roadmap in addition to the functionality described above.

- [ ] MacOS support for `launchd` (issue [#1](https://github.com/Lifailon/lazyjournal/issues/1)).
- [ ] Windows support (`Windows Events` via `PowerShell` and log files from `Program Files` and others directories).
- [ ] Syntax coloring for logging output (like [tailspin](https://github.com/bensadeh/tailspin)).
- [ ] Interface for scrolling and mouse support.
- [ ] Support remote machines via `ssh` protocol.

## Install

Binaries for the Linux operating system are available on the [releases](https://github.com/Lifailon/lazyjournal/releases) page.

> Development is done on the Ubuntu Server system, also tested in WSL environment on Debian system (`x64` platform) and Raspberry Pi (`aarch64` platform).

Run the command in your console to quickly install or update:

```shell
curl https://raw.githubusercontent.com/Lifailon/lazyjournal/main/install.sh | bash
```

This command will run a script that will download the latest executable from the GitHub repository into your current user's home directory along with other executables (or create a directory) and grant execution permission.

You can also use Go for installation. To do this, the Go interpreter must be installed on the system, for example, in Ubuntu you can use the SnapCraft package manager:

```shell
sudo snap install go --classic
grep -F 'export PATH=$PATH:$HOME/go/bin' $HOME/.bashrc || echo 'export PATH=$PATH:$HOME/go/bin' >> $HOME/.bashrc && source $HOME/.bashrc

go install github.com/Lifailon/lazyjournal@latest
```

You can launch the interface anywhere (no parameters are used):

```shell
lazyjournal
```

Access to all system logs and containers may require elevated privileges for the current user.

Windows and MacOS is not currently supported, but the project will work to access Docker and Podman containers logs.

## Build

Clone the repository, install dependencies from `go.mod` and run the project:

```shell
git clone https://github.com/Lifailon/lazyjournal
cd lazyjournal
go mod tidy
go run main.go
```

Building the executable files for different platforms:

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

## Alternatives

- [lnav](https://github.com/tstack/lnav) - The Logfile Navigator is a **log file** viewer for the terminal.
- [Dozzle](https://github.com/amir20/dozzle) - is a small lightweight application with a web based interface to monitor **Docker logs**.

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