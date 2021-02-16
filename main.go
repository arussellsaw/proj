package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/machinebox/graphql"
)

func main() {
	projectNumber := flag.Int("p", 0, "project number")
	user := flag.String("u", "", "filter by user")
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
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", color.BlueString(fmt.Sprintf("%v", card.Content.Number)), color.MagentaString(getOwner(card.Content)), capStr(card.Content.Title, 60), color.CyanString(card.Content.URL))
		}
	}
	w.Flush()
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
