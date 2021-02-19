package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func doTUI(res *ProjectQueryResponse) {
	app := tview.NewApplication()
	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Clear()
		return false
	})
	table := tview.NewTable()
	table.SetBackgroundColor(tcell.ColorDefault)
	selected := tcell.Style{}.Reverse(true)
	//selected := tcell.Style{}.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	table.SetSelectedStyle(selected)
	n := -1
	issues := make(map[string]Content)
	for _, col := range res.Organization.Project.Columns.Nodes {
		n++
		name := col.Name
		table.SetCell(n, 1, tview.NewTableCell(name).SetTextColor(tcell.ColorGreen))
		table.SetCell(n, 2, tview.NewTableCell(res.Organization.Project.Name).SetTextColor(tcell.ColorGreen))
		for _, card := range col.Cards.Nodes {
			n++
			if card.Content.Number == 0 {
				table.SetCell(
					n, 0,
					tview.NewTableCell("note").SetTextColor(tcell.ColorWhite),
				)
				table.SetCell(
					n, 2,
					tview.NewTableCell(capStr(card.Note, 60)),
				)
				continue
			}
			number := fmt.Sprint(card.Content.Number)
			owner := getOwner(card.Content)
			title := capStr(card.Content.Title, 60)
			url := card.Content.URL

			issues[number] = card.Content

			table.SetCell(
				n, 0,
				tview.NewTableCell(number).SetTextColor(tcell.ColorBlue),
			)
			table.SetCell(
				n, 1,
				tview.NewTableCell(owner).SetTextColor(tcell.ColorFuchsia),
			)
			table.SetCell(
				n, 2,
				tview.NewTableCell(title),
			)
			table.SetCell(
				n, 3,
				tview.NewTableCell(url).SetTextColor(tcell.ColorLavender),
			)
		}
	}

	table.SetSelectable(true, false)

	flex := tview.NewFlex()
	flex.AddItem(table, 0, 3, true)

	textbox := tview.NewTextView()
	textbox.Box.SetBorder(true)
	textbox.SetBackgroundColor(tcell.ColorDefault)

	var focusIssue string

	table.SetSelectedFunc(func(row int, column int) {
		cell := table.GetCell(row, 3)
		if cell.Text == "" {
			table.Select(row+1, column)
		}
		resource := ""
		if strings.Contains(cell.Text, "pull") {
			resource = "pr"
		} else {
			resource = "issue"
		}
		buf := bytes.Buffer{}
		errBuf := bytes.Buffer{}
		cmd := exec.Command("gh", resource, "view", cell.Text)
		cmd.Stdout = &buf
		cmd.Stderr = &errBuf
		err := cmd.Run()
		if err != nil {
			panic(err.Error() + errBuf.String())
		}
		textbox.SetText(buf.String())
		if focusIssue == "" {
			flex.AddItem(textbox, 0, 3, true)
			focusIssue = table.GetCell(row, 0).Text
			if err != nil {
				panic(err)
			}
		}
		textbox.ScrollToBeginning()
		app.SetFocus(textbox)
	})

	inputField := tview.NewInputField().SetFieldTextColor(tcell.ColorBlack)
	inputField.SetDoneFunc(func(key tcell.Key) {
		defer app.SetFocus(table)
		if key != tcell.KeyEnter {
			return
		}
		args := strings.Split(inputField.GetText(), " ")
		if len(args) == 0 {
			return
		}
		row, _ := table.GetSelection()
		issue := table.GetCell(row, 0).Text
		if focusIssue != "" {
			issue = focusIssue
		}
		switch args[0] {
		case ":assign":
			if len(args) >= 3 {
				issue = args[2]
			}
			inputField.SetText(fmt.Sprintf("assigning %s to %s", args[1], issue))
			err := AssignIssue(context.Background(), args[1], issues[issue])
			if err != nil {
				panic(err)
			}
		case ":close":
		case ":q":
			app.Stop()
		}
	})

	vstack := tview.NewFlex().SetDirection(tview.FlexRow)
	vstack.AddItem(flex, 0, 1000, true)
	vstack.AddItem(inputField, 0, 1, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape && focusIssue != "" {
			flex.RemoveItem(textbox)
			app.SetFocus(table)
			focusIssue = ""
			return nil
		}
		if event.Rune() == ':' {
			inputField.SetText(":")
			app.SetFocus(inputField)
			return nil
		}
		return event
	})

	if err := app.SetRoot(vstack, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func getURL(table *tview.Table, row int) string {
	cell := table.GetCell(row, 3)
	if cell == nil {
		return ""
	}
	return cell.Text
}
