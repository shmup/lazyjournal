package main

import (
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
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
				// (2) Читаем журнал, выводим путь, количество строк в массиве (прочитанных из файла) и время чтения
				app.loadFileLogs(strings.TrimSpace(logFileName), true)
				endTime := time.Since(startTime)
				t.Log("[READ]  Path:", app.lastLogPath, "--- LINE:\x1b[0;33m", len(app.currentLogLines), "\x1b[0;0m--- TIME:\x1b[0;33m", endTime, "\x1b[0;0m")

				startTime = time.Now()
				app.applyFilter(true)
				endTime = time.Since(startTime)
				t.Log("[COLOR] Path:", app.lastLogPath, "--- LINE:\x1b[0;33m", len(app.filteredLogLines), "\x1b[0;0m--- TIME:\x1b[0;33m", endTime, "\x1b[0;0m")
			}
		})
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
		{"Var Log", "/var/log/"},
		{"Home", "/home/"},
		{"Descriptors", "descriptor"},
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
				t.Log("[READ]  Path:", app.lastLogPath, "--- LINE:\x1b[0;33m", len(app.currentLogLines), "\x1b[0;0m--- TIME:\x1b[0;33m", endTime, "\x1b[0;0m")

				startTime = time.Now()
				app.applyFilter(true)
				endTime = time.Since(startTime)
				t.Log("[COLOR] Path:", app.lastLogPath, "--- LINE:\x1b[0;33m", len(app.filteredLogLines), "\x1b[0;0m--- TIME:\x1b[0;33m", endTime, "\x1b[0;0m")
			}
		})
	}
}

func TestJournal(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skip Linux test")
	}
	testCases := []struct {
		name        string
		journalName string
	}{
		{"Services", "services"},
		{"Units", "UNIT"},
		{"User units", "USER_UNIT"},
		{"Kernel", "kernel"},
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
				t.Log("[READ]  Journal:", serviceName, "--- LINE:\x1b[0;33m", len(app.currentLogLines), "\x1b[0;0m--- TIME:\x1b[0;33m", endTime, "\x1b[0;0m")

				startTime = time.Now()
				app.applyFilter(true)
				endTime = time.Since(startTime)
				t.Log("[COLOR] Journal:", serviceName, "--- LINE:\x1b[0;33m", len(app.filteredLogLines), "\x1b[0;0m--- TIME:\x1b[0;33m", endTime, "\x1b[0;0m")
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
				t.Log("[READ]  Container:", dockerContainer.name, "--- LINE:\x1b[0;33m", len(app.currentLogLines), "\x1b[0;0m--- TIME:\x1b[0;33m", endTime, "\x1b[0;0m")

				startTime = time.Now()
				app.applyFilter(true)
				endTime = time.Since(startTime)
				t.Log("[COLOR] Container:", dockerContainer.name, "--- LINE:\x1b[0;33m", len(app.filteredLogLines), "\x1b[0;0m--- TIME:\x1b[0;33m", endTime, "\x1b[0;0m")
			}
		})
	}
}

func TestFilterColor(t *testing.T) {
	testCases := []struct {
		name             string
		selectFilterMode string
	}{
		{"Default", "default"},
		{"Fuzzy", "fuzzy"},
		{"Regex", "regex"},
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
