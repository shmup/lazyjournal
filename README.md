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

## Roadmap

- [ ] Support for fuzzy find and regex
- [ ] Backgound color of the found word when filtering
- [ ] Filter for log lists
- [ ] Scrolling interface
- [ ] Background checking and updating the log when data changes
- [ ] Syntax coloring for systemctl output
- [ ] Add a switch to load other logs (for example, USER_UNIT) and other log paths in the file system
- [ ] Windows support via PowerShell (events and logs from program files)
- [ ] Podman log support