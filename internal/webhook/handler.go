package webhook

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	webhookGithub "github.com/go-playground/webhooks/v6/github"
	apiGithub "github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"

	"example.com/nopejsbot/internal/ai"
)

var hook *webhookGithub.Webhook

func Handler() http.Handler {
	secret := os.Getenv("WEBHOOK_SECRET")
	h, err := webhookGithub.New(webhookGithub.Options.Secret(secret))
	if err != nil {
		panic(err)
	}
	hook = h

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, err := hook.Parse(r, webhookGithub.IssueCommentEvent)
		if err != nil {
			if err == webhookGithub.ErrEventNotFound {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Webhook parse error: %s", err)
			return
		}

		switch event := payload.(type) {
		case webhookGithub.IssueCommentPayload:
			if event.Action == "created" &&
				event.Issue.PullRequest.URL != "" &&
				containsTrigger(event.Comment.Body) {
				go handleComment(&event)
			}
		}

		w.WriteHeader(http.StatusOK)
	})
}

func containsTrigger(body string) bool {
	return strings.Contains(strings.ToLower(body), "@nopejsbot")
}

func extractPrompt(body string) string {
	lower := strings.ToLower(body)
	index := strings.Index(lower, "@nopejsbot")
	if index == -1 {
		return strings.TrimSpace(body)
	}
	return strings.TrimSpace(body[index+len("@nopejsbot"):])
}

func handleComment(event *webhookGithub.IssueCommentPayload) {
	ctx := context.Background()

	diff, err := fetchDiff(ctx, event)
	if err != nil {
		fmt.Println("Error fetching diff:", err)
		return
	}

	prompt := extractPrompt(event.Comment.Body)

	reply, err := ai.ExplainDiff(ctx, diff, prompt)
	if err != nil {
		fmt.Println("AI error:", err)
		return
	}

	if err := postComment(ctx, event, reply); err != nil {
		fmt.Println("Post comment error:", err)
	}
}

func fetchDiff(ctx context.Context, ev *webhookGithub.IssueCommentPayload) (string, error) {
	client := apiGithub.NewClient(nil)

	prNum := int(ev.Issue.Number)
	repo := ev.Repository.Name
	owner := ev.Repository.Owner.Login

	opts := &apiGithub.ListOptions{PerPage: 100}
	files, _, err := client.PullRequests.ListFiles(ctx, owner, repo, prNum, opts)
	if err != nil {
		return "", err
	}

	var diff string
	for _, f := range files {
		if patch := f.GetPatch(); patch != "" {
			diff += patch + "\n"
		}
	}
	return diff, nil
}

func postComment(ctx context.Context, ev *webhookGithub.IssueCommentPayload, body string) error {
	token := os.Getenv("GH_TOKEN")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := apiGithub.NewClient(tc)

	comment := &apiGithub.IssueComment{
		Body: apiGithub.String("NopeJSbot:\n" + body),
	}

	_, _, err := client.Issues.CreateComment(
		ctx,
		ev.Repository.Owner.Login,
		ev.Repository.Name,
		int(ev.Issue.Number),
		comment,
	)
	return err
}