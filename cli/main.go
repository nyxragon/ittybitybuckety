package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nyxragon/bitybuckety"
)

func main() {
	// Parse CLI flags
	totalCommits := flag.Int("total", 100, "Total number of commits to fetch")
	date := flag.String("date", "", "Fetch commits after this date (format: 2024-12-20T00:00:00+00:00). If not present, default will be used.")
	flag.Parse()

	// Validate 'total' flag
	if *totalCommits <= 0 {
		fmt.Println("Error: 'total' must be greater than 0.")
		os.Exit(1)
	}

	// Fetch commits and write to file
	filename, err := bitybuckety.FetchCommitsAndWriteFile(*totalCommits, *date)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}

	// Output the success message with the file where commits are stored
	fmt.Printf("Commits successfully written to %s\n", filename)
}
