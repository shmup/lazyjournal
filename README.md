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

Terminal user interface for `journalctl` (tool for reading logs from [systemd](https://github.com/systemd/systemd)), written in Go with the [gocui](https://github.com/jroimartin/gocui) (fork [awesome-gocui](https://github.com/awesome-gocui/gocui)) library.

Displays a list of all available system and user journals for quick viewing and filtering with regex support (like grep).

This tool is inspired by and with love for [lazydocker](https://github.com/jesseduffield/lazydocker) and [lazygit](https://github.com/jesseduffield/lazygit).

## Install from source

Clone the repository, install dependencies from `go.mod` and run the project or building the executable file:

```shell
git clone https://github.com/Lifailon/lazyjournal
cd lazyjournal/src

# snap install go --classic
# go version

go mod tidy
go run main.go

GOOS=linux GOARCH=amd64 go build -o bin/
bin/lazyjournal
```
