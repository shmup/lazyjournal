package main

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strings"

	"github.com/jroimartin/gocui"
	// "github.com/jesseduffield/gocui"
	// "github.com/awesome-gocui/gocui"
)

// Структура хранения информации о журналах
type Journal struct {
	name    string   // название журнала (имя службы)
	content []string // содержимое журнала (массив строк)
}

// Структура основного приложения (графический интерфейс и данные журналов)
type App struct {
	gui             *gocui.Gui // графический интерфейс (gocui)
	journals        []Journal  // список (массив) журналов для отображения
	selectedJournal int        // индекс выбранного журнала

	maxVisibleServices int // Максимальное количество видимых элементов в окне списка служб
	startServices      int // Индекс первого видимого элемента
	endServices        int // Индекс последнего видимого элемента

	filterText       string   // текст для фильтрации записей журнала
	currentLogLines  []string // набор строк (срез) для хранения журнала без фильтрации
	filteredLogLines []string // набор строк (срез) для хранения журнала после фильтра
	logScrollPos     int      // позиция прокрутки для отображаемых строк журнала
}

func main() {
	app := &App{
		journals:        make([]Journal, 0), // инициализация списка журналов (пустой массив)
		selectedJournal: 0,                  // начальный индекс выбранного журнала
		startServices:   0,
		endServices:     0,
	}

	// Создаем GUI
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	// g := gocui.NewGui() // fork
	// Закрываем GUI после завершения
	defer g.Close()

	app.gui = g
	// Функция, которая будет вызываться при обновлении интерфейса
	g.SetManagerFunc(app.layout)
	// g.SetCurrentView("viewName") // fork
	g.Mouse = true // включить поддержку мыши

	// Цветовая схема GUI (ColorBlack, ColorGreen, ColorRed, ColorYellow, ColorBlue, ColorCyan, ColorMagenta)
	g.FgColor = gocui.ColorDefault // поля окон и цвет текста
	g.BgColor = gocui.ColorDefault // фон

	// Привязка клавиш для работы с интерфейсом из функции setupKeybindings()
	if err := app.setupKeybindings(); err != nil {
		log.Panicln(err)
	}

	// Выполняем layout для инициализации окна services
	if err := app.layout(g); err != nil {
		log.Panicln(err)
	}

	// Фиксируем текущее количество видимых строк в терминале (-1 заголовок)
	if v, err := g.View("services"); err == nil {
		_, viewHeight := v.Size()
		app.maxVisibleServices = viewHeight - 1
	}

	// Загружаем список доступных журналов
	app.loadServices()
	// Устанавливаем фокус на окно с журналами по умолчанию
	g.SetCurrentView("services")

	// Запус GUI
	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

// Функция для определения структуры интерфейса (окон) GUI
func (app *App) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size() // Получаем текущий размер интерфейса терминала

	// Окно для отображения списка доступных журналов
	// Размеры окна (позиция слева, сверху, четверть от максимальной ширины, вся высота окна)
	if v, err := g.SetView("services", 0, 0, maxX/4, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Services" // заголовок окна
		v.Highlight = true   // выделение активного элемента

		v.SelBgColor = gocui.ColorGreen // Цвет фона при выборе в списке
		v.SelFgColor = gocui.ColorBlack // Цвет текста
		// v.BgColor = gocui.ColorRed      // Цвет текста
		// v.FgColor = gocui.ColorYellow   // Цвет фона

		v.Wrap = false           // отключаем перенос строк
		v.Autoscroll = true      // включаем автопрокрутку
		app.updateServicesList() // выводим список журналов в это окно
	}

	// Окно ввода текста для фильтрации
	if v, err := g.SetView("filter", maxX/4+1, 0, maxX-1, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Filter"
		v.Editable = true                   // включить окно редактируемым для ввода текста
		v.Editor = app.createFilterEditor() // редактор для обработки ввода (удалить для fork)
		v.Wrap = true
	}

	// Окно для вывода записей выбранного журнала
	if v, err := g.SetView("logs", maxX/4+1, 3, maxX-1, maxY-1); err != nil {
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

// Функция для загрузки списка журналов служб из journalctl
func (app *App) loadServices() {
	cmd := exec.Command("journalctl", "--no-pager", "-F", "UNIT")
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
				content: make([]string, 0),
			})
		}
	}
	// Сортируем список служб по алфавиту
	sort.Slice(app.journals, func(i, j int) bool {
		return app.journals[i].name < app.journals[j].name
	})
	// Обновляем список служб в интерфейсе
	app.updateServicesList()
}

// Функция для обновления окна со списком служб
func (app *App) updateServicesList() {
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
func (app *App) nextService(g *gocui.Gui, v *gocui.View, step int) error {
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
		// Если сдвинули видимую область, корректируем индекс
		if app.selectedJournal < app.startServices+app.maxVisibleServices {
			// Выбираем журнал по скорректированному индексу
			return app.selectServiceByIndex(app.selectedJournal - app.startServices)
		}
	}
	return nil
}

// Функция для перемещения по списку журналов вверх
func (app *App) prevService(g *gocui.Gui, v *gocui.View, step int) error {
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

// Функция для выбора журнала по индексу
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
	return nil
}

// Функция для загрузки записей журнала выбранной службы через journalctl
func (app *App) loadJournalLogs(serviceName string) {
	cmd := exec.Command("journalctl", "-u", serviceName, "--no-pager", "-n", "5000")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error getting logs: %v", err)
		return
	}
	// Сохраняем строки журнала в массив
	app.currentLogLines = strings.Split(string(output), "\n")
	// Очищаем поле ввода для фильтрации
	app.filterText = ""
	// Применяем текущий фильтр к записям для обновления вывода
	app.applyFilter()
}

// Функция для фильтрации записей текущего журнала
func (app *App) applyFilter() {
	app.filteredLogLines = make([]string, 0)
	// Опускаем регистр
	filter := strings.ToLower(app.filterText)
	for _, line := range app.currentLogLines {
		if filter == "" || strings.Contains(strings.ToLower(line), filter) {
			app.filteredLogLines = append(app.filteredLogLines, line) // сохраняем строки, соответствующие фильтру
		}
	}
	app.logScrollPos = 0
	// Обновляем окно для отображения отфильтрованных записей
	app.updateLogsView()
}

// Функция для обновления вывода журнала
func (app *App) updateLogsView() {
	// Получаем доступ к выводу журнала
	v, err := app.gui.View("logs")
	if err != nil {
		return
	}
	// Очищаем окно для отображения новых строк
	v.Clear()
	// Определяем строки для отображения, начиная с позиции logScrollPos
	linesToDisplay := app.filteredLogLines[app.logScrollPos:]
	// Проходим по отфильтрованным строкам и выводим их
	for _, line := range linesToDisplay {
		fmt.Fprintln(v, line) // Печатаем каждую строку
	}
}

// Функция для скроллинга вниз
func (app *App) scrollDownLogs(g *gocui.Gui, v *gocui.View) error {
	// Увеличиваем позицию прокрутки на одну строку, если не достигнут конец списка
	if app.logScrollPos < len(app.filteredLogLines)-1 {
		// Увеличиваем позицию прокрутки
		app.logScrollPos++
		// Вызываем функцию для обновления отображения журнала
		app.updateLogsView()
	}
	return nil
}

// Функция для скроллинга вверх
func (app *App) scrollUpLogs(g *gocui.Gui, v *gocui.View) error {
	// Уменьшаем позицию прокрутки, если текущая позиция больше нуля
	if app.logScrollPos > 0 {
		app.logScrollPos--
		app.updateLogsView()
	}
	return nil
}

// Функция редактора обработки ввода текста для фильтрации
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
			v.MoveCursor(-1, 0, false)
		// Перемещение курсора вправо
		case key == gocui.KeyArrowRight:
			v.MoveCursor(1, 0, false)
		}
		// Обновляем текст в буфере
		app.filterText = strings.TrimSpace(v.Buffer())
		// Применяем функцию фильтрации к выводу записей журнала
		app.applyFilter()
	})
}

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
	// Вниз (KeyArrowDown) для перемещения к следующей службе в списке журналов (функция nextService)
	app.gui.SetKeybinding("services", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.nextService(g, v, 1)
	})
	// Быстрое пролистывание (через 10 записей)
	app.gui.SetKeybinding("services", gocui.KeyArrowRight, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error { // ModAlt
		return app.nextService(g, v, 10)
	})
	// Пролистывание вверх
	app.gui.SetKeybinding("services", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevService(g, v, 1)
	})
	app.gui.SetKeybinding("services", gocui.KeyArrowLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return app.prevService(g, v, 10)
	})
	// Пролистывание вывода журнала
	if err := app.gui.SetKeybinding("logs", gocui.KeyArrowDown, gocui.ModNone, app.scrollDownLogs); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("logs", gocui.KeyArrowUp, gocui.ModNone, app.scrollUpLogs); err != nil {
		return err
	}
	return nil
}

// Функция для переключения окон через Tab
func (app *App) nextView(g *gocui.Gui, v *gocui.View) error {
	selectServices, err := g.View("services")
	if err != nil {
		log.Panicln(err)
	}
	selectFilter, err := g.View("filter")
	if err != nil {
		log.Panicln(err)
	}
	selectLogs, err := g.View("logs")
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
			selectServices.Title = "Services"
			selectFilter.Title = "Filter 🔸"
			selectLogs.Title = "Logs"
			nextView = "filter"
		case "filter":
			selectServices.Title = "Services"
			selectFilter.Title = "Filter"
			selectLogs.Title = "Logs 🔸"
			nextView = "logs"
		case "logs":
			selectServices.Title = "Services 🔸"
			selectFilter.Title = "Filter"
			selectLogs.Title = "Logs"
			nextView = "services"
		default:
			selectServices.Title = "Services 🔸"
			selectFilter.Title = "Filter"
			selectLogs.Title = "Logs"
			nextView = "services"
		}
	}
	// Устанавливаем новое активное окно
	if _, err := g.SetCurrentView(nextView); err != nil {
		// if err := g.SetCurrentView(nextView); err != nil { // fork
		return err
	}
	return nil
}

// Функция для выхода
func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
