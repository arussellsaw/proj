package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/machinebox/graphql"
)

func GetProject(ctx context.Context, id int) (*ProjectQueryResponse, error) {
	client := graphql.NewClient("https://api.github.com/graphql")
	req := graphql.NewRequest(viewProjectQuery)
	req.Var("project", id)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("GITHUB_TOKEN"))

	res := ProjectQueryResponse{}
	err := client.Run(ctx, req, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
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
	Name    string  `json:"name"`
	Number  int     `json:"number"`
}
type Organization struct {
	Project Project `json:"project"`
}
type ProjectQueryResponse struct {
	Organization Organization `json:"organization"`
}

const viewProjectQuery = `query viewProject($project: Int!) {
  organization(login: "sourcegraph") {
    project(number: $project) {
			name
			number
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
	`

func AssignIssue(ctx context.Context, user string, issue Content) error {
	userID, err := getUserID(ctx, user)
	if err != nil {
		return err
	}
	client := graphql.NewClient("https://api.github.com/graphql")
	req := graphql.NewRequest(`mutation assignUser($userid: ID! $assignableid: ID!) {
		addAssigneesToAssignable(input: {clientMutationId: "proj", assignableId: $assignableid, assigneeIds: [$userid]}) {
				clientMutationId
			}
	}`)
	req.Var("userid", userID)
	req.Var("assignableid", issue.ID)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("GITHUB_TOKEN"))

	res := struct{}{}
	err = client.Run(ctx, req, &res)
	if err != nil {
		return err
	}
	return nil
}

func UnassignIssue(ctx context.Context, user string, issue Content) error {
	userID, err := getUserID(ctx, user)
	if err != nil {
		return err
	}
	client := graphql.NewClient("https://api.github.com/graphql")
	req := graphql.NewRequest(`mutation unassignUser($userid: ID! $assignableid: ID!) {
		removeAssigneesFromAssignable(input: {clientMutationId: "proj", assignableId: $assignableid, assigneeIds: [$userid]}) {
				clientMutationId
			}
	}`)
	req.Var("userid", userID)
	req.Var("assignableid", issue.ID)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("GITHUB_TOKEN"))

	res := struct{}{}
	err = client.Run(ctx, req, &res)
	if err != nil {
		return err
	}
	return nil
}

func CloseIssue(ctx context.Context, issue Content) error {
	client := graphql.NewClient("https://api.github.com/graphql")
	req := graphql.NewRequest(`mutation closeIssue($issueid: String!) {
			closeIssue(input: {clientMutationId: "proj", issueId: $issueid}) {
				clientMutationId
			}
	}`)
	req.Var("issueid", issue.ID)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("GITHUB_TOKEN"))

	res := struct{}{}
	err := client.Run(ctx, req, &res)
	if err != nil {
		return err
	}
	return nil
}

func ReopenIssue(ctx context.Context, issue Content) error {
	client := graphql.NewClient("https://api.github.com/graphql")
	req := graphql.NewRequest(`mutation reopenIssue($issueid: String!) {
			reopenIssue(input: {clientMutationId: "proj", issueId: $issueid}) {
				clientMutationId
			}
	}`)
	req.Var("issueid", issue.ID)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("GITHUB_TOKEN"))

	res := struct{}{}
	err := client.Run(ctx, req, &res)
	if err != nil {
		return err
	}
	return nil
}

func MoveCard(ctx context.Context, issue Content, projectID int, colName string) error {
	proj, err := GetProject(ctx, projectID)
	if err != nil {
		return err
	}
	var (
		colID  string
		cardID string
	)
	for _, col := range proj.Organization.Project.Columns.Nodes {
		if strings.ToLower(colName) == strings.ToLower(col.Name) ||
			strings.ToLower(strings.Replace(col.Name, " ", "", -1)) == strings.ToLower(colName) {
			colID = col.ID
		}
		for _, card := range col.Cards.Nodes {
			if card.Content.Number == issue.Number {
				cardID = card.ID
			}
		}
	}
	if colID == "" || cardID == "" {
		return fmt.Errorf("couldn't move card: cardid: %s colid: %s", cardID, colID)
	}
	client := graphql.NewClient("https://api.github.com/graphql")
	req := graphql.NewRequest(`mutation moveCard($cardid: ID!, $colid: ID!) {
			moveProjectCard(input: {cardId: $cardid, columnId: $colid}) {
				clientMutationId
			}
	}`)
	req.Var("colid", colID)
	req.Var("cardid", cardID)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("GITHUB_TOKEN"))

	res := struct{}{}
	err = client.Run(ctx, req, &res)
	if err != nil {
		return err
	}
	return nil
}

func getUserID(ctx context.Context, user string) (string, error) {
	client := graphql.NewClient("https://api.github.com/graphql")
	req := graphql.NewRequest(`query getUserID($login: String!){
		user(login: $login) {
			id
		}
	}`)
	req.Var("login", user)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("GITHUB_TOKEN"))

	res := struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}{}
	err := client.Run(ctx, req, &res)
	if err != nil {
		return "", err
	}

	return res.User.ID, nil
}
