package readme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
