package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nyxragon/bitybuckety"
)

func main() {
	// Define default values
	defaultTotal := 100
	defaultDate := ""

	// Parse CLI flags
	totalCommits := flag.Int("total", defaultTotal, "Total number of commits to fetch")
	date := flag.String("date", defaultDate, "Fetch commits after this date (format: 2024-12-20T00:00:00+00:00). If not present, default will be used.")
	flag.Parse()

	// Check if the flags are set by the user
	argsProvided := len(flag.Args()) > 0

	// Check if neither 'date' nor 'total' is provided and print a message
	if !argsProvided {
		fmt.Println("No arguments provided for date or total commits. Using default values.")
	}

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
