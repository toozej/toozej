// Command readme-gen renders templates/README.md.tpl into README.md
// using data fetched from the GitHub API.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/toozej/toozej/internal/readme"
)

func main() {
	var (
		templatePath = flag.String("template", "templates/README.md.tpl", "path to README template file")
		outputPath   = flag.String("output", "README.md", "path to write generated README")
		githubUser   = flag.String("user", "toozej", "GitHub username to fetch data for")
		token        = flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token (defaults to GITHUB_TOKEN env var)")
	)
	flag.Parse()

	if *token == "" {
		log.Println("warning: no GitHub token provided; unauthenticated requests are heavily rate-limited")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client := readme.NewClient(ctx, *token)
	data, err := client.Fetch(ctx, *githubUser)
	if err != nil {
		log.Fatalf("failed to fetch GitHub data: %v", err)
	}

	if err := readme.Render(*templatePath, *outputPath, data); err != nil {
		log.Fatalf("failed to render template: %v", err)
	}

	fmt.Printf("Wrote %s from %s\n", *outputPath, *templatePath)
}
