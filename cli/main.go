package main

import (
	"flag"
	"fmt"
	"log"

	"bitbucket"
)

func main() {
	// Parse CLI flags
	totalCommits := flag.Int("total", 100, "Total number of commits to fetch")
	date := flag.String("date", "2024-01-01T00:00:00+00:00", "Fetch commits before this date")
	flag.Parse()

	// Fetch commits and write to file
	filename, err := bitbucket.FetchCommitsAndWriteFile(*totalCommits, *date)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}

	fmt.Printf("Commits successfully written to %s\n", filename)
}
