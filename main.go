package main

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strings"

	"github.com/jroimartin/gocui"
)

type Journal struct {
	name    string
	content []string
}

type App struct {
	gui              *gocui.Gui
	journals         []Journal
	selectedJournal  int
	filterText       string
	currentLogLines  []string
	filteredLogLines []string
}

func main() {
	app := &App{
		journals:        make([]Journal, 0),
		selectedJournal: 0,
	}

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	app.gui = g
	g.SetManagerFunc(app.layout)
	g.Mouse = true

	g.FgColor = gocui.ColorWhite
	g.BgColor = gocui.ColorBlack

	if err := app.setupKeybindings(); err != nil {
		log.Panicln(err)
	}

	app.loadServices()
	g.SetCurrentView("services")

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func (app *App) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView("services", 0, 0, maxX/4, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Services"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		v.Wrap = true

		app.updateServicesList()
	}

	if v, err := g.SetView("filter", maxX/4+1, 0, maxX-1, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Filter (Tab to focus)"
		v.Editable = true
		v.Editor = app.createFilterEditor()
		v.Wrap = true
	}

	if v, err := g.SetView("logs", maxX/4+1, 3, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Logs"
		v.Wrap = true
		v.Autoscroll = true
	}

	// Управляем отображением курсора в зависимости от активного вида
	currentView := g.CurrentView()
	if currentView != nil {
		switch currentView.Name() {
		case "filter":
			g.Cursor = true
		default:
			g.Cursor = false
		}
	}

	return nil
}

// Создаем кастомный редактор для поля фильтра
func (app *App) createFilterEditor() gocui.Editor {
	return gocui.EditorFunc(func(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
		switch {
		case ch != 0 && mod == 0:
			v.EditWrite(ch)
			app.filterText = strings.TrimSpace(v.Buffer())
			app.applyFilter()
		case key == gocui.KeySpace:
			v.EditWrite(' ')
			app.filterText = strings.TrimSpace(v.Buffer())
			app.applyFilter()
		case key == gocui.KeyBackspace || key == gocui.KeyBackspace2:
			v.EditDelete(true)
			app.filterText = strings.TrimSpace(v.Buffer())
			app.applyFilter()
		case key == gocui.KeyDelete:
			v.EditDelete(false)
			app.filterText = strings.TrimSpace(v.Buffer())
			app.applyFilter()
		}
	})
}

func (app *App) loadServices() {
	cmd := exec.Command("journalctl", "--no-pager", "-F", "_SYSTEMD_UNIT")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error getting services: %v", err)
		return
	}

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

	sort.Slice(app.journals, func(i, j int) bool {
		return app.journals[i].name < app.journals[j].name
	})

	app.updateServicesList()
}

func (app *App) updateServicesList() {
	v, err := app.gui.View("services")
	if err != nil {
		return
	}
	v.Clear()

	for _, journal := range app.journals {
		fmt.Fprintln(v, journal.name)
	}
}

func (app *App) loadJournalLogs(serviceName string) {
	cmd := exec.Command("journalctl", "-u", serviceName, "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error getting logs: %v", err)
		return
	}

	app.currentLogLines = strings.Split(string(output), "\n")
	app.applyFilter()
}

func (app *App) applyFilter() {
	app.filteredLogLines = make([]string, 0)
	filter := strings.ToLower(app.filterText)

	for _, line := range app.currentLogLines {
		if filter == "" || strings.Contains(strings.ToLower(line), filter) {
			app.filteredLogLines = append(app.filteredLogLines, line)
		}
	}

	app.updateLogsView()
}

func (app *App) updateLogsView() {
	v, err := app.gui.View("logs")
	if err != nil {
		return
	}

	v.Clear()
	for _, line := range app.filteredLogLines {
		fmt.Fprintln(v, line)
	}
}

func (app *App) setupKeybindings() error {
	if err := app.gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	if err := app.gui.SetKeybinding("services", gocui.KeyArrowDown, gocui.ModNone, app.nextService); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("services", gocui.KeyArrowUp, gocui.ModNone, app.prevService); err != nil {
		return err
	}
	if err := app.gui.SetKeybinding("services", gocui.KeyEnter, gocui.ModNone, app.selectService); err != nil {
		return err
	}

	// Удалено событие Enter для фильтра, так как теперь фильтрация происходит динамически

	if err := app.gui.SetKeybinding("", gocui.KeyTab, gocui.ModNone, app.nextView); err != nil {
		return err
	}

	return nil
}

func (app *App) nextService(g *gocui.Gui, v *gocui.View) error {
	if len(app.journals) == 0 {
		return nil
	}

	if app.selectedJournal < len(app.journals)-1 {
		app.selectedJournal++
		return app.selectServiceByIndex(app.selectedJournal)
	}
	return nil
}

func (app *App) prevService(g *gocui.Gui, v *gocui.View) error {
	if len(app.journals) == 0 {
		return nil
	}

	if app.selectedJournal > 0 {
		app.selectedJournal--
		return app.selectServiceByIndex(app.selectedJournal)
	}
	return nil
}

func (app *App) selectService(g *gocui.Gui, v *gocui.View) error {
	if v == nil || len(app.journals) == 0 {
		return nil
	}

	_, cy := v.Cursor()
	line, err := v.Line(cy)
	if err != nil {
		return err
	}

	app.loadJournalLogs(strings.TrimSpace(line))
	return nil
}

func (app *App) selectServiceByIndex(index int) error {
	v, err := app.gui.View("services")
	if err != nil {
		return err
	}

	v.SetCursor(0, index)
	return nil
}

func (app *App) nextView(g *gocui.Gui, v *gocui.View) error {
	currentView := g.CurrentView()
	var nextView string

	if currentView == nil {
		nextView = "services"
	} else {
		switch currentView.Name() {
		case "services":
			nextView = "filter"
		case "filter":
			nextView = "logs"
		case "logs":
			nextView = "services"
		default:
			nextView = "services"
		}
	}

	if _, err := g.SetCurrentView(nextView); err != nil {
		return err
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
