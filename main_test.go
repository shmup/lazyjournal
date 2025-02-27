package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/awesome-gocui/gocui"
)

func TestCreatReport(t *testing.T) {
	file, _ := os.Create("test-report.md")
	defer file.Close()
}

func TestWinFiles(t *testing.T) {
	// Пропускаем тест целиком для Linux/macOS/bsd
	if runtime.GOOS != "windows" {
		t.Skip("Skip Windows test")
	}

	// Создаем файл отчета
	file, _ := os.OpenFile("test-report.md", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	file.WriteString("## Windows File Logs\n")
	file.WriteString("| Path | Lines | Read | Color |\n")
	file.WriteString("|------|-------|------|-------|\n")

	// Тестируемые параметры для функции
	testCases := []struct {
		name       string
		selectPath string
	}{
		{"Program Files", "ProgramFiles"},
		{"Program Files 86", "ProgramFiles86"},
		{"ProgramData", "ProgramData"},
		// {"AppData/Local", "AppDataLocal"},
		// {"AppData/Roaming", "AppDataRoaming"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Заполняем базовые параметры структуры
			app := &App{
				selectPath:       tc.selectPath,
				testMode:         true,
				logViewCount:     "100000",
				getOS:            "windows",
				systemDisk:       "C",
				userName:         "lifailon",
				selectFilterMode: "fuzzy", // режим фильтрации
				filterText:       "",      // текст для фильтрации
				// Инициализируем переменные с регулярными выражениями
				trimHttpRegex:        trimHttpRegex,
				trimHttpsRegex:       trimHttpsRegex,
				trimPrefixPathRegex:  trimPrefixPathRegex,
				trimPostfixPathRegex: trimPostfixPathRegex,
				hexByteRegex:         hexByteRegex,
				dateTimeRegex:        dateTimeRegex,
				timeMacAddressRegex:  timeMacAddressRegex,
				timeRegex:            timeRegex,
				macAddressRegex:      macAddressRegex,
				dateIpAddressRegex:   dateIpAddressRegex,
				dateRegex:            dateRegex,
				ipAddressRegex:       ipAddressRegex,
				procRegex:            procRegex,
				syslogUnitRegex:      syslogUnitRegex,
			}

			// (1) Заполняем массив из названий файлов и путей к ним
			app.loadWinFiles(app.selectPath)
			// Если список файлов пустой, тест будет провален
			if len(app.logfiles) == 0 {
				t.Errorf("File list is null")
			} else {
				t.Log("Log files count:", len(app.logfiles))
			}

			var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)
			// Проходимся по всем путям в массиве
			for _, logfile := range app.logfiles {
				// Удаляем покраску из названия файла в массиве (в интерфейсе строка читается без покраски при выборе)
				logFileName := ansiEscape.ReplaceAllString(logfile.name, "")
				// Фиксируем время запуска функции
				startTime := time.Now()
				// (2) Читаем журнал
				app.loadFileLogs(strings.TrimSpace(logFileName), true)
				endTime := time.Since(startTime)
				// (3) Фильтруем и красим
				startTime2 := time.Now()
				app.applyFilter(true)
				endTime2 := time.Since(startTime2)
				// Записываем в отчет путь, количество строк в массиве прочитанных из файла, время чтения и фильтрации + покраски
				file.WriteString(fmt.Sprintf("| %s | %d | %s | %s |\n", app.lastLogPath, len(app.currentLogLines), endTime, endTime2))
			}
		})
	}
}

func TestWinEvents(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skip Windows test")
	}

	file, _ := os.OpenFile("test-report.md", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	file.WriteString("## Windows Event Logs\n")
	file.WriteString("| Event Name | Lines | Read | Color |\n")
	file.WriteString("|------------|-------|------|-------|\n")

	app := &App{
		testMode:             true,
		logViewCount:         "100000",
		getOS:                "windows",
		systemDisk:           "C",
		userName:             "lifailon",
		selectFilterMode:     "fuzzy",
		filterText:           "",
		trimHttpRegex:        trimHttpRegex,
		trimHttpsRegex:       trimHttpsRegex,
		trimPrefixPathRegex:  trimPrefixPathRegex,
		trimPostfixPathRegex: trimPostfixPathRegex,
		hexByteRegex:         hexByteRegex,
		dateTimeRegex:        dateTimeRegex,
		timeMacAddressRegex:  timeMacAddressRegex,
		timeRegex:            timeRegex,
		macAddressRegex:      macAddressRegex,
		dateIpAddressRegex:   dateIpAddressRegex,
		dateRegex:            dateRegex,
		ipAddressRegex:       ipAddressRegex,
		procRegex:            procRegex,
		syslogUnitRegex:      syslogUnitRegex,
	}

	app.loadWinEvents()
	if len(app.journals) == 0 {
		t.Errorf("File list is null")
	} else {
		t.Log("Windows Event Logs count:", len(app.journals))
	}

	var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	for _, journal := range app.journals {
		app.updateFile = true
		serviceName := ansiEscape.ReplaceAllString(journal.name, "")
		startTime := time.Now()
		app.loadJournalLogs(strings.TrimSpace(serviceName), true)
		endTime := time.Since(startTime)

		startTime2 := time.Now()
		app.applyFilter(true)
		endTime2 := time.Since(startTime2)

		file.WriteString(fmt.Sprintf("| %s | %d | %s | %s |\n", serviceName, len(app.currentLogLines), endTime, endTime2))
	}
}

func TestUnixFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skip Linux test")
	}

	file, _ := os.OpenFile("test-report.md", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	file.WriteString("## Unix File Logs\n")
	file.WriteString("| Path | Lines | Read | Color |\n")
	file.WriteString("|------|-------|------|-------|\n")

	testCases := []struct {
		name       string
		selectPath string
	}{
		{"System var logs", "/var/log/"},
		// {"Optional package logs", "/opt/"},
		{"Users home logs", "/home/"},
		{"Process descriptor logs", "descriptor"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := &App{
				selectPath:           tc.selectPath,
				testMode:             true,
				logViewCount:         "100000",
				getOS:                "linux",
				userName:             "lifailon",
				selectFilterMode:     "fuzzy",
				filterText:           "",
				trimHttpRegex:        trimHttpRegex,
				trimHttpsRegex:       trimHttpsRegex,
				trimPrefixPathRegex:  trimPrefixPathRegex,
				trimPostfixPathRegex: trimPostfixPathRegex,
				hexByteRegex:         hexByteRegex,
				dateTimeRegex:        dateTimeRegex,
				timeMacAddressRegex:  timeMacAddressRegex,
				timeRegex:            timeRegex,
				macAddressRegex:      macAddressRegex,
				dateIpAddressRegex:   dateIpAddressRegex,
				dateRegex:            dateRegex,
				ipAddressRegex:       ipAddressRegex,
				procRegex:            procRegex,
				syslogUnitRegex:      syslogUnitRegex,
			}

			app.loadFiles(app.selectPath)
			if len(app.logfiles) == 0 {
				t.Errorf("File list is null")
			} else {
				t.Log("Log files count:", len(app.logfiles))
			}

			var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)
			for _, logfile := range app.logfiles {
				logFileName := ansiEscape.ReplaceAllString(logfile.name, "")
				startTime := time.Now()
				app.loadFileLogs(strings.TrimSpace(logFileName), true)
				endTime := time.Since(startTime)

				startTime2 := time.Now()
				app.applyFilter(true)
				endTime2 := time.Since(startTime2)

				file.WriteString(fmt.Sprintf("| %s | %d | %s | %s |\n", app.lastLogPath, len(app.currentLogLines), endTime, endTime2))
			}
		})
	}
}

func TestLinuxJournal(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skip Linux test")
	}

	file, _ := os.OpenFile("test-report.md", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	file.WriteString("## Linux journals\n")
	file.WriteString("| Journal Name | Lines | Read | Color |\n")
	file.WriteString("|--------------|-------|------|-------|\n")

	testCases := []struct {
		name        string
		journalName string
	}{
		{"Unit list", "services"},
		{"System journals", "UNIT"},
		{"User journals", "USER_UNIT"},
		{"Kernel boot", "kernel"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := &App{
				selectUnits:          tc.journalName,
				testMode:             true,
				logViewCount:         "100000",
				getOS:                "linux",
				selectFilterMode:     "fuzzy",
				filterText:           "",
				trimHttpRegex:        trimHttpRegex,
				trimHttpsRegex:       trimHttpsRegex,
				trimPrefixPathRegex:  trimPrefixPathRegex,
				trimPostfixPathRegex: trimPostfixPathRegex,
				hexByteRegex:         hexByteRegex,
				dateTimeRegex:        dateTimeRegex,
				timeMacAddressRegex:  timeMacAddressRegex,
				timeRegex:            timeRegex,
				macAddressRegex:      macAddressRegex,
				dateIpAddressRegex:   dateIpAddressRegex,
				dateRegex:            dateRegex,
				ipAddressRegex:       ipAddressRegex,
				procRegex:            procRegex,
				syslogUnitRegex:      syslogUnitRegex,
			}

			app.loadServices(app.selectUnits)
			if len(app.journals) == 0 {
				t.Errorf("File list is null")
			} else {
				t.Log("Journal count:", len(app.journals))
			}

			var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)
			for _, journal := range app.journals {
				serviceName := ansiEscape.ReplaceAllString(journal.name, "")
				startTime := time.Now()
				app.loadJournalLogs(strings.TrimSpace(serviceName), true)
				endTime := time.Since(startTime)

				startTime2 := time.Now()
				app.applyFilter(true)
				endTime2 := time.Since(startTime2)

				file.WriteString(fmt.Sprintf("| %s | %d | %s | %s |\n", serviceName, len(app.currentLogLines), endTime, endTime2))
			}
		})
	}
}

func TestDockerContainer(t *testing.T) {
	file, _ := os.OpenFile("test-report.md", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	file.WriteString("## Containers\n")
	file.WriteString("| Container Name | Lines | Read | Color |\n")
	file.WriteString("|----------------|-------|------|-------|\n")

	testCases := []struct {
		name                         string
		selectContainerizationSystem string
	}{
		{"Docker", "docker"},
		// {"Podman", "podman"},
		// {"Kubernetes", "kubectl"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Пропускаем не установленые системы
			_, err := exec.LookPath(tc.selectContainerizationSystem)
			if err != nil {
				t.Skip("Skip: ", tc.selectContainerizationSystem, " not installed (environment not found)")
			}
			app := &App{
				selectContainerizationSystem: tc.selectContainerizationSystem,
				testMode:                     true,
				logViewCount:                 "100000",
				selectFilterMode:             "fuzzy",
				filterText:                   "",
				trimHttpRegex:                trimHttpRegex,
				trimHttpsRegex:               trimHttpsRegex,
				trimPrefixPathRegex:          trimPrefixPathRegex,
				trimPostfixPathRegex:         trimPostfixPathRegex,
				hexByteRegex:                 hexByteRegex,
				dateTimeRegex:                dateTimeRegex,
				timeMacAddressRegex:          timeMacAddressRegex,
				timeRegex:                    timeRegex,
				macAddressRegex:              macAddressRegex,
				dateIpAddressRegex:           dateIpAddressRegex,
				dateRegex:                    dateRegex,
				ipAddressRegex:               ipAddressRegex,
				procRegex:                    procRegex,
				syslogUnitRegex:              syslogUnitRegex,
			}

			app.loadDockerContainer(app.selectContainerizationSystem)
			if len(app.dockerContainers) == 0 {
				t.Errorf("Container list is null")
			} else {
				t.Log("Container count:", len(app.dockerContainers))
			}

			var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)
			for _, dockerContainer := range app.dockerContainers {
				containerName := ansiEscape.ReplaceAllString(dockerContainer.name, "")
				startTime := time.Now()
				app.loadDockerLogs(strings.TrimSpace(containerName), true)
				endTime := time.Since(startTime)

				startTime2 := time.Now()
				app.applyFilter(true)
				endTime2 := time.Since(startTime2)

				file.WriteString(fmt.Sprintf("| %s | %d | %s | %s |\n", containerName, len(app.currentLogLines), endTime, endTime2))
			}
		})
	}
}

func TestFilterColor(t *testing.T) {
	testCases := []struct {
		name             string
		selectFilterMode string
	}{
		{"Default filter mode", "default"},
		{"Fuzzy filter mode", "fuzzy"},
		{"Regex filter mode", "regex"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := &App{
				testMode:             true,
				logViewCount:         "100000",
				selectFilterMode:     tc.selectFilterMode,
				filterText:           "line",
				trimHttpRegex:        trimHttpRegex,
				trimHttpsRegex:       trimHttpsRegex,
				trimPrefixPathRegex:  trimPrefixPathRegex,
				trimPostfixPathRegex: trimPostfixPathRegex,
				hexByteRegex:         hexByteRegex,
				dateTimeRegex:        dateTimeRegex,
				timeMacAddressRegex:  timeMacAddressRegex,
				timeRegex:            timeRegex,
				macAddressRegex:      macAddressRegex,
				dateIpAddressRegex:   dateIpAddressRegex,
				dateRegex:            dateRegex,
				ipAddressRegex:       ipAddressRegex,
				procRegex:            procRegex,
				syslogUnitRegex:      syslogUnitRegex,
			}

			app.currentLogLines = []string{
				"1  line: http://localhost:8443",
				"2  line: https://github.com/Lifailon/lazyjournal",
				"3  line: /etc/ssh/sshd_config",
				"4  line: root runner",
				"5  line: warning",
				"6  line: stderr: disconnected and crashed",
				"7  line: kernel: deletion removed stopped invalidated aborted blocked deactivated",
				"8  line: rsyslogd: exited critical failed rejection fataling closed ended dropped killing",
				"9  line: sudo: cancelation unavailable unsuccessful found denied conflict false none",
				"10 line: /dev/null",
				"11 line: null success complete accept connection finish started created enable allowing posted",
				"12 line: routing forward passed running added opened patching ok available accessible done true",
				"13 line: stdout input GET SET head request upload listen launch change clear skip missing mount",
				"14 line: authorization configuration option writing saving boot paused filter normal notice alert",
				"15 line: information update shutdown status debug verbose trace protocol level",
				"16 line: 2025-02-26T21:38:35.956968+03:00 ⎯⎯⎯ 0x04 ⎯⎯⎯",
				"17 line: 25.02.2025 11:11 11:11:11:11:11:11 11-11-11-11-11-11",
				"18 line: TCP UDP ICMP IP 192.168.1.1:8443",
				"19 line: 25.02.2025 01:14:42 [INFO]: not data",
				"20 line: cron[123]: running",
			}

			app.applyFilter(true)
			t.Log("Lines: ", len(app.filteredLogLines))
			for _, line := range app.filteredLogLines {
				t.Log(line)
			}
		})
	}
}

func TestFlag(t *testing.T) {
	app := &App{}
	showHelp()
	app.showVersion()
}

func TestMainInterface(t *testing.T) {
	go runGoCui(true)
	time.Sleep(3 * time.Second)
	quit(g, nil)
}

func TestMockInterface(t *testing.T) {
	app := &App{
		testMode:                     false,
		startServices:                0,
		selectedJournal:              0,
		startFiles:                   0,
		selectedFile:                 0,
		startDockerContainers:        0,
		selectedDockerContainer:      0,
		selectUnits:                  "services",
		selectPath:                   "/var/log/",
		selectContainerizationSystem: "docker",
		selectFilterMode:             "default",
		logViewCount:                 "200000",
		journalListFrameColor:        gocui.ColorDefault,
		fileSystemFrameColor:         gocui.ColorDefault,
		dockerFrameColor:             gocui.ColorDefault,
		autoScroll:                   true,
		trimHttpRegex:                trimHttpRegex,
		trimHttpsRegex:               trimHttpsRegex,
		trimPrefixPathRegex:          trimPrefixPathRegex,
		trimPostfixPathRegex:         trimPostfixPathRegex,
		hexByteRegex:                 hexByteRegex,
		dateTimeRegex:                dateTimeRegex,
		timeMacAddressRegex:          timeMacAddressRegex,
		timeRegex:                    timeRegex,
		macAddressRegex:              macAddressRegex,
		dateIpAddressRegex:           dateIpAddressRegex,
		dateRegex:                    dateRegex,
		ipAddressRegex:               ipAddressRegex,
		procRegex:                    procRegex,
		syslogUnitRegex:              syslogUnitRegex,
		keybindingsEnabled:           true,
	}

	app.getOS = runtime.GOOS
	app.getArch = runtime.GOARCH

	var err error
	// Отключение tcell для CI
	g, err = gocui.NewGui(gocui.OutputSimulator, true)
	// Включить отображение интерфейса
	// g, err = gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	app.gui = g
	g.SetManagerFunc(app.layout)
	g.Mouse = false

	g.FgColor = gocui.ColorDefault
	g.BgColor = gocui.ColorDefault

	if err := app.setupKeybindings(); err != nil {
		log.Panicln("Error key bindings", err)
	}

	if err := app.layout(g); err != nil {
		log.Panicln(err)
	}

	app.hostName, _ = os.Hostname()
	if strings.Contains(app.hostName, ".") {
		app.hostName = strings.Split(app.hostName, ".")[0]
	}
	currentUser, _ := user.Current()
	app.userName = currentUser.Username
	if strings.Contains(app.userName, "\\") {
		app.userName = strings.Split(app.userName, "\\")[1]
	}
	app.systemDisk = os.Getenv("SystemDrive")
	if len(app.systemDisk) >= 1 {
		app.systemDisk = string(app.systemDisk[0])
	} else {
		app.systemDisk = "C"
	}
	passwd, _ := os.Open("/etc/passwd")
	scanner := bufio.NewScanner(passwd)
	for scanner.Scan() {
		line := scanner.Text()
		userName := strings.Split(line, ":")
		if len(userName) > 0 {
			app.userNameArray = append(app.userNameArray, userName[0])
		}
	}
	files, _ := os.ReadDir("/")
	for _, file := range files {
		if file.IsDir() {
			app.rootDirArray = append(app.rootDirArray, file.Name())
		}
	}

	if v, err := g.View("services"); err == nil {
		_, viewHeight := v.Size()
		app.maxVisibleServices = viewHeight
	}
	if app.getOS == "windows" {
		v, err := g.View("services")
		if err != nil {
			log.Panicln(err)
		}
		v.Title = " < Windows Event Logs (0) > "
		go func() {
			app.loadWinEvents()
		}()
	} else {
		app.loadServices(app.selectUnits)
	}

	if v, err := g.View("varLogs"); err == nil {
		_, viewHeight := v.Size()
		app.maxVisibleFiles = viewHeight
	}

	if app.getOS == "windows" {
		selectedVarLog, err := g.View("varLogs")
		if err != nil {
			log.Panicln(err)
		}
		g.Update(func(g *gocui.Gui) error {
			selectedVarLog.Clear()
			fmt.Fprintln(selectedVarLog, "Searching log files...")
			selectedVarLog.Highlight = false
			return nil
		})
		selectedVarLog.Title = " < Program Files (0) > "
		app.selectPath = "ProgramFiles"
		go func() {
			app.loadWinFiles(app.selectPath)
		}()
	} else {
		app.loadFiles(app.selectPath)
	}

	if v, err := g.View("docker"); err == nil {
		_, viewHeight := v.Size()
		app.maxVisibleDockerContainers = viewHeight
	}
	app.loadDockerContainer(app.selectContainerizationSystem)

	if _, err := g.SetCurrentView("filterList"); err != nil {
		return
	}

	go func() {
		app.updateLogOutput(3)
	}()

	go func() {
		app.updateWindowSize(1)
	}()

	// Отображение GUI в режиме OutputNormal
	go g.MainLoop()

	time.Sleep(3 * time.Second)

	// Проверяем покраску
	app.currentLogLines = []string{
		"1  line: http://localhost:8443",
		"2  line: https://github.com/Lifailon/lazyjournal",
		"3  line: /etc/ssh/sshd_config",
		"4  line: root runner",
		"5  line: warning",
		"6  line: stderr: disconnected and crashed",
		"7  line: kernel: deletion removed stopped invalidated aborted blocked deactivated",
		"8  line: rsyslogd: exited critical failed rejection fataling closed ended dropped killing",
		"9  line: sudo: cancelation unavailable unsuccessful found denied conflict false none",
		"10 line: /dev/null",
		"11 line: null success complete accept connection finish started created enable allowing posted",
		"12 line: routing forward passed running added opened patching ok available accessible done true",
		"13 line: stdout input GET SET head request upload listen launch change clear skip missing mount",
		"14 line: authorization configuration option writing saving boot paused filter normal notice alert",
		"15 line: information update shutdown status debug verbose trace protocol level",
		"16 line: 2025-02-26T21:38:35.956968+03:00 ⎯⎯⎯ 0x04 ⎯⎯⎯",
		"17 line: 25.02.2025 11:11 11:11:11:11:11:11 11-11-11-11-11-11",
		"18 line: TCP UDP ICMP IP 192.168.1.1:8443",
		"19 line: 25.02.2025 01:14:42 [INFO]: not data",
		"20 line: cron[123]: running",
	}
	app.updateDelimiter(true)
	app.applyFilter(true)
	time.Sleep(3 * time.Second)
	t.Log("Test coloring - passed")

	// Проверяем фильтрацию текста для списков
	app.filterListText = "a"
	app.createFilterEditor("lists")
	time.Sleep(1 * time.Second)
	app.filterListText = ""
	app.applyFilterList()
	time.Sleep(1 * time.Second)
	t.Log("Test filter list - passed")

	// TAB journal
	app.nextView(g, nil)
	time.Sleep(1 * time.Second)
	if v, err := g.View("services"); err == nil {
		// DOWN
		app.nextService(v, 100)
		time.Sleep(1 * time.Second)
		// Загружаем журнал
		app.selectService(g, v)
		time.Sleep(3 * time.Second)
		// UP
		app.prevService(v, 100)
		time.Sleep(1 * time.Second)
		// Переключаем списки
		if runtime.GOOS != "windows" {
			// Right
			app.setUnitListRight(g, v)
			time.Sleep(3 * time.Second)
			app.setUnitListRight(g, v)
			time.Sleep(3 * time.Second)
			app.setUnitListRight(g, v)
			time.Sleep(3 * time.Second)
			app.setUnitListRight(g, v)
			time.Sleep(3 * time.Second)
			// Left
			app.setUnitListLeft(g, v)
			time.Sleep(3 * time.Second)
			app.setUnitListLeft(g, v)
			time.Sleep(3 * time.Second)
			app.setUnitListLeft(g, v)
			time.Sleep(3 * time.Second)
			app.setUnitListLeft(g, v)
			time.Sleep(3 * time.Second)
		}
	}
	t.Log("Test journals - passed")

	// TAB filesystem
	app.nextView(g, nil)
	time.Sleep(1 * time.Second)
	if v, err := g.View("varLogs"); err == nil {
		app.nextFileName(v, 100)
		time.Sleep(1 * time.Second)
		app.selectFile(g, v)
		time.Sleep(3 * time.Second)
		app.prevFileName(v, 100)
		time.Sleep(1 * time.Second)
		if runtime.GOOS != "windows" {
			app.setLogFilesListRight(g, v)
			time.Sleep(3 * time.Second)
			app.setLogFilesListRight(g, v)
			time.Sleep(3 * time.Second)
			app.setLogFilesListRight(g, v)
			time.Sleep(3 * time.Second)
			app.setLogFilesListRight(g, v)
			time.Sleep(3 * time.Second)
			app.setLogFilesListLeft(g, v)
			time.Sleep(3 * time.Second)
			app.setLogFilesListLeft(g, v)
			time.Sleep(3 * time.Second)
			app.setLogFilesListLeft(g, v)
			time.Sleep(3 * time.Second)
			app.setLogFilesListLeft(g, v)
			time.Sleep(3 * time.Second)
		}
	}
	t.Log("Test filesystem - passed")

	// TAB docker
	app.nextView(g, nil)
	time.Sleep(1 * time.Second)
	if v, err := g.View("docker"); err == nil {
		app.nextDockerContainer(v, 100)
		time.Sleep(1 * time.Second)
		app.prevDockerContainer(v, 100)
		time.Sleep(1 * time.Second)
		if runtime.GOOS != "windows" {
			app.setContainersListRight(g, v)
			time.Sleep(1 * time.Second)
			app.setContainersListRight(g, v)
			time.Sleep(1 * time.Second)
			app.setContainersListRight(g, v)
			time.Sleep(1 * time.Second)
			app.setContainersListLeft(g, v)
			time.Sleep(1 * time.Second)
			app.setContainersListLeft(g, v)
			time.Sleep(1 * time.Second)
			app.setContainersListLeft(g, v)
		}
		time.Sleep(1 * time.Second)
		app.selectDocker(g, v)
		time.Sleep(3 * time.Second)
	}
	t.Log("Test containers - passed")

	// TAB filter logs
	app.nextView(g, nil)

	// Проверяем фильтрацию текста для вывода журнала
	app.filterText = "a"
	app.applyFilter(true)
	time.Sleep(3 * time.Second)
	// Ctrl+W
	app.clearFilterEditor(g)
	app.applyFilter(true)
	time.Sleep(3 * time.Second)
	t.Log("Test filter logs - passed")

	// Проверяем режимы фильтрации
	if v, err := g.View("filter"); err == nil {
		// fuzzy
		app.setFilterModeRight(g, v)
		time.Sleep(1 * time.Second)
		// regex
		app.setFilterModeRight(g, v)
		time.Sleep(1 * time.Second)
		// default
		app.setFilterModeRight(g, v)
		time.Sleep(1 * time.Second)
		// regex
		app.setFilterModeLeft(g, v)
		time.Sleep(1 * time.Second)
		// fuzzy
		app.setFilterModeLeft(g, v)
		time.Sleep(1 * time.Second)
		// default
		app.setFilterModeLeft(g, v)
		time.Sleep(1 * time.Second)
	}
	t.Log("Test filter modes - passed")

	// TAB logs output
	app.nextView(g, nil)
	time.Sleep(1 * time.Second)
	if v, err := g.View("logs"); err == nil {
		// Right tail count +
		app.setCountLogViewUp(g, v)
		time.Sleep(1 * time.Second)
		app.setCountLogViewUp(g, v)
		time.Sleep(1 * time.Second)
		// Left tail count -
		app.setCountLogViewDown(g, v)
		time.Sleep(1 * time.Second)
		app.setCountLogViewDown(g, v)
		time.Sleep(1 * time.Second)
		app.setCountLogViewDown(g, v)
		time.Sleep(1 * time.Second)
		app.setCountLogViewDown(g, v)
		time.Sleep(1 * time.Second)
		app.setCountLogViewDown(g, v)
		time.Sleep(1 * time.Second)
		app.setCountLogViewDown(g, v)
		time.Sleep(1 * time.Second)
		// Right
		app.setCountLogViewUp(g, v)
		time.Sleep(1 * time.Second)
		app.setCountLogViewUp(g, v)
		time.Sleep(1 * time.Second)
		app.setCountLogViewUp(g, v)
		time.Sleep(1 * time.Second)
		app.setCountLogViewDown(g, v)
		time.Sleep(1 * time.Second)
		// UP output
		app.scrollUpLogs(1)
		time.Sleep(1 * time.Second)
		// DOWN output
		app.scrollDownLogs(1)
		time.Sleep(1 * time.Second)
		// Ctrl+A
		app.pageUpLogs()
		time.Sleep(1 * time.Second)
		// Ctrl+E
		app.updateLogsView(true)
		time.Sleep(1 * time.Second)
	}
	t.Log("Test log output - passed")

	// TAB filter list
	app.nextView(g, nil)
	time.Sleep(1 * time.Second)

	// Shift+TAB
	app.backView(g, nil)
	time.Sleep(1 * time.Second)
	app.backView(g, nil)
	time.Sleep(1 * time.Second)
	app.backView(g, nil)
	time.Sleep(1 * time.Second)
	app.backView(g, nil)
	time.Sleep(1 * time.Second)
	app.backView(g, nil)
	time.Sleep(1 * time.Second)
	app.backView(g, nil)
	time.Sleep(1 * time.Second)
	app.backView(g, nil)
	time.Sleep(1 * time.Second)

	quit(g, nil)
}
