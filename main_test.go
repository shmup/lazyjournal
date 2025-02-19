package main

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestWinFiles(t *testing.T) {
	// Определяем тестовые пути
	testCases := []struct {
		name       string
		selectPath string
	}{
		// {"Program Files", "ProgramFiles"},
		{"Program Files 86", "ProgramFiles86"},
		{"ProgramData", "ProgramData"},
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
				t.Errorf("The file list is empty")
			} else {
				t.Log("Main path:", app.selectPath, "(", len(app.logfiles), "count log files )")
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
				t.Log("Path:", app.lastLogPath, "[ lines:", len(app.currentLogLines), "& time:", endTime, "]")
			}
		})
	}
}
