package main

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"hextiller/pkg/process"
)

var uiTheme = struct {
	background tcell.Color
	surface    tcell.Color
	stripe     tcell.Color
	headerBg   tcell.Color
	text       tcell.Color
	subtleText tcell.Color
	accent     tcell.Color
	warm       tcell.Color
	danger     tcell.Color
	selection  tcell.Color
	inputBg    tcell.Color
}{
	background: tcell.NewHexColor(0x0f0f14),
	surface:    tcell.NewHexColor(0x11131a),
	stripe:     tcell.NewHexColor(0x161924),
	headerBg:   tcell.NewHexColor(0x181c26),
	text:       tcell.NewHexColor(0xe7e7eb),
	subtleText: tcell.NewHexColor(0x9aa0b2),
	accent:     tcell.NewHexColor(0x2fb4ad),
	warm:       tcell.NewHexColor(0xffb347),
	danger:     tcell.NewHexColor(0xff6b6b),
	selection:  tcell.NewHexColor(0x1f6f78),
	inputBg:    tcell.NewHexColor(0x151824),
}

type processInfo struct {
	pid  int
	name string
}

func applyTableTheme(t *tview.Table) {
	t.SetBackgroundColor(uiTheme.surface)
	t.SetBorderColor(uiTheme.accent)
	t.SetTitleColor(uiTheme.accent)
	t.SetSelectedStyle(tcell.StyleDefault.Background(uiTheme.selection).Foreground(uiTheme.text))
}

func applyFormTheme(f *tview.Form) {
	f.SetBackgroundColor(uiTheme.surface)
	f.SetBorderColor(uiTheme.accent)
	f.SetTitleColor(uiTheme.accent)
	f.SetFieldBackgroundColor(uiTheme.inputBg)
	f.SetFieldTextColor(uiTheme.text)
	f.SetLabelColor(uiTheme.subtleText)
	f.SetButtonBackgroundColor(uiTheme.accent)
	f.SetButtonTextColor(uiTheme.background)
}

func stripeColor(row int) tcell.Color {
	if row%2 == 1 {
		return uiTheme.stripe
	}
	return uiTheme.surface
}

func bodyCell(text string, row int) *tview.TableCell {
	return tview.NewTableCell(text).
		SetTextColor(uiTheme.text).
		SetBackgroundColor(stripeColor(row))
}

type ui struct {
	app           *tview.Application
	procs         []processInfo
	table         *tview.Table
	watched       *tview.Table
	log           *tview.TextView
	status        *tview.TextView
	sets          []*searchSet
	activeSetIdx  int
	selectedPID   int
	selectedExe   string
	watchedRows   []resultRow
	watchedTitle  string
	logLines      []string
	lastLog       string
	lastCount     int
	lastNavRune   rune
	spinnerIdx    int
	spinnerFrames []string
}

type searchSet struct {
	ui           *ui
	form         *tview.Form
	typeDrop     *tview.DropDown
	valueField   *tview.InputField
	results      *tview.Table
	resultsTitle string
	rows         []resultRow
	activeType   string
	formItems    []tview.FormItem
	formIndex    int
}

type numericValue struct {
	i64 int64
	u64 uint64
	f64 float64
}

type resultRow struct {
	addr    uintptr
	dtype   string
	current numericValue
	desired numericValue
	pinned  bool
}

func main() {
	app := tview.NewApplication()
	u := newUI(app)

	if err := app.SetRoot(u.layout(), true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func newUI(app *tview.Application) *ui {
	u := &ui{
		app:          app,
		activeSetIdx: 0,
	}

	u.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	applyTableTheme(u.table)
	u.table.SetTitle(" Processes (r=refresh) ").SetBorder(true)

	setA := newSearchSet(u)
	setB := newSearchSet(u)
	setC := newSearchSet(u)
	u.sets = []*searchSet{setA, setB, setC}

	u.watched = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	u.watchedTitle = " Watched (e=edit, p=pin, w=write, u=unwatch) "
	applyTableTheme(u.watched)
	u.watched.SetTitle(u.watchedTitle).SetBorder(true)

	u.log = tview.NewTextView().
		SetScrollable(true).
		SetWrap(true)
	u.log.SetBorder(true).SetTitle(" Log (c=clear) ")
	u.log.SetBackgroundColor(uiTheme.surface)
	u.log.SetBorderColor(uiTheme.accent)
	u.log.SetTitleColor(uiTheme.accent)
	u.log.SetTextColor(uiTheme.text)
	u.log.SetDynamicColors(true)

	u.status = tview.NewTextView().
		SetScrollable(false).
		SetWrap(false)
	u.status.SetDynamicColors(false)
	u.status.SetBorder(false)
	u.status.SetBackgroundColor(uiTheme.headerBg)
	u.status.SetTextColor(uiTheme.accent)

	u.spinnerFrames = []string{"-", "\\", "|", "/"}

	u.showWelcome()
	u.loadProcesses()
	u.bindKeys()
	u.renderWatched(-1)
	u.focusTable()
	u.updateStatus(false, "")
	go u.pinnedLoop()

	return u
}

func newSearchSet(u *ui) *searchSet {
	s := &searchSet{ui: u}

	s.form = tview.NewForm()
	s.form.SetBorder(true).SetTitle(" Search ")
	applyFormTheme(s.form)

	s.typeDrop = tview.NewDropDown().
		SetLabel("Type ").
		SetOptions([]string{"int32", "int64", "uint32", "uint64", "float32", "float64"}, nil)
	s.typeDrop.SetCurrentOption(0)

	s.valueField = tview.NewInputField().
		SetLabel("Value ").
		SetPlaceholder("42")

	s.form.AddFormItem(s.typeDrop)
	s.form.AddFormItem(s.valueField)
	s.form.AddButton("Search", func() { s.doSearch() })
	s.form.AddButton("Refine", func() { s.doRefine() })
	s.form.SetButtonsAlign(tview.AlignLeft)
	s.formItems = []tview.FormItem{s.typeDrop, s.valueField}
	s.formIndex = 0

	s.results = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	s.resultsTitle = " Results (w=watch) "
	applyTableTheme(s.results)
	s.results.SetTitle(s.resultsTitle).SetBorder(true)

	return s
}

func (u *ui) layout() tview.Primitive {
	if len(u.sets) == 0 {
		content := tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(u.table, 40, 0, true)
		return tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(content, 0, 1, true).
			AddItem(u.status, 1, 0, false)
	}

	setsRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	for _, set := range u.sets {
		col := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(set.form, 9, 0, false).
			AddItem(set.results, 0, 1, true)
		setsRow.AddItem(col, 0, 1, true)
	}

	bottom := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(u.watched, 0, 2, false).
		AddItem(u.log, 0, 1, false)

	right := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(setsRow, 0, 1, true).
		AddItem(bottom, 0, 1, false)

	content := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(u.table, 40, 0, true).
		AddItem(right, 0, 1, false)

	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(content, 0, 1, true).
		AddItem(u.status, 1, 0, false)
}

func (u *ui) populateTable() {
	u.table.Clear()
	u.table.SetCell(0, 0, header("PID"))
	u.table.SetCell(0, 1, header("Name"))

	for i, p := range u.procs {
		row := i + 1
		u.table.SetCell(row, 0, bodyCell(fmt.Sprintf("%d", p.pid), row))
		u.table.SetCell(row, 1, bodyCell(p.name, row))
	}

	if len(u.procs) > 0 {
		u.table.Select(1, 0)
		u.updateSelection(1)
	}

	u.table.SetSelectedFunc(func(row, _ int) {
		u.updateSelection(row)
	})

	u.table.SetSelectionChangedFunc(func(row, _ int) {
		u.updateSelection(row)
	})
}

func (u *ui) updateSelection(row int) {
	if row <= 0 || row-1 >= len(u.procs) {
		u.selectedPID = 0
		u.selectedExe = ""
		u.updateFormTitles()
		u.updateStatus(false, "")
		return
	}
	p := u.procs[row-1]
	u.selectedPID = p.pid
	u.selectedExe = p.name
	u.updateFormTitles()
	u.updateStatus(false, "")
}

func (u *ui) updateFormTitles() {
	for _, set := range u.sets {
		set.updateFormTitle(u.selectedPID, u.selectedExe)
	}
}

func (u *ui) currentSet() *searchSet {
	if len(u.sets) == 0 {
		return nil
	}
	if u.activeSetIdx < 0 || u.activeSetIdx >= len(u.sets) {
		u.activeSetIdx = 0
	}
	return u.sets[u.activeSetIdx]
}

func (u *ui) setActiveSet(idx int) {
	if idx < 0 || idx >= len(u.sets) {
		return
	}
	u.activeSetIdx = idx
}

func (u *ui) bindKeys() {
	if len(u.sets) == 0 {
		return
	}

	u.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRight:
			u.setActiveSet(0)
			u.focusForm()
			return nil
		case tcell.KeyLeft:
			return nil
		}
		switch r := event.Rune(); {
		case r != 0 && unicode.IsLetter(r):
			if r == 'r' || r == 'R' {
				u.loadProcesses()
				return nil
			}
			u.quickNavigateProcesses(r)
			return nil
		}
		return event
	})

	for idx, set := range u.sets {
		i := idx
		set.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			u.setActiveSet(i)
			if u.app.GetFocus() == set.typeDrop {
				switch event.Key() {
				case tcell.KeyLeft:
					u.focusTable()
					return nil
				case tcell.KeyRight:
					u.app.SetFocus(set.results)
					return nil
				default:
					return event // allow dropdown to handle open/navigation
				}
			}
			switch event.Key() {
			case tcell.KeyLeft:
				u.focusTable()
				return nil
			case tcell.KeyRight:
				u.app.SetFocus(set.results)
				return nil
			case tcell.KeyUp:
				set.moveFormFocus(-1)
				return nil
			case tcell.KeyDown:
				set.moveFormFocus(1)
				return nil
			case tcell.KeyEnter:
				set.doSearch()
				return nil
			}
			return event
		})

		set.results.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			u.setActiveSet(i)
			switch event.Key() {
			case tcell.KeyLeft:
				u.app.SetFocus(set.form)
				return nil
			case tcell.KeyRight:
				u.app.SetFocus(u.watched)
				return nil
			}
			switch event.Rune() {
			case 'w', 'W':
				u.watchSelected()
				return nil
			}
			return event
		})
	}

	u.watched.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			if cur := u.currentSet(); cur != nil {
				u.app.SetFocus(cur.form)
			}
			return nil
		case tcell.KeyRight:
			return nil
		}
		switch event.Rune() {
		case 'e', 'E':
			u.editDesired()
			return nil
		case 'p', 'P':
			u.togglePin()
			return nil
		case 'w', 'W':
			u.writeDesired()
			return nil
		case 'u', 'U':
			u.unwatchSelected()
			return nil
		}
		return event
	})

	u.log.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'c', 'C':
			u.clearLog()
			return nil
		}
		return event
	})
}

func (u *ui) focusTable() {
	u.app.SetFocus(u.table)
}

func (u *ui) focusForm() {
	set := u.currentSet()
	if set == nil {
		return
	}
	set.focusForm()
}

func (u *ui) loadProcesses() {
	infos, err := process.List()
	if err != nil {
		u.procs = nil
		if set := u.currentSet(); set != nil {
			set.showResultsError(fmt.Sprintf("load error: %v", err))
		}
		u.table.Clear()
		u.table.SetCell(0, 0, header("PID"))
		u.table.SetCell(0, 1, header("Name"))
		return
	}

	procs := make([]processInfo, 0, len(infos))
	for _, p := range infos {
		procs = append(procs, processInfo{pid: int(p.PID), name: p.Exe})
	}
	sort.Slice(procs, func(i, j int) bool {
		// sort ascending by name, case-insensitive
		return strings.ToLower(procs[i].name) < strings.ToLower(procs[j].name)
	})

	u.procs = procs
	u.populateTable()
}

func (s *searchSet) updateFormTitle(pid int, exe string) {
	title := " Search "
	if pid != 0 {
		title = fmt.Sprintf(" Search (PID %d) ", pid)
	}
	s.form.SetTitle(title)
}

func (s *searchSet) focusForm() {
	s.formIndex = 0
	s.focusFormItem(s.formIndex)
}

func (s *searchSet) moveFormFocus(delta int) {
	if len(s.formItems) == 0 {
		return
	}
	s.formIndex = (s.formIndex + delta + len(s.formItems)) % len(s.formItems)
	s.focusFormItem(s.formIndex)
}

func (s *searchSet) focusFormItem(idx int) {
	if idx < 0 || idx >= len(s.formItems) {
		return
	}
	s.ui.app.SetFocus(s.formItems[idx])
}

func (s *searchSet) doSearch() {
	_, dtype := s.typeDrop.GetCurrentOption()
	valStr := s.valueField.GetText()

	if s.ui.selectedPID == 0 {
		s.showResultsError("Select a process first")
		return
	}

	s.searchNumeric(dtype, valStr, false)
}

func (s *searchSet) doRefine() {
	_, dtype := s.typeDrop.GetCurrentOption()
	valStr := s.valueField.GetText()

	if s.ui.selectedPID == 0 {
		s.showResultsError("Select a process first")
		return
	}

	s.searchNumeric(dtype, valStr, true)
}

func (s *searchSet) searchNumeric(dtype, valStr string, refine bool) {
	if refine && (s.activeType == "" || s.activeType != dtype) {
		s.showResultsError("refine requires the same type as the last search")
		return
	}

	val, err := s.ui.parseValue(dtype, valStr)
	if err != nil {
		s.showResultsError(err.Error())
		return
	}

	proc, err := process.Open(uint32(s.ui.selectedPID))
	if err != nil {
		s.showResultsError(fmt.Sprintf("open: %v", err))
		return
	}
	defer proc.Close()

	if refine {
		s.doRefineWith(proc, dtype, val)
		return
	}

	addrs, err := s.ui.scanByType(proc, dtype, val)
	if err != nil {
		s.showResultsError(fmt.Sprintf("scan: %v", err))
		return
	}

	rows := make([]resultRow, 0, len(addrs))
	for _, addr := range addrs {
		cur, err := s.ui.readByType(proc, dtype, addr)
		if err != nil {
			cur = numericValue{}
		}
		rows = append(rows, resultRow{addr: addr, dtype: dtype, current: cur, desired: cur})
	}

	s.activeType = dtype
	s.rows = rows
	s.renderResults(0)
}

func (s *searchSet) doRefineWith(proc *process.Process, dtype string, val numericValue) {
	if len(s.rows) == 0 {
		s.showResultsMessage("no previous results to refine")
		return
	}
	cmp := s.ui.makeComparator(dtype, val)

	var filtered []resultRow
	for _, r := range s.rows {
		cur, err := s.ui.readByType(proc, dtype, r.addr)
		if err != nil {
			continue
		}
		if cmp(cur) {
			r.current = cur
			r.desired = cur
			filtered = append(filtered, r)
		}
	}

	if len(filtered) == 0 {
		s.rows = nil
		s.showResultsMessage("no matches after refine")
		return
	}

	// Compact backing storage so the slice capacity matches the filtered size.
	s.rows = make([]resultRow, len(filtered))
	copy(s.rows, filtered)

	s.activeType = dtype
	s.renderResults(0)
}

func (s *searchSet) showResultsMessage(msg string) {
	s.setResultsMessage(msg, uiTheme.subtleText)
}

func (s *searchSet) showResultsError(msg string) {
	s.setResultsMessage(msg, uiTheme.danger)
}

func (s *searchSet) setResultsMessage(msg string, color tcell.Color) {
	s.results.Clear()
	s.results.SetCell(0, 0, tview.NewTableCell(msg).
		SetSelectable(false).
		SetTextColor(color).
		SetBackgroundColor(uiTheme.surface))
}

func (u *ui) setTableTitle(table *tview.Table, base, extra string) {
	title := base
	if extra != "" {
		title = fmt.Sprintf("%s %s", base, extra)
	}
	table.SetTitle(title)
}

const maxLogLines = 200

func (u *ui) logf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if msg == u.lastLog {
		u.lastCount++
		if len(u.logLines) > 0 {
			u.logLines[len(u.logLines)-1] = collapseMsg(u.lastLog, u.lastCount)
		}
	} else {
		u.lastLog = msg
		u.lastCount = 1
		u.logLines = append(u.logLines, msg)
		if len(u.logLines) > maxLogLines {
			u.logLines = u.logLines[len(u.logLines)-maxLogLines:]
		}
	}

	u.log.SetText(strings.Join(u.logLines, "\n"))
	u.log.ScrollToEnd()
}

func (u *ui) clearLog() {
	u.logLines = nil
	u.lastLog = ""
	u.lastCount = 0
	u.log.SetText("")
}

func (u *ui) showWelcome() {
	help := []string{
		"[lightgreen]Welcome to Hextiller!",
		"[lightcyan]Use the process list on the left to select a target process.",
		"[lightcyan]Then use the search forms to scan for values in the target process's memory.",
	}
	u.logLines = append(help, u.logLines...)
	u.log.SetText(strings.Join(u.logLines, "\n"))
}

func collapseMsg(msg string, count int) string {
	if count <= 1 {
		return msg
	}
	return fmt.Sprintf("%s (x%d)", msg, count)
}

func (u *ui) spinnerNext() string {
	if len(u.spinnerFrames) == 0 {
		return ""
	}
	frame := u.spinnerFrames[u.spinnerIdx%len(u.spinnerFrames)]
	u.spinnerIdx = (u.spinnerIdx + 1) % len(u.spinnerFrames)
	return frame
}

func (u *ui) updateStatus(active bool, warn string) {
	text := "No process selected"
	color := uiTheme.accent
	if warn != "" {
		text = warn
		color = uiTheme.danger
	} else if u.selectedPID != 0 {
		spin := ""
		if active {
			spin = u.spinnerNext()
		}
		text = fmt.Sprintf("%s PID %d %s", spin, u.selectedPID, u.selectedExe)
	}
	u.status.SetTextColor(color)
	u.status.SetText(text)
}

func restoreSelection(table *tview.Table, selectIdx, prevIdx, prevCol, rowOff, colOff, length, maxCol int) {
	rowToSelect := 0
	if selectIdx >= 0 && selectIdx < length {
		rowToSelect = selectIdx
	} else if selectIdx < 0 && prevIdx >= 0 && prevIdx < length {
		rowToSelect = prevIdx
	}

	table.Select(rowToSelect+1, prevCol)

	if rowOff > length {
		rowOff = length
	}
	if colOff > maxCol {
		colOff = maxCol
	}
	table.SetOffset(rowOff, colOff)
}

func (s *searchSet) renderResults(selectIdx int) {
	prevRow, prevCol := s.results.GetSelection()
	prevIdx := prevRow - 1
	rowOff, colOff := s.results.GetOffset()

	s.results.Clear()
	s.results.SetCell(0, 0, header("#"))
	s.results.SetCell(0, 1, header("Address"))
	s.results.SetCell(0, 2, header("Current"))
	s.ui.setTableTitle(s.results, s.resultsTitle, "")

	for i, r := range s.rows {
		row := i + 1
		s.results.SetCell(row, 0, bodyCell(fmt.Sprintf("%d", row), row))
		s.results.SetCell(row, 1, bodyCell(fmt.Sprintf("0x%X", r.addr), row))
		s.results.SetCell(row, 2, bodyCell(s.ui.formatValFor(r.dtype, r.current), row))
	}

	if len(s.rows) == 0 {
		s.showResultsMessage("no matches")
		return
	}

	restoreSelection(s.results, selectIdx, prevIdx, prevCol, rowOff, colOff, len(s.rows), 2)

	if len(s.ui.watchedRows) == 0 {
		s.ui.renderWatched(-1)
	}
}

func (u *ui) renderWatched(selectIdx int) {
	prevRow, prevCol := u.watched.GetSelection()
	prevIdx := prevRow - 1
	rowOff, colOff := u.watched.GetOffset()

	u.watched.Clear()
	u.watched.SetCell(0, 0, header("#"))
	u.watched.SetCell(0, 1, header("Address"))
	u.watched.SetCell(0, 2, header("Type"))
	u.watched.SetCell(0, 3, header("Current"))
	u.watched.SetCell(0, 4, header("Desired"))
	u.watched.SetCell(0, 5, header("Pin"))
	u.setTableTitle(u.watched, u.watchedTitle, "")

	if len(u.watchedRows) == 0 {
		u.watched.SetCell(1, 0, tview.NewTableCell("no watched addresses").
			SetSelectable(false).
			SetTextColor(uiTheme.subtleText).
			SetBackgroundColor(uiTheme.surface))
		return
	}

	for i, r := range u.watchedRows {
		row := i + 1
		u.watched.SetCell(row, 0, bodyCell(fmt.Sprintf("%d", row), row))
		u.watched.SetCell(row, 1, bodyCell(fmt.Sprintf("0x%X", r.addr), row))
		u.watched.SetCell(row, 2, bodyCell(r.dtype, row))
		u.watched.SetCell(row, 3, bodyCell(u.formatValFor(r.dtype, r.current), row))
		u.watched.SetCell(row, 4, bodyCell(u.formatValFor(r.dtype, r.desired), row))
		pin := "[ ]"
		pinCell := bodyCell(pin, row)
		if r.pinned {
			pin = "[X]"
			pinCell = bodyCell(pin, row)
			pinCell.SetTextColor(uiTheme.warm)
		}
		u.watched.SetCell(row, 5, pinCell)
	}

	restoreSelection(u.watched, selectIdx, prevIdx, prevCol, rowOff, colOff, len(u.watchedRows), 5)
}

func (u *ui) pinnedLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		u.app.QueueUpdateDraw(func() {
			u.applyPinnedWrites()
		})
	}
}

func (u *ui) applyPinnedWrites() {
	if u.selectedPID == 0 {
		u.updateStatus(false, "")
		return
	}
	hasRows := len(u.watchedRows) > 0
	if !hasRows {
		for _, set := range u.sets {
			if len(set.rows) > 0 {
				hasRows = true
				break
			}
		}
	}
	if !hasRows {
		u.updateStatus(false, "")
		return
	}

	proc, err := process.Open(uint32(u.selectedPID))
	if err != nil {
		u.logf("refresh open error: %v", err)
		u.setTableTitle(u.watched, u.watchedTitle, "")
		for _, set := range u.sets {
			u.setTableTitle(set.results, set.resultsTitle, "")
		}
		u.updateStatus(false, fmt.Sprintf("PID %d unavailable", u.selectedPID))
		return
	}
	defer proc.Close()

	for i := range u.watchedRows {
		r := &u.watchedRows[i]
		if r.pinned {
			cur, err := u.writeByType(proc, r.dtype, r.addr, r.desired)
			if err != nil {
				u.logf("pin write error: %v", err)
				u.updateStatus(false, fmt.Sprintf("PID %d error", u.selectedPID))
				return
			}
			r.current = cur
			continue
		}
		cur, err := u.readByType(proc, r.dtype, r.addr)
		if err != nil {
			u.logf("refresh read error: %v", err)
			u.updateStatus(false, fmt.Sprintf("PID %d error", u.selectedPID))
			return
		}
		r.current = cur
	}

	u.renderWatched(-1)

	for _, set := range u.sets {
		if len(set.rows) == 0 {
			continue
		}
		for i := range set.rows {
			cur, err := u.readByType(proc, set.rows[i].dtype, set.rows[i].addr)
			if err != nil {
				u.logf("refresh read error: %v", err)
				return
			}
			set.rows[i].current = cur
		}
		set.renderResults(-1)
	}

	u.updateStatus(true, "")
}

func (u *ui) selectedWatchedIndex() int {
	return selectedIndex(u.watched, len(u.watchedRows))
}

func (s *searchSet) selectedResultIndex() int {
	return selectedIndex(s.results, len(s.rows))
}

func selectedIndex(table *tview.Table, length int) int {
	row, _ := table.GetSelection()
	idx := row - 1
	if idx < 0 || idx >= length {
		return -1
	}
	return idx
}

func (u *ui) watchSelected() {
	set := u.currentSet()
	if set == nil {
		return
	}
	idx := set.selectedResultIndex()
	if idx < 0 {
		return
	}
	r := set.rows[idx]
	for i, w := range u.watchedRows {
		if w.addr == r.addr && w.dtype == r.dtype {
			u.logf("already watching 0x%X (%s)", r.addr, r.dtype)
			u.renderWatched(i)
			u.app.SetFocus(u.watched)
			return
		}
	}
	u.watchedRows = append(u.watchedRows, resultRow{addr: r.addr, dtype: r.dtype, current: r.current, desired: r.desired})
	u.logf("watching 0x%X (%s)", r.addr, r.dtype)
	u.renderWatched(len(u.watchedRows) - 1)
	u.app.SetFocus(u.watched)
}

func (u *ui) unwatchSelected() {
	idx := u.selectedWatchedIndex()
	if idx < 0 {
		return
	}
	u.watchedRows = append(u.watchedRows[:idx], u.watchedRows[idx+1:]...)
	prev := idx - 1
	if prev < 0 && len(u.watchedRows) > 0 {
		prev = 0
	}
	u.renderWatched(prev)
}

func (u *ui) togglePin() {
	idx := u.selectedWatchedIndex()
	if idx < 0 {
		return
	}
	u.watchedRows[idx].pinned = !u.watchedRows[idx].pinned
	u.renderWatched(idx)
}

func (u *ui) writeDesired() {
	idx := u.selectedWatchedIndex()
	if idx < 0 {
		return
	}
	row := &u.watchedRows[idx]
	if u.selectedPID == 0 {
		u.logf("write skipped: no process selected")
		return
	}
	proc, err := process.Open(uint32(u.selectedPID))
	if err != nil {
		u.logf("write open error: %v", err)
		return
	}
	defer proc.Close()

	cur, err := u.writeByType(proc, row.dtype, row.addr, row.desired)
	if err != nil {
		u.logf("write error: %v", err)
		return
	}
	row.current = cur
	u.logf("wrote 0x%X (%s) -> %s", row.addr, row.dtype, u.formatValFor(row.dtype, row.desired))
	u.renderWatched(idx)
}

func (u *ui) parseValue(dtype, valStr string) (numericValue, error) {
	valStr = strings.TrimSpace(valStr)
	if valStr == "" {
		return numericValue{}, fmt.Errorf("enter a value to search")
	}

	switch dtype {
	case "int32":
		v, err := strconv.ParseInt(valStr, 10, 32)
		if err != nil {
			return numericValue{}, parseNumericError("int32", valStr, err)
		}
		return numericValue{i64: int64(int32(v))}, nil
	case "int64":
		v, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return numericValue{}, parseNumericError("int64", valStr, err)
		}
		return numericValue{i64: v}, nil
	case "uint32":
		v, err := strconv.ParseUint(valStr, 10, 32)
		if err != nil {
			return numericValue{}, parseNumericError("uint32", valStr, err)
		}
		return numericValue{u64: uint64(uint32(v))}, nil
	case "uint64":
		v, err := strconv.ParseUint(valStr, 10, 64)
		if err != nil {
			return numericValue{}, parseNumericError("uint64", valStr, err)
		}
		return numericValue{u64: v}, nil
	case "float32":
		v, err := strconv.ParseFloat(valStr, 32)
		if err != nil {
			return numericValue{}, parseNumericError("float32", valStr, err)
		}
		f32 := float32(v)
		return numericValue{f64: float64(f32)}, nil
	case "float64":
		v, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return numericValue{}, parseNumericError("float64", valStr, err)
		}
		return numericValue{f64: v}, nil
	default:
		return numericValue{}, fmt.Errorf("unsupported type: %s", dtype)
	}
}

func parseNumericError(dtype, valStr string, err error) error {
	var nerr *strconv.NumError
	if errors.As(err, &nerr) {
		switch {
		case errors.Is(nerr.Err, strconv.ErrRange):
			return fmt.Errorf("invalid %s: value out of range", dtype)
		case errors.Is(nerr.Err, strconv.ErrSyntax):
			return fmt.Errorf("invalid %s: enter a %s value (got %q)", dtype, dtype, valStr)
		}
	}

	return fmt.Errorf("invalid %s: %v", dtype, err)
}

func (u *ui) scanByType(proc *process.Process, dtype string, val numericValue) ([]uintptr, error) {
	const maxResults = 0

	switch dtype {
	case "int32":
		return proc.ScanInt32(int32(val.i64), maxResults, false)
	case "int64":
		return proc.ScanInt64(val.i64, maxResults, false)
	case "uint32":
		return proc.ScanUint32(uint32(val.u64), maxResults, false)
	case "uint64":
		return proc.ScanUint64(val.u64, maxResults, false)
	case "float32":
		return proc.ScanFloat32Approx(float32(val.f64), 1e-4, maxResults, false)
	case "float64":
		return proc.ScanFloat64Approx(val.f64, 1e-6, maxResults, false)
	default:
		return nil, fmt.Errorf("unsupported type: %s", dtype)
	}
}

func (u *ui) readByType(proc *process.Process, dtype string, addr uintptr) (numericValue, error) {
	switch dtype {
	case "int32":
		v, err := proc.ReadInt32(addr)
		return numericValue{i64: int64(v)}, err
	case "int64":
		v, err := proc.ReadInt64(addr)
		return numericValue{i64: v}, err
	case "uint32":
		v, err := proc.ReadUint32(addr)
		return numericValue{u64: uint64(v)}, err
	case "uint64":
		v, err := proc.ReadUint64(addr)
		return numericValue{u64: v}, err
	case "float32":
		v, err := proc.ReadFloat32(addr)
		return numericValue{f64: float64(v)}, err
	case "float64":
		v, err := proc.ReadFloat64(addr)
		return numericValue{f64: v}, err
	default:
		return numericValue{}, fmt.Errorf("unsupported type: %s", dtype)
	}
}

func (u *ui) writeByType(proc *process.Process, dtype string, addr uintptr, val numericValue) (numericValue, error) {
	switch dtype {
	case "int32":
		cur, err := proc.WriteInt32AndRead(addr, int32(val.i64))
		return numericValue{i64: int64(cur)}, err
	case "int64":
		cur, err := proc.WriteInt64AndRead(addr, val.i64)
		return numericValue{i64: cur}, err
	case "uint32":
		cur, err := proc.WriteUint32AndRead(addr, uint32(val.u64))
		return numericValue{u64: uint64(cur)}, err
	case "uint64":
		cur, err := proc.WriteUint64AndRead(addr, val.u64)
		return numericValue{u64: cur}, err
	case "float32":
		cur, err := proc.WriteFloat32AndRead(addr, float32(val.f64))
		return numericValue{f64: float64(cur)}, err
	case "float64":
		cur, err := proc.WriteFloat64AndRead(addr, val.f64)
		return numericValue{f64: cur}, err
	default:
		return numericValue{}, fmt.Errorf("unsupported type: %s", dtype)
	}
}

func (u *ui) makeComparator(dtype string, target numericValue) func(cur numericValue) bool {
	switch dtype {
	case "float32":
		const eps = 1e-4
		return func(cur numericValue) bool {
			return math.Abs(cur.f64-target.f64) <= eps
		}
	case "float64":
		const eps = 1e-6
		return func(cur numericValue) bool {
			return math.Abs(cur.f64-target.f64) <= eps
		}
	default:
		return func(cur numericValue) bool {
			return cur.i64 == target.i64 && cur.u64 == target.u64
		}
	}
}

func (u *ui) formatValFor(dtype string, v numericValue) string {
	switch dtype {
	case "int32", "int64":
		return fmt.Sprintf("%d", v.i64)
	case "uint32", "uint64":
		return fmt.Sprintf("%d", v.u64)
	case "float32":
		return fmt.Sprintf("%.4f", v.f64)
	case "float64":
		return fmt.Sprintf("%.6f", v.f64)
	default:
		return fmt.Sprintf("%.4f", v.f64)
	}
}

func (u *ui) editDesired() {
	idx := u.selectedWatchedIndex()
	if idx < 0 {
		return
	}
	row := u.watchedRows[idx]
	dtype := row.dtype
	label := fmt.Sprintf("Desired %s ", dtype)
	input := tview.NewInputField().
		SetLabel(label).
		SetText(u.formatValFor(dtype, row.desired))
	pin := tview.NewCheckbox().
		SetLabel("Pin ").
		SetChecked(row.pinned)

	form := tview.NewForm().
		AddFormItem(input).
		AddFormItem(pin).
		AddButton("Save", func() {
			val, err := u.parseValue(dtype, input.GetText())
			if err != nil {
				input.SetLabel("Invalid value ")
				return
			}
			u.watchedRows[idx].desired = val
			u.watchedRows[idx].pinned = pin.IsChecked()
			u.app.SetRoot(u.layout(), true)
			u.renderWatched(idx)
			u.app.SetFocus(u.watched)
		}).
		AddButton("Cancel", func() {
			u.app.SetRoot(u.layout(), true)
			u.app.SetFocus(u.watched)
		})
	form.SetBorder(true).SetTitle("Edit Desired")

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 50, 0, true).
			AddItem(nil, 0, 1, false), 10, 0, true).
		AddItem(nil, 0, 1, false)

	u.app.SetRoot(modal, true)
	u.app.SetFocus(input)
}

func header(text string) *tview.TableCell {
	return tview.NewTableCell(text).
		SetSelectable(false).
		SetAttributes(tcell.AttrBold).
		SetTextColor(uiTheme.accent).
		SetBackgroundColor(uiTheme.headerBg)
}

func (u *ui) quickNavigateProcesses(ch rune) {
	if len(u.procs) == 0 {
		return
	}

	target := unicode.ToLower(ch)
	start := 0
	if target == u.lastNavRune {
		if row, _ := u.table.GetSelection(); row > 0 {
			start = (row - 1) + 1
		}
	}

	for i := 0; i < len(u.procs); i++ {
		idx := (start + i) % len(u.procs)
		name := strings.ToLower(u.procs[idx].name)
		if strings.HasPrefix(name, string(target)) {
			u.table.Select(idx+1, 0)
			u.updateSelection(idx + 1)
			u.lastNavRune = target
			return
		}
	}

	u.lastNavRune = 0
}
