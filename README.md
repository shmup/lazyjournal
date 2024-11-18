# lazyjournal

<!--
```d
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

Terminal user interface for `journalctl` (tool for reading logs from [systemd](https://github.com/systemd/systemd)), logs in the file system (including archival, for example, apache or nginx) and docker containers for quick viewing and filtering with fuzzy find and regex support (like `fzf` and `grep`), written in Go with the [awesome-gocui](https://github.com/awesome-gocui/gocui) (fork [gocui](https://github.com/jroimartin/gocui)) library.

This tool is inspired by and with love for [lazydocker](https://github.com/jesseduffield/lazydocker) and [lazygit](https://github.com/jesseduffield/lazygit).

## Roadmap

- [X] Sorting logs by modification date and support archived logs from `/var/log` directory
- [X] Support fuzzy find and regular expression to filter output
- [ ] Backgound color of the found word when filtering
- [ ] Filter for log lists
- [ ] Background checking and updating the log when data changes
- [ ] Syntax coloring for logging output
- [ ] Scrolling interface
- [ ] Mouse support
- [ ] Add a switch to load other logs (for example, `USER_UNIT`) and other log paths in the file system
- [ ] Windows support via PowerShell (events and logs from program files)
- [ ] Podman log support
- [ ] Support remote machines via `ssh` protocol

## Install from source

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

### Alternatives

- [Dozzle](https://github.com/amir20/dozzle) - is a small lightweight application with a web based interface to monitor Docker logs. It doesn’t store any log files. It is for live monitoring of your container logs only.

If you like using TUI tools, try [multranslate](https://github.com/Lifailon/multranslate) for translating text in multiple translators simultaneously, with support for translation history and automatic language detection.