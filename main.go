package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/awesome-gocui/gocui"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
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

	getOS         string   // название ОС
	getArch       string   // архитектура процессора
	hostName      string   // текущее имя хоста для покраски в логах
	userName      string   // текущее имя пользователя
	userNameArray []string // список всех пользователей
	rootDirArray  []string // список всех корневых каталогов

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

	filterListText string // текст для фильтрации список журналов

	// Массивы для хранения списка журналов без фильтрации
	journalsNotFilter         []Journal
	logfilesNotFilter         []Logfile
	dockerContainersNotFilter []DockerContainers

	filterText       string   // текст для фильтрации записей журнала
	currentLogLines  []string // набор строк (срез) для хранения журнала без фильтрации
	filteredLogLines []string // набор строк (срез) для хранения журнала после фильтра
	logScrollPos     int      // позиция прокрутки для отображаемых строк журнала
	lastFilterText   string   // фиксируем содержимое последнего ввода текста для фильтрации

	autoScroll     bool   // используется для автоматического скроллинга вниз при обновлении (если это не ручной скроллинг)
	newUpdateIndex int    // фиксируем текущую длинну массива (индекс) для вставки строки обновления (если это ручной выбор из списка)
	updateTime     string // время загрузки журнала для делиметра

	lastDateUpdateFile time.Time // последняя дата изменения файла
	lastSizeFile       int64     // размер файла
	updateFile         bool

	lastWindow   string // фиксируем последний используемый источник для вывода логов
	lastSelected string // фиксируем название последнего выбранного журнала или контейнера

	// Переменные для хранения значений автообновления вывода при смене окна
	lastSelectUnits            string
	lastBootId                 string
	lastLogPath                string
	lastContainerizationSystem string
	lastContainerId            string

	// Цвета окон по умолчанию (изменяется в зависимости от доступности журналов)
	journalListFrameColor gocui.Attribute
	fileSystemFrameColor  gocui.Attribute
	dockerFrameColor      gocui.Attribute

	// Фиксируем последнее время загрузки журнала
	debugLoadTime string

	// Отключение привязки горячих клавиш на время загрузки списка
	keybindingsEnabled bool

	// Регулярные выражения для покраски строк
	trimHttpRegex        *regexp.Regexp
	trimHttpsRegex       *regexp.Regexp
	trimPrefixPathRegex  *regexp.Regexp
	trimPostfixPathRegex *regexp.Regexp
	hexByteRegex         *regexp.Regexp
	dateTimeRegex        *regexp.Regexp
	timeMacAddressRegex  *regexp.Regexp
	timeRegex            *regexp.Regexp
	macAddressRegex      *regexp.Regexp
	dateIpAddressRegex   *regexp.Regexp
	dateRegex            *regexp.Regexp
	ipAddressRegex       *regexp.Regexp
	procRegex            *regexp.Regexp
	syslogUnitRegex      *regexp.Regexp
}

func showHelp() {
	fmt.Println("lazydocker - terminal user interface  for journalctl, file system logs, as well Docker and Podman containers")
	fmt.Println("Source code: https://github.com/Lifailon/lazyjournal")
	fmt.Println("")
	fmt.Println("  Flags:")
	fmt.Println("    lazyjournal                Run interface")
	fmt.Println("    lazyjournal --help, -h     Show help")
	fmt.Println("    lazyjournal --version, -v  Show version")
}

func (app *App) showVersion() {
	fmt.Println("Version:", "0.7.0") // Текущая версия
	if app.getOS != "windows" {
		data, err := os.ReadFile("/etc/os-release")
		if err != nil {
			fmt.Printf("OS: %s\n", app.getOS)
			return
		} else {
			var name, version string
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "NAME=") {
					name = strings.Trim(line[5:], "\"")
				}
				if strings.HasPrefix(line, "VERSION=") {
					version = strings.Trim(line[8:], "\"")
				}
			}
			fmt.Printf("OS: %s %s %s\n", app.getOS, name, version)
		}
	} else {
		fmt.Printf("OS: %s\n", app.getOS)
	}
	fmt.Printf("Arch: %s\n", app.getArch)
	execPath, err := os.Executable()
	if err != nil {
		return
	}
	if strings.Contains(execPath, "tmp/go-build") || strings.Contains(execPath, "Temp\\go-build") {
		fmt.Printf("Executable type: source code (%s)\n", execPath)
	} else {
		fmt.Printf("Executable type: binary file (%s)\n", execPath)
	}
	fmt.Println("If you have problems with the application, please open issue: https://github.com/Lifailon/lazyjournal/issues")
}

func main() {
	// Инициализация значений по умолчанию + компиляция регулярных выражений для покраски
	app := &App{
		startServices:                0, // начальная позиция списка юнитов
		selectedJournal:              0, // начальный индекс выбранного журнала
		startFiles:                   0,
		selectedFile:                 0,
		startDockerContainers:        0,
		selectedDockerContainer:      0,
		selectUnits:                  "services",  // "UNIT" || "USER_UNIT" || "kernel"
		selectPath:                   "/var/log/", // "/home/" ("/Users/" - для MacOS)
		selectContainerizationSystem: "docker",    // "podman"
		selectFilterMode:             "default",   // "fuzzy" || "regex"
		logViewCount:                 "200000",    // 5000-300000
		journalListFrameColor:        gocui.ColorDefault,
		fileSystemFrameColor:         gocui.ColorDefault,
		dockerFrameColor:             gocui.ColorDefault,
		autoScroll:                   true,
		trimHttpRegex:                regexp.MustCompile(`^.*http://|([^a-zA-Z0-9:/._?&=-].*)$`),                                                                                             // исключаем все до http:// (включительно) в начале строки
		trimHttpsRegex:               regexp.MustCompile(`^.*https://|([^a-zA-Z0-9:/._?&=-].*)$`),                                                                                            // и после любого символа, который не может содержать в себе url
		trimPrefixPathRegex:          regexp.MustCompile(`^[^/]+`),                                                                                                                           // иключаем все до первого символа слэша (не включительно)
		trimPostfixPathRegex:         regexp.MustCompile(`[=:'"(){}\[\]]+.*$`),                                                                                                               // исключаем все после первого символа, который не должен (но может) содержаться в пути
		hexByteRegex:                 regexp.MustCompile(`\b0x[0-9A-Fa-f]+\b`),                                                                                                               // Байты или числа в шестнадцатеричном формате: 0x2 || 0xc0000001
		dateTimeRegex:                regexp.MustCompile(`\b(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?([+-]\d{2}:\d{2})?)\b`),                                                              // YYYY-MM-DDTHH:MM:SS.MS+HH:MM
		timeMacAddressRegex:          regexp.MustCompile(`\b(?:\d{1,2}:\d{2}(:\d{2}([.,+]\d{2,6})?)?|\b(?:[0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}\b)\b`),                                        // Time + MAC address
		timeRegex:                    regexp.MustCompile(`\b\d{1,2}:\d{2}(:\d{2}([.,+]\d{1,6})?)?([+-]\d{2}(:\d{2})?)?\b`),                                                                   // Time: H:MM || HH:MM || HH:MM:SS || XX:XX:XX:XX || HH:MM:SS,XXX || HH:MM:SS.XXX || HH:MM:SS+03
		macAddressRegex:              regexp.MustCompile(`\b([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}\b`),                                                                                        // MAC: XX:XX:XX:XX:XX:XX || XX-XX-XX-XX-XX-XX
		dateIpAddressRegex:           regexp.MustCompile(`\b(\d{1,2}[-.]\d{1,2}[-.]\d{4}|\d{4}[-.]\d{1,2}[-.]\d{1,2}|(?:\d{1,3}\.){3}\d{1,3}(?::\d+|\.\d+|/\d+)?|\d+\.\d+\.\d+|\d+\.\d+)\b`), // Date + IP address + version
		dateRegex:                    regexp.MustCompile(`\b(\d{1,2}[-.]\d{1,2}[-.]\d{4}|\d{4}[-.]\d{1,2}[-.]\d{1,2}|\d+\.\d+\.\d+|\d+\.\d+)\b`),                                             // Date: DD-MM-YYYY || DD.MM.YYYY || YYYY-MM-DD || YYYY.MM.DD || 5.7.5 (version) || 5.709076
		ipAddressRegex:               regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}(?::\d+|\.\d+|/\d+)?\b`),                                                                                  // IP: 255.255.255.255 || 255.255.255.255:443 || 255.255.255.255.443 || 255.255.255.255/24
		procRegex:                    regexp.MustCompile(`(\d+)%`),                                                                                                                           // int%
		syslogUnitRegex:              regexp.MustCompile(`^[a-zA-Z-_.]+\[\d+\]:$`),                                                                                                           // unit_daemon-name.service[1341]:
		keybindingsEnabled:           true,
	}

	// Определяем используемую ОС (linux/darwin/*bsd/windows) и архитектуру
	app.getOS = runtime.GOOS
	app.getArch = runtime.GOARCH

	// Аргументы
	help := flag.Bool("help", false, "Show help")
	flag.BoolVar(help, "h", false, "Show help")
	version := flag.Bool("version", false, "Show version")
	flag.BoolVar(version, "v", false, "Show version")

	// Обработка аргументов
	flag.Parse()
	if *help {
		showHelp()
		os.Exit(0)
	}
	if *version {
		app.showVersion()
		os.Exit(0)
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
		log.Panicln("Error key bindings", err)
	}

	// Выполняем layout для инициализации интерфейса
	if err := app.layout(g); err != nil {
		log.Panicln(err)
	}

	// Определяем переменные и массивы для покраски вывода
	// Текущее имя хоста
	app.hostName, _ = os.Hostname()
	// Удаляем доменную часть, если она есть
	if strings.Contains(app.hostName, ".") {
		app.hostName = strings.Split(app.hostName, ".")[0]
	}
	// Текущее имя пользователя
	currentUser, _ := user.Current()
	app.userName = currentUser.Username
	// Удаляем доменную часть, если она есть
	if strings.Contains(app.userName, "\\") {
		app.userName = strings.Split(app.userName, "\\")[1]
	}
	// Имена пользователей
	passwd, _ := os.Open("/etc/passwd")
	scanner := bufio.NewScanner(passwd)
	for scanner.Scan() {
		line := scanner.Text()
		userName := strings.Split(line, ":")
		if len(userName) > 0 {
			app.userNameArray = append(app.userNameArray, userName[0])
		}
	}
	// Список корневых каталогов (ls -d /*/)
	files, _ := os.ReadDir("/")
	for _, file := range files {
		if file.IsDir() {
			app.rootDirArray = append(app.rootDirArray, file.Name())
		}
	}

	// Фиксируем текущее количество видимых строк в терминале (-1 заголовок)
	if v, err := g.View("services"); err == nil {
		_, viewHeight := v.Size()
		app.maxVisibleServices = viewHeight
	}
	// Загрузка списков журналов
	app.loadServices(app.selectUnits)

	// Filesystem
	if v, err := g.View("varLogs"); err == nil {
		_, viewHeight := v.Size()
		app.maxVisibleFiles = viewHeight
	}

	// Определяем ОС и загружаем файловые журналы
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

	// Docker
	if v, err := g.View("docker"); err == nil {
		_, viewHeight := v.Size()
		app.maxVisibleDockerContainers = viewHeight
	}
	app.loadDockerContainer(app.selectContainerizationSystem)

	// Устанавливаем фокус на окно с журналами по умолчанию
	if _, err := g.SetCurrentView("filterList"); err != nil {
		return
	}

	// Горутина для автоматического обновления вывода журнала каждын 3 секунды
	errCh := make(chan error)
	go func() {
		if err := app.updateLogOutput(3); err != nil {
			errCh <- err
		}
	}()

	// Запус GUI
	if err := g.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		log.Panicln(err)
	}
}

// Структура интерфейса окон GUI
func (app *App) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()                // получаем текущий размер интерфейса терминала (ширина, высота)
	leftPanelWidth := maxX / 4            // ширина левой колонки
	inputHeight := 3                      // высота поля ввода для фильтрации список
	availableHeight := maxY - inputHeight // общая высота всех трех окон слева
	panelHeight := availableHeight / 3    // высота каждого окна

	// Поле ввода для фильтрации списков
	if v, err := g.SetView("filterList", 0, 0, leftPanelWidth-1, inputHeight-1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = "Filtering lists"
		v.Editable = true
		v.Wrap = true
		v.FrameColor = gocui.ColorGreen // Цвет границ окна
		v.TitleColor = gocui.ColorGreen // Цвет заголовка
		v.Editor = app.createFilterEditor("lists")
	}

	// Окно для отображения списка доступных журналов (UNIT)
	// Размеры окна: заголовок, отступ слева, отступ сверху, ширина, высота, 5-й параметр из форка для продолжение окна (2)
	if v, err := g.SetView("services", 0, inputHeight, leftPanelWidth-1, inputHeight+panelHeight-1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = " < Unit list (0) > " // заголовок окна
		v.Highlight = true              // выделение активного элемента в списке
		v.Wrap = false                  // отключаем перенос строк
		v.Autoscroll = true             // включаем автопрокрутку
		// Цветовая схема из форка awesome-gocui/gocui
		v.SelBgColor = gocui.ColorGreen // Цвет фона при выборе в списке
		v.SelFgColor = gocui.ColorBlack // Цвет текста
		app.updateServicesList()        // выводим список журналов в это окно
	}

	// Окно для списка логов из файловой системы
	if v, err := g.SetView("varLogs", 0, inputHeight+panelHeight, leftPanelWidth-1, inputHeight+2*panelHeight-1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = " < System var logs (0) > "
		v.Highlight = true
		v.Wrap = false
		v.Autoscroll = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		app.updateLogsList()
	}

	// Окно для списка контейнеров Docker и Podman
	if v, err := g.SetView("docker", 0, inputHeight+2*panelHeight, leftPanelWidth-1, maxY-1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = " < Docker containers (0) > "
		v.Highlight = true
		v.Wrap = false
		v.Autoscroll = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
	}

	// Окно ввода текста для фильтрации
	if v, err := g.SetView("filter", leftPanelWidth+1, 0, maxX-1, 2, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = "Filter (Default)"
		v.Editable = true                         // включить окно редактируемым для ввода текста
		v.Editor = app.createFilterEditor("logs") // редактор для обработки ввода
		v.Wrap = true
	}

	// Окно для вывода записей выбранного журнала
	if v, err := g.SetView("logs", leftPanelWidth+1, 3, maxX-1, maxY-1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = "Logs"
		v.Wrap = true
		v.Autoscroll = false
	}

	// Включение курсора в режиме фильтра и отключение в остальных окнах
	currentView := g.CurrentView()
	if currentView != nil && (currentView.Name() == "filter" || currentView.Name() == "filterList") {
		g.Cursor = true
	} else {
		g.Cursor = false
	}

	return nil
}

// ---------------------------------------- journalctl ----------------------------------------

// Функция для загрузки списка журналов служб или загрузок системы из journalctl
func (app *App) loadServices(journalName string) {
	// Проверка, что в системе установлен/поддерживается утилита journalctl
	checkJournald := exec.Command("journalctl", "--version")
	// Проверяем на ошибки (очищаем список служб, отключаем курсор и выводим ошибку)
	_, err := checkJournald.Output()
	if err != nil {
		vError, _ := app.gui.View("services")
		vError.Clear()
		app.journalListFrameColor = gocui.ColorRed
		vError.FrameColor = app.journalListFrameColor
		vError.Highlight = false
		fmt.Fprintln(vError, "\033[31msystemd-journald not supported\033[0m")
		return
	}
	switch {
	case journalName == "services":
		// Получаем список всех юнитов в системе через systemctl в формате JSON
		unitsList := exec.Command("systemctl", "list-units", "--all", "--plain", "--no-legend", "--no-pager", "--output=json") // "--type=service"
		output, err := unitsList.Output()
		if err != nil {
			vError, _ := app.gui.View("services")
			vError.Clear()
			app.journalListFrameColor = gocui.ColorRed
			vError.FrameColor = app.journalListFrameColor
			vError.Highlight = false
			fmt.Fprintln(vError, "\033[31mAccess denied in systemd\033[0m")
			return
		}
		v, _ := app.gui.View("services")
		app.journalListFrameColor = gocui.ColorDefault
		if v.FrameColor != gocui.ColorDefault {
			v.FrameColor = gocui.ColorGreen
		}
		v.Highlight = true
		// Чтение данных в формате JSON
		var units []map[string]interface{}
		err = json.Unmarshal(output, &units)
		// Если ошибка JSON, создаем массив вручную
		if err != nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				// Разбиваем строку на поля (эквивалентно: awk '{print $1,$3,$4}')
				fields := strings.Fields(line)
				// Пропускаем строки с недостаточным количеством полей
				if len(fields) < 3 {
					continue
				}
				// Заполняем временный массив из строки
				unit := map[string]interface{}{
					"unit":   fields[0],
					"active": fields[2],
					"sub":    fields[3],
				}
				// Добавляем временный массив строки в основной массив
				units = append(units, unit)
			}
		}
		serviceMap := make(map[string]bool)
		// Обработка записей
		for _, unit := range units {
			// Извлечение данных в формате JSON и проверка статуса для покраски
			unitName, _ := unit["unit"].(string)
			active, _ := unit["active"].(string)
			if active == "active" {
				active = "\033[32m" + active + "\033[0m"
			} else {
				active = "\033[31m" + active + "\033[0m"
			}
			sub, _ := unit["sub"].(string)
			if sub == "exited" || sub == "dead" {
				sub = "\033[31m" + sub + "\033[0m"
			} else {
				sub = "\033[32m" + sub + "\033[0m"
			}
			name := unitName + " (" + active + "/" + sub + ")"
			bootID := unitName
			// Уникальный ключ для проверки
			uniqueKey := name + ":" + bootID
			if !serviceMap[uniqueKey] {
				serviceMap[uniqueKey] = true
				// Добавление записи в массив
				app.journals = append(app.journals, Journal{
					name:    name,
					boot_id: bootID,
				})
			}
		}
	case journalName == "kernel":
		// Получаем список загрузок системы
		bootCmd := exec.Command("journalctl", "--list-boots", "-o", "json")
		bootOutput, err := bootCmd.Output()
		if err != nil {
			vError, _ := app.gui.View("services")
			vError.Clear()
			app.journalListFrameColor = gocui.ColorRed
			vError.FrameColor = app.journalListFrameColor
			vError.Highlight = false
			fmt.Fprintln(vError, "\033[31mError getting boot information from journald\033[0m")
			return
		} else {
			vError, _ := app.gui.View("services")
			app.journalListFrameColor = gocui.ColorDefault
			if vError.FrameColor != gocui.ColorDefault {
				vError.FrameColor = gocui.ColorGreen
			}
			vError.Highlight = true
		}
		// Структура для парсинга JSON
		type BootInfo struct {
			BootID     string `json:"boot_id"`
			FirstEntry int64  `json:"first_entry"`
			LastEntry  int64  `json:"last_entry"`
		}
		var bootRecords []BootInfo
		err = json.Unmarshal(bootOutput, &bootRecords)
		// Если JSON невалидный
		if err != nil {
			// Парсим вывод построчно
			lines := strings.Split(string(bootOutput), "\n")
			for _, line := range lines {
				// Разбиваем строку на массив
				wordsArray := strings.Fields(line)
				// 0 d914ebeb67c6428a87f9cfe3861c295d Mon 2024-11-25 12:15:07 MSK—Mon 2024-11-25 18:34:53 MSK
				if len(wordsArray) >= 8 {
					bootId := wordsArray[1]
					// Забираем дату, проверяем и изменяем формат
					var parseDate []string
					var bootDate string
					parseDate = strings.Split(wordsArray[3], "-")
					if len(parseDate) == 3 {
						bootDate = fmt.Sprintf("%s.%s.%s", parseDate[2], parseDate[1], parseDate[0])
					} else {
						continue
					}
					var stopDate string
					parseDate = strings.Split(wordsArray[6], "-")
					if len(parseDate) == 3 {
						stopDate = fmt.Sprintf("%s.%s.%s", parseDate[2], parseDate[1], parseDate[0])
					} else {
						continue
					}
					// Заполняем массив
					bootDateTime := bootDate + " " + wordsArray[4]
					stopDateTime := stopDate + " " + wordsArray[7]
					app.journals = append(app.journals, Journal{
						name:    fmt.Sprintf("%s - %s", bootDateTime, stopDateTime),
						boot_id: bootId,
					})
				}
			}
		} else {
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
			}
		}
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
	default:
		cmd := exec.Command("journalctl", "--no-pager", "-F", journalName)
		output, err := cmd.Output()
		if err != nil {
			vError, _ := app.gui.View("services")
			vError.Clear()
			app.journalListFrameColor = gocui.ColorRed
			vError.FrameColor = app.journalListFrameColor
			vError.Highlight = false
			fmt.Fprintln(vError, "\033[31mError getting services from journald\033[0m")
			return
		} else {
			vError, _ := app.gui.View("services")
			app.journalListFrameColor = gocui.ColorDefault
			if vError.FrameColor != gocui.ColorDefault {
				vError.FrameColor = gocui.ColorGreen
			}
			vError.Highlight = true
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
	// Сохраняем неотфильтрованный список
	app.journalsNotFilter = app.journals
	// Применяем фильтр при загрузки и обновляем список служб в интерфейсе через updateServicesList() внутри функции
	app.applyFilterList()
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
	app.maxVisibleServices = viewHeight
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
	app.maxVisibleServices = viewHeight
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
	// Обновляем счетчик в заголовке
	re := regexp.MustCompile(`\s\(.+\) >`)
	updateTitle := " (0) >"
	if len(app.journals) != 0 {
		updateTitle = " (" + strconv.Itoa(app.selectedJournal+1) + "/" + strconv.Itoa(len(app.journals)) + ") >"
	}
	v.Title = re.ReplaceAllString(v.Title, updateTitle)
	// Устанавливаем курсор на нужный индекс (строку)
	// Первый столбец (0), индекс строки
	if err := v.SetCursor(0, index); err != nil {
		return nil
	}
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
	app.loadJournalLogs(strings.TrimSpace(line), true, g)
	// Фиксируем для ручного или автоматического обновления вывода журнала
	app.lastWindow = "services"
	app.lastSelected = strings.TrimSpace(line)
	return nil
}

// Функция для загрузки записей журнала выбранной службы через journalctl
// Второй параметр для обнолвения позиции делимитра нового вывода лога а также сброса автоскролл
func (app *App) loadJournalLogs(serviceName string, newUpdate bool, g *gocui.Gui) {
	var output []byte
	var err error
	selectUnits := app.selectUnits
	if newUpdate {
		app.lastSelectUnits = app.selectUnits
	} else {
		selectUnits = app.lastSelectUnits
	}
	// Загрузки системы для логов ядра
	if selectUnits == "kernel" {
		var boot_id string
		for _, journal := range app.journals {
			if journal.name == serviceName {
				boot_id = journal.boot_id
				break
			}
		}
		// Сохраняем название для обновления вывода журнала при фильтрации списков
		if newUpdate {
			app.lastBootId = boot_id
		} else {
			boot_id = app.lastBootId
		}
		cmd := exec.Command("journalctl", "-k", "-b", boot_id, "--no-pager", "-n", app.logViewCount)
		output, err = cmd.Output()
		if err != nil {
			v, _ := app.gui.View("logs")
			v.Clear()
			fmt.Fprintln(v, "\033[31mError getting logs:", err, "\033[0m")
			return
		}
		// Для юнитов systemd
	} else {
		if selectUnits == "services" {
			// Удаляем статусы с покраской из навзания
			var ansiEscape = regexp.MustCompile(`\s\(.+\)`)
			serviceName = ansiEscape.ReplaceAllString(serviceName, "")
		}
		cmd := exec.Command("journalctl", "-u", serviceName, "--no-pager", "-n", app.logViewCount)
		output, err = cmd.Output()
		if err != nil {
			v, _ := app.gui.View("logs")
			v.Clear()
			fmt.Fprintln(v, "\033[31mError getting logs:", err, "\033[0m")
			return
		}
	}
	// Сохраняем строки журнала в массив
	app.currentLogLines = strings.Split(string(output), "\n")
	app.updateDelimiter(newUpdate, g)
	// Очищаем поле ввода для фильтрации, что бы не применять фильтрацию к новому журналу
	// app.filterText = ""
	// Применяем текущий фильтр к записям для обновления вывода
	app.applyFilter(false)
}

// ---------------------------------------- Filesystem ----------------------------------------

func (app *App) loadFiles(logPath string) {
	var output []byte
	switch {
	case logPath == "descriptor":
		// n - имя файла (путь)
		// c - имя команды (процесса)
		cmd := exec.Command("lsof", "-Fn")
		// Подавить вывод ошибок при отсутствиее прав доступа (opendir: Permission denied)
		cmd.Stderr = nil
		output, _ = cmd.Output()
		// Разбиваем вывод на строки
		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		// Если список файлов пустой, возвращаем ошибку Permission denied
		if len(files) == 0 || (len(files) == 1 && files[0] == "") {
			vError, _ := app.gui.View("varLogs")
			vError.Clear()
			// Меняем цвет окна на красный
			app.fileSystemFrameColor = gocui.ColorRed
			vError.FrameColor = app.fileSystemFrameColor
			// Отключаем курсор и выводим сообщение об ошибке
			vError.Highlight = false
			fmt.Fprintln(vError, "\033[31mPermission denied (files not found)\033[0m")
			return
		} else {
			vError, _ := app.gui.View("varLogs")
			app.fileSystemFrameColor = gocui.ColorDefault
			if vError.FrameColor != gocui.ColorDefault {
				vError.FrameColor = gocui.ColorGreen
			}
			vError.Highlight = true
		}
		// Очищаем массив перед добавлением отфильтрованных файлов
		output = []byte{}
		// Фильтруем строки, которые заканчиваются на ".log" и удаляем префикс (имя файла)
		for _, file := range files {
			if strings.HasSuffix(file, ".log") {
				file = strings.TrimPrefix(file, "n")
				output = append(output, []byte(file+"\n")...)
			}
		}
	case logPath == "/var/log/":
		var cmd *exec.Cmd
		// Загрузка системных журналов для MacOS
		if app.getOS == "darwin" {
			cmd = exec.Command(
				"find", logPath, "/Library/Logs", "/opt/",
				"-type", "f",
				"-name", "*.log", "-o",
				"-name", "*log*", "-o",
				"-name", "*.1", "-o",
				"-name", "*.gz", "-o",
				"-name", "*.bz2", "-o",
				"-name", "*.pcap",
			)
		} else {
			// Загрузка системных журналов для Linux (все файлы, которые содержат log в расширение или названии, а также расширение .1, gz, bz2 и pcap)
			cmd = exec.Command(
				"find", logPath, "/opt/",
				"-type", "f",
				"-name", "*.log", "-o",
				"-name", "*log*", "-o",
				"-name", "*.1", "-o",
				"-name", "*.gz", "-o",
				"-name", "*.bz2", "-o",
				"-name", "*.pcap",
			)
		}
		output, _ = cmd.Output()
		// Преобразуем вывод команды в строку и делим на массив строк
		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		// Если список файлов пустой, возвращаем ошибку Permission denied
		if len(files) == 0 || (len(files) == 1 && files[0] == "") {
			vError, _ := app.gui.View("varLogs")
			vError.Clear()
			// Меняем цвет окна на красный
			app.fileSystemFrameColor = gocui.ColorRed
			vError.FrameColor = app.fileSystemFrameColor
			// Отключаем курсор и выводим сообщение об ошибке
			vError.Highlight = false
			fmt.Fprintln(vError, "\033[31mPermission denied\033[0m")
			return
		} else {
			vError, _ := app.gui.View("varLogs")
			app.fileSystemFrameColor = gocui.ColorDefault
			if vError.FrameColor != gocui.ColorDefault {
				vError.FrameColor = gocui.ColorGreen
			}
			vError.Highlight = true
		}
		// Добавляем пути по умолчанию для /var/log
		logPaths := []string{
			// Ядро
			"/var/log/dmesg\n",
			// Информация о входах и выходах пользователей, перезагрузках и остановках системы
			"/var/log/wtmp\n",
			// Информация о неудачных попытках входа в систему (например, неправильные пароли)
			"/var/log/btmp\n",
			// Информация о текущих пользователях, их сеансах и входах в систему
			"/var/run/utmp\n",
			"/run/utmp\n",
			// MacOS/BSD/RHEL
			"/var/log/secure\n",
			"/var/log/messages\n",
			"/var/log/daemon\n",
			"/var/log/lpd-errs\n",
			"/var/log/security.out\n",
			"/var/log/daily.out\n",
			// Службы
			"/var/log/cron\n",
			"/var/log/ftpd\n",
			"/var/log/ntpd\n",
			"/var/log/named\n",
			"/var/log/dhcpd\n",
		}
		for _, path := range logPaths {
			output = append([]byte(path), output...)
		}
	default:
		// Домашние каталоги пользователей: /home/ для Linux и /Users/ для MacOS
		if app.getOS == "darwin" {
			logPath = "/Users/"
		}
		// Ищем файлы с помощью системной утилиты find
		cmd := exec.Command(
			"find", logPath,
			"-type", "d",
			"(",
			"-name", "Library", "-o",
			"-name", "Pictures", "-o",
			"-name", "Movies", "-o",
			"-name", "Music", "-o",
			"-name", ".Trash", "-o",
			"-name", ".cache",
			")",
			"-prune", "-o",
			"-type", "f",
			"(",
			"-name", "*.log", "-o",
			"-name", "*.pcap",
			")",
			"-print",
		)
		output, _ = cmd.Output()
		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(files) == 0 || (len(files) == 1 && files[0] == "") {
			vError, _ := app.gui.View("varLogs")
			vError.Clear()
			vError.Highlight = false
			fmt.Fprintln(vError, "\033[32mNo logs available\033[0m")
			return
		} else {
			vError, _ := app.gui.View("varLogs")
			app.fileSystemFrameColor = gocui.ColorDefault
			if vError.FrameColor != gocui.ColorDefault {
				vError.FrameColor = gocui.ColorGreen
			}
			vError.Highlight = true
		}
		// Получаем содержимое файлов из домашнего каталога пользователя root
		cmdRootDir := exec.Command(
			"find", "/root/",
			"-type", "f",
			"-name", "*.log", "-o",
			"-name", "*.pcap",
		)
		outputRootDir, err := cmdRootDir.Output()
		// Добавляем содержимое директории /root/ в общий массив, если есть доступ
		if err == nil {
			output = append(output, outputRootDir...)
		}
		if app.fileSystemFrameColor == gocui.ColorRed {
			vError, _ := app.gui.View("varLogs")
			app.fileSystemFrameColor = gocui.ColorDefault
			if vError.FrameColor != gocui.ColorDefault {
				vError.FrameColor = gocui.ColorGreen
			}
			vError.Highlight = true
		}
	}
	serviceMap := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		// Получаем строку полного пути
		logFullPath := scanner.Text()
		// Удаляем префикс пути и расширение файла в конце
		logName := logFullPath
		if logPath != "descriptor" {
			logName = strings.TrimPrefix(logFullPath, logPath)
		}
		logName = strings.TrimSuffix(logName, ".log")
		logName = strings.TrimSuffix(logName, ".gz")
		logName = strings.TrimSuffix(logName, ".bz2")
		logName = strings.ReplaceAll(logName, "/", " ")
		logName = strings.ReplaceAll(logName, ".log.", "")
		logName = strings.TrimPrefix(logName, " ")
		if logPath == "/home/" || logPath == "/Users/" {
			// Разбиваем строку на слова
			words := strings.Fields(logName)
			// Берем первое и последнее слово
			firstWord := words[0]
			lastWord := words[len(words)-1]
			logName = "\x1b[0;33m" + firstWord + "\033[0m" + ": " + lastWord
		}
		// Получаем информацию о файле
		// cmd := exec.Command("bash", "-c", "stat --format='%y' /var/log/apache2/access.log | awk '{print $1}' | awk -F- '{print $3\".\"$2\".\"$1}'")
		fileInfo, err := os.Stat(logFullPath)
		if err != nil {
			// Пропускаем файл, если к нему нет доступа (актуально для статических файлов из logPath)
			continue
		}
		// Проверяем, что файл не пустой
		if fileInfo.Size() == 0 {
			// Пропускаем пустой файл
			continue
		}
		// Получаем дату изменения
		modTime := fileInfo.ModTime()
		// Форматирование даты в формат DD.MM.YYYY
		formattedDate := modTime.Format("02.01.2006")
		// Проверяем, что полного пути до файла еще нет в списке
		if logName != "" && !serviceMap[logFullPath] {
			// Добавляем путь в массив для проверки уникальных путей
			serviceMap[logFullPath] = true
			// Получаем имя процесса для файла дескриптора
			if logPath == "descriptor" {
				cmd := exec.Command("lsof", "-Fc", logFullPath)
				cmd.Stderr = nil
				outputLsof, _ := cmd.Output()
				processLines := strings.Split(strings.TrimSpace(string(outputLsof)), "\n")
				// Ищем строку, которая содержит имя процесса (только первый процесс)
				for _, line := range processLines {
					if strings.HasPrefix(line, "c") {
						// Удаляем префикс
						processName := line[1:]
						logName = "\x1b[0;33m" + processName + "\033[0m" + ": " + logName
						break
					}
				}
			}
			// Добавляем в список
			app.logfiles = append(app.logfiles, Logfile{
				name: "[" + "\033[34m" + formattedDate + "\033[0m" + "] " + logName,
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
	app.logfilesNotFilter = app.logfiles
	app.applyFilterList()
}

func (app *App) loadWinFiles(logPath string) {
	// Узнать имя пользователя (app.userName) и диск с виндой
	switch {
	case logPath == "ProgramFiles":
		logPath = "C:\\Program Files"
	case logPath == "ProgramFiles86":
		logPath = "C:\\Program Files (x86)"
	case logPath == "AppDataLocal":
		logPath = "C:\\Users\\" + app.userName + "\\AppData\\Local"
	case logPath == "AppDataRoaming":
		logPath = "C:\\Users\\" + app.userName + "\\AppData\\Roaming"
	}
	// Ищем файлы с помощью WalkDir
	var files []string
	// Доступ к срезу files из нескольких горутин
	var mu sync.Mutex
	// Группа ожидания для отслеживания завершения всех горутин
	var wg sync.WaitGroup
	// Получаем список корневых директорий
	rootDirs, _ := os.ReadDir(logPath)
	for _, rootDir := range rootDirs {
		// Проверяем, является ли текущий элемент директорие
		if rootDir.IsDir() {
			// Увеличиваем счетчик ожидаемых горутин
			wg.Add(1)
			go func(dir string) {
				// Уменьшаем счетчик горутин после завершения текущей
				defer wg.Done()
				// Рекурсивно обходим все файлы и подкаталоги в текущей директории
				err := filepath.WalkDir(filepath.Join(logPath, dir), func(path string, d os.DirEntry, err error) error {
					if err != nil {
						// Игнорируем ошибки, чтобы не прерывать поиск
						return nil
					}
					// Проверяем, что текущий элемент не является директорией и имеет расширение .log
					if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".log") {
						// Получаем относительный путь (без корневого пути logPath)
						relPath, _ := filepath.Rel(logPath, path)
						// Используем мьютекс для добавления файла в срез
						mu.Lock()
						files = append(files, relPath)
						mu.Unlock()
					}
					return nil
				})
				if err != nil {
					return
				}
			}(
				// Передаем имя текущей директории в горутину
				rootDir.Name(),
			)
		}
	}
	// Ждем завершения всех запущенных горутин
	wg.Wait()
	// Объединяем все пути в одну строку, разделенную символом новой строки
	output := strings.Join(files, "\n")
	// Если список файлов пустой, возвращаем ошибку
	if len(files) == 0 || (len(files) == 1 && files[0] == "") {
		vError, _ := app.gui.View("varLogs")
		vError.Clear()
		app.fileSystemFrameColor = gocui.ColorRed
		vError.FrameColor = app.fileSystemFrameColor
		vError.Highlight = false
		fmt.Fprintln(vError, "\033[31mPermission denied (files not found)\033[0m")
		return
	} else {
		vError, _ := app.gui.View("varLogs")
		app.fileSystemFrameColor = gocui.ColorDefault
		if vError.FrameColor != gocui.ColorDefault {
			vError.FrameColor = gocui.ColorGreen
		}
		vError.Highlight = true
	}
	serviceMap := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		// Формируем полный путь к файлу
		logFullPath := logPath + "\\" + scanner.Text()
		// Формируем имя файла для списка
		logName := scanner.Text()
		logName = strings.TrimSuffix(logName, ".log")
		logName = strings.ReplaceAll(logName, "\\", " ")
		// Получаем информацию о файле
		fileInfo, err := os.Stat(logFullPath)
		// Пропускаем файлы, к которым нет доступа
		if err != nil {
			continue
		}
		// Пропускаем пустые файлы
		if fileInfo.Size() == 0 {
			continue
		}
		// Получаем дату изменения
		modTime := fileInfo.ModTime()
		// Форматирование даты в формат DD.MM.YYYY
		formattedDate := modTime.Format("02.01.2006")
		// Проверяем, что полного пути до файла еще нет в списке
		if logName != "" && !serviceMap[logFullPath] {
			// Добавляем путь в массив для проверки уникальных путей
			serviceMap[logFullPath] = true
			// Добавляем в список
			app.logfiles = append(app.logfiles, Logfile{
				name: "[" + "\033[34m" + formattedDate + "\033[0m" + "] " + logName,
				path: logFullPath,
			})
		}
	}
	// Сортируем по дате
	sort.Slice(app.logfiles, func(i, j int) bool {
		layout := "02.01.2006"
		dateI, _ := time.Parse(layout, extractDate(app.logfiles[i].name))
		dateJ, _ := time.Parse(layout, extractDate(app.logfiles[j].name))
		return dateI.After(dateJ)
	})
	app.logfilesNotFilter = app.logfiles
	app.applyFilterList()
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
	app.maxVisibleFiles = viewHeight
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
	app.maxVisibleFiles = viewHeight
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
	// Обновляем счетчик в заголовке
	re := regexp.MustCompile(`\s\(.+\) >`)
	updateTitle := " (0) >"
	if len(app.logfiles) != 0 {
		updateTitle = " (" + strconv.Itoa(app.selectedFile+1) + "/" + strconv.Itoa(len(app.logfiles)) + ") >"
	}
	v.Title = re.ReplaceAllString(v.Title, updateTitle)
	if err := v.SetCursor(0, index); err != nil {
		return nil
	}
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
	app.loadFileLogs(strings.TrimSpace(line), true, g)
	app.lastWindow = "varLogs"
	app.lastSelected = strings.TrimSpace(line)
	return nil
}

// Функция для чтения файла с опредиление кодировки в Windows
func (app *App) loadWinFileLog(filePath string) (output []byte, stringErrors string) {
	// Открываем файл
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Sprintf("open file: %v", err)
	}
	defer file.Close()
	// Получаем информацию о файле
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Sprintf("get file stat: %v", err)
	}
	// Получаем размер файла
	fileSize := stat.Size()
	// Буфер для хранения последних строк
	var buffer []byte
	lineCount := 0
	// Размер буфера чтения (читаем по 1КБ за раз)
	readSize := int64(1024)
	// Преобразуем строку с максимальным количеством строк в int
	logViewCountInt, _ := strconv.Atoi(app.logViewCount)
	// Читаем файл с конца
	for fileSize > 0 && lineCount < logViewCountInt {
		if fileSize < readSize {
			readSize = fileSize
		}
		_, err := file.Seek(fileSize-readSize, 0)
		if err != nil {
			return nil, fmt.Sprintf("detect the end of a file via seek: %v", err)
		}
		tempBuffer := make([]byte, readSize)
		_, err = file.Read(tempBuffer)
		if err != nil {
			return nil, fmt.Sprintf("read file: %v", err)
		}
		buffer = append(tempBuffer, buffer...)
		lineCount = strings.Count(string(buffer), "\n")
		fileSize -= int64(readSize)
	}
	// Проверка на UTF-16 с BOM
	utf16withBOM := func(data []byte) bool {
		return len(data) >= 2 && ((data[0] == 0xFF && data[1] == 0xFE) || (data[0] == 0xFE && data[1] == 0xFF))
	}
	// Проверка на UTF-16 LE без BOM
	utf16withoutBOM := func(data []byte) bool {
		if len(data)%2 != 0 {
			return false
		}
		for i := 1; i < len(data); i += 2 {
			if data[i] != 0x00 {
				return false
			}
		}
		return true
	}
	var decodedOutput []byte
	switch {
	case utf16withBOM(buffer):
		// Декодируем UTF-16 с BOM
		decodedOutput, err = unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM).NewDecoder().Bytes(buffer)
		if err != nil {
			return nil, fmt.Sprintf("decoding from UTF-16 with BOM: %v", err)
		}
	case utf16withoutBOM(buffer):
		// Декодируем UTF-16 LE без BOM
		decodedOutput, err = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder().Bytes(buffer)
		if err != nil {
			return nil, fmt.Sprintf("decoding from UTF-16 LE without BOM: %v", err)
		}
	case utf8.Valid(buffer):
		// Декодируем UTF-8
		decodedOutput = buffer
	default:
		// Декодируем Windows-1251
		decodedOutput, err = charmap.Windows1251.NewDecoder().Bytes(buffer)
		if err != nil {
			return nil, fmt.Sprintf("decoding from Windows-1251: %v", err)
		}
	}
	return decodedOutput, "nil"
}

func (app *App) loadFileLogs(logName string, newUpdate bool, g *gocui.Gui) {
	// В параметре logName имя файла при выборе возвращяется без символов покраски
	// Получаем путь из массива по имени
	var logFullPath string
	var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	for _, logfile := range app.logfiles {
		// Удаляем покраску из имени файла в сохраненном массиве
		logfileName := ansiEscape.ReplaceAllString(logfile.name, "")
		if logfileName == logName {
			logFullPath = logfile.path
		}
	}
	if newUpdate {
		app.lastLogPath = logFullPath
		// Фиксируем новую дату изменения и размер для выбранного файла
		fileInfo, err := os.Stat(logFullPath)
		if err != nil {
			return
		}
		fileModTime := fileInfo.ModTime()
		fileSize := fileInfo.Size()
		app.lastDateUpdateFile = fileModTime
		app.lastSizeFile = fileSize
		app.updateFile = true
	} else {
		logFullPath = app.lastLogPath
		// Проверяем дату изменения
		fileInfo, err := os.Stat(logFullPath)
		if err != nil {
			return
		}
		fileModTime := fileInfo.ModTime()
		fileSize := fileInfo.Size()
		// Обновлять файл в горутине, только если есть изменения (проверяем дату модификации и размер)
		if fileModTime != app.lastDateUpdateFile || fileSize != app.lastSizeFile {
			app.lastDateUpdateFile = fileModTime
			app.lastSizeFile = fileSize
			app.updateFile = true
		} else {
			app.updateFile = false
		}
	}
	// Читаем файл, толькое если были изменения
	if app.updateFile {
		// Читаем логи в системе Windows
		if app.getOS == "windows" {
			decodedOutput, stringErrors := app.loadWinFileLog(logFullPath)
			if stringErrors != "nil" {
				v, _ := app.gui.View("logs")
				v.Clear()
				fmt.Fprintln(v, "\033[31mError", stringErrors, "\033[0m")
				return
			}
			app.currentLogLines = strings.Split(string(decodedOutput), "\n")
		} else {
			// Читаем логи в системах UNIX (Linux/Darwin/*BSD)
			switch {
			// Читаем архивные логи (decompress + stdout)
			case strings.HasSuffix(logFullPath, ".gz"):
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
					v, _ := app.gui.View("logs")
					v.Clear()
					fmt.Fprintln(v, " \033[31mError reading archive log using gzip tool.\n", err, "\033[0m")
					return
				}
				if err := cmdTail.Wait(); err != nil {
					v, _ := app.gui.View("logs")
					v.Clear()
					fmt.Fprintln(v, " \033[31mError reading log using tail tool.\n", err, "\033[0m")
					return
				}
				// Выводим содержимое
				app.currentLogLines = strings.Split(string(output), "\n")
			// Читаем архивные логи в формате bz2 в FreeBSD
			case strings.HasSuffix(logFullPath, ".bz2"):
				cmdBzip2 := exec.Command("bzip2", "-dc", logFullPath)
				cmdTail := exec.Command("tail", "-n", app.logViewCount)
				pipe, err := cmdBzip2.StdoutPipe()
				if err != nil {
					log.Fatalf("Error creating pipe: %v", err)
				}
				cmdTail.Stdin = pipe
				out, err := cmdTail.StdoutPipe()
				if err != nil {
					log.Fatalf("Error creating stdout pipe for tail: %v", err)
				}
				if err := cmdBzip2.Start(); err != nil {
					log.Fatalf("Error starting gzip: %v", err)
				}
				if err := cmdTail.Start(); err != nil {
					log.Fatalf("Error starting tail: %v", err)
				}
				output, err := io.ReadAll(out)
				if err != nil {
					log.Fatalf("Error reading output from tail: %v", err)
				}
				if err := cmdBzip2.Wait(); err != nil {
					v, _ := app.gui.View("logs")
					v.Clear()
					fmt.Fprintln(v, " \033[31mError reading archive log using gzip tool.\n", err, "\033[0m")
					return
				}
				if err := cmdTail.Wait(); err != nil {
					v, _ := app.gui.View("logs")
					v.Clear()
					fmt.Fprintln(v, " \033[31mError reading log using tail tool.\n", err, "\033[0m")
					return
				}
				app.currentLogLines = strings.Split(string(output), "\n")
			// Читаем бинарные файлы с помощью last/lastb для wtmp/btmp, а также utmp (OpenBSD) и utx.log (FreeBSD)
			case strings.Contains(logFullPath, "wtmp") || strings.Contains(logFullPath, "utmp") || strings.Contains(logFullPath, "utx.log"):
				cmd := exec.Command("last", "-f", logFullPath)
				output, err := cmd.Output()
				if err != nil {
					v, _ := app.gui.View("logs")
					v.Clear()
					fmt.Fprintln(v, " \033[31mError reading log using last tool.\n", err, "\033[0m")
					return
				}
				// Разбиваем вывод на строки
				lines := strings.Split(string(output), "\n")
				var filteredLines []string
				// Фильтруем строки, исключая последнюю строку и пустые строки
				for _, line := range lines {
					trimmedLine := strings.TrimSpace(line)
					if trimmedLine != "" && !strings.Contains(trimmedLine, "begins") {
						filteredLines = append(filteredLines, trimmedLine)
					}
				}
				// Переворачиваем порядок строк
				for i, j := 0, len(filteredLines)-1; i < j; i, j = i+1, j-1 {
					filteredLines[i], filteredLines[j] = filteredLines[j], filteredLines[i]
				}
				app.currentLogLines = filteredLines
			case strings.Contains(logFullPath, "btmp"):
				cmd := exec.Command("lastb", "-f", logFullPath)
				output, err := cmd.Output()
				if err != nil {
					v, _ := app.gui.View("logs")
					v.Clear()
					fmt.Fprintln(v, " \033[31mError reading log using lastb tool.\n", err, "\033[0m")
					return
				}
				lines := strings.Split(string(output), "\n")
				var filteredLines []string
				for _, line := range lines {
					trimmedLine := strings.TrimSpace(line)
					if trimmedLine != "" && !strings.Contains(trimmedLine, "begins") {
						filteredLines = append(filteredLines, trimmedLine)
					}
				}
				for i, j := 0, len(filteredLines)-1; i < j; i, j = i+1, j-1 {
					filteredLines[i], filteredLines[j] = filteredLines[j], filteredLines[i]
				}
				app.currentLogLines = filteredLines
			// Выводим содержимое из команды lastlog
			case strings.HasSuffix(logFullPath, "lastlog"):
				cmd := exec.Command("lastlog")
				output, err := cmd.Output()
				if err != nil {
					v, _ := app.gui.View("logs")
					v.Clear()
					fmt.Fprintln(v, " \033[31mError reading log using lastlog tool.\n", err, "\033[0m")
					return
				}
				app.currentLogLines = strings.Split(string(output), "\n")
			// lastlogin for FreeBSD
			case strings.HasSuffix(logFullPath, "lastlogin"):
				cmd := exec.Command("lastlogin")
				output, err := cmd.Output()
				if err != nil {
					v, _ := app.gui.View("logs")
					v.Clear()
					fmt.Fprintln(v, " \033[31mError reading log using lastlogin tool.\n", err, "\033[0m")
					return
				}
				app.currentLogLines = strings.Split(string(output), "\n")
			// Packet Filter (PF) Firewall OpenBSD
			case strings.HasSuffix(logFullPath, "pflog"):
				cmd := exec.Command("tcpdump", "-e", "-n", "-r", logFullPath)
				output, err := cmd.Output()
				if err != nil {
					v, _ := app.gui.View("logs")
					v.Clear()
					fmt.Fprintln(v, " \033[31mError reading log using tcpdump tool.\n", err, "\033[0m")
					return
				}
				app.currentLogLines = strings.Split(string(output), "\n")
			// Packet Capture (pcap format)
			case strings.HasSuffix(logFullPath, "pcap"):
				cmd := exec.Command("tcpdump", "-n", "-r", logFullPath)
				output, err := cmd.Output()
				if err != nil {
					v, _ := app.gui.View("logs")
					v.Clear()
					fmt.Fprintln(v, " \033[31mError reading log using tcpdump tool.\n", err, "\033[0m")
					return
				}
				app.currentLogLines = strings.Split(string(output), "\n")
			default:
				cmd := exec.Command("tail", "-n", app.logViewCount, logFullPath)
				output, err := cmd.Output()
				if err != nil {
					v, _ := app.gui.View("logs")
					v.Clear()
					fmt.Fprintln(v, " \033[31mError reading log using tail tool.\n", err, "\033[0m")
					return
				}
				app.currentLogLines = strings.Split(string(output), "\n")
			}
		}
		app.updateDelimiter(newUpdate, g)
		app.applyFilter(false)
	}
}

// ---------------------------------------- Docker/Podman ----------------------------------------

func (app *App) loadDockerContainer(containerizationSystem string) {
	// Получаем версию для проверки, что система контейнеризации установлена
	cmd := exec.Command(containerizationSystem, "--version")
	_, err := cmd.Output()
	if err != nil {
		vError, _ := app.gui.View("docker")
		vError.Clear()
		app.dockerFrameColor = gocui.ColorRed
		vError.FrameColor = app.dockerFrameColor
		vError.Highlight = false
		fmt.Fprintln(vError, "\033[31m"+containerizationSystem+" not installed (environment not found)\033[0m")
		return
	}
	cmd = exec.Command(containerizationSystem, "ps", "-a", "--format", "{{.ID}} {{.Names}} {{.State}}")
	output, err := cmd.Output()
	if err != nil {
		vError, _ := app.gui.View("docker")
		vError.Clear()
		app.dockerFrameColor = gocui.ColorRed
		vError.FrameColor = app.dockerFrameColor
		vError.Highlight = false
		fmt.Fprintln(vError, "\033[31mAccess denied or "+containerizationSystem+" not running\033[0m")
		return
	} else {
		vError, _ := app.gui.View("docker")
		app.dockerFrameColor = gocui.ColorDefault
		vError.Highlight = true
		if vError.FrameColor != gocui.ColorDefault {
			vError.FrameColor = gocui.ColorGreen
		}
	}
	containers := strings.Split(strings.TrimSpace(string(output)), "\n")
	// Проверяем, что список контейнеров не пустой
	if len(containers) == 0 || (len(containers) == 1 && containers[0] == "") {
		vError, _ := app.gui.View("docker")
		vError.Clear()
		vError.Highlight = false
		fmt.Fprintln(vError, "\033[32mNo running containers\033[0m")
		return
	} else {
		vError, _ := app.gui.View("docker")
		app.fileSystemFrameColor = gocui.ColorDefault
		if vError.FrameColor != gocui.ColorDefault {
			vError.FrameColor = gocui.ColorGreen
		}
		vError.Highlight = true
	}
	serviceMap := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		idName := scanner.Text()
		parts := strings.Fields(idName)
		if idName != "" && !serviceMap[idName] {
			serviceMap[idName] = true
			containerStatus := parts[2]
			if containerStatus == "running" {
				containerStatus = "\033[32m" + containerStatus + "\033[0m"
			} else {
				containerStatus = "\033[31m" + containerStatus + "\033[0m"
			}
			containerName := parts[1] + " (" + containerStatus + ")"
			app.dockerContainers = append(app.dockerContainers, DockerContainers{
				name: containerName,
				id:   parts[0],
			})
		}
	}
	sort.Slice(app.dockerContainers, func(i, j int) bool {
		return app.dockerContainers[i].name < app.dockerContainers[j].name
	})
	app.dockerContainersNotFilter = app.dockerContainers
	app.applyFilterList()
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
	app.maxVisibleDockerContainers = viewHeight
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
	app.maxVisibleDockerContainers = viewHeight
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
	// Обновляем счетчик в заголовке
	re := regexp.MustCompile(`\s\(.+\) >`)
	updateTitle := " (0) >"
	if len(app.dockerContainers) != 0 {
		updateTitle = " (" + strconv.Itoa(app.selectedDockerContainer+1) + "/" + strconv.Itoa(len(app.dockerContainers)) + ") >"
	}
	v.Title = re.ReplaceAllString(v.Title, updateTitle)
	if err := v.SetCursor(0, index); err != nil {
		return nil
	}
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
	app.loadDockerLogs(strings.TrimSpace(line), true, g)
	app.lastWindow = "docker"
	app.lastSelected = strings.TrimSpace(line)
	return nil
}

func (app *App) loadDockerLogs(containerName string, newUpdate bool, g *gocui.Gui) {
	containerizationSystem := app.selectContainerizationSystem
	// Сохраняем систему контейнеризации для автообновления при смене окна
	if newUpdate {
		app.lastContainerizationSystem = app.selectContainerizationSystem
	} else {
		containerizationSystem = app.lastContainerizationSystem
	}
	var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	var containerId string
	for _, dockerContainer := range app.dockerContainers {
		dockerContainerName := ansiEscape.ReplaceAllString(dockerContainer.name, "")
		if dockerContainerName == containerName {
			containerId = dockerContainer.id
		}
	}
	// Сохраняем id контейнера для автообновления при смене окна
	if newUpdate {
		app.lastContainerId = containerId
	} else {
		containerId = app.lastContainerId
	}
	// Читаем локальный лог Docker в формате JSON
	var readFileContainer bool = false
	if containerizationSystem == "docker" {
		basePath := "/var/lib/docker/containers"
		var logFilePath string
		// Ищем файл лога в локальной системе по id
		_ = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err == nil && strings.Contains(info.Name(), containerId) && strings.HasSuffix(info.Name(), "-json.log") {
				logFilePath = path
				// Фиксируем, если найден файловый журнал
				readFileContainer = true
				// Останавливаем поиск
				return filepath.SkipDir
			}
			return nil
		})
		// Читаем файл с конца с помощью tail
		if readFileContainer {
			cmd := exec.Command("tail", "-n", app.logViewCount, logFilePath)
			output, err := cmd.Output()
			if err != nil {
				v, _ := app.gui.View("logs")
				v.Clear()
				fmt.Fprintln(v, "\033[31mError reading log:", err, "\033[0m")
				return
			}
			// Разбиваем строки на массив
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			var formattedLines []string
			// Обрабатываем вывод в формате JSON построчно
			for i, line := range lines {
				// JSON-структура для парсинга
				var jsonData map[string]interface{}
				err := json.Unmarshal([]byte(line), &jsonData)
				if err != nil {
					continue
				}
				// Извлекаем JSON данные
				stream, _ := jsonData["stream"].(string)
				timeStr, _ := jsonData["time"].(string)
				logMessage, _ := jsonData["log"].(string)
				// Удаляем встроенный экранированный символ переноса строки
				logMessage = strings.TrimSuffix(logMessage, "\n")
				// Парсим строку времени в объект time.Time
				parsedTime, err := time.Parse(time.RFC3339Nano, timeStr)
				if err == nil {
					// Форматируем дату в формат: DD:MM:YYYY HH:MM:SS
					timeStr = parsedTime.Format("02.01.2006 15:04:05")
				}
				// Заполняем строку в формате: stream time: log
				formattedLine := fmt.Sprintf("%s %s: %s", stream, timeStr, logMessage)
				formattedLines = append(formattedLines, formattedLine)
				// Если это последняя строка в выводе, добавляем перенос строки
				if i == len(lines)-1 {
					formattedLines = append(formattedLines, "\n")
				}
			}
			app.currentLogLines = formattedLines
		}
	}
	// Читаем лог через Podman или Docker cli (если файл не найден)
	if containerizationSystem == "podman" || !readFileContainer {
		cmd := exec.Command(containerizationSystem, "logs", "--tail", app.logViewCount, containerId)
		output, err := cmd.Output()
		if err != nil {
			v, _ := app.gui.View("logs")
			v.Clear()
			fmt.Fprintln(v, "\033[31mError getting logs from", containerName, "(id:", containerId, ")", "container.", err, "\033[0m")
			return
		}
		app.currentLogLines = strings.Split(string(output), "\n")
	}
	app.updateDelimiter(newUpdate, g)
	app.applyFilter(false)
}

// ---------------------------------------- Filter ----------------------------------------

// Редактор обработки ввода текста для фильтрации
func (app *App) createFilterEditor(window string) gocui.Editor {
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
		if window == "logs" {
			// Обновляем текст в буфере
			app.filterText = strings.TrimSpace(v.Buffer())
			// Применяем функцию фильтрации к выводу записей журнала
			app.applyFilter(true)
		} else if window == "lists" {
			app.filterListText = strings.TrimSpace(v.Buffer())
			app.applyFilterList()
		}
	})
}

// Функция для фильтрации всех списоков журналов
func (app *App) applyFilterList() {
	filter := strings.ToLower(app.filterListText)
	// Временные массивы для отфильтрованных журналов
	var filteredJournals []Journal
	var filteredLogFiles []Logfile
	var filteredDockerContainers []DockerContainers
	for _, j := range app.journalsNotFilter {
		if strings.Contains(strings.ToLower(j.name), filter) {
			filteredJournals = append(filteredJournals, j)
		}
	}
	for _, j := range app.logfilesNotFilter {
		if strings.Contains(strings.ToLower(j.name), filter) {
			filteredLogFiles = append(filteredLogFiles, j)
		}
	}
	for _, j := range app.dockerContainersNotFilter {
		if strings.Contains(strings.ToLower(j.name), filter) {
			filteredDockerContainers = append(filteredDockerContainers, j)
		}
	}
	// Сбрасываем индексы выбранного журнала для правильного позиционирования
	app.selectedJournal = 0
	app.selectedFile = 0
	app.selectedDockerContainer = 0
	app.startServices = 0
	app.startFiles = 0
	app.startDockerContainers = 0
	// Сохраняем отфильтрованные и отсортированные данные
	app.journals = filteredJournals
	app.logfiles = filteredLogFiles
	app.dockerContainers = filteredDockerContainers
	// Обновляем списки в интерфейсе
	app.updateServicesList()
	app.updateLogsList()
	app.updateDockerContainerList()
	// Обновляем статус количества служб
	v, _ := app.gui.View("services")
	// Обновляем счетчик в заголовке
	re := regexp.MustCompile(`\s\(.+\) >`)
	updateTitle := " (0) >"
	if len(app.journals) != 0 {
		updateTitle = " (" + strconv.Itoa(app.selectedJournal+1) + "/" + strconv.Itoa(len(app.journals)) + ") >"
	}
	v.Title = re.ReplaceAllString(v.Title, updateTitle)
	// Обновляем статус количества файлов
	v, _ = app.gui.View("varLogs")
	// Обновляем счетчик в заголовке
	re = regexp.MustCompile(`\s\(.+\) >`)
	updateTitle = " (0) >"
	if len(app.logfiles) != 0 {
		updateTitle = " (" + strconv.Itoa(app.selectedFile+1) + "/" + strconv.Itoa(len(app.logfiles)) + ") >"
	}
	v.Title = re.ReplaceAllString(v.Title, updateTitle)
	// Обновляем статус количества контейнеров
	v, _ = app.gui.View("docker")
	// Обновляем счетчик в заголовке
	re = regexp.MustCompile(`\s\(.+\) >`)
	updateTitle = " (0) >"
	if len(app.dockerContainers) != 0 {
		updateTitle = " (" + strconv.Itoa(app.selectedDockerContainer+1) + "/" + strconv.Itoa(len(app.dockerContainers)) + ") >"
	}
	v.Title = re.ReplaceAllString(v.Title, updateTitle)
}

// Функция для фильтрации записей текущего журнала
func (app *App) applyFilter(color bool) {
	v, err := app.gui.View("filter")
	if err != nil {
		return
	}
	if color {
		v.FrameColor = gocui.ColorGreen
	}
	filter := app.filterText
	// Debug: если текст фильтра не менялся и позиция курсора не в самом конце журнала, то пропускаем фильтрацию и покраску при пролистывании
	var skip bool = false
	vLogs, _ := app.gui.View("logs")
	_, viewHeight := vLogs.Size()
	size := app.logScrollPos + viewHeight + 1
	if app.lastFilterText == filter && size < len(app.filteredLogLines) {
		skip = true
	}
	// Фиксируем текущий текст из фильтра
	app.lastFilterText = filter
	// Фильтруем и красим, только если это не строллинг
	if !skip {
		// Debug start time
		startTime := time.Now()
		// Debug: если текст фильтра пустой или равен любому символу, возвращяем вывод без фильтрации
		if filter == "" || filter == "." {
			app.filteredLogLines = app.currentLogLines
		} else {
			app.filteredLogLines = make([]string, 0)
			// Опускаем регистр ввода текста для фильтра
			filter = strings.ToLower(filter)
			// Проверка регулярного выражения
			var regex *regexp.Regexp
			if app.selectFilterMode == "regex" {
				// Добавляем флаг для нечувствительности к регистру по умолчанию
				filter = "(?i)" + filter
				// Компилируем регулярное выражение
				regex, err = regexp.Compile(filter)
				if err != nil {
					// В случае синтаксической ошибки регулярного выражения, красим окно красным цветом и завершаем цикл
					v.FrameColor = gocui.ColorRed
					return
				}
			}
			// Проходимся по каждой строке
			for _, line := range app.currentLogLines {
				// Fuzzy (неточный поиск без учета регистра)
				switch {
				case app.selectFilterMode == "fuzzy":
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
						originalLine = strings.ReplaceAll(originalLine, startColor, "\x1b[0;44m")
						originalLine = strings.ReplaceAll(originalLine, endColor, "\033[0m")
						app.filteredLogLines = append(app.filteredLogLines, originalLine)
					}
					// Regex (с использованием регулярных выражений Go и без учета регистра по умолчанию)
				case app.selectFilterMode == "regex":
					// Проверяем, что строка подходит под регулярное выражение
					if regex.MatchString(line) {
						originalLine := line
						// Находим все найденные совпадени
						matches := regex.FindAllString(originalLine, -1)
						// Красим только первое найденное совпадение
						originalLine = strings.ReplaceAll(originalLine, matches[0], "\x1b[0;44m"+matches[0]+"\033[0m")
						app.filteredLogLines = append(app.filteredLogLines, originalLine)
					}
					// Default (точный поиск с учетом регистра)
				default:
					filter = app.filterText
					if filter == "" || strings.Contains(line, filter) {
						lineColor := strings.ReplaceAll(line, filter, "\x1b[0;44m"+filter+"\033[0m")
						app.filteredLogLines = append(app.filteredLogLines, lineColor)
					}
				}
			}
		}
		// Пропускаем вывод построчно (синхронно) после фильтрации для покраски
		// var colorLogLines []string
		// for _, line := range app.filteredLogLines {
		// 	colorLine := app.lineColor(line)
		// 	colorLogLines = append(colorLogLines, colorLine)
		// }
		// app.filteredLogLines = colorLogLines
		// Максимальное количество потоков
		const maxWorkers = 10
		// Канал для передачи индексов всех строк
		tasks := make(chan int, len(app.filteredLogLines))
		// Срез для хранения обработанных строк
		colorLogLines := make([]string, len(app.filteredLogLines))
		// Объявляем группу ожидания для синхронизации всех горутин (воркеров)
		var wg sync.WaitGroup
		// Создаем maxWorkers горутин, где каждая будет обрабатывать задачи из канала tasks
		for i := 0; i < maxWorkers; i++ {
			go func() {
				// Горутина будет работать, пока в канале tasks есть задачи
				for index := range tasks {
					// Обрабатываем строку и сохраняем результат по соответствующему индексу
					colorLogLines[index] = app.lineColor(app.filteredLogLines[index])
					// Уменьшаем счетчик задач в группе ожидания.
					wg.Done()
				}
			}()
		}
		// Добавляем задачи в канал
		for i := range app.filteredLogLines {
			// Увеличиваем счетчик задач в группе ожидания.
			wg.Add(1)
			// Передаем индекс строки в канал tasks
			tasks <- i
		}
		// Закрываем канал задач, чтобы воркеры завершили работу после обработки всех задач
		close(tasks)
		// Ждем завершения всех задач
		wg.Wait()
		app.filteredLogLines = colorLogLines
		// Debug end time
		endTime := time.Since(startTime)
		app.debugLoadTime = endTime.Truncate(time.Millisecond).String()
	}
	// Debug: корректируем текущую позицию скролла, если размер массива стал меньше
	if size > len(app.filteredLogLines) {
		newScrollPos := len(app.filteredLogLines) - viewHeight
		if newScrollPos > 0 {
			app.logScrollPos = newScrollPos
		} else {
			app.logScrollPos = 0
		}
	}
	// Обновляем окно для отображения отфильтрованных записей
	if app.autoScroll {
		app.logScrollPos = 0
		app.updateLogsView(true)
	} else {
		app.updateLogsView(false)
	}
}

// ---------------------------------------- Coloring ----------------------------------------

// Функция для покраски строки
func (app *App) lineColor(inputLine string) string {
	// Разбиваем строку на слова
	words := strings.Fields(inputLine)
	var colorLine string
	var filterColor bool = false
	for _, word := range words {
		// Исключаем строки с покраской при поиске (Background)
		if strings.Contains(word, "\x1b[0;44m") {
			filterColor = true
		}
		// Красим слово в функции
		if !filterColor {
			word = app.wordColor(word)
		}
		// Возобновляем покраску
		if strings.Contains(word, "\033[0m") {
			filterColor = false
		}
		colorLine += word + " "
	}
	return strings.TrimSpace(colorLine)
}

// Игнорируем регистр и проверяем, что слово окружено границами (не буквы и цифры)
func (app *App) replaceWordLower(word, keyword, color string) string {
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(keyword) + `\b`)
	return re.ReplaceAllStringFunc(word, func(match string) string {
		return color + match + "\033[0m"
	})
}

// Поиск пользователей
func (app *App) containsUser(searchWord string) bool {
	for _, user := range app.userNameArray {
		if user == searchWord {
			return true
		}
	}
	return false
}

// Поиск корневых директорий
func (app *App) containsPath(searchWord string) bool {
	for _, dir := range app.rootDirArray {
		if strings.Contains(searchWord, dir) {
			return true
		}
	}
	return false
}

// Функция для покраски словосочетаний
func (app *App) wordColor(inputWord string) string {
	// Опускаем регистр слова
	inputWordLower := strings.ToLower(inputWord)
	// Значение по умолчанию
	var coloredWord string = inputWord
	switch {
	// Желтый (известные имена: hostname и username) [33m]
	case strings.Contains(inputWord, app.hostName):
		coloredWord = strings.ReplaceAll(inputWord, app.hostName, "\033[33m"+app.hostName+"\033[0m")
	case strings.Contains(inputWord, app.userName):
		coloredWord = strings.ReplaceAll(inputWord, app.userName, "\033[33m"+app.userName+"\033[0m")
	case app.containsUser(inputWord):
		coloredWord = app.replaceWordLower(inputWord, inputWord, "\033[33m")
	case strings.Contains(inputWordLower, "warn"):
		words := []string{"warnings", "warning", "warn"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[33m")
				break
			}
		}
	// Custom (UNIX processes)
	case app.syslogUnitRegex.MatchString(inputWord):
		unitSplit := strings.Split(inputWord, "[")
		unitName := unitSplit[0]
		unitId := strings.ReplaceAll(unitSplit[1], "]:", "")
		coloredWord = strings.ReplaceAll(inputWord, inputWord, "\033[36m"+unitName+"\033[0m"+"\033[33m"+"["+"\033[0m"+"\033[34m"+unitId+"\033[0m"+"\033[33m"+"]"+"\033[0m"+":")
	case strings.HasPrefix(inputWordLower, "kernel:"):
		coloredWord = app.replaceWordLower(inputWord, "kernel", "\033[36m")
	case strings.HasPrefix(inputWordLower, "rsyslogd:"):
		coloredWord = app.replaceWordLower(inputWord, "rsyslogd", "\033[36m")
	case strings.HasPrefix(inputWordLower, "sudo:"):
		coloredWord = app.replaceWordLower(inputWord, "sudo", "\033[36m")
	// Красный (ошибки) [31m]
	case strings.Contains(inputWordLower, "err"):
		words := []string{"stderr", "errors", "error", "erro", "err"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "dis"):
		words := []string{"disconnected", "disconnection", "disconnects", "disconnect", "disabled", "disabling", "disable"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "crash"):
		words := []string{"crashed", "crashing", "crash"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "delet"):
		words := []string{"deletion", "deleted", "deleting", "deletes", "delete"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "remov"):
		words := []string{"removing", "removed", "removes", "remove"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "stop"):
		words := []string{"stopping", "stopped", "stops", "stop"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "invalid"):
		words := []string{"invalidation", "invalidating", "invalidated", "invalidate", "invalid"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "abort"):
		words := []string{"aborted", "aborting", "abort"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "block"):
		words := []string{"blocked", "blocker", "blocking", "blocks", "block"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "activ"):
		words := []string{"inactive", "deactivated", "deactivating", "deactivate"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "exit"):
		words := []string{"exited", "exiting", "exits", "exit"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "crit"):
		words := []string{"critical", "crit"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "fail"):
		words := []string{"failed", "failure", "failing", "fails", "fail"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "reject"):
		words := []string{"rejection", "rejecting", "reject"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "fatal"):
		words := []string{"fatality", "fataling", "fatals", "fatal"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "close"):
		words := []string{"closed", "closing", "close"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "end"):
		words := []string{"ended", "ending", "ends", "end"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "drop"):
		words := []string{"dropped", "droping", "drops", "drop"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "kill"):
		words := []string{"killer", "killing", "kills", "kill"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[31m")
				break
			}
		}
	case strings.Contains(inputWordLower, "unavailable"):
		coloredWord = app.replaceWordLower(inputWord, "unavailable", "\033[31m")
	case strings.Contains(inputWordLower, "unsuccessful"):
		coloredWord = app.replaceWordLower(inputWord, "unsuccessful", "\033[31m")
	case strings.Contains(inputWordLower, "found"):
		coloredWord = app.replaceWordLower(inputWord, "found", "\033[31m")
	case strings.Contains(inputWordLower, "denied"):
		coloredWord = app.replaceWordLower(inputWord, "denied", "\033[31m")
	case strings.Contains(inputWordLower, "conflict"):
		coloredWord = app.replaceWordLower(inputWord, "conflict", "\033[31m")
	case strings.Contains(inputWordLower, "false"):
		coloredWord = app.replaceWordLower(inputWord, "false", "\033[31m")
	case strings.Contains(inputWordLower, "none"):
		coloredWord = app.replaceWordLower(inputWord, "none", "\033[31m")
	// Исключения
	case strings.Contains(inputWordLower, "/dev/null"):
		coloredWord = strings.ReplaceAll(inputWord, "/dev/null", "\033[35m"+"/dev/null"+"\033[0m")
	case strings.Contains(inputWordLower, "null"):
		coloredWord = app.replaceWordLower(inputWord, "null", "\033[31m")
	// Зеленый (успех) [32m]
	case strings.Contains(inputWordLower, "succe"):
		words := []string{"successfully", "successfull", "successful", "succeeded", "succeed", "success"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "complet"):
		words := []string{"completed", "completing", "completion", "complets", "complete"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "accept"):
		words := []string{"accepted", "accepting", "acception", "acceptance", "acceptable", "acceptably", "accepte", "accepts", "accept"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "connect"):
		words := []string{"connected", "connecting", "connection", "connects", "connect"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "finish"):
		words := []string{"finished", "finishing", "finish"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "start"):
		words := []string{"started", "starting", "start"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "creat"):
		words := []string{"created", "creating", "creates", "create"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "enable"):
		words := []string{"enabled", "enables", "enable"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "allow"):
		words := []string{"allowed", "allowing", "allow"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "post"):
		words := []string{"posting", "posted", "prerouting", "postrouting", "post"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "rout"):
		words := []string{"routing", "routes", "route"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "forward"):
		words := []string{"forwarding", "forwards", "forward"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "pass"):
		words := []string{"passed", "passing", "password"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "run"):
		words := []string{"running", "runs", "run"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "add"):
		words := []string{"added", "add"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "open"):
		words := []string{"opened", "open"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "patch"):
		words := []string{"patched", "patching", "patch"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[32m")
				break
			}
		}
	case strings.Contains(inputWordLower, "ok"):
		coloredWord = app.replaceWordLower(inputWord, "ok", "\033[32m")
	case strings.Contains(inputWordLower, "available"):
		coloredWord = app.replaceWordLower(inputWord, "available", "\033[32m")
	case strings.Contains(inputWordLower, "accessible"):
		coloredWord = app.replaceWordLower(inputWord, "accessible", "\033[32m")
	case strings.Contains(inputWordLower, "done"):
		coloredWord = app.replaceWordLower(inputWord, "done", "\033[32m")
	case strings.Contains(inputWordLower, "true"):
		coloredWord = app.replaceWordLower(inputWord, "true", "\033[32m")
	// Пурпурный (url и директории) [35m]
	case strings.Contains(inputWord, "http://"):
		cleanedWord := app.trimHttpRegex.ReplaceAllString(inputWord, "")
		coloredWord = strings.ReplaceAll(inputWord, "http://"+cleanedWord, "\033[35m"+"http://"+cleanedWord+"\033[0m")
	case strings.Contains(inputWord, "https://"):
		cleanedWord := app.trimHttpsRegex.ReplaceAllString(inputWord, "")
		coloredWord = strings.ReplaceAll(inputWord, "https://"+cleanedWord, "\033[35m"+"https://"+cleanedWord+"\033[0m")
	case app.containsPath(inputWord):
		cleanedWord := app.trimPrefixPathRegex.ReplaceAllString(inputWord, "")
		cleanedWord = app.trimPostfixPathRegex.ReplaceAllString(cleanedWord, "")
		coloredWord = strings.ReplaceAll(inputWord, cleanedWord, "\033[35m"+cleanedWord+"\033[0m")
	// Синий (статусы) [36m]
	case strings.Contains(inputWordLower, "out"):
		words := []string{"stdout", "timeout", "output"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "put"):
		words := []string{"input", "put"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "get"):
		words := []string{"getting", "get"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "set"):
		words := []string{"settings", "setting", "setup", "set"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "head"):
		words := []string{"headers", "header", "heades", "head"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "re"):
		words := []string{"requested", "requests", "request", "resolved", "resolving", "resolve", "reply", "restarting", "restarted", "restart", "reboot", "registered", "registeration", "reading", "readed", "read"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "lo"):
		words := []string{"uploading", "uploaded", "uploads", "upload", "downloading", "downloaded", "downloads", "download", "loading", "loaded", "load", "logged", "login"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "listen"):
		words := []string{"listening", "listener", "listen"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "launch"):
		words := []string{"launched", "launching", "launch"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "chang"):
		words := []string{"changed", "changing", "change"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "clea"):
		words := []string{"cleaning", "cleaner", "clearing", "cleared", "clear"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "skip"):
		words := []string{"skipping", "skipped", "skip"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "miss"):
		words := []string{"missing", "missed"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "mount"):
		words := []string{"mountpoint", "mounted", "mounting", "mount"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "auth"):
		words := []string{"authenticating", "authentication", "authorization"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "conf"):
		words := []string{"configurations", "configuration", "configuring", "configured", "configure", "config", "conf"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "option"):
		words := []string{"options", "option"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "writ"):
		words := []string{"writing", "writed", "write"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "sav"):
		words := []string{"saved", "saving", "save"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "boot"):
		words := []string{"booting", "boot"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "paus"):
		words := []string{"paused", "pausing", "pause"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "filt"):
		words := []string{"filtration", "filtr", "filtering", "filtered", "filter"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "norm"):
		words := []string{"normal", "norm"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "noti"):
		words := []string{"notifications", "notification", "notify", "noting", "notice"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "alert"):
		words := []string{"alerting", "alert"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "in"):
		words := []string{"informations", "information", "informing", "informed", "info", "installation", "installed", "installing", "install", "initialization", "initial", "using"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "up"):
		words := []string{"updates", "updated", "updating", "update", "upgrades", "upgraded", "upgrading", "upgrade", "backup", "up"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "down"):
		words := []string{"shutdown", "down"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "us"):
		words := []string{"status", "used", "use"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[36m")
				break
			}
		}
	case strings.Contains(inputWordLower, "debug"):
		coloredWord = app.replaceWordLower(inputWord, "debug", "\033[36m")
	case strings.HasPrefix(inputWordLower, "trace"):
		coloredWord = app.replaceWordLower(inputWord, "trace", "\033[36m")
	case strings.HasPrefix(inputWordLower, "protocol"):
		coloredWord = app.replaceWordLower(inputWord, "protocol", "\033[36m")
	case strings.Contains(inputWordLower, "level"):
		coloredWord = app.replaceWordLower(inputWord, "level", "\033[36m")
	// Голубой (цифры: byte/time+mac/date+ip/%) [34m]
	case app.hexByteRegex.MatchString(inputWord):
		coloredWord = app.hexByteRegex.ReplaceAllStringFunc(inputWord, func(match string) string {
			return "\033[34m" + match + "\033[0m"
		})
	case app.dateTimeRegex.MatchString(inputWord):
		coloredWord = app.dateTimeRegex.ReplaceAllStringFunc(inputWord, func(match string) string {
			return "\033[34m" + match + "\033[0m"
		})
	case app.timeMacAddressRegex.MatchString(inputWord):
		coloredWord = app.timeMacAddressRegex.ReplaceAllStringFunc(inputWord, func(match string) string {
			return "\033[34m" + match + "\033[0m"
		})
	case app.dateIpAddressRegex.MatchString(inputWord):
		coloredWord = app.dateIpAddressRegex.ReplaceAllStringFunc(inputWord, func(match string) string {
			return "\033[34m" + match + "\033[0m"
		})
	case strings.Contains(inputWordLower, "%"):
		coloredWord = app.procRegex.ReplaceAllStringFunc(inputWord, func(match string) string {
			return "\033[34m" + match + "\033[0m"
		})
	// Update delimiter
	case strings.Contains(inputWord, "⎯"):
		coloredWord = app.replaceWordLower(inputWord, "⎯", "\033[32m")
	// tcpdump
	case strings.Contains(inputWordLower, "tcp"):
		coloredWord = app.replaceWordLower(inputWord, "tcp", "\033[33m")
	case strings.Contains(inputWordLower, "udp"):
		coloredWord = app.replaceWordLower(inputWord, "udp", "\033[33m")
	case strings.Contains(inputWordLower, "icmp"):
		coloredWord = app.replaceWordLower(inputWord, "udp", "\033[33m")
	case strings.Contains(inputWordLower, "ip"):
		words := []string{"ip6", "ipv6", "ip4", "ipv4", "ip"}
		for _, word := range words {
			if strings.Contains(inputWordLower, word) {
				coloredWord = app.replaceWordLower(inputWord, word, "\033[33m")
				break
			}
		}
	// Исключения
	case strings.Contains(inputWordLower, "not"):
		coloredWord = app.replaceWordLower(inputWord, "not", "\033[31m")
	}
	return coloredWord
}

// ---------------------------------------- Log output ----------------------------------------

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
	// Учитываем auto wrap (только в конце лога)
	if app.logScrollPos == len(app.filteredLogLines)-viewHeight-1 {
		var viewLines int = 0                             // количество строк для вывода
		var viewCounter int = 0                           // обратный счетчик видимых строк для остановки
		var viewIndex int = len(app.filteredLogLines) - 1 // начальный индекс для строк с конца
		for {
			// Фиксируем текущую входную строку и счетчик
			viewLines += 1
			viewCounter += 1
			// Получаем длинну видимых символов в строке с конца
			var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)
			lengthLine := len([]rune(ansiEscape.ReplaceAllString(app.filteredLogLines[viewIndex], "")))
			// Если длинна строки больше ширины окна, получаем количество дополнительных строк
			if lengthLine > viewWidth {
				// Увеличивая счетчик и пропускаем строки
				viewCounter += lengthLine / viewWidth
			}
			// Если счетчик привысил количество видимых строк, вычетаем последнюю строку из видимости
			if viewCounter > viewHeight {
				viewLines -= 1
			}
			if viewCounter >= viewHeight {
				break
			}
			// Уменьшаем индекс
			viewIndex -= 1
		}
		for i := len(app.filteredLogLines) - viewLines - 1; i < endLine; i++ {
			fmt.Fprintln(v, app.filteredLogLines[i])
		}
	} else {
		// Проходим по отфильтрованным строкам и выводим их
		for i := startLine; i < endLine; i++ {
			fmt.Fprintln(v, app.filteredLogLines[i])
		}
	}
	// Вычисляем процент прокрутки и обновляем заголовок
	if len(app.filteredLogLines) > 0 {
		// Стартовая позиция + размер текущего вывода логов и округляем в большую сторону (math)
		percentage := int(math.Ceil(float64((startLine+viewHeight)*100) / float64(len(app.filteredLogLines))))
		if percentage > 100 {
			v.Title = fmt.Sprintf("Logs: 100%% (%d) ["+app.debugLoadTime+"]", len(app.filteredLogLines)) // "Logs: 100%% (%d) [Max lines: "+app.logViewCount+"/Load time: "+app.debugLoadTime+"]"
		} else {
			v.Title = fmt.Sprintf("Logs: %d%% (%d/%d) ["+app.debugLoadTime+"]", percentage, startLine+1+viewHeight, len(app.filteredLogLines))
		}
	} else {
		v.Title = "Logs: 0% (0) [" + app.debugLoadTime + "]"
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
			// Включаем автоскролл
			app.autoScroll = true
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
	// Отключаем автоскролл
	app.autoScroll = false
	app.updateLogsView(false)
	return nil
}

// Функция для очистки поля ввода фильтра
func (app *App) clearFilterEditor(g *gocui.Gui) {
	v, _ := g.View("filter")
	// Очищаем содержимое View
	v.Clear()
	// Устанавливаем курсор на начальную позицию
	if err := v.SetCursor(0, 0); err != nil {
		return
	}
	// Очищаем буфер фильтра
	app.filterText = ""
	app.applyFilter(false)
}

// Функция для обновления последнего выбранного вывода лога
func (app *App) updateLogOutput(seconds int) error {
	for {
		// Выполняем обновление интерфейса через метод Update для иницилизации перерисовки интерфейса
		app.gui.Update(func(g *gocui.Gui) error {
			// Сбрасываем автоскролл, если это ручное обновление, что бы опустить журнал вниз
			if seconds == 0 {
				app.autoScroll = true
			}
			switch app.lastWindow {
			case "services":
				app.loadJournalLogs(app.lastSelected, false, g)
			case "varLogs":
				app.loadFileLogs(app.lastSelected, false, g)
			case "docker":
				app.loadDockerLogs(app.lastSelected, false, g)
			}
			return nil
		})
		if seconds == 0 {
			break
		}
		time.Sleep(time.Duration(seconds) * time.Second)
	}
	return nil
}

// Функция для фиксации места загрузки журнала с помощью делиметра
func (app *App) updateDelimiter(newUpdate bool, g *gocui.Gui) {
	// Фиксируем текущую длинну массива (индекс) для вставки строки обновления, если это ручной выбор из списка
	if newUpdate {
		app.newUpdateIndex = len(app.currentLogLines) - 1
		// Сбрасываем автоскролл
		app.autoScroll = true
		// Фиксируем время загрузки журнала
		app.updateTime = time.Now().Format("15:04:05")
	}
	// Проверяем, что массив не пустой и уже привысил длинну новых сообщений
	if app.newUpdateIndex > 0 && len(app.currentLogLines)-1 > app.newUpdateIndex {
		// Формируем длинну делимитра
		v, _ := g.View("logs")
		width, _ := v.Size()
		lengthDelimiter := width/2 - 5
		delimiter1 := strings.Repeat("⎯", lengthDelimiter)
		delimiter2 := delimiter1
		if width > lengthDelimiter+lengthDelimiter+10 {
			delimiter2 = strings.Repeat("⎯", lengthDelimiter+1)
		}
		var delimiterString string = delimiter1 + " " + app.updateTime + " " + delimiter2
		// Вставляем новую строку после указанного индекса, сдвигая остальные строки массива
		app.currentLogLines = append(app.currentLogLines[:app.newUpdateIndex],
			append([]string{delimiterString}, app.currentLogLines[app.newUpdateIndex:]...)...)
	}
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
	// Shift+Tab для переключения между окнами в обратном порядке
	if err := app.gui.SetKeybinding("", gocui.KeyBacktab, gocui.ModNone, app.backView); err != nil {
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
	if err := app.gui.SetKeybinding("services", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextService(v, 1)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("varLogs", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextFileName(v, 1)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("docker", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextDockerContainer(v, 1)
	}); err != nil {
		return err
	}
	// Быстрое пролистывание вниз через 10 и 100 записей (<Shift/Alt>+Down)
	if err := app.gui.SetKeybinding("services", gocui.KeyArrowDown, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextService(v, 10)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("varLogs", gocui.KeyArrowDown, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextFileName(v, 10)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("docker", gocui.KeyArrowDown, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextDockerContainer(v, 10)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("services", gocui.KeyArrowDown, gocui.ModAlt, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextService(v, 100)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("varLogs", gocui.KeyArrowDown, gocui.ModAlt, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextFileName(v, 100)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("docker", gocui.KeyArrowDown, gocui.ModAlt, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextDockerContainer(v, 100)
	}); err != nil {
		return err
	}
	// Пролистывание вверх (<Shift/Alt>+Up)
	if err := app.gui.SetKeybinding("services", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevService(v, 1)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("varLogs", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevFileName(v, 1)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("docker", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevDockerContainer(v, 1)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("services", gocui.KeyArrowUp, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevService(v, 10)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("varLogs", gocui.KeyArrowUp, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevFileName(v, 10)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("docker", gocui.KeyArrowUp, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevDockerContainer(v, 10)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("services", gocui.KeyArrowUp, gocui.ModAlt, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevService(v, 100)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("varLogs", gocui.KeyArrowUp, gocui.ModAlt, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevFileName(v, 100)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("docker", gocui.KeyArrowUp, gocui.ModAlt, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevDockerContainer(v, 100)
	}); err != nil {
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
	if app.keybindingsEnabled {
		// Установка привязок
		if err := app.gui.SetKeybinding("varLogs", gocui.KeyArrowRight, gocui.ModNone, app.setLogFilesListRight); err != nil {
			return err
		}
		if err := app.gui.SetKeybinding("varLogs", gocui.KeyArrowLeft, gocui.ModNone, app.setLogFilesListLeft); err != nil {
			return err
		}
	} else {
		// Удаление привязок
		if err := app.gui.DeleteKeybinding("varLogs", gocui.KeyArrowRight, gocui.ModNone); err != nil {
			return err
		}
		if err := app.gui.DeleteKeybinding("varLogs", gocui.KeyArrowLeft, gocui.ModNone); err != nil {
			return err
		}
	}
	// Переключение выбора журналов для Containerization System
	if err := app.gui.SetKeybinding("docker", gocui.KeyArrowRight, gocui.ModNone, app.setContainersList); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("docker", gocui.KeyArrowLeft, gocui.ModNone, app.setContainersList); err != nil {
		return err
	}
	// Переключение между режимами фильтрации через Up/Down для выбранного окна (filter)
	if err := app.gui.SetKeybinding("filter", gocui.KeyArrowUp, gocui.ModNone, app.setFilterModeRight); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("filter", gocui.KeyArrowDown, gocui.ModNone, app.setFilterModeLeft); err != nil {
		return err
	}
	// Переключение для количества выводимых строк через Left/Right для выбранного окна (logs)
	if err := app.gui.SetKeybinding("logs", gocui.KeyArrowLeft, gocui.ModNone, app.setCountLogViewDown); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("logs", gocui.KeyArrowRight, gocui.ModNone, app.setCountLogViewUp); err != nil {
		return err
	}
	// Пролистывание вывода журнала
	if err := app.gui.SetKeybinding("logs", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.scrollDownLogs(1)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("logs", gocui.KeyArrowDown, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		return app.scrollDownLogs(10)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("logs", gocui.KeyArrowDown, gocui.ModAlt, func(g *gocui.Gui, v *gocui.View) error {
		return app.scrollDownLogs(500)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("logs", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.scrollUpLogs(1)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("logs", gocui.KeyArrowUp, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		return app.scrollUpLogs(10)
	}); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("logs", gocui.KeyArrowUp, gocui.ModAlt, func(g *gocui.Gui, v *gocui.View) error {
		return app.scrollUpLogs(500)
	}); err != nil {
		return err
	}
	// Ручное обновление вывода журнала (Ctrl+R)
	if err := app.gui.SetKeybinding("", gocui.KeyCtrlR, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.updateLogOutput(0)
	}); err != nil {
		return err
	}
	// Очистка поля ввода для фильтра (Ctrl+D)
	if err := app.gui.SetKeybinding("", gocui.KeyCtrlD, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		app.clearFilterEditor(g)
		return nil
	}); err != nil {
		return err
	}
	// Очистка поля ввода для фильтра (Ctrl+W)
	if err := app.gui.SetKeybinding("", gocui.KeyCtrlW, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		app.clearFilterEditor(g)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// Функции для переключения количества строк для вывода логов

func (app *App) setCountLogViewUp(g *gocui.Gui, v *gocui.View) error {
	switch app.logViewCount {
	case "5000":
		app.logViewCount = "10000"
	case "10000":
		app.logViewCount = "50000"
	case "50000":
		app.logViewCount = "100000"
	case "100000":
		app.logViewCount = "200000"
	case "200000":
		app.logViewCount = "300000"
	case "300000":
		app.logViewCount = "300000"
	}
	app.applyFilter(false)
	if err := app.updateLogOutput(0); err != nil {
		return nil
	}
	return nil
}

func (app *App) setCountLogViewDown(g *gocui.Gui, v *gocui.View) error {
	switch app.logViewCount {
	case "300000":
		app.logViewCount = "200000"
	case "200000":
		app.logViewCount = "100000"
	case "100000":
		app.logViewCount = "50000"
	case "50000":
		app.logViewCount = "10000"
	case "10000":
		app.logViewCount = "5000"
	case "5000":
		app.logViewCount = "5000"
	}
	app.applyFilter(false)
	if err := app.updateLogOutput(0); err != nil {
		return nil
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
	case "Filter (Default)":
		selectedFilter.Title = "Filter (Fuzzy)"
		app.selectFilterMode = "fuzzy"
	case "Filter (Fuzzy)":
		selectedFilter.Title = "Filter (Regex)"
		app.selectFilterMode = "regex"
	case "Filter (Regex)":
		selectedFilter.Title = "Filter (Default)"
		app.selectFilterMode = "default"
	}
	app.applyFilter(false)
	return nil
}

func (app *App) setFilterModeLeft(g *gocui.Gui, v *gocui.View) error {
	selectedFilter, err := g.View("filter")
	if err != nil {
		log.Panicln(err)
	}
	switch selectedFilter.Title {
	case "Filter (Default)":
		selectedFilter.Title = "Filter (Regex)"
		app.selectFilterMode = "regex"
	case "Filter (Regex)":
		selectedFilter.Title = "Filter (Fuzzy)"
		app.selectFilterMode = "fuzzy"
	case "Filter (Fuzzy)":
		selectedFilter.Title = "Filter (Default)"
		app.selectFilterMode = "default"
	}
	app.applyFilter(false)
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
	switch app.selectUnits {
	case "services":
		app.selectUnits = "UNIT"
		selectedServices.Title = " < System journals (0) > "
		app.loadServices(app.selectUnits)
	case "UNIT":
		app.selectUnits = "USER_UNIT"
		selectedServices.Title = " < User journals (0) > "
		app.loadServices(app.selectUnits)
	case "USER_UNIT":
		app.selectUnits = "kernel"
		selectedServices.Title = " < Kernel boot (0) > "
		app.loadServices(app.selectUnits)
	case "kernel":
		app.selectUnits = "services"
		selectedServices.Title = " < Unit list (0) > "
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
	switch app.selectUnits {
	case "services":
		app.selectUnits = "kernel"
		selectedServices.Title = " < Kernel boot (0) > "
		app.loadServices(app.selectUnits)
	case "kernel":
		app.selectUnits = "USER_UNIT"
		selectedServices.Title = " < User journals (0) > "
		app.loadServices(app.selectUnits)
	case "USER_UNIT":
		app.selectUnits = "UNIT"
		selectedServices.Title = " < System journals (0) > "
		app.loadServices(app.selectUnits)
	case "UNIT":
		app.selectUnits = "services"
		selectedServices.Title = " < Unit list (0) > "
		app.loadServices(app.selectUnits)
	}
	return nil
}

// Функция для переключения выбора журналов файловой системы
func (app *App) setLogFilesListRight(g *gocui.Gui, v *gocui.View) error {
	selectedVarLog, err := g.View("varLogs")
	if err != nil {
		log.Panicln(err)
	}
	// Добавляем сообщение о загрузке журнала
	g.Update(func(g *gocui.Gui) error {
		selectedVarLog.Clear()
		fmt.Fprintln(selectedVarLog, "Searching log files...")
		selectedVarLog.Highlight = false
		return nil
	})
	// Отключаем переключение списков
	app.keybindingsEnabled = false
	if err := app.setupKeybindings(); err != nil {
		log.Panicln("Error key bindings", err)
	}
	// Полсекундная задержка, для корректного обновления интерфейса после выполнения функции
	time.Sleep(500 * time.Millisecond)
	app.logfiles = app.logfiles[:0]
	app.startFiles = 0
	app.selectedFile = 0
	// Запускаем функцию загрузки журнала в горутине
	if app.getOS == "windows" {
		go func() {
			switch app.selectPath {
			case "ProgramFiles":
				app.selectPath = "ProgramFiles86"
				selectedVarLog.Title = " < Program Files x86 (0) > "
				app.loadWinFiles(app.selectPath)
			case "ProgramFiles86":
				app.selectPath = "AppDataLocal"
				selectedVarLog.Title = " < AppData Local (0) > "
				app.loadWinFiles(app.selectPath)
			case "AppDataLocal":
				app.selectPath = "AppDataRoaming"
				selectedVarLog.Title = " < AppData Roaming (0) > "
				app.loadWinFiles(app.selectPath)
			case "AppDataRoaming":
				app.selectPath = "ProgramFiles"
				selectedVarLog.Title = " < Program Files (0) > "
				app.loadWinFiles(app.selectPath)
			}
			// Включаем переключение списков
			app.keybindingsEnabled = true
			if err := app.setupKeybindings(); err != nil {
				log.Panicln("Error key bindings", err)
			}
		}()
	} else {
		go func() {
			switch app.selectPath {
			case "/var/log/":
				app.selectPath = "/home/"
				selectedVarLog.Title = " < Users home logs (0) > "
				app.loadFiles(app.selectPath)
			case "/home/":
				app.selectPath = "descriptor"
				selectedVarLog.Title = " < Process descriptor logs (0) > "
				app.loadFiles(app.selectPath)
			case "descriptor":
				app.selectPath = "/var/log/"
				selectedVarLog.Title = " < System var logs (0) > "
				app.loadFiles(app.selectPath)
			}
			// Включаем переключение списков
			app.keybindingsEnabled = true
			if err := app.setupKeybindings(); err != nil {
				log.Panicln("Error key bindings", err)
			}
		}()
	}
	return nil
}

func (app *App) setLogFilesListLeft(g *gocui.Gui, v *gocui.View) error {
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
	app.keybindingsEnabled = false
	if err := app.setupKeybindings(); err != nil {
		log.Panicln("Error key bindings", err)
	}
	time.Sleep(500 * time.Millisecond)
	app.logfiles = app.logfiles[:0]
	app.startFiles = 0
	app.selectedFile = 0
	if app.getOS == "windows" {
		go func() {
			switch app.selectPath {
			case "ProgramFiles":
				app.selectPath = "AppDataRoaming"
				selectedVarLog.Title = " < AppData Roaming (0) > "
				app.loadWinFiles(app.selectPath)
			case "AppDataRoaming":
				app.selectPath = "AppDataLocal"
				selectedVarLog.Title = " < AppData Local (0) > "
				app.loadWinFiles(app.selectPath)
			case "AppDataLocal":
				app.selectPath = "ProgramFiles86"
				selectedVarLog.Title = " < Program Files x86 (0) > "
				app.loadWinFiles(app.selectPath)
			case "ProgramFiles86":
				app.selectPath = "ProgramFiles"
				selectedVarLog.Title = " < Program Files (0) > "
				app.loadWinFiles(app.selectPath)
			}
			app.keybindingsEnabled = true
			if err := app.setupKeybindings(); err != nil {
				log.Panicln("Error key bindings", err)
			}
		}()
	} else {
		go func() {
			switch app.selectPath {
			case "/var/log/":
				app.selectPath = "descriptor"
				selectedVarLog.Title = " < Process descriptor logs (0) > "
				app.loadFiles(app.selectPath)
			case "descriptor":
				app.selectPath = "/home/"
				selectedVarLog.Title = " < Users home logs (0) > "
				app.loadFiles(app.selectPath)
			case "/home/":
				app.selectPath = "/var/log/"
				selectedVarLog.Title = " < System var logs (0) > "
				app.loadFiles(app.selectPath)
			}
			app.keybindingsEnabled = true
			if err := app.setupKeybindings(); err != nil {
				log.Panicln("Error key bindings", err)
			}
		}()
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
	switch app.selectContainerizationSystem {
	case "docker":
		app.selectContainerizationSystem = "podman"
		selectedDocker.Title = " < Podman containers (0) > "
		app.loadDockerContainer(app.selectContainerizationSystem)
	case "podman":
		app.selectContainerizationSystem = "docker"
		selectedDocker.Title = " < Docker containers (0) > "
		app.loadDockerContainer(app.selectContainerizationSystem)
	}
	return nil
}

// Функция для переключения окон через Tab
func (app *App) nextView(g *gocui.Gui, v *gocui.View) error {
	selectedFilterList, err := g.View("filterList")
	if err != nil {
		log.Panicln(err)
	}
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
		case "filterList":
			nextView = "services"
			selectedFilterList.FrameColor = gocui.ColorDefault
			selectedFilterList.TitleColor = gocui.ColorDefault
			selectedServices.FrameColor = gocui.ColorGreen
			selectedServices.TitleColor = gocui.ColorGreen
			selectedVarLog.FrameColor = app.fileSystemFrameColor
			selectedVarLog.TitleColor = gocui.ColorDefault
			selectedDocker.FrameColor = app.dockerFrameColor
			selectedDocker.TitleColor = gocui.ColorDefault
			selectedFilter.FrameColor = gocui.ColorDefault
			selectedFilter.TitleColor = gocui.ColorDefault
			selectedLogs.FrameColor = gocui.ColorDefault
			selectedLogs.TitleColor = gocui.ColorDefault
		case "services":
			nextView = "varLogs"
			selectedFilterList.FrameColor = gocui.ColorDefault
			selectedFilterList.TitleColor = gocui.ColorDefault
			selectedServices.FrameColor = app.journalListFrameColor
			selectedServices.TitleColor = gocui.ColorDefault
			selectedVarLog.FrameColor = gocui.ColorGreen
			selectedVarLog.TitleColor = gocui.ColorGreen
			selectedDocker.FrameColor = app.dockerFrameColor
			selectedDocker.TitleColor = gocui.ColorDefault
			selectedFilter.FrameColor = gocui.ColorDefault
			selectedFilter.TitleColor = gocui.ColorDefault
			selectedLogs.FrameColor = gocui.ColorDefault
			selectedLogs.TitleColor = gocui.ColorDefault
		case "varLogs":
			nextView = "docker"
			selectedFilterList.FrameColor = gocui.ColorDefault
			selectedFilterList.TitleColor = gocui.ColorDefault
			selectedServices.FrameColor = app.journalListFrameColor
			selectedServices.TitleColor = gocui.ColorDefault
			selectedVarLog.FrameColor = app.fileSystemFrameColor
			selectedVarLog.TitleColor = gocui.ColorDefault
			selectedDocker.FrameColor = gocui.ColorGreen
			selectedDocker.TitleColor = gocui.ColorGreen
			selectedFilter.FrameColor = gocui.ColorDefault
			selectedFilter.TitleColor = gocui.ColorDefault
			selectedLogs.FrameColor = gocui.ColorDefault
			selectedLogs.TitleColor = gocui.ColorDefault
		case "docker":
			nextView = "filter"
			selectedFilterList.FrameColor = gocui.ColorDefault
			selectedFilterList.TitleColor = gocui.ColorDefault
			selectedServices.FrameColor = app.journalListFrameColor
			selectedServices.TitleColor = gocui.ColorDefault
			selectedVarLog.FrameColor = app.fileSystemFrameColor
			selectedVarLog.TitleColor = gocui.ColorDefault
			selectedDocker.FrameColor = app.dockerFrameColor
			selectedDocker.TitleColor = gocui.ColorDefault
			selectedFilter.FrameColor = gocui.ColorGreen
			selectedFilter.TitleColor = gocui.ColorGreen
			selectedLogs.FrameColor = gocui.ColorDefault
			selectedLogs.TitleColor = gocui.ColorDefault
		case "filter":
			nextView = "logs"
			selectedFilterList.FrameColor = gocui.ColorDefault
			selectedFilterList.TitleColor = gocui.ColorDefault
			selectedServices.FrameColor = app.journalListFrameColor
			selectedServices.TitleColor = gocui.ColorDefault
			selectedVarLog.FrameColor = app.fileSystemFrameColor
			selectedVarLog.TitleColor = gocui.ColorDefault
			selectedDocker.FrameColor = app.dockerFrameColor
			selectedDocker.TitleColor = gocui.ColorDefault
			selectedFilter.FrameColor = gocui.ColorDefault
			selectedFilter.TitleColor = gocui.ColorDefault
			selectedLogs.FrameColor = gocui.ColorGreen
			selectedLogs.TitleColor = gocui.ColorGreen
		case "logs":
			nextView = "filterList"
			selectedFilterList.FrameColor = gocui.ColorGreen
			selectedFilterList.TitleColor = gocui.ColorGreen
			selectedServices.FrameColor = app.journalListFrameColor
			selectedServices.TitleColor = gocui.ColorDefault
			selectedVarLog.FrameColor = app.fileSystemFrameColor
			selectedVarLog.TitleColor = gocui.ColorDefault
			selectedDocker.FrameColor = app.dockerFrameColor
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

// Функция для переключения окон в обратном порядке через Shift+Tab
func (app *App) backView(g *gocui.Gui, v *gocui.View) error {
	selectedFilterList, err := g.View("filterList")
	if err != nil {
		log.Panicln(err)
	}
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
	if currentView == nil {
		nextView = "services"
	} else {
		switch currentView.Name() {
		case "filterList":
			nextView = "logs"
			selectedFilterList.FrameColor = gocui.ColorDefault
			selectedFilterList.TitleColor = gocui.ColorDefault
			selectedServices.FrameColor = app.journalListFrameColor
			selectedServices.TitleColor = gocui.ColorDefault
			selectedVarLog.FrameColor = app.fileSystemFrameColor
			selectedVarLog.TitleColor = gocui.ColorDefault
			selectedDocker.FrameColor = app.dockerFrameColor
			selectedDocker.TitleColor = gocui.ColorDefault
			selectedFilter.FrameColor = gocui.ColorDefault
			selectedFilter.TitleColor = gocui.ColorDefault
			selectedLogs.FrameColor = gocui.ColorGreen
			selectedLogs.TitleColor = gocui.ColorGreen
		case "services":
			nextView = "filterList"
			selectedFilterList.FrameColor = gocui.ColorGreen
			selectedFilterList.TitleColor = gocui.ColorGreen
			selectedServices.FrameColor = app.journalListFrameColor
			selectedServices.TitleColor = gocui.ColorDefault
			selectedVarLog.FrameColor = app.fileSystemFrameColor
			selectedVarLog.TitleColor = gocui.ColorDefault
			selectedDocker.FrameColor = app.dockerFrameColor
			selectedDocker.TitleColor = gocui.ColorDefault
			selectedFilter.FrameColor = gocui.ColorDefault
			selectedFilter.TitleColor = gocui.ColorDefault
			selectedLogs.FrameColor = gocui.ColorDefault
			selectedLogs.TitleColor = gocui.ColorDefault
		case "logs":
			nextView = "filter"
			selectedFilterList.FrameColor = gocui.ColorDefault
			selectedFilterList.TitleColor = gocui.ColorDefault
			selectedServices.FrameColor = app.journalListFrameColor
			selectedServices.TitleColor = gocui.ColorDefault
			selectedVarLog.FrameColor = app.fileSystemFrameColor
			selectedVarLog.TitleColor = gocui.ColorDefault
			selectedDocker.FrameColor = app.dockerFrameColor
			selectedDocker.TitleColor = gocui.ColorDefault
			selectedFilter.FrameColor = gocui.ColorGreen
			selectedFilter.TitleColor = gocui.ColorGreen
			selectedLogs.FrameColor = gocui.ColorDefault
			selectedLogs.TitleColor = gocui.ColorDefault
		case "filter":
			nextView = "docker"
			selectedFilterList.FrameColor = gocui.ColorDefault
			selectedFilterList.TitleColor = gocui.ColorDefault
			selectedServices.FrameColor = app.journalListFrameColor
			selectedServices.TitleColor = gocui.ColorDefault
			selectedVarLog.FrameColor = app.fileSystemFrameColor
			selectedVarLog.TitleColor = gocui.ColorDefault
			selectedDocker.FrameColor = gocui.ColorGreen
			selectedDocker.TitleColor = gocui.ColorGreen
			selectedFilter.FrameColor = gocui.ColorDefault
			selectedFilter.TitleColor = gocui.ColorDefault
			selectedLogs.FrameColor = gocui.ColorDefault
			selectedLogs.TitleColor = gocui.ColorDefault
		case "docker":
			nextView = "varLogs"
			selectedFilterList.FrameColor = gocui.ColorDefault
			selectedFilterList.TitleColor = gocui.ColorDefault
			selectedServices.FrameColor = app.journalListFrameColor
			selectedServices.TitleColor = gocui.ColorDefault
			selectedVarLog.FrameColor = gocui.ColorGreen
			selectedVarLog.TitleColor = gocui.ColorGreen
			selectedDocker.FrameColor = app.dockerFrameColor
			selectedDocker.TitleColor = gocui.ColorDefault
			selectedFilter.FrameColor = gocui.ColorDefault
			selectedFilter.TitleColor = gocui.ColorDefault
			selectedLogs.FrameColor = gocui.ColorDefault
			selectedLogs.TitleColor = gocui.ColorDefault
		case "varLogs":
			nextView = "services"
			selectedFilterList.FrameColor = gocui.ColorDefault
			selectedFilterList.TitleColor = gocui.ColorDefault
			selectedServices.FrameColor = gocui.ColorGreen
			selectedServices.TitleColor = gocui.ColorGreen
			selectedVarLog.FrameColor = app.fileSystemFrameColor
			selectedVarLog.TitleColor = gocui.ColorDefault
			selectedDocker.FrameColor = app.dockerFrameColor
			selectedDocker.TitleColor = gocui.ColorDefault
			selectedFilter.FrameColor = gocui.ColorDefault
			selectedFilter.TitleColor = gocui.ColorDefault
			selectedLogs.FrameColor = gocui.ColorDefault
			selectedLogs.TitleColor = gocui.ColorDefault
		}
	}
	if _, err := g.SetCurrentView(nextView); err != nil {
		return err
	}
	return nil
}

// Функция для выхода
func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
