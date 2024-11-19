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

## Roadmap

- [X] Sorting logs by modification date and support archived logs from `/var/log` directory.
- [X] Support fuzzy find and regular expression to filter output.
- [X] Backgound color of the found word when filtering.
- [ ] Filter for log lists.
- [ ] Background checking and updating the log when data changes.
- [ ] Syntax coloring for logging output.
- [ ] Scrolling interface.
- [ ] Mouse support.
- [ ] Add a switch to load other logs (for example, `USER_UNIT`) and other log paths in the file system.
- [ ] Windows support via PowerShell (events and logs from program files).
- [ ] Podman log support.
- [ ] Support remote machines via `ssh` protocol.

## Install

Download the executable from the GitHub repository to the current user's home directory, and grant execution permissions:

```shell
version="0.1.0"
mkdir -p ~/.local/bin
curl -s https://github.com/Lifailon/lazyjournal/releases/download/$version/lazyjorunal-$version-linux-arm64 -o ~/.local/bin/lazyjorunal
chmod +x ~/.local/bin/lazyjorunal
```

## Build

Go must be installed on the system, for example, for Ubuntu you can use the SnapCraft package manager:

```shell
snap install go --classic
go version
```

Clone the repository, install dependencies from `go.mod` and run the project:

```shell
git clone https://github.com/Lifailon/lazyjournal
cd lazyjournal/src
go mod tidy
go run main.go
```

Building the executable file:

```shell
GOOS=linux GOARCH=amd64 go build -o bin/
bin/lazyjournal
```

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