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

func TestWinFiles(t *testing.T) {
	// Пропускаем тест целиком для Linux/macOS/bsd
	if runtime.GOOS != "windows" {
		t.Skip("Skip Windows test")
	}
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
				// Выводим путь, количество строк в массиве (прочитанных из файла), время чтения и фильтрации+покраски
				t.Log("Path:", app.lastLogPath, "--- LINE:\x1b[0;34m", len(app.currentLogLines), "\x1b[0;0m--- READ:\x1b[0;32m", endTime, "\x1b[0;0m--- COLOR:\x1b[0;33m", endTime2, "\x1b[0;0m")
			}
		})
	}
}

func TestWinEvents(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skip Windows test")
	}
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

		t.Log("Event:", serviceName, "--- LINE:\x1b[0;34m", len(app.currentLogLines), "\x1b[0;0m--- READ:\x1b[0;32m", endTime, "\x1b[0;0m--- COLOR:\x1b[0;33m", endTime2, "\x1b[0;0m")
	}
}

func TestUnixFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skip Linux test")
	}
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

				t.Log("Path:", app.lastLogPath, "--- LINE:\x1b[0;34m", len(app.currentLogLines), "\x1b[0;0m--- READ:\x1b[0;32m", endTime, "\x1b[0;0m--- COLOR:\x1b[0;33m", endTime2, "\x1b[0;0m")
			}
		})
	}
}

func TestLinuxJournal(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skip Linux test")
	}
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

				t.Log("Journal:", serviceName, "--- LINE:\x1b[0;34m", len(app.currentLogLines), "\x1b[0;0m--- READ:\x1b[0;32m", endTime, "\x1b[0;0m--- COLOR:\x1b[0;33m", endTime2, "\x1b[0;0m")
			}
		})
	}
}

func TestDockerContainer(t *testing.T) {
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

				t.Log("Container:", dockerContainer.name, "--- LINE:\x1b[0;34m", len(app.currentLogLines), "\x1b[0;0m--- READ:\x1b[0;32m", endTime, "\x1b[0;0m--- COLOR:\x1b[0;33m", endTime2, "\x1b[0;0m")
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
				"1  line: stdout true",
				"2  line: stderr false",
				"3  line: warning",
				"4  line: POST request",
				"5  line: http://localhost:8443",
				"6  line: https://github.com/Lifailon/lazyjournal",
				"7  line: 0x04",
				"8  line: 11:11:11:11:11:11 11-11-11-11-11-11",
				"9  line: TCP UDP 192.168.1.1:8443",
				"10 line: stdout 25.02.2025 01:14:42: [INFO]: not data",
				"11 line: cron[123]: running",
				"12 line: root: /etc/ssh/sshd_config",
			}

			app.applyFilter(true)
			t.Log("Lines: ", len(app.filteredLogLines))
			for _, line := range app.filteredLogLines {
				t.Log(line)
			}
		})
	}
}

func TestInterface(t *testing.T) {
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

	g, _ = gocui.NewGui(gocui.OutputNormal, false)
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

	// Включить отображение GUI
	// go g.MainLoop()

	time.Sleep(3 * time.Second)

	// Проверяем фильтрацию текста для списков
	app.filterListText = "a"
	app.applyFilterList()
	time.Sleep(1 * time.Second)
	app.filterListText = ""
	app.applyFilterList()
	time.Sleep(1 * time.Second)

	// TAB journal
	app.nextView(g, nil)
	time.Sleep(1 * time.Second)
	if v, err := g.View("services"); err == nil {
		// DOWN
		app.nextService(v, 100)
		time.Sleep(1 * time.Second)
		// UP
		app.prevService(v, 100)
	}

	// TAB filesystem
	app.nextView(g, nil)
	time.Sleep(1 * time.Second)
	if v, err := g.View("varLogs"); err == nil {
		app.nextFileName(v, 100)
		time.Sleep(1 * time.Second)
		app.prevFileName(v, 100)
	}

	// TAB docker
	app.nextView(g, nil)
	time.Sleep(1 * time.Second)
	if v, err := g.View("docker"); err == nil {
		app.nextDockerContainer(v, 100)
		time.Sleep(1 * time.Second)
		app.prevDockerContainer(v, 100)
		// Загружаем журнал
		app.selectDocker(g, v)
		time.Sleep(1 * time.Second)
	}

	// TAB filter
	app.nextView(g, nil)

	// Проверяем фильтрацию текста для вывода журнала
	app.filterText = "a"
	app.applyFilter(true)
	time.Sleep(1 * time.Second)

	// Проверяем режимы фильтрации
	time.Sleep(1 * time.Second)
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

	app.filterText = ""
	app.applyFilter(true)
	time.Sleep(1 * time.Second)

	// TAB logs output
	app.nextView(g, nil)
	time.Sleep(1 * time.Second)
	if v, err := g.View("logs"); err == nil {
		app.setCountLogViewDown(g, v)
		time.Sleep(1 * time.Second)
		app.setCountLogViewUp(g, v)

		// UP
		app.scrollUpLogs(1)
		time.Sleep(1 * time.Second)
		// DOWN
		app.scrollDownLogs(1)
		time.Sleep(1 * time.Second)

		// Ctrl+A
		app.pageUpLogs()
		time.Sleep(1 * time.Second)
		// Ctrl+E
		app.updateLogsView(true)
		time.Sleep(1 * time.Second)
	}

	// Shift+TAB
	app.backView(g, nil)
	time.Sleep(1 * time.Second)

	quit(g, nil)
}
