package bitybuckety

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"io"
	"time"
	"github.com/0x4f53/textsubs"
)

// Commit structure remains the same
type Commit struct {
	Hash            string   `json:"hash"`
	AuthorName      string   `json:"author_name"`
	Date            string   `json:"date"`
	Message         string   `json:"message"`
	PatchLink       string   `json:"patch_link"`
	CommitURL       string   `json:"commit_url"`
	RepositoryLink  string   `json:"repository_link"`
	ProjectKey      string   `json:"project_key"`
	ProjectName     string   `json:"project_name"`
	ProjectURL      string   `json:"project_url"`
	Subdomains      []string `json:"subdomains"` // Updated to be a slice of strings
}


// fetchRepositories fetches repositories from Bitbucket.
func fetchRepositories(pagelen int, after string) ([]map[string]string, error) {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories?pagelen=%d&after=%s", pagelen, after)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching from Bitbucket API: %w", err)
	}
	defer resp.Body.Close()

	var bitbucketResponse struct {
		Values []struct {
			Name      string `json:"name"`
			FullName  string `json:"full_name"`
			UpdatedOn string `json:"updated_on"`
			Project   struct {
				Key   string `json:"key"`
				Name  string `json:"name"`
				Links struct {
					HTML struct {
						Href string `json:"href"`
					} `json:"html"`
				} `json:"links"`
			} `json:"project"`
		} `json:"values"`
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

func fetchCommits(repositoryFullName string, pagelen int, projectKey, projectName, projectURL string) ([]Commit, error) {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/commits?pagelen=%d", repositoryFullName, pagelen)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching commits: %w", err)
	}
	defer resp.Body.Close()

	var commitResponse struct {
		Values []struct {
			Hash    string `json:"hash"`
			Date    string `json:"date"`
			Author  struct {
				User struct {
					DisplayName string `json:"display_name"`
				} `json:"user"`
			} `json:"author"`
			Message string `json:"message"`
			Links   struct {
				Self  struct{ Href string } `json:"self"`
				Patch struct{ Href string } `json:"patch"`
			} `json:"links"`
		} `json:"values"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&commitResponse); err != nil {
		return nil, fmt.Errorf("error decoding commit response: %w", err)
	}

	var commits []Commit
	for _, commit := range commitResponse.Values {
		// Validate the PatchLink
		patchResp, err := http.Get(commit.Links.Patch.Href)
		if err != nil {
			fmt.Printf("Error fetching patch link for commit %s: %v\n", commit.Hash, err)
			continue
		}
		defer patchResp.Body.Close()

		if patchResp.StatusCode != http.StatusOK {
			fmt.Printf("Invalid patch link for commit %s: %d\n", commit.Hash, patchResp.StatusCode)
			continue
		}

		// Read the patch content
		patchContent, err := io.ReadAll(patchResp.Body)
		if err != nil {
			fmt.Printf("Error reading patch content for commit %s: %v\n", commit.Hash, err)
			continue
		}

		// Process the patch content with textsubs
		subdomains, err := textsubs.SubdomainsOnly(string(patchContent), true)
		if err != nil {
			fmt.Printf("Error processing patch content for commit %s: %v\n", commit.Hash, err)
			continue
		}

		// Create Commit instance and append to slice
		commits = append(commits, Commit{
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
			Subdomains:     subdomains, // Store the subdomains in the commit struct
		})
	}
	return commits, nil
}

// writeCommitToFile writes a commit to a file.
func writeCommitToFile(commit Commit, filename string) error {
	fmt.Println("Preparing to write commit:", commit)

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
        return fmt.Errorf("error opening file: %w", err)
    }
	defer file.Close()

    encoder := json.NewEncoder(file)
    if err := encoder.Encode(commit); err != nil {
        return fmt.Errorf("error writing to file: %w", err)
    }
    
    return nil
}

// FetchCommitsAndWriteFile handles everything and writes commits to a file.
func FetchCommitsAndWriteFile(totalCommits int, date string) (string, error) {
	pagelen := 100

	if totalCommits == 0{
		totalCommits=100
	}
	if date == "" {
        threeMonthsAgo := time.Now().AddDate(0, -3, 0).UTC().Format("2006-01-02T15:04:05.000000+00:00")
        date = threeMonthsAgo
    }

	filename := fmt.Sprintf("commits_%s.json", time.Now().Format("2006-01-02_15-04-05"))

	repos, err := fetchRepositories(pagelen, date)
	if err != nil {
        return "", fmt.Errorf("error fetching repositories: %w", err)
    }

	totalFetched := 0
	for _, repo := range repos {
        if totalFetched >= totalCommits {
            break
        }

        commits, err := fetchCommits(repo["full_name"], pagelen, repo["project_key"], repo["project_name"], repo["project_url"])
        if err != nil {
            return "", fmt.Errorf("error fetching commits for repository %s: %w", repo["full_name"], err)
        }

        for _, commit := range commits {
            if totalFetched >= totalCommits { 
                break 
            }
            if writeErr := writeCommitToFile(commit, filename); writeErr != nil { 
                return "", writeErr 
            }
            totalFetched++
        }
    }

	fmt.Println("Total fetched:", totalFetched)

	return filename, nil
}
