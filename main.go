package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/awesome-gocui/gocui"
)

// Структура хранения информации о журналах
type Journal struct {
	name    string // название журнала (имя службы) или дата загрузки
	boot_id string // id загрузки системы
}

type Logfile struct {
	name string
	path string
}

type DockerContainers struct {
	name string
	id   string
}

// Структура основного приложения (графический интерфейс и данные журналов)
type App struct {
	gui *gocui.Gui // графический интерфейс (gocui)

	selectUnits                  string // название журнала (UNIT/USER_UNIT)
	selectPath                   string // путь к логам (/var/log/)
	selectContainerizationSystem string // название системы контейнеризации (docker/podman)
	selectFilterMode             string // режим фильтрации (default/fuzzy/regex)
	logViewCount                 string // количество логов для просмотра (5000)

	journals           []Journal // список (массив/срез) журналов для отображения
	maxVisibleServices int       // максимальное количество видимых элементов в окне списка служб
	startServices      int       // индекс первого видимого элемента
	selectedJournal    int       // индекс выбранного журнала

	logfiles        []Logfile
	maxVisibleFiles int
	startFiles      int
	selectedFile    int

	dockerContainers           []DockerContainers
	maxVisibleDockerContainers int
	startDockerContainers      int
	selectedDockerContainer    int

	filterText       string   // текст для фильтрации записей журнала
	currentLogLines  []string // набор строк (срез) для хранения журнала без фильтрации
	filteredLogLines []string // набор строк (срез) для хранения журнала после фильтра
	logScrollPos     int      // позиция прокрутки для отображаемых строк журнала

	lastWindow   string // фиксируем последний используемый источник для вывода логов
	lastSelected string // фиксируем название последнего выбранного журнала или контейнера
}

func main() {
	// Инициализация значений по умолчанию
	app := &App{
		startServices:                0, // начальная позиция списка юнитов
		selectedJournal:              0, // начальный индекс выбранного журнала
		startFiles:                   0,
		selectedFile:                 0,
		startDockerContainers:        0,
		selectedDockerContainer:      0,
		selectUnits:                  "UNIT", // "USER_UNIT"
		selectPath:                   "/var/log/",
		selectContainerizationSystem: "docker",
		selectFilterMode:             "default", // "fuzzy" || "regex"
		logViewCount:                 "5000",
	}

	// Создаем GUI
	g, err := gocui.NewGui(gocui.OutputNormal, true) // 2-й параметр для форка
	if err != nil {
		log.Panicln(err)
	}
	// Закрываем GUI после завершения
	defer g.Close()

	app.gui = g
	// Функция, которая будет вызываться при обновлении интерфейса
	g.SetManagerFunc(app.layout)
	// Включить поддержку мыши
	g.Mouse = false

	// Цветовая схема GUI
	g.FgColor = gocui.ColorDefault // поля всех окон и цвет текста
	g.BgColor = gocui.ColorDefault // фон

	// Привязка клавиш для работы с интерфейсом из функции setupKeybindings()
	if err := app.setupKeybindings(); err != nil {
		log.Panicln(err)
	}

	// Выполняем layout для инициализации интерфейса
	if err := app.layout(g); err != nil {
		log.Panicln(err)
	}

	// Фиксируем текущее количество видимых строк в терминале (-1 заголовок)
	if v, err := g.View("services"); err == nil {
		_, viewHeight := v.Size()
		app.maxVisibleServices = viewHeight - 1
	}
	// Загрузка списков журналов
	app.loadServices(app.selectUnits)

	// /var/logs
	if v, err := g.View("varLogs"); err == nil {
		_, viewHeight := v.Size()
		app.maxVisibleFiles = viewHeight - 1
	}
	app.loadFiles(app.selectPath)

	// Docker
	if v, err := g.View("docker"); err == nil {
		_, viewHeight := v.Size()
		app.maxVisibleDockerContainers = viewHeight - 1
	}
	app.loadDockerContainer(app.selectContainerizationSystem)

	// Устанавливаем фокус на окно с журналами по умолчанию
	g.SetCurrentView("services")

	// Горутина для автоматического обновления вывода журнала
	// go app.updateLogOutput(2)

	// Запус GUI
	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

// Структура интерфейса окон GUI
func (app *App) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()  // Получаем текущий размер интерфейса терминала (ширина, высота)
	panelHeight := maxY / 3 // Высота каждой панели
	// Окно для отображения списка доступных журналов (UNIT)
	// Размеры окна: заголовок, отступ слева, отступ сверху, ширина, высота, 5-й параметр из форка для продолжение окна (2)
	if v, err := g.SetView("services", 0, 0, maxX/4-1, panelHeight-1, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = " < System units > " // заголовок окна
		v.Highlight = true             // выделение активного элемента
		// Цветовая схема из форка awesome-gocui/gocui
		v.FrameColor = gocui.ColorGreen // Цвет границ окна
		v.TitleColor = gocui.ColorGreen // Цвет заголовка
		v.SelBgColor = gocui.ColorGreen // Цвет фона при выборе в списке
		v.SelFgColor = gocui.ColorBlack // Цвет текста
		// v.BgColor = gocui.ColorRed      // Цвет текста
		// v.FgColor = gocui.ColorYellow   // Цвет фона внутри окна
		v.Wrap = false           // отключаем перенос строк
		v.Autoscroll = true      // включаем автопрокрутку
		app.updateServicesList() // выводим список журналов в это окно
	}

	// Окно для списка из /var/log
	if v, err := g.SetView("varLogs", 0, panelHeight, maxX/4-1, 2*panelHeight-1, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = " < Var logs > "
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		v.Wrap = false
		v.Autoscroll = true
		app.updateLogsList()
	}

	// Окно для списка контейнеров Docker
	if v, err := g.SetView("docker", 0, 2*panelHeight, maxX/4-1, maxY-1, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = " < Docker containers > "
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		v.Wrap = false
		v.Autoscroll = true
	}

	// Окно ввода текста для фильтрации
	if v, err := g.SetView("filter", maxX/4+1, 0, maxX-1, 2, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Filter [Default]"
		v.Editable = true                   // включить окно редактируемым для ввода текста
		v.Editor = app.createFilterEditor() // редактор для обработки ввода
		v.Wrap = true
	}

	// Окно для вывода записей выбранного журнала
	if v, err := g.SetView("logs", maxX/4+1, 3, maxX-1, maxY-1, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Logs"
		v.Wrap = true
		v.Autoscroll = false
	}

	// Включение курсора в режиме фильтра и отключение в остальных окнах
	currentView := g.CurrentView()
	if currentView != nil && currentView.Name() == "filter" {
		g.Cursor = true
	} else {
		g.Cursor = false
	}

	return nil
}

// ---------------------------------------- journalctl ----------------------------------------

// Функция для загрузки списка журналов служб или загрузок системы из journalctl
func (app *App) loadServices(journalName string) {
	if journalName == "kernel" {
		// Получаем список загрузок с помощью journalctl
		bootCmd := exec.Command("journalctl", "--list-boots", "-o", "json")
		bootOutput, err := bootCmd.Output()
		if err != nil {
			log.Printf("Error getting boot information: %v", err)
			return
		}
		// Структура для парсинга JSON
		type BootInfo struct {
			BootID     string `json:"boot_id"`
			FirstEntry int64  `json:"first_entry"`
			LastEntry  int64  `json:"last_entry"`
		}
		var bootRecords []BootInfo
		err = json.Unmarshal(bootOutput, &bootRecords)
		if err != nil {
			log.Printf("Error parsing JSON: %v", err)
			return
		}
		// Добавляем информацию о загрузках в app.journals
		for _, bootRecord := range bootRecords {
			// Преобразуем наносекунды в секунды
			firstEntryTime := time.Unix(bootRecord.FirstEntry/1000000, bootRecord.FirstEntry%1000000)
			lastEntryTime := time.Unix(bootRecord.LastEntry/1000000, bootRecord.LastEntry%1000000)
			// Форматируем строку в формате "DD.MM.YYYY HH:MM:SS"
			const dateFormat = "02.01.2006 15:04:05"
			name := fmt.Sprintf("%s - %s", firstEntryTime.Format(dateFormat), lastEntryTime.Format(dateFormat))
			// Добавляем в массив
			app.journals = append(app.journals, Journal{
				name:    name,
				boot_id: bootRecord.BootID,
			})
			// Сортируем по второй дате
			sort.Slice(app.journals, func(i, j int) bool {
				// Разделяем строки на части (до и после дефиса)
				dateFormat := "02.01.2006 15:04:05"
				// Получаем вторую дату (после дефиса) и парсим её
				endDate1, _ := time.Parse(dateFormat, app.journals[i].name[22:])
				endDate2, _ := time.Parse(dateFormat, app.journals[j].name[22:])
				// Сравниваем по второй дате в обратном порядке
				return endDate1.After(endDate2) // Используем After для сортировки по убыванию
			})
		}
	} else {
		cmd := exec.Command("journalctl", "--no-pager", "-F", journalName)
		output, err := cmd.Output()
		if err != nil {
			log.Printf("Error getting services: %v", err)
			return
		}
		// Создаем массив (хеш-таблица с доступом по ключу) для уникальных имен служб
		serviceMap := make(map[string]bool)
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			serviceName := strings.TrimSpace(scanner.Text())
			if serviceName != "" && !serviceMap[serviceName] {
				serviceMap[serviceName] = true
				app.journals = append(app.journals, Journal{
					name:    serviceName,
					boot_id: "",
				})
			}
		}
		// Сортируем список служб по алфавиту
		sort.Slice(app.journals, func(i, j int) bool {
			return app.journals[i].name < app.journals[j].name
		})
	}
	// Обновляем список служб в интерфейсе
	app.updateServicesList()
}

// Функция для обновления окна со списком служб
func (app *App) updateServicesList() {
	// Выбираем окно для заполнения в зависимости от используемого журнала
	v, err := app.gui.View("services")
	if err != nil {
		return
	}
	// Очищаем окно
	v.Clear()
	// Вычисляем конечную позицию видимой области (стартовая позиция + максимальное количество видимых строк)
	visibleEnd := app.startServices + app.maxVisibleServices
	if visibleEnd > len(app.journals) {
		visibleEnd = len(app.journals)
	}
	// Отображаем только элементы в пределах видимой области
	for i := app.startServices; i < visibleEnd; i++ {
		fmt.Fprintln(v, app.journals[i].name)
	}
}

// Функция для перемещения по списку журналов вниз
func (app *App) nextService(v *gocui.View, step int) error {
	// Обновляем текущее количество видимых строк в терминале (-1 заголовок)
	_, viewHeight := v.Size()
	app.maxVisibleServices = viewHeight - 1
	// Если список журналов пустой, ничего не делаем
	if len(app.journals) == 0 {
		return nil
	}
	// Переходим к следующему, если текущий выбранный журнал не последний
	if app.selectedJournal < len(app.journals)-1 {
		// Увеличиваем индекс выбранного журнала
		app.selectedJournal += step
		// Проверяем, чтобы не выйти за пределы списка
		if app.selectedJournal >= len(app.journals) {
			app.selectedJournal = len(app.journals) - 1
		}
		// Проверяем, вышли ли за пределы видимой области (увеличиваем стартовую позицию видимости, только если дошли до 0 + maxVisibleServices)
		if app.selectedJournal >= app.startServices+app.maxVisibleServices {
			// Сдвигаем видимую область вниз
			app.startServices += step
			// Проверяем, чтобы не выйти за пределы списка
			if app.startServices > len(app.journals)-app.maxVisibleServices {
				app.startServices = len(app.journals) - app.maxVisibleServices
			}
			// Обновляем отображение списка служб
			app.updateServicesList()
		}
		// Если сдвинули видимую область, корректируем индекс для смещения курсора в интерфейсе
		if app.selectedJournal < app.startServices+app.maxVisibleServices {
			// Выбираем журнал по скорректированному индексу
			return app.selectServiceByIndex(app.selectedJournal - app.startServices)
		}
	}
	return nil
}

// Функция для перемещения по списку журналов вверх
func (app *App) prevService(v *gocui.View, step int) error {
	_, viewHeight := v.Size()
	app.maxVisibleServices = viewHeight - 1
	if len(app.journals) == 0 {
		return nil
	}
	// Переходим к предыдущему, если текущий выбранный журнал не первый
	if app.selectedJournal > 0 {
		app.selectedJournal -= step
		// Если ушли в минус (за начало журнала), приводим к нулю
		if app.selectedJournal < 0 {
			app.selectedJournal = 0
		}
		// Проверяем, вышли ли за пределы видимой области
		if app.selectedJournal < app.startServices {
			app.startServices -= step
			if app.startServices < 0 {
				app.startServices = 0
			}
			app.updateServicesList()
		}
		if app.selectedJournal >= app.startServices {
			return app.selectServiceByIndex(app.selectedJournal - app.startServices)
		}
	}
	return nil
}

// Функция для визуального выбора журнала по индексу (смещение курсора выделения)
func (app *App) selectServiceByIndex(index int) error {
	// Получаем доступ к представлению списка служб
	v, err := app.gui.View("services")
	if err != nil {
		return err
	}
	// Устанавливаем курсор на нужный индекс (строку)
	v.SetCursor(0, index) // первый столбец (0), индекс строки
	return nil
}

// Функция для выбора журнала в списке сервисов по нажатию Enter
func (app *App) selectService(g *gocui.Gui, v *gocui.View) error {
	// Проверка, что есть доступ к представлению и список журналов не пустой
	if v == nil || len(app.journals) == 0 {
		return nil
	}
	// Получаем текущую позицию курсора
	_, cy := v.Cursor()
	// Читаем строку, на которой находится курсор
	line, err := v.Line(cy)
	if err != nil {
		return err
	}
	// Загружаем журналы выбранной службы, обрезая пробелы в названии
	app.loadJournalLogs(strings.TrimSpace(line))
	// Фиксируем для ручного или автоматического обновления вывода журнала
	app.lastWindow = "services"
	app.lastSelected = strings.TrimSpace(line)
	return nil
}

// Функция для загрузки записей журнала выбранной службы через journalctl
func (app *App) loadJournalLogs(serviceName string) {
	var output []byte
	var err error
	if app.selectUnits == "kernel" {
		var boot_id string
		for _, journal := range app.journals {
			if journal.name == serviceName {
				boot_id = journal.boot_id
				break
			}
		}
		cmd := exec.Command("journalctl", "-k", "-b", boot_id, "--no-pager", "-n", app.logViewCount)
		output, err = cmd.Output()
		if err != nil {
			log.Printf("Error getting logs: %v", err)
			return
		}
	} else {
		cmd := exec.Command("journalctl", "-u", serviceName, "--no-pager", "-n", app.logViewCount)
		output, err = cmd.Output()
		if err != nil {
			log.Printf("Error getting logs: %v", err)
			return
		}
	}
	// Сохраняем строки журнала в массив
	app.currentLogLines = strings.Split(string(output), "\n")
	// Очищаем поле ввода для фильтрации
	app.filterText = ""
	// Применяем текущий фильтр к записям для обновления вывода
	app.applyFilter()
}

// ---------------------------------------- Var Logs ----------------------------------------

func (app *App) loadFiles(logPath string) {
	var output []byte
	var err error
	if logPath == "/var/log/" {
		cmd := exec.Command("find", logPath, "-type", "f", "-name", "*.log", "-o", "-name", "*.gz")
		output, err = cmd.Output()
		if err != nil {
			log.Printf("Error getting log files: %v", err)
			return
		}
		// Добавляем пути по умолчанию для /var/log
		logPaths := []string{
			"/var/log/syslog\n",
			"/var/log/dmesg\n",
			// Информация о входах и выходах пользователей, перезагрузках и остановках системы
			"/var/log/wtmp\n",
			// Информация о неудачных попытках входа в систему (например, неправильные пароли)
			"/var/log/btmp\n",
		}
		for _, path := range logPaths {
			output = append([]byte(path), output...)
		}
	} else {
		cmd := exec.Command("find", logPath, "-type", "f", "-name", "*.log")
		output, err = cmd.Output()
		if err != nil {
			log.Printf("Error getting log files: %v", err)
			return
		}
	}
	serviceMap := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		// Получаем строку полного пути
		logFullPath := scanner.Text()
		// Удаляем префикс пути и расширение файла в конце
		logName := strings.TrimPrefix(logFullPath, logPath)
		logName = strings.TrimSuffix(logName, ".log")
		logName = strings.TrimSuffix(logName, ".gz")
		logName = strings.ReplaceAll(logName, "/", " ")
		logName = strings.ReplaceAll(logName, ".log.", " ")
		if logPath == "/home/" {
			// Разбиваем строку на слова
			words := strings.Fields(logName)
			// Берем первое и последнее слово
			firstWord := words[0]
			lastWord := words[len(words)-1]
			logName = firstWord + ": " + lastWord
		}
		// Получаем дату изменения файла
		// cmd := exec.Command("bash", "-c", "stat --format='%y' /var/log/apache2/access.log | awk '{print $1}' | awk -F- '{print $3\".\"$2\".\"$1}'")
		// Получаем информацию о файле
		fileInfo, err := os.Stat(logFullPath)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		// Получаем дату изменения
		modTime := fileInfo.ModTime()
		// Форматирование даты в формат DD.MM.YYYY
		formattedDate := modTime.Format("02.01.2006")
		// Проверяем, если такого имени ещё нет
		if logName != "" && !serviceMap[logName] {
			serviceMap[logName] = true
			// Добавляем в список
			app.logfiles = append(app.logfiles, Logfile{
				name: "[" + formattedDate + "] " + logName,
				path: logFullPath,
			})
		}
	}
	// Сортируем по дате
	sort.Slice(app.logfiles, func(i, j int) bool {
		// Извлечение дат из имени
		layout := "02.01.2006"
		dateI, _ := time.Parse(layout, extractDate(app.logfiles[i].name))
		dateJ, _ := time.Parse(layout, extractDate(app.logfiles[j].name))
		// return dateI.Before(dateJ)
		// Сортировка в обратном порядке
		return dateI.After(dateJ)
	})
	// Обновляем отображение списка журналов
	app.updateLogsList()
}

// Функция для извлечения первой втречающейся даты в формате DD.MM.YYYY
func extractDate(name string) string {
	re := regexp.MustCompile(`\d{2}\.\d{2}\.\d{4}`)
	return re.FindString(name)
}

func (app *App) updateLogsList() {
	v, err := app.gui.View("varLogs")
	if err != nil {
		return
	}
	v.Clear()
	visibleEnd := app.startFiles + app.maxVisibleFiles
	if visibleEnd > len(app.logfiles) {
		visibleEnd = len(app.logfiles)
	}
	for i := app.startFiles; i < visibleEnd; i++ {
		fmt.Fprintln(v, app.logfiles[i].name)
	}
}

func (app *App) nextFileName(v *gocui.View, step int) error {
	_, viewHeight := v.Size()
	app.maxVisibleFiles = viewHeight - 1
	if len(app.logfiles) == 0 {
		return nil
	}
	if app.selectedFile < len(app.logfiles)-1 {
		app.selectedFile += step
		if app.selectedFile >= len(app.logfiles) {
			app.selectedFile = len(app.logfiles) - 1
		}
		if app.selectedFile >= app.startFiles+app.maxVisibleFiles {
			app.startFiles += step
			if app.startFiles > len(app.logfiles)-app.maxVisibleFiles {
				app.startFiles = len(app.logfiles) - app.maxVisibleFiles
			}
			app.updateLogsList()
		}
		if app.selectedFile < app.startFiles+app.maxVisibleFiles {
			return app.selectFileByIndex(app.selectedFile - app.startFiles)
		}
	}
	return nil
}

func (app *App) prevFileName(v *gocui.View, step int) error {
	_, viewHeight := v.Size()
	app.maxVisibleFiles = viewHeight - 1
	if len(app.logfiles) == 0 {
		return nil
	}
	if app.selectedFile > 0 {
		app.selectedFile -= step
		if app.selectedFile < 0 {
			app.selectedFile = 0
		}
		if app.selectedFile < app.startFiles {
			app.startFiles -= step
			if app.startFiles < 0 {
				app.startFiles = 0
			}
			app.updateLogsList()
		}
		if app.selectedFile >= app.startFiles {
			return app.selectFileByIndex(app.selectedFile - app.startFiles)
		}
	}
	return nil
}

func (app *App) selectFileByIndex(index int) error {
	v, err := app.gui.View("varLogs")
	if err != nil {
		return err
	}
	v.SetCursor(0, index)
	return nil
}

func (app *App) selectFile(g *gocui.Gui, v *gocui.View) error {
	if v == nil || len(app.logfiles) == 0 {
		return nil
	}
	_, cy := v.Cursor()
	line, err := v.Line(cy)
	if err != nil {
		return err
	}
	app.loadFileLogs(strings.TrimSpace(line))
	app.lastWindow = "varLogs"
	app.lastSelected = strings.TrimSpace(line)
	return nil
}

func (app *App) loadFileLogs(logName string) {
	// Парсим имя обратно
	// logName = strings.ReplaceAll(logName, " ", "/")
	// logFullPath := app.selectPath + logName + ".log"
	// Получаем путь из массива по имени
	var logFullPath string
	for _, logfile := range app.logfiles {
		if logfile.name == logName {
			logFullPath = logfile.path
		}
	}
	// Читаем архивные логи (decompress + stdout)
	// gzip -dc access.log.10.gz
	// zcat access.log.10.gz
	// gunzip -c access.log.10.gz
	if strings.HasSuffix(logFullPath, ".gz") {
		cmdGzip := exec.Command("gzip", "-dc", logFullPath)
		cmdTail := exec.Command("tail", "-n", app.logViewCount)
		pipe, err := cmdGzip.StdoutPipe()
		if err != nil {
			log.Fatalf("Error creating pipe: %v", err)
		}
		// Стандартный вывод gzip передаем в stdin tail
		cmdTail.Stdin = pipe
		out, err := cmdTail.StdoutPipe()
		if err != nil {
			log.Fatalf("Error creating stdout pipe for tail: %v", err)
		}
		// Запуск команд
		if err := cmdGzip.Start(); err != nil {
			log.Fatalf("Error starting gzip: %v", err)
		}
		if err := cmdTail.Start(); err != nil {
			log.Fatalf("Error starting tail: %v", err)
		}
		// Чтение вывода
		output, err := io.ReadAll(out)
		if err != nil {
			log.Fatalf("Error reading output from tail: %v", err)
		}
		// Ожидание завершения команд
		if err := cmdGzip.Wait(); err != nil {
			log.Fatalf("Error waiting for gzip: %v", err)
		}
		if err := cmdTail.Wait(); err != nil {
			log.Fatalf("Error waiting for tail: %v", err)
		}
		// Выводим содержимое
		app.currentLogLines = strings.Split(string(output), "\n")
		// Читаем бинарные файлы с помощью last/lastb
	} else if strings.HasSuffix(logFullPath, "wtmp") {
		cmd := exec.Command("last", "-f", logFullPath)
		output, err := cmd.Output()
		if err != nil {
			log.Printf("Error getting logs: %v", err)
			return
		}
		app.currentLogLines = strings.Split(string(output), "\n")
	} else if strings.HasSuffix(logFullPath, "btmp") {
		cmd := exec.Command("lastb", "-f", logFullPath)
		output, err := cmd.Output()
		if err != nil {
			log.Printf("Error getting logs: %v", err)
			return
		}
		app.currentLogLines = strings.Split(string(output), "\n")
	} else {
		cmd := exec.Command("tail", logFullPath, "-n", app.logViewCount)
		output, err := cmd.Output()
		if err != nil {
			log.Printf("Error getting logs: %v", err)
			return
		}
		app.currentLogLines = strings.Split(string(output), "\n")
	}
	app.filterText = ""
	app.applyFilter()
}

// ---------------------------------------- Docker ----------------------------------------

func (app *App) loadDockerContainer(ContainerizationSystem string) {
	cmd := exec.Command(ContainerizationSystem, "ps", "--format", "{{.ID}} {{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error getting docker containers: %v", err)
		return
	}
	serviceMap := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		idName := scanner.Text()
		parts := strings.Fields(idName)
		if idName != "" && !serviceMap[idName] {
			serviceMap[idName] = true
			app.dockerContainers = append(app.dockerContainers, DockerContainers{
				name: parts[1],
				id:   parts[0],
			})
		}
	}
	sort.Slice(app.dockerContainers, func(i, j int) bool {
		return app.dockerContainers[i].name < app.dockerContainers[j].name
	})
	app.updateDockerContainerList()
}

func (app *App) updateDockerContainerList() {
	v, err := app.gui.View("docker")
	if err != nil {
		return
	}
	v.Clear()
	visibleEnd := app.startDockerContainers + app.maxVisibleDockerContainers
	if visibleEnd > len(app.dockerContainers) {
		visibleEnd = len(app.dockerContainers)
	}
	for i := app.startDockerContainers; i < visibleEnd; i++ {
		fmt.Fprintln(v, app.dockerContainers[i].name)
	}
}

func (app *App) nextDockerContainer(v *gocui.View, step int) error {
	_, viewHeight := v.Size()
	app.maxVisibleDockerContainers = viewHeight - 1
	if len(app.dockerContainers) == 0 {
		return nil
	}
	if app.selectedDockerContainer < len(app.dockerContainers)-1 {
		app.selectedDockerContainer += step
		if app.selectedDockerContainer >= len(app.dockerContainers) {
			app.selectedDockerContainer = len(app.dockerContainers) - 1
		}
		if app.selectedDockerContainer >= app.startDockerContainers+app.maxVisibleDockerContainers {
			app.startDockerContainers += step
			if app.startDockerContainers > len(app.dockerContainers)-app.maxVisibleDockerContainers {
				app.startDockerContainers = len(app.dockerContainers) - app.maxVisibleDockerContainers
			}
			app.updateDockerContainerList()
		}
		if app.selectedDockerContainer < app.startDockerContainers+app.maxVisibleDockerContainers {
			return app.selectDockerByIndex(app.selectedDockerContainer - app.startDockerContainers)
		}
	}
	return nil
}

func (app *App) prevDockerContainer(v *gocui.View, step int) error {
	_, viewHeight := v.Size()
	app.maxVisibleDockerContainers = viewHeight - 1
	if len(app.dockerContainers) == 0 {
		return nil
	}
	if app.selectedDockerContainer > 0 {
		app.selectedDockerContainer -= step
		if app.selectedDockerContainer < 0 {
			app.selectedDockerContainer = 0
		}
		if app.selectedDockerContainer < app.startDockerContainers {
			app.startDockerContainers -= step
			if app.startDockerContainers < 0 {
				app.startDockerContainers = 0
			}
			app.updateDockerContainerList()
		}
		if app.selectedDockerContainer >= app.startDockerContainers {
			return app.selectDockerByIndex(app.selectedDockerContainer - app.startDockerContainers)
		}
	}
	return nil
}

func (app *App) selectDockerByIndex(index int) error {
	v, err := app.gui.View("docker")
	if err != nil {
		return err
	}
	v.SetCursor(0, index)
	return nil
}

func (app *App) selectDocker(g *gocui.Gui, v *gocui.View) error {
	if v == nil || len(app.dockerContainers) == 0 {
		return nil
	}
	_, cy := v.Cursor()
	line, err := v.Line(cy)
	if err != nil {
		return err
	}
	app.loadDockerLogs(strings.TrimSpace(line))
	app.lastWindow = "docker"
	app.lastSelected = strings.TrimSpace(line)
	return nil
}

func (app *App) loadDockerLogs(containerName string) {
	var ContainerizationSystem string = app.selectContainerizationSystem
	var containerId string
	for _, dockerContainer := range app.dockerContainers {
		if dockerContainer.name == containerName {
			containerId = dockerContainer.id
		}
	}
	cmd := exec.Command(ContainerizationSystem, "logs", "--tail", app.logViewCount, containerId)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error getting logs: %v", err)
		return
	}
	app.currentLogLines = strings.Split(string(output), "\n")
	app.filterText = ""
	app.applyFilter()
}

// ---------------------------------------- Filter/Logs ----------------------------------------

// Функция для фильтрации записей текущего журнала
func (app *App) applyFilter() {
	v, err := app.gui.View("filter")
	if err != nil {
		return
	}
	v.FrameColor = gocui.ColorGreen
	app.filteredLogLines = make([]string, 0)
	// Опускаем регистр ввода текста для фильтра
	filter := strings.ToLower(app.filterText)
	for _, line := range app.currentLogLines {
		// Fuzzy (неточный поиск без учета регистра)
		if app.selectFilterMode == "fuzzy" {
			// Разбиваем текст фильтра на массив из строк
			filterWords := strings.Fields(filter)
			// Опускаем регистр текущей строки цикла
			lineLower := strings.ToLower(line)
			var match bool = true
			// Проверяем, если строка не содержит хотя бы одно слово из фильтра, то пропускаем строку
			for _, word := range filterWords {
				if !strings.Contains(lineLower, word) {
					match = false
					break
				}
			}
			// Если строка подходит под фильтр, возвращаем её с покраской
			if match {
				// Временные символы для обозначения начала и конца покраски найденных символов
				startColor := "►"
				endColor := "◄"
				originalLine := line
				// Проходимся по всем словосочетаниям фильтра (массив через пробел) для позиционирования покраски
				for _, word := range filterWords {
					wordLower := strings.ToLower(word)
					start := 0
					// Ищем все вхождения слова в строке с учётом регистра
					for {
						// Находим индекс вхождения с учетом регистра
						idx := strings.Index(strings.ToLower(originalLine[start:]), wordLower)
						if idx == -1 {
							break // Если больше нет вхождений, выходим
						}
						start += idx // корректируем индекс с учетом текущей позиции
						// Вставляем временные символы для покраски
						originalLine = originalLine[:start] + startColor + originalLine[start:start+len(word)] + endColor + originalLine[start+len(word):]
						// Сдвигаем индекс для поиска в оставшейся части строки
						start += len(startColor) + len(word) + len(endColor)
					}
				}
				// Заменяем временные символы на ANSI escape-последовательности
				originalLine = strings.ReplaceAll(originalLine, startColor, "\x1b[0;33m")
				originalLine = strings.ReplaceAll(originalLine, endColor, "\033[0m")
				app.filteredLogLines = append(app.filteredLogLines, originalLine)
			}
			// Regex (с использованием регулярных выражений Go и без учета регистра по умолчанию)
		} else if app.selectFilterMode == "regex" {
			// Добавляем флаг для нечувствительности к регистру по умолчанию
			filter = "(?i)" + filter
			// Компилируем регулярное выражение
			regex, err := regexp.Compile(filter)
			if err != nil {
				// В случае синтаксической ошибки регулярного выражения, красим окно красным цветом и завершаем цикл
				v.FrameColor = gocui.ColorRed
				break
			}
			// Проверяем, что строка подходит под регулярное выражение
			if regex.MatchString(line) {
				originalLine := line
				// Находим все найденные совпадени
				matches := regex.FindAllString(originalLine, -1)
				// Красим только первое найденное совпадение
				originalLine = strings.ReplaceAll(originalLine, matches[0], "\x1b[0;33m"+matches[0]+"\033[0m")
				app.filteredLogLines = append(app.filteredLogLines, originalLine)
			}
			// Default (точный поиск с учетом регистра)
		} else {
			filter = app.filterText
			if filter == "" || strings.Contains(line, filter) {
				lineColor := strings.ReplaceAll(line, filter, "\x1b[0;33m"+filter+"\033[0m")
				app.filteredLogLines = append(app.filteredLogLines, lineColor)
			}
		}
	}
	app.logScrollPos = 0
	// Обновляем окно для отображения отфильтрованных записей
	app.updateLogsView(true)
}

// Функция для обновления вывода журнала (параметр для прокрутки в самый вниз)
func (app *App) updateLogsView(lowerDown bool) {
	// Получаем доступ к выводу журнала
	v, err := app.gui.View("logs")
	if err != nil {
		return
	}
	// Очищаем окно для отображения новых строк
	v.Clear()
	// Получаем ширину и высоту окна
	viewWidth, viewHeight := v.Size()
	// Опускаем в самый низ, только если это не ручной скролл (отключается параметром)
	if lowerDown {
		// Если количество строк больше высоты окна, опускаем в самый низ
		if len(app.filteredLogLines) > viewHeight-1 {
			app.logScrollPos = len(app.filteredLogLines) - viewHeight - 1
		} else {
			app.logScrollPos = 0
		}
	}
	// Определяем количество строк для отображения, начиная с позиции logScrollPos
	startLine := app.logScrollPos
	endLine := startLine + viewHeight
	if endLine > len(app.filteredLogLines) {
		endLine = len(app.filteredLogLines)
	}
	// Учитываем auto wrap
	var steps int
	if app.logScrollPos >= len(app.filteredLogLines)-viewHeight-1 {
		for i := startLine; i < endLine; i++ {
			// Получаем длинну видимых символов в строке
			var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)
			lengthLine := len([]rune(ansiEscape.ReplaceAllString(app.filteredLogLines[i], "")))
			// Если длинна строки больше ширины окна
			if lengthLine > viewWidth {
				// Добавляем количество дополнительных строк в позицию для сдвига вниз
				steps += lengthLine / viewWidth
			}
		}
	}
	// Проходим по отфильтрованным строкам и выводим их
	for i := startLine + steps; i < endLine; i++ {
		fmt.Fprintln(v, app.filteredLogLines[i])
	}
	// Вычисляем процент прокрутки и обновляем заголовок
	if len(app.filteredLogLines) > 0 {
		// Стартовая позиция + размер текущего вывода логов и округляем в большую сторону (math)
		percentage := int(math.Ceil(float64((startLine+viewHeight)*100) / float64(len(app.filteredLogLines))))
		if percentage > 100 {
			v.Title = fmt.Sprintf("Logs: 100%% (%d)", len(app.filteredLogLines))
		} else {
			v.Title = fmt.Sprintf("Logs: %d%% (%d/%d)", percentage, startLine+1+viewHeight, len(app.filteredLogLines))
		}
	} else {
		v.Title = "Logs: 0% (0)"
	}
}

// Функция для скроллинга вниз
func (app *App) scrollDownLogs(step int) error {
	v, err := app.gui.View("logs")
	if err != nil {
		return err
	}
	// Получаем высоту окна, что бы не опускать лог с пустыми строками
	_, viewHeight := v.Size()
	// Проверяем, что размер журнала больше размера окна
	if len(app.filteredLogLines) > viewHeight {
		// Увеличиваем позицию прокрутки
		app.logScrollPos += step
		// Если достигнут конец списка, останавливаем на максимальной длинне с учетом высоты окна
		if app.logScrollPos > len(app.filteredLogLines)-1-viewHeight {
			app.logScrollPos = len(app.filteredLogLines) - 1 - viewHeight
		}
		// Вызываем функцию для обновления отображения журнала
		app.updateLogsView(false)
	}
	return nil
}

// Функция для скроллинга вверх
func (app *App) scrollUpLogs(step int) error {
	app.logScrollPos -= step
	if app.logScrollPos < 0 {
		app.logScrollPos = 0
	}
	app.updateLogsView(false)
	return nil
}

// Редактор обработки ввода текста для фильтрации
func (app *App) createFilterEditor() gocui.Editor {
	return gocui.EditorFunc(func(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
		switch {
		// добавляем символ в поле ввода
		case ch != 0 && mod == 0:
			v.EditWrite(ch)
		// добавляем пробел
		case key == gocui.KeySpace:
			v.EditWrite(' ')
		// удаляем символ слева от курсора
		case key == gocui.KeyBackspace || key == gocui.KeyBackspace2:
			v.EditDelete(true)
		// Удаляем символ справа от курсора
		case key == gocui.KeyDelete:
			v.EditDelete(false)
		// Перемещение курсора влево
		case key == gocui.KeyArrowLeft:
			v.MoveCursor(-1, 0) // удалить 3-й булевой параметр для форка
		// Перемещение курсора вправо
		case key == gocui.KeyArrowRight:
			v.MoveCursor(1, 0)
		}
		// Обновляем текст в буфере
		app.filterText = strings.TrimSpace(v.Buffer())
		// Применяем функцию фильтрации к выводу записей журнала
		app.applyFilter()
	})
}

// ---------------------------------------- Key Binding ----------------------------------------

// Функция для биндинга клавиш
func (app *App) setupKeybindings() error {
	// Ctrl+C для выхода из приложения
	if err := app.gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	// Tab для переключения между окнами
	if err := app.gui.SetKeybinding("", gocui.KeyTab, gocui.ModNone, app.nextView); err != nil {
		return err
	}
	// Enter для выбора службы и загрузки журналов
	if err := app.gui.SetKeybinding("services", gocui.KeyEnter, gocui.ModNone, app.selectService); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("varLogs", gocui.KeyEnter, gocui.ModNone, app.selectFile); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("docker", gocui.KeyEnter, gocui.ModNone, app.selectDocker); err != nil {
		return err
	}
	// Вниз (KeyArrowDown) для перемещения к следующей службе в списке журналов (функция nextService)
	app.gui.SetKeybinding("services", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextService(v, 1)
	})
	app.gui.SetKeybinding("varLogs", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextFileName(v, 1)
	})
	app.gui.SetKeybinding("docker", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextDockerContainer(v, 1)
	})
	// Быстрое пролистывание (через 10 записей) Shift+Down
	app.gui.SetKeybinding("services", gocui.KeyArrowDown, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextService(v, 10)
	})
	app.gui.SetKeybinding("varLogs", gocui.KeyArrowDown, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextFileName(v, 10)
	})
	app.gui.SetKeybinding("docker", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextDockerContainer(v, 10)
	})
	// Пролистывание вверх
	app.gui.SetKeybinding("services", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevService(v, 1)
	})
	app.gui.SetKeybinding("varLogs", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevFileName(v, 1)
	})
	app.gui.SetKeybinding("docker", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevDockerContainer(v, 1)
	})
	// Shift+Up
	app.gui.SetKeybinding("services", gocui.KeyArrowUp, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevService(v, 10)
	})
	app.gui.SetKeybinding("varLogs", gocui.KeyArrowUp, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevFileName(v, 10)
	})
	app.gui.SetKeybinding("docker", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevDockerContainer(v, 10)
	})
	// Переключение между режимами фильтрации <Shift/Alt>+<Left/Right>
	if err := app.gui.SetKeybinding("", gocui.KeyArrowRight, gocui.ModShift, app.setFilterModeRight); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("", gocui.KeyArrowRight, gocui.ModAlt, app.setFilterModeRight); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModShift, app.setFilterModeLeft); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModAlt, app.setFilterModeLeft); err != nil {
		return err
	}
	// Переключение выбора журналов для journalctl (systemd)
	if err := app.gui.SetKeybinding("services", gocui.KeyArrowRight, gocui.ModNone, app.setUnitListRight); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("services", gocui.KeyArrowLeft, gocui.ModNone, app.setUnitListLeft); err != nil {
		return err
	}
	// Переключение выбора журналов для File System
	if err := app.gui.SetKeybinding("varLogs", gocui.KeyArrowRight, gocui.ModNone, app.setLogFilesList); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("varLogs", gocui.KeyArrowLeft, gocui.ModNone, app.setLogFilesList); err != nil {
		return err
	}
	// Переключение выбора журналов для Containerization System
	if err := app.gui.SetKeybinding("docker", gocui.KeyArrowRight, gocui.ModNone, app.setContainersList); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("docker", gocui.KeyArrowLeft, gocui.ModNone, app.setContainersList); err != nil {
		return err
	}
	// Пролистывание вывода журнала
	app.gui.SetKeybinding("logs", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.scrollDownLogs(1)
	})
	app.gui.SetKeybinding("logs", gocui.KeyArrowDown, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		return app.scrollDownLogs(10)
	})
	app.gui.SetKeybinding("logs", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.scrollUpLogs(1)
	})
	app.gui.SetKeybinding("logs", gocui.KeyArrowUp, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		return app.scrollUpLogs(10)
	})
	// Ручное обновление вывода журнала (Ctrl+R)
	app.gui.SetKeybinding("", gocui.KeyCtrlR, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.updateLogOutput(0)
	})
	return nil
}

// Функция для обновления последнего выбранного вывода лога
func (app *App) updateLogOutput(seconds int) error {
	for {
		switch app.lastWindow {
		case "services":
			app.loadJournalLogs(app.lastSelected)
		case "varLogs":
			app.loadFileLogs(app.lastSelected)
		case "docker":
			app.loadDockerLogs(app.lastSelected)
		}
		if seconds == 0 {
			break
		}
		timer := time.Duration(seconds) * time.Second
		time.Sleep(timer)
	}
	return nil
}

// Функции для переключения режима фильтрации

func (app *App) setFilterModeRight(g *gocui.Gui, v *gocui.View) error {
	selectedFilter, err := g.View("filter")
	if err != nil {
		log.Panicln(err)
	}
	switch selectedFilter.Title {
	case "Filter [Default]":
		selectedFilter.Title = "Filter [Fuzzy]"
		app.selectFilterMode = "fuzzy"
	case "Filter [Fuzzy]":
		selectedFilter.Title = "Filter [Regex]"
		app.selectFilterMode = "regex"
	case "Filter [Regex]":
		selectedFilter.Title = "Filter [Default]"
		app.selectFilterMode = "default"
	}
	app.applyFilter()
	return nil
}

func (app *App) setFilterModeLeft(g *gocui.Gui, v *gocui.View) error {
	selectedFilter, err := g.View("filter")
	if err != nil {
		log.Panicln(err)
	}
	switch selectedFilter.Title {
	case "Filter [Default]":
		selectedFilter.Title = "Filter [Regex]"
		app.selectFilterMode = "regex"
	case "Filter [Regex]":
		selectedFilter.Title = "Filter [Fuzzy]"
		app.selectFilterMode = "fuzzy"
	case "Filter [Fuzzy]":
		selectedFilter.Title = "Filter [Default]"
		app.selectFilterMode = "default"
	}
	app.applyFilter()
	return nil
}

// Функции для переключения выбора журналов из journalctl

func (app *App) setUnitListRight(g *gocui.Gui, v *gocui.View) error {
	selectedServices, err := g.View("services")
	if err != nil {
		log.Panicln(err)
	}
	// Сбрасываем содержимое массива и положение курсора
	app.journals = app.journals[:0]
	app.startServices = 0
	app.selectedJournal = 0
	// Меняем журнал и обновляем список
	switch selectedServices.Title {
	case " < System units > ":
		selectedServices.Title = " < User units > "
		app.selectUnits = "USER_UNIT"
		app.loadServices(app.selectUnits)
	case " < User units > ":
		selectedServices.Title = " < Kernel boot > "
		app.selectUnits = "kernel"
		app.loadServices(app.selectUnits)
	case " < Kernel boot > ":
		selectedServices.Title = " < System units > "
		app.selectUnits = "UNIT"
		app.loadServices(app.selectUnits)
	}
	return nil
}

func (app *App) setUnitListLeft(g *gocui.Gui, v *gocui.View) error {
	selectedServices, err := g.View("services")
	if err != nil {
		log.Panicln(err)
	}
	app.journals = app.journals[:0]
	app.startServices = 0
	app.selectedJournal = 0
	switch selectedServices.Title {
	case " < System units > ":
		selectedServices.Title = " < Kernel boot > "
		app.selectUnits = "kernel"
		app.loadServices(app.selectUnits)
	case " < Kernel boot > ":
		selectedServices.Title = " < User units > "
		app.selectUnits = "USER_UNIT"
		app.loadServices(app.selectUnits)
	case " < User units > ":
		selectedServices.Title = " < System units > "
		app.selectUnits = "UNIT"
		app.loadServices(app.selectUnits)
	}
	return nil
}

// Функция для переключения выбора журналов из файловой системы
func (app *App) setLogFilesList(g *gocui.Gui, v *gocui.View) error {
	selectedVarLog, err := g.View("varLogs")
	if err != nil {
		log.Panicln(err)
	}
	app.logfiles = app.logfiles[:0]
	app.startFiles = 0
	app.selectedFile = 0
	switch selectedVarLog.Title {
	case " < Var logs > ":
		selectedVarLog.Title = " < Home logs > "
		app.selectPath = "/home/"
		app.loadFiles(app.selectPath)
	case " < Home logs > ":
		selectedVarLog.Title = " < Var logs > "
		app.selectPath = "/var/log/"
		app.loadFiles(app.selectPath)
	}
	return nil
}

// Функция для переключения выбора системы контейнеризации (Docker/Podman)
func (app *App) setContainersList(g *gocui.Gui, v *gocui.View) error {
	selectedDocker, err := g.View("docker")
	if err != nil {
		log.Panicln(err)
	}
	app.dockerContainers = app.dockerContainers[:0]
	app.startDockerContainers = 0
	app.selectedDockerContainer = 0
	switch selectedDocker.Title {
	case " < Docker containers > ":
		selectedDocker.Title = " < Podman containers > "
		app.selectContainerizationSystem = "podman"
		app.loadDockerContainer(app.selectContainerizationSystem)
	case " < Podman containers > ":
		selectedDocker.Title = " < Docker containers > "
		app.selectContainerizationSystem = "docker"
		app.loadDockerContainer(app.selectContainerizationSystem)
	}
	return nil
}

// Функция для переключения окон через Tab
func (app *App) nextView(g *gocui.Gui, v *gocui.View) error {
	selectedServices, err := g.View("services")
	if err != nil {
		log.Panicln(err)
	}
	selectedVarLog, err := g.View("varLogs")
	if err != nil {
		log.Panicln(err)
	}
	selectedDocker, err := g.View("docker")
	if err != nil {
		log.Panicln(err)
	}
	selectedFilter, err := g.View("filter")
	if err != nil {
		log.Panicln(err)
	}
	selectedLogs, err := g.View("logs")
	if err != nil {
		log.Panicln(err)
	}
	currentView := g.CurrentView()
	var nextView string
	// Начальное окно
	if currentView == nil {
		nextView = "services"
	} else {
		switch currentView.Name() {
		// Если текущее окно services, переходим к filter
		case "services":
			nextView = "varLogs"
			selectedServices.FrameColor = gocui.ColorDefault
			selectedServices.TitleColor = gocui.ColorDefault
			selectedVarLog.FrameColor = gocui.ColorGreen
			selectedVarLog.TitleColor = gocui.ColorGreen
			selectedDocker.FrameColor = gocui.ColorDefault
			selectedDocker.TitleColor = gocui.ColorDefault
			selectedFilter.FrameColor = gocui.ColorDefault
			selectedFilter.TitleColor = gocui.ColorDefault
			selectedLogs.FrameColor = gocui.ColorDefault
			selectedLogs.TitleColor = gocui.ColorDefault
		case "varLogs":
			nextView = "docker"
			selectedServices.FrameColor = gocui.ColorDefault
			selectedServices.TitleColor = gocui.ColorDefault
			selectedVarLog.FrameColor = gocui.ColorDefault
			selectedVarLog.TitleColor = gocui.ColorDefault
			selectedDocker.FrameColor = gocui.ColorGreen
			selectedDocker.TitleColor = gocui.ColorGreen
			selectedFilter.FrameColor = gocui.ColorDefault
			selectedFilter.TitleColor = gocui.ColorDefault
			selectedLogs.FrameColor = gocui.ColorDefault
			selectedLogs.TitleColor = gocui.ColorDefault
		case "docker":
			nextView = "filter"
			selectedServices.FrameColor = gocui.ColorDefault
			selectedServices.TitleColor = gocui.ColorDefault
			selectedVarLog.FrameColor = gocui.ColorDefault
			selectedVarLog.TitleColor = gocui.ColorDefault
			selectedDocker.FrameColor = gocui.ColorDefault
			selectedDocker.TitleColor = gocui.ColorDefault
			selectedFilter.FrameColor = gocui.ColorGreen
			selectedFilter.TitleColor = gocui.ColorGreen
			selectedLogs.FrameColor = gocui.ColorDefault
			selectedLogs.TitleColor = gocui.ColorDefault
		case "filter":
			nextView = "logs"
			selectedServices.FrameColor = gocui.ColorDefault
			selectedServices.TitleColor = gocui.ColorDefault
			selectedVarLog.FrameColor = gocui.ColorDefault
			selectedVarLog.TitleColor = gocui.ColorDefault
			selectedDocker.FrameColor = gocui.ColorDefault
			selectedDocker.TitleColor = gocui.ColorDefault
			selectedFilter.FrameColor = gocui.ColorDefault
			selectedFilter.TitleColor = gocui.ColorDefault
			selectedLogs.FrameColor = gocui.ColorGreen
			selectedLogs.TitleColor = gocui.ColorGreen
		case "logs":
			nextView = "services"
			selectedServices.FrameColor = gocui.ColorGreen
			selectedServices.TitleColor = gocui.ColorGreen
			selectedVarLog.FrameColor = gocui.ColorDefault
			selectedVarLog.TitleColor = gocui.ColorDefault
			selectedDocker.FrameColor = gocui.ColorDefault
			selectedDocker.TitleColor = gocui.ColorDefault
			selectedFilter.FrameColor = gocui.ColorDefault
			selectedFilter.TitleColor = gocui.ColorDefault
			selectedLogs.FrameColor = gocui.ColorDefault
			selectedLogs.TitleColor = gocui.ColorDefault
		}
	}
	// Устанавливаем новое активное окно
	if _, err := g.SetCurrentView(nextView); err != nil {
		return err
	}
	return nil
}

// Функция для выхода
func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
