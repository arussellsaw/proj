package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
)

func main() {
	projectNumber := flag.Int("p", 0, "project number")
	user := flag.String("u", "", "filter by user")
	interactive := flag.Bool("i", false, "interactive mode")
	flag.Parse()

	ctx := context.Background()
	if !*interactive {
	res, err := GetProject(ctx, *projectNumber)
	if err != nil {
		log.Fatal(err)
	}
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
		doTUI(ctx, *projectNumber)
	}
}

func getOwner(c Content) string {
	if strings.Contains(c.URL, "pull") {
		return c.Author.Login
	}
	if len(c.Assignees.Edges) == 0 {
		return ""
	}
	return c.Assignees.Edges[0].Node.Login
}

func capStr(s string, max int) string {
	if len(s) < max {
		return s
	}
	return s[:max] + "..."
}
