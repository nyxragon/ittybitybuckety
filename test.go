package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// Define the structure for the commit
type Commit struct {
	Hash            string `json:"hash"`
	AuthorName      string `json:"author_name"`
	Date            string `json:"date"`
	Message         string `json:"message"`
	PatchLink       string `json:"patch_link"`
	CommitURL       string `json:"commit_url"`
	RepositoryLink  string `json:"repository_link"`
	ProjectKey      string `json:"project_key"`
	ProjectName     string `json:"project_name"`
	ProjectURL      string `json:"project_url"`
}

// Define the structure for the Bitbucket response for repositories
type BitbucketResponse struct {
	Values []struct {
		Name        string `json:"name"`
		FullName    string `json:"full_name"`
		UpdatedOn   string `json:"updated_on"`
		Project     struct {
			Key  string `json:"key"`
			Name string `json:"name"`
			Links struct {
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
			} `json:"links"`
		} `json:"project"`
	} `json:"values"`
}

// Function to get repositories and their details
func getRepos(pagelen int, before string) ([]map[string]string, error) {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories?pagelen=%d&before=%s", pagelen, before)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching from Bitbucket API: %w", err)
	}
	defer resp.Body.Close()

	var bitbucketResponse BitbucketResponse
	if err := json.NewDecoder(resp.Body).Decode(&bitbucketResponse); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	var repos []map[string]string
	for _, repo := range bitbucketResponse.Values {
		// Include project details here
		repoInfo := map[string]string{
			"name":           repo.Name,
			"full_name":      repo.FullName,
			"updated_on":     repo.UpdatedOn,
			"repository_link": fmt.Sprintf("https://bitbucket.org/%s", repo.FullName),
			"project_key":    repo.Project.Key,  // Project Key
			"project_name":   repo.Project.Name, // Project Name
			"project_url":    repo.Project.Links.HTML.Href, // Project URL
		}
		repos = append(repos, repoInfo)
	}

	return repos, nil
}

// Function to fetch commits from a specific repository
func getCommits(repositoryFullName string, pagelen int, projectKey string, projectName string, projectURL string, commitCh chan<- Commit, wg *sync.WaitGroup) {
	defer wg.Done()

	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/commits?pagelen=%d", repositoryFullName, pagelen)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error fetching commits from Bitbucket API: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var commitResponse struct {
		Values []struct {
			Hash    string `json:"hash"`
			Date    string `json:"date"`
			Author  struct {
				Raw    string `json:"raw"`
				User   struct {
					DisplayName string `json:"display_name"`
				} `json:"user"`
			} `json:"author"`
			Message string `json:"message"`
			Links   struct {
				Self struct {
					Href string `json:"href"`
				} `json:"self"`
				Patch struct {
					Href string `json:"href"`
				} `json:"patch"`
			} `json:"links"`
			Repository struct {
				Links struct {
					Self struct {
						Href string `json:"href"`
					} `json:"self"`
				} `json:"links"`
			} `json:"repository"`
		} `json:"values"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&commitResponse); err != nil {
		fmt.Printf("Error decoding commit response: %v\n", err)
		return
	}

	// Send each commit to the channel for concurrent writing
	for _, commit := range commitResponse.Values {
		commitCh <- Commit{
			Hash:            commit.Hash,
			AuthorName:      commit.Author.User.DisplayName,
			Date:            commit.Date,
			Message:         commit.Message,
			PatchLink:       commit.Links.Patch.Href,
			CommitURL:       commit.Links.Self.Href,
			RepositoryLink:  commit.Repository.Links.Self.Href,
			ProjectKey:      projectKey,
			ProjectName:     projectName,
			ProjectURL:      projectURL, // Include project URL
		}
	}
}

// Function to write commit data concurrently to a JSON file with each commit on a new line
func writeCommitToJSON(commit Commit, filename string, mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()

	// Open the file in append mode
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening file for appending: %v\n", err)
		return
	}
	defer file.Close()

	// Write each commit to the file on a new line (without array brackets)
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "") // No indentation, compact form
	if err := encoder.Encode(commit); err != nil {
		fmt.Printf("Error writing commit to file: %v\n", err)
	}
}

// Main function to orchestrate fetching repos, commits and saving to JSON
func main() {
	// Set the initial variables
	pagelen := 10
	before := "2024-11-15T00:00:00+00:00" // Adjust based on your needs

	// Start fetching repos
	repos, err := getRepos(pagelen, before)
	if err != nil {
		fmt.Println("Error fetching repositories:", err)
		return
	}

	var wg sync.WaitGroup
	commitCh := make(chan Commit, 10) // Buffered channel for commits
	mu := &sync.Mutex{}

	// Get the current timestamp to create a filename
	timestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s.json", timestamp)

	// Loop through each repository and fetch commits
	for _, repo := range repos {
		wg.Add(1)

		go func(repo map[string]string) {
			defer wg.Done()

			// Fetch commits for the repository
			getCommits(repo["full_name"], pagelen, repo["project_key"], repo["project_name"], repo["project_url"], commitCh, &wg)
		}(repo)
	}

	// Go routine for writing commits to file concurrently
	go func() {
		for commit := range commitCh {
			writeCommitToJSON(commit, filename, mu)
		}
	}()

	// Wait for all go routines to finish
	wg.Wait()
	close(commitCh) // Close the channel once all commits are processed

	// Wait for the commit writing go routine to finish
	fmt.Printf("Commits stored to %s\n", filename)
}
