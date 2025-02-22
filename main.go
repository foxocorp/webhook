package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"webhook/events"
)

type Discord struct {
	Content   string `json:"content"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

type Credentials struct {
	ID    string
	Token string
}

const (
	discordBaseURL = "https://discord.com/api"
	githubEvent    = "X-GitHub-Event"
)

func main() {
	http.HandleFunc("/{id}/{token}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}

		creds := Credentials{
			ID:    r.PathValue("id"),
			Token: r.PathValue("token"),
		}

		event := r.Header.Get(githubEvent)
		if strings.TrimSpace(creds.ID) == "" || strings.TrimSpace(creds.Token) == "" || strings.TrimSpace(event) == "" {
			return
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				slog.Error(err.Error())
			}
		}(r.Body)

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error(err.Error())
		}

		go parseEvent(event, bodyBytes, creds)
		w.WriteHeader(204)
	})

	slog.Info("Starting server")
	slog.Error(http.ListenAndServe(":8080", nil).Error())
}

func parseEvent(event string, data []byte, creds Credentials) {
	var content, username, avatar string

	switch event {
	case "push":
		e := events.Push{}
		err := parseJSON(data, &e)
		if err != nil {
			slog.Error(err.Error())
			return
		}

		builder := strings.Builder{}

		for _, c := range e.Commits {
			builder.WriteString(fmt.Sprintf("[`%s`](<%s>) %s\n", c.Id[:7], c.Url, c.Message))
		}

		branch := strings.TrimPrefix(e.Ref, "refs/heads/")

		builder.WriteString(fmt.Sprintf(
			"\n- [%s](<%s>) on [%s](<%s>)/[%s](<%s>)",
			e.Pusher.Name,
			e.Sender.HtmlUrl,
			e.Repository.Name,
			e.Repository.HtmlUrl,
			branch,
			e.Repository.HtmlUrl+"/tree/"+branch,
		))

		content, username, avatar = builder.String(), e.Pusher.Name, e.Sender.AvatarUrl
	case "workflow_run":
		e := events.WorkflowRun{}
		err := parseJSON(data, &e)
		if err != nil {
			slog.Error(err.Error())
			return
		}

		if e.Action != "completed" {
			return
		}

		if (strings.HasPrefix(e.Workflow.Name, "CodeQL") || e.Workflow.Name == "Dependabot Updates") && *e.WorkflowRun.Conclusion == "success" {
			return
		}

		emoji := "<:pepethinking:1330806911141941249>"

		if *e.WorkflowRun.Conclusion == "failure" {
			emoji = "<:catscream:1325122976575655936>"
		}

		formattedContent := fmt.Sprintf("%s Workflow [%s](<%s>) on [%s](<%s>)/[%s](<%s>)",
			emoji,
			*e.WorkflowRun.Conclusion,
			e.WorkflowRun.HtmlUrl,
			e.Repository.Name,
			e.Repository.HtmlUrl,
			*e.WorkflowRun.HeadBranch,
			e.Repository.HtmlUrl+"/tree/"+*e.WorkflowRun.HeadBranch,
		)

		content, username, avatar = formattedContent, e.Workflow.Name, e.Sender.AvatarUrl
	case "star":
		e := events.Star{}
		err := parseJSON(data, &e)
		if err != nil {
			slog.Error(err.Error())
			return
		}

		if e.Action != "created" {
			return
		}

		formattedContent := fmt.Sprintf("[%s](<%s>) starred [%s](<%s>) <:foxtada:1311327105300172882>",
			e.Sender.Login,
			e.Sender.HtmlUrl,
			e.Repository.Name,
			e.Repository.HtmlUrl,
		)

		content, username, avatar = formattedContent, e.Sender.Login, e.Sender.AvatarUrl
	case "fork":
		e := events.Fork{}
		err := parseJSON(data, &e)
		if err != nil {
			slog.Error(err.Error())
			return
		}

		formattedContent := fmt.Sprintf("[%s](<%s>) forked [%s](<%s>)",
			e.Sender.Login,
			e.Sender.HtmlUrl,
			e.Forkee.Name,
			e.Forkee.HtmlUrl,
		)

		content, username, avatar = formattedContent, e.Sender.Login, e.Sender.AvatarUrl
	}

	executeWebhook(content, username, avatar, creds)
}

func executeWebhook(content, username, avatar string, creds Credentials) {
	body := Discord{
		Content:   content,
		Username:  username,
		AvatarURL: avatar,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	url := fmt.Sprintf("%s/webhooks/%s/%s", discordBaseURL, creds.ID, creds.Token)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		slog.Error(err.Error())
		return
	}

	if resp.StatusCode != 204 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error(err.Error())
		}

		slog.Error("discord error", "err", respBody)
	}
}

func parseJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
