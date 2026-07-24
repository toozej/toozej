package readme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v89/github"
)

func TestHumanize(t *testing.T) {
	now := time.Now()
	cases := []struct {
		in   time.Time
		want string
	}{
		{now.Add(-30 * time.Second), "just now"},
		{now.Add(-2 * time.Minute), "2 minutes ago"},
		{now.Add(-1 * time.Minute), "1 minute ago"},
		{now.Add(-3 * time.Hour), "3 hours ago"},
		{now.Add(-24 * time.Hour), "1 day ago"},
		{now.Add(-48 * time.Hour), "2 days ago"},
		{now.Add(-14 * 24 * time.Hour), "14 days ago"},
		{now.Add(-60 * 24 * time.Hour), "8 weeks ago"},
		{now.Add(-400 * 24 * time.Hour), "1 year ago"},
	}
	for _, c := range cases {
		got := humanize(c.in)
		if got != c.want {
			t.Errorf("humanize(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHumanizeZero(t *testing.T) {
	if got := humanize(time.Time{}); got != "" {
		t.Errorf("humanize(zero) = %q, want empty", got)
	}
}

func TestRender(t *testing.T) {
	dir := t.TempDir()
	tplPath := filepath.Join(dir, "tpl")
	outPath := filepath.Join(dir, "out")
	if err := os.WriteFile(tplPath, []byte("Hello {{.Username}}!"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Render(tplPath, outPath, &Data{Username: "toozej"}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "Hello toozej!") {
		t.Errorf("output = %q, want it to contain %q", got, "Hello toozej!")
	}
}

// strPtr is a small helper to take the address of a string literal.
func strPtr(s string) *string { return &s }

func newEvent(eventType, repoName, htmlURL, description string, at time.Time) *github.Event {
	return &github.Event{
		Type: strPtr(eventType),
		Repo: &github.Repository{
			Name:        strPtr(repoName),
			HTMLURL:     &htmlURL,
			Description: &description,
		},
		CreatedAt: &github.Timestamp{Time: at},
	}
}

func TestFilterAndDedupeEvents_ExcludesWatchEvents(t *testing.T) {
	now := time.Now()
	events := []*github.Event{
		newEvent("PushEvent", "toozej/toozej", "https://github.com/toozej/toozej", "self", now.Add(-1*time.Hour)),
		newEvent("WatchEvent", "stupside/castor", "https://github.com/stupside/castor", "starred", now.Add(-30*time.Minute)),
		newEvent("PullRequestEvent", "toozej/tools", "https://github.com/toozej/tools", "tools", now.Add(-2*time.Hour)),
		newEvent("WatchEvent", "perplexityai/bumblebee", "https://github.com/perplexityai/bumblebee", "starred", now.Add(-10*time.Minute)),
	}

	got := filterAndDedupeEvents(events)

	if len(got) != 2 {
		t.Fatalf("expected 2 contributions after filtering WatchEvents, got %d: %+v", len(got), names(got))
	}
	for _, c := range got {
		if c.Repo.Name == "stupside/castor" || c.Repo.Name == "perplexityai/bumblebee" {
			t.Errorf("starred repo %q should be excluded, got %+v", c.Repo.Name, c)
		}
	}
}

func TestFilterAndDedupeEvents_DedupesByRepo(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-2 * time.Hour)
	events := []*github.Event{
		newEvent("PushEvent", "toozej/toozej", "https://github.com/toozej/toozej", "self", now),
		newEvent("PullRequestEvent", "toozej/toozej", "https://github.com/toozej/toozej", "self", earlier),
		newEvent("PushEvent", "toozej/tools", "https://github.com/toozej/tools", "tools", now.Add(-1*time.Hour)),
	}

	got := filterAndDedupeEvents(events)

	if len(got) != 2 {
		t.Fatalf("expected 2 unique repos, got %d: %+v", len(got), names(got))
	}
	// First entry should be the most recent push on toozej/toozej (now).
	if got[0].Repo.Name != "toozej/toozej" || !got[0].OccurredAt.Equal(now) {
		t.Errorf("first entry = %+v, want toozej/toozej at %v", got[0], now)
	}
}

func TestFilterAndDedupeEvents_SortedDescendingByOccurredAt(t *testing.T) {
	now := time.Now()
	events := []*github.Event{
		newEvent("PushEvent", "toozej/a", "https://github.com/toozej/a", "a", now.Add(-3*time.Hour)),
		newEvent("PushEvent", "toozej/b", "https://github.com/toozej/b", "b", now.Add(-1*time.Hour)),
		newEvent("PushEvent", "toozej/c", "https://github.com/toozej/c", "c", now.Add(-2*time.Hour)),
	}

	got := filterAndDedupeEvents(events)

	if len(got) != 3 {
		t.Fatalf("expected 3 contributions, got %d", len(got))
	}
	want := []string{"toozej/b", "toozej/c", "toozej/a"}
	for i, w := range want {
		if got[i].Repo.Name != w {
			t.Errorf("got[%d].Repo.Name = %q, want %q", i, got[i].Repo.Name, w)
		}
		if i > 0 && got[i].OccurredAt.After(got[i-1].OccurredAt) {
			t.Errorf("result not sorted descending at index %d", i)
		}
	}
}

func TestFilterAndDedupeEvents_SkipsNilAndMissingRepo(t *testing.T) {
	now := time.Now()
	events := []*github.Event{
		nil,
		{Type: strPtr("PushEvent")}, // no Repo
		newEvent("PushEvent", "toozej/toozej", "https://github.com/toozej/toozej", "self", now),
	}

	got := filterAndDedupeEvents(events)

	if len(got) != 1 {
		t.Fatalf("expected 1 contribution, got %d: %+v", len(got), names(got))
	}
	if got[0].Repo.Name != "toozej/toozej" {
		t.Errorf("got %q, want toozej/toozej", got[0].Repo.Name)
	}
}

func TestFilterAndDedupeEvents_Empty(t *testing.T) {
	if got := filterAndDedupeEvents(nil); len(got) != 0 {
		t.Errorf("expected empty result for nil input, got %+v", got)
	}
	if got := filterAndDedupeEvents([]*github.Event{}); len(got) != 0 {
		t.Errorf("expected empty result for empty input, got %+v", got)
	}
}

func names(cs []Contribution) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Repo.Name
	}
	return out
}
