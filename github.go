package main

import (
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"strings"
	"context"
)

type githubClient struct {
	client *github.Client
	owner  string
	repo   string
	ref    string
}

func NewClient(owner, repo, ref, token string) githubClient {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	return githubClient{
		owner:  owner,
		repo:   repo,
		ref:    ref,
		client: github.NewClient(tc),
	}
}

func createStatus(client *github.Client, owner, repo, ref string, status *github.RepoStatus) error {
	ctx := context.Background()
	_, _, err := client.Repositories.CreateStatus(ctx, owner, repo, ref, status)

	return err
}

func targetURL(g githubClient) string {
	return fmt.Sprintf("https://github.com/%s/%s/commit/%s", g.owner, g.repo, g.ref)
}

func (g githubClient) pendingStatus() error {
	status := NewRepoStatus("pending", targetURL(g), "The build is pending")

	return createStatus(g.client, g.owner, g.repo, g.ref, status)
}

func (g githubClient) successStatus(target string) error {
	if target == "" {
		target = targetURL(g)
	}

	log.Info(target)

	status := NewRepoStatus("success", target, "The build succeeded!")

	return createStatus(g.client, g.owner, g.repo, g.ref, status)
}

func (g githubClient) failureStatus(target string) error {
	if target == "" {
		target = targetURL(g)
	}

	log.Info(target)

	status := NewRepoStatus("failure", target, "The build failed!")

	return createStatus(g.client, g.owner, g.repo, g.ref, status)
}

func NewRepoStatus(state, target, description string) *github.RepoStatus {
	return &github.RepoStatus{
		State:       &state,
		TargetURL:   &target,
		Description: &description,
	}
}

func includeActions(action string, includes []string) bool {
	if len(includes) == 0 {
		return true
	}

	for _, i := range includes {
		if action == i {
			return true
		}
	}

	return false
}

func excludeActions(action string, excludes []string) bool {
	if len(excludes) == 0 {
		return false
	}

	for _, e := range excludes {
		if action == e {
			return true
		}
	}

	return false
}

func parseBranch(payload interface{}) string {
	j := payload.(map[string]interface{})
	if _, ok := j["ref"]; !ok {
		return ""
	}

	branches := strings.SplitN(j["ref"].(string), "/", 3)

	if len(branches) != 3 {
		return ""
	}

	return branches[2]
}

// Note: https://developer.github.com/v3/activity/events/types/
// Include action field: return action field
// Not Incluade action field: return parse created, deleted, and forced field when push
func parseAction(payload interface{}) string {
	j := payload.(map[string]interface{})

	// include action field
	if _, ok := j["action"]; ok {
		return j["action"].(string)
	}

	// not include action field
	if created, ok := j["created"]; ok && created.(bool) {
		return "created"
	}

	if deleted, ok := j["deleted"]; ok && deleted.(bool) {
		return "deleted"
	}

	if forced, ok := j["forced"]; ok && forced.(bool) {
		return "forced"
	}

	return ""
}

func parsePullRequestStatus(payload interface{}) (string, string, string) {
	j := payload.(map[string]interface{})
	if _, ok := j["pull_request"]; !ok {
		return "", "", ""
	}

	s := j["pull_request"].(map[string]interface{})["_links"].(map[string]interface{})["statuses"].(map[string]interface{})["href"].(string)
	statuses := strings.Split(s, "/")

	return statuses[4], statuses[5], statuses[7]
}
