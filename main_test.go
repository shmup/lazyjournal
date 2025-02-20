package main

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestWinFiles(t *testing.T) {
	testCases := []struct {
		name       string
		selectPath string
	}{
		// {"Program Files", "ProgramFiles"},
		{"Program Files 86", "ProgramFiles86"},
		// {"ProgramData", "ProgramData"},
		// {"AppData/Local", "AppDataLocal"},
		// {"AppData/Roaming", "AppDataRoaming"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := &App{
				testMode:     true,
				systemDisk:   "C",
				userName:     "lifailon",
				getOS:        "windows",
				logViewCount: "100000",
				selectPath:   tc.selectPath,
			}

			// (1) Заполняем массив из названий файлов и путей к ним
			app.loadWinFiles(app.selectPath)
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
				t.Log("Path:", app.lastLogPath, ">>> LINE:\x1b[0;33m", len(app.currentLogLines), "\x1b[0;0m& TIME:\x1b[0;33m", endTime, "\x1b[0;0m")
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
		// {"Kubernetes", "kubernetes"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := &App{
				testMode:                     true,
				selectContainerizationSystem: tc.selectContainerizationSystem,
				logViewCount:                 "100000",
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
				t.Log("Container:", dockerContainer.name, ">>> LINE:\x1b[0;33m", len(app.currentLogLines), "\x1b[0;0m& TIME:\x1b[0;33m", endTime, "\x1b[0;0m")
			}
		})
	}
}
