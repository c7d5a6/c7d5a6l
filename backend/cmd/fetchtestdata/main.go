package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
)

var fixtureURLs = []string{
	"https://liquipedia.net/starcraft/ASL/20",
	"https://liquipedia.net/starcraft/AfreecaTV/StarCraft_League_Remastered/14",
	"https://liquipedia.net/starcraft/AfreecaTV/StarCraft_League_Remastered/8",
	"https://liquipedia.net/starcraft/KCM/Race_Survival/2026/1",
}

func main() {
	root := filepath.Join("testdata", "liquipedia")
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		log.Fatal(err)
	}

	client := liquipedia.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	for i, pageURL := range fixtureURLs {
		log.Printf("[%d/%d] fetching %s", i+1, len(fixtureURLs), pageURL)
		page, err := client.FetchPage(ctx, pageURL)
		if err != nil {
			log.Fatalf("fetch failed: %v", err)
		}
		path, err := liquipedia.WriteFixture(root, page)
		if err != nil {
			log.Fatalf("write fixture failed: %v", err)
		}
		log.Printf("wrote %s (%d bytes, title=%q)", path, len(page.HTML), page.Title)
	}

	fmt.Printf("done: %d fixtures in %s\n", len(fixtureURLs), root)
}
