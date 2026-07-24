// Package readme renders the README template using data fetched from GitHub.
package readme

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"text/template"
	"time"

	"github.com/google/go-github/v89/github"
	"golang.org/x/oauth2"
)

// Repo is a minimal view of a GitHub repository used in the README template.
type Repo struct {
	Name        string
	URL         string
	Description string
}

// Contribution is a recent activity entry on a repository.
type Contribution struct {
	Repo       Repo
	OccurredAt time.Time
}

// Data is the data passed to the README template.
type Data struct {
	Username string
	// RecentCreatedRepos is populated by the recentCreatedRepos template func.
	// RecentContributions is populated by the recentContributions template func.
}

// Client is a thin wrapper around the GitHub API client.
type Client struct {
	gh *github.Client
}

// NewClient returns a Client authenticated with the given token (if non-empty).
func NewClient(ctx context.Context, token string) *Client {
	_ = ctx
	var opts []github.ClientOptionsFunc
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		httpClient := oauth2.NewClient(ctx, ts)
		opts = append(opts, github.WithHTTPClient(httpClient))
	}
	c, err := github.NewClient(opts...)
	if err != nil {
		log.Fatalf("github.NewClient: %v", err)
	}
	return &Client{gh: c}
}

// Fetch collects the data needed to render the README for the given user.
func (c *Client) Fetch(ctx context.Context, user string) (*Data, error) {
	_ = ctx
	_ = user
	// The template pulls data via template functions which lazily call
	// back into the client; we just hand the client to the template here.
	return &Data{Username: user}, nil
}

// recentCreatedRepos returns the n most recently created public repositories
// owned by user.
func (c *Client) recentCreatedRepos(user string, n int) ([]Repo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opt := &github.RepositoryListByUserOptions{
		Sort:        "created",
		Direction:   "desc",
		ListOptions: github.ListOptions{PerPage: max(n*4, 100)},
		Type:        "public",
	}
	repos, _, err := c.gh.Repositories.ListByUser(ctx, user, opt)
	if err != nil {
		return nil, fmt.Errorf("list user repos: %w", err)
	}
	if n > 0 && len(repos) > n {
		repos = repos[:n]
	}
	out := make([]Repo, 0, len(repos))
	for _, r := range repos {
		if r == nil {
			continue
		}
		out = append(out, Repo{
			Name:        r.GetName(),
			URL:         r.GetHTMLURL(),
			Description: r.GetDescription(),
		})
	}
	return out, nil
}

// recentContributions returns up to n recent public activity events for user,
// deduplicated by repository (most recent occurrence per repo wins).
func (c *Client) recentContributions(user string, n int) ([]Contribution, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	events, _, err := c.gh.Activity.ListEventsPerformedByUser(ctx, user, false, &github.ListOptions{PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("list user events: %w", err)
	}
	// Deduplicate by repo name, keeping the most recent occurrence.
	byRepo := make(map[string]Contribution, len(events))
	for _, ev := range events {
		if ev == nil || ev.Repo == nil {
			continue
		}
		name := ev.Repo.GetName()
		if _, seen := byRepo[name]; seen {
			continue
		}
		htmlURL := ev.Repo.GetHTMLURL()
		if htmlURL == "" && name != "" {
			htmlURL = "https://github.com/" + name
		}
		byRepo[name] = Contribution{
			Repo: Repo{
				Name:        name,
				URL:         htmlURL,
				Description: ev.Repo.GetDescription(),
			},
			OccurredAt: ev.GetCreatedAt().Time,
		}
	}
	out := make([]Contribution, 0, len(byRepo))
	for _, c := range byRepo {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].OccurredAt.After(out[j].OccurredAt)
	})
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out, nil
}

// humanize returns a short human-readable string for the time delta, like
// "just now", "5 minutes ago", "1 day ago", "2 weeks ago".
func humanize(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case d < 365*24*time.Hour:
		weeks := int(d.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	default:
		years := int(d.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// Render reads the template at templatePath, executes it with data, and
// writes the result to outputPath.
func Render(templatePath, outputPath string, data *Data) error {
	tplBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}
	// We re-create a client here so the template can call the data-fetching
	// functions; in practice Fetch returns the username and the funcs do the
	// real work.
	c := NewClient(context.Background(), os.Getenv("GITHUB_TOKEN"))
	funcs := template.FuncMap{
		"recentCreatedRepos": func(user string, n int) ([]Repo, error) {
			return c.recentCreatedRepos(user, n)
		},
		"recentContributions": func(n int) ([]Contribution, error) {
			return c.recentContributions(data.Username, n)
		},
		"humanize": humanize,
	}
	tpl, err := template.New("readme").Funcs(funcs).Parse(string(tplBytes))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	if err := os.WriteFile(outputPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
