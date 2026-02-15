package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

// SearchResult represents a plugin found via GitHub search.
type SearchResult struct {
	Name        string
	FullName    string
	Description string
	Author      string
	Stars       int
	URL         string
	UpdatedAt   string
}

// Search queries GitHub for mine plugin repositories matching the query.
func Search(query, tag string) ([]SearchResult, error) {
	q := "mine-plugin-"
	if query != "" {
		q += query
	}
	q += " in:name"

	// Always require the mine-plugin topic; add extra tag filter if specified.
	q += " topic:mine-plugin"
	if tag != "" {
		q += " topic:" + tag
	}

	apiURL := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&sort=stars&per_page=20",
		url.QueryEscape(q))

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "mine-cli")

	// Use GITHUB_TOKEN for authenticated requests (higher rate limits)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("GitHub API rate limit exceeded — try again later or set GITHUB_TOKEN")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var result struct {
		Items []struct {
			Name        string `json:"name"`
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			Owner       struct {
				Login string `json:"login"`
			} `json:"owner"`
			StargazersCount int    `json:"stargazers_count"`
			HTMLURL         string `json:"html_url"`
			UpdatedAt       string `json:"updated_at"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing GitHub response: %w", err)
	}

	var results []SearchResult
	for _, item := range result.Items {
		results = append(results, SearchResult{
			Name:        item.Name,
			FullName:    item.FullName,
			Description: item.Description,
			Author:      item.Owner.Login,
			Stars:       item.StargazersCount,
			URL:         item.HTMLURL,
			UpdatedAt:   item.UpdatedAt,
		})
	}

	return results, nil
}
