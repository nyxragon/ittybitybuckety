package bitybuckety

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// Commit structure remains the same
type Commit struct {
	Hash            string json:"hash"
	AuthorName      string json:"author_name"
	Date            string json:"date"
	Message         string json:"message"
	PatchLink       string json:"patch_link"
	CommitURL       string json:"commit_url"
	RepositoryLink  string json:"repository_link"
	ProjectKey      string json:"project_key"
	ProjectName     string json:"project_name"
	ProjectURL      string json:"project_url"
}

// fetchRepositories fetches repositories from Bitbucket.
func fetchRepositories(pagelen int, before string) ([]map[string]string, error) {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories?pagelen=%d&before=%s", pagelen, before)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching from Bitbucket API: %w", err)
	}
	defer resp.Body.Close()

	var bitbucketResponse struct {
		Values []struct {
			Name      string json:"name"
			FullName  string json:"full_name"
			UpdatedOn string json:"updated_on"
			Project   struct {
				Key   string json:"key"
				Name  string json:"name"
				Links struct {
					HTML struct {
						Href string json:"href"
					} json:"html"
				} json:"links"
			} json:"project"
		} json:"values"
	}

	if err := json.NewDecoder(resp.Body).Decode(&bitbucketResponse); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	var repos []map[string]string
	for _, repo := range bitbucketResponse.Values {
		repoInfo := map[string]string{
			"name":           repo.Name,
			"full_name":      repo.FullName,
			"updated_on":     repo.UpdatedOn,
			"repository_link": fmt.Sprintf("https://bitbucket.org/%s", repo.FullName),
			"project_key":    repo.Project.Key,
			"project_name":   repo.Project.Name,
			"project_url":    repo.Project.Links.HTML.Href,
		}
		repos = append(repos, repoInfo)
	}
	return repos, nil
}

// fetchCommits fetches commits from a specific repository.
func fetchCommits(repositoryFullName string, pagelen int, projectKey, projectName, projectURL string, commitCh chan<- Commit, wg *sync.WaitGroup) {
	defer wg.Done()

	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/commits?pagelen=%d", repositoryFullName, pagelen)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error fetching commits: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var commitResponse struct {
		Values []struct {
			Hash    string json:"hash"
			Date    string json:"date"
			Author  struct {
				User struct {
					DisplayName string json:"display_name"
				} json:"user"
			} json:"author"
			Message string json:"message"
			Links   struct {
				Self  struct{ Href string } json:"self"
				Patch struct{ Href string } json:"patch"
			} json:"links"
		} json:"values"
	}
	if err := json.NewDecoder(resp.Body).Decode(&commitResponse); err != nil {
		fmt.Printf("Error decoding commit response: %v\n", err)
		return
	}

	for _, commit := range commitResponse.Values {
		commitCh <- Commit{
			Hash:           commit.Hash,
			AuthorName:     commit.Author.User.DisplayName,
			Date:           commit.Date,
			Message:        commit.Message,
			PatchLink:      commit.Links.Patch.Href,
			CommitURL:      commit.Links.Self.Href,
			RepositoryLink: fmt.Sprintf("https://bitbucket.org/%s", repositoryFullName),
			ProjectKey:     projectKey,
			ProjectName:    projectName,
			ProjectURL:     projectURL,
		}
	}
}

// writeCommitToFile writes a commit to a file.
func writeCommitToFile(commit Commit, filename string, mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(commit); err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
	}
}

// FetchCommitsAndWriteFile handles everything and writes commits to a file.
func FetchCommitsAndWriteFile(totalCommits int, date string) (string, error) {
	pagelen := 10 // Number of items to fetch per request
	var wg sync.WaitGroup
	commitCh := make(chan Commit, pagelen)
	mu := &sync.Mutex{}

	// Generate the output filename based on the current timestamp
	filename := fmt.Sprintf("commits_%s.json", time.Now().Format("2006-01-02"))

	// Fetch repositories
	repos, err := fetchRepositories(pagelen, date)
	if err != nil {
		return "", fmt.Errorf("error fetching repositories: %w", err)
	}

	// Start fetching commits
	go func() {
		for commit := range commitCh {
			writeCommitToFile(commit, filename, mu)
		}
	}()

	totalFetched := 0
	for _, repo := range repos {
		wg.Add(1)
		go fetchCommits(repo["full_name"], pagelen, repo["project_key"], repo["project_name"], repo["project_url"], commitCh, &wg)

		totalFetched += pagelen
		if totalFetched >= totalCommits {
			break
		}
	}

	wg.Wait()
	close(commitCh)

	return filename, nil
}
