package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/gdamore/tcell/v2"
	"github.com/machinebox/graphql"
	"github.com/rivo/tview"
)

func main() {
	projectNumber := flag.Int("p", 0, "project number")
	user := flag.String("u", "", "filter by user")
	interactive := flag.Bool("i", false, "interactive mode")
	flag.Parse()
	client := graphql.NewClient("https://api.github.com/graphql")
	req := graphql.NewRequest(`query viewProject($project: Int!) {
  organization(login: "sourcegraph") {
    project(number: $project) {
      columns(first: 10) {
        nodes {
          cards {
            nodes {
              id
							note
              content {
                ... on Issue {
                  id
                  author {
                    login
                  }
                  number
                  title
									url
									assignees(first: 10) {
                    edges {
                      node {
                        login
                      }
                    }
                  }
                }
                ... on PullRequest {
                  id
                  author {
                    login
                  }
                  number
                  title
									url
                }
              }
            }
          }
          name
          id
        }
      }
    }
  }
}
	`)
	req.Var("project", *projectNumber)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("GITHUB_TOKEN"))

	ctx := context.Background()
	res := Data{}
	err := client.Run(ctx, req, &res)
	if err != nil {
		panic(err)
	}
	err = client.Run(ctx, req, &res)
	if err != nil {
		panic(err)
	}
	if !*interactive {
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 1, ' ', 0)
		for _, col := range res.Organization.Project.Columns.Nodes {
			fmt.Fprintf(w, "%s\t%s\t\t\n", color.GreenString(" "), color.GreenString(col.Name))
			for _, card := range col.Cards.Nodes {
				if *user != "" && getOwner(card.Content) != *user {
					continue
				}
				if card.Note != "" {
					fmt.Fprintf(w, "%s\t%s\t%s\t\n", color.GreenString(" "), color.GreenString(" "), capStr(card.Note, 60))
					continue
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					color.BlueString(fmt.Sprintf("%v", card.Content.Number)),
					color.MagentaString(getOwner(card.Content)),
					capStr(card.Content.Title, 60),
					color.CyanString(card.Content.URL))
			}
		}
		w.Flush()
	} else {
		doTUI(res)
	}
}

func doTUI(res Data) {
	app := tview.NewApplication()
	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Clear()
		return false
	})
	table := tview.NewTable()
	table.SetBackgroundColor(tcell.ColorBlack)
	n := -1
	for _, col := range res.Organization.Project.Columns.Nodes {
		n++
		name := col.Name
		table.SetCell(n, 1, tview.NewTableCell(name).SetTextColor(tcell.ColorGreen))
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
				tview.NewTableCell(url).SetTextColor(tcell.ColorBlueViolet),
			)
		}
	}

	table.SetSelectable(true, false)

	flex := tview.NewFlex()
	flex.AddItem(table, 0, 3, true)

	textbox := tview.NewTextView()
	textbox.Box.SetBorder(true)

	hasBox := false

	table.SetSelectedFunc(func(row int, column int) {
		cell := table.GetCell(row, 3)
		if cell.Text == "" {
			table.Select(row+1, column)
		}
		if cell.Text == "" || strings.Contains(cell.Text, "pull") {
			return
		}
		buf := bytes.Buffer{}
		errBuf := bytes.Buffer{}
		cmd := exec.Command("gh", "issue", "view", cell.Text, "--comments")
		cmd.Stdout = &buf
		cmd.Stderr = &errBuf
		err := cmd.Run()
		if err != nil {
			panic(err.Error() + errBuf.String())
		}
		textbox.SetText(buf.String())
		if !hasBox {
			flex.AddItem(textbox, 0, 3, true)
			hasBox = true
		}
		textbox.ScrollToBeginning()
		app.SetFocus(textbox)
	})

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape && hasBox {
			flex.RemoveItem(textbox)
			app.SetFocus(table)
			hasBox = false
			return nil
		}
		return event
	})

	if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
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

func getOwner(c Content) string {
	if len(c.Assignees.Edges) == 0 {
		return c.Author.Login
	}
	return c.Assignees.Edges[0].Node.Login
}

func capStr(s string, max int) string {
	if len(s) < max {
		return s
	}
	return s[:max] + "..."
}

type Author struct {
	Login string `json:"login"`
}
type Content struct {
	ID        string    `json:"id"`
	Author    Author    `json:"author"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Assignees Assignees `json:"assignees"`
}

type Assignees struct {
	Edges []AssigneeNode `json:"edges"`
}
type AssigneeNode struct {
	Node Assignee `json:"node"`
}

type Assignee struct {
	Login string `json:"login"`
}

type Node struct {
	ID      string  `json:"id"`
	Content Content `json:"content"`
	Note    string  `json:"note"`
}
type Cards struct {
	Nodes []Node `json:"nodes"`
}
type ColumnNode struct {
	Cards Cards  `json:"cards"`
	Name  string `json:"name"`
	ID    string `json:"id"`
}
type Columns struct {
	Nodes []ColumnNode `json:"nodes"`
}
type Project struct {
	Columns Columns `json:"columns"`
}
type Organization struct {
	Project Project `json:"project"`
}
type Data struct {
	Organization Organization `json:"organization"`
}
