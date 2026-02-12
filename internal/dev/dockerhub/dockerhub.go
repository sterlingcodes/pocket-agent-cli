package dockerhub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/pkg/output"
)

var baseURL = "https://hub.docker.com/v2"

var client = &http.Client{Timeout: 10 * time.Second}

// Image is LLM-friendly image info
type Image struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Description string `json:"desc,omitempty"`
	Stars       int    `json:"stars"`
	Pulls       int64  `json:"pulls"`
	Official    bool   `json:"official,omitempty"`
	Automated   bool   `json:"automated,omitempty"`
	Updated     string `json:"updated,omitempty"`
}

// SearchResult is LLM-friendly search result
type SearchResult struct {
	Name        string `json:"name"`
	Description string `json:"desc,omitempty"`
	Stars       int    `json:"stars"`
	Official    bool   `json:"official,omitempty"`
	Automated   bool   `json:"automated,omitempty"`
}

// Tag is LLM-friendly tag info
type Tag struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	Digest     string `json:"digest,omitempty"`
	Updated    string `json:"updated,omitempty"`
	Compressed int64  `json:"compressed,omitempty"`
}

// Manifest is LLM-friendly manifest/inspect info
type Manifest struct {
	Name         string   `json:"name"`
	Tag          string   `json:"tag"`
	Digest       string   `json:"digest"`
	Architecture string   `json:"arch,omitempty"`
	OS           string   `json:"os,omitempty"`
	Size         int64    `json:"size"`
	Layers       int      `json:"layers"`
	Created      string   `json:"created,omitempty"`
	Platforms    []string `json:"platforms,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dockerhub",
		Aliases: []string{"docker", "dh"},
		Short:   "Docker Hub registry commands",
	}

	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newImageCmd())
	cmd.AddCommand(newTagsCmd())
	cmd.AddCommand(newInspectCmd())

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search Docker images",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.QueryEscape(args[0])
			reqURL := fmt.Sprintf("%s/search/repositories/?query=%s&page_size=%d", baseURL, query, limit)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, http.NoBody)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}
			resp, err := client.Do(req)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}
			defer resp.Body.Close()

			var data struct {
				Results []struct {
					RepoName    string `json:"repo_name"`
					ShortDesc   string `json:"short_description"`
					StarCount   int    `json:"star_count"`
					IsOfficial  bool   `json:"is_official"`
					IsAutomated bool   `json:"is_automated"`
				} `json:"results"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			results := make([]SearchResult, 0, len(data.Results))
			for _, r := range data.Results {
				results = append(results, SearchResult{
					Name:        r.RepoName,
					Description: truncate(r.ShortDesc, 100),
					Stars:       r.StarCount,
					Official:    r.IsOfficial,
					Automated:   r.IsAutomated,
				})
			}

			return output.Print(results)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of results")

	return cmd
}

func newImageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image [name]",
		Short: "Get image details",
		Long:  "Get image details (e.g., library/nginx or username/image)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := normalizeImageName(args[0])
			parts := strings.SplitN(name, "/", 2)
			if len(parts) != 2 {
				return output.PrintError("invalid_name", "Image name must be in format namespace/image", nil)
			}

			namespace, repo := parts[0], parts[1]
			reqURL := fmt.Sprintf("%s/repositories/%s/%s/", baseURL, namespace, repo)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, http.NoBody)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}
			resp, err := client.Do(req)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}
			defer resp.Body.Close()

			if resp.StatusCode == 404 {
				return output.PrintError("not_found", "Image not found: "+name, nil)
			}

			var data struct {
				Name           string   `json:"name"`
				Namespace      string   `json:"namespace"`
				Description    string   `json:"description"`
				StarCount      int      `json:"star_count"`
				PullCount      int64    `json:"pull_count"`
				IsPrivate      bool     `json:"is_private"`
				IsAutomated    bool     `json:"is_automated"`
				LastUpdated    string   `json:"last_updated"`
				RepositoryType string   `json:"repository_type"`
				ContentTypes   []string `json:"content_types"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			img := Image{
				Name:        data.Name,
				Namespace:   data.Namespace,
				Description: truncate(data.Description, 200),
				Stars:       data.StarCount,
				Pulls:       data.PullCount,
				Official:    data.Namespace == "library",
				Automated:   data.IsAutomated,
				Updated:     parseTimeAgo(data.LastUpdated),
			}

			return output.Print(img)
		},
	}

	return cmd
}

func newTagsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "tags [name]",
		Short: "List tags for an image",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := normalizeImageName(args[0])
			parts := strings.SplitN(name, "/", 2)
			if len(parts) != 2 {
				return output.PrintError("invalid_name", "Image name must be in format namespace/image", nil)
			}

			namespace, repo := parts[0], parts[1]
			reqURL := fmt.Sprintf("%s/repositories/%s/%s/tags/?page_size=%d&ordering=last_updated", baseURL, namespace, repo, limit)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, http.NoBody)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}
			resp, err := client.Do(req)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}
			defer resp.Body.Close()

			if resp.StatusCode == 404 {
				return output.PrintError("not_found", "Image not found: "+name, nil)
			}

			var data struct {
				Results []struct {
					Name        string `json:"name"`
					FullSize    int64  `json:"full_size"`
					LastUpdated string `json:"last_updated"`
					Digest      string `json:"digest"`
					Images      []struct {
						Size   int64  `json:"size"`
						Digest string `json:"digest"`
					} `json:"images"`
				} `json:"results"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			tags := make([]Tag, 0, len(data.Results))
			for _, t := range data.Results {
				tag := Tag{
					Name:       t.Name,
					Size:       t.FullSize,
					Updated:    parseTimeAgo(t.LastUpdated),
					Compressed: t.FullSize,
				}
				if t.Digest != "" {
					tag.Digest = truncateDigest(t.Digest)
				} else if len(t.Images) > 0 {
					tag.Digest = truncateDigest(t.Images[0].Digest)
				}
				tags = append(tags, tag)
			}

			return output.Print(tags)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of tags")

	return cmd
}

func newInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect [name:tag]",
		Short: "Get detailed image manifest info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameTag := args[0]
			name, tag := parseNameTag(nameTag)
			name = normalizeImageName(name)

			parts := strings.SplitN(name, "/", 2)
			if len(parts) != 2 {
				return output.PrintError("invalid_name", "Image name must be in format namespace/image[:tag]", nil)
			}

			namespace, repo := parts[0], parts[1]
			reqURL := fmt.Sprintf("%s/repositories/%s/%s/tags/%s", baseURL, namespace, repo, tag)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, http.NoBody)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}
			resp, err := client.Do(req)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}
			defer resp.Body.Close()

			if resp.StatusCode == 404 {
				return output.PrintError("not_found", fmt.Sprintf("Image not found: %s:%s", name, tag), nil)
			}

			var data struct {
				Name        string `json:"name"`
				FullSize    int64  `json:"full_size"`
				LastUpdated string `json:"last_updated"`
				Digest      string `json:"digest"`
				Images      []struct {
					Architecture string `json:"architecture"`
					OS           string `json:"os"`
					Size         int64  `json:"size"`
					Digest       string `json:"digest"`
					LastPushed   string `json:"last_pushed"`
				} `json:"images"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			manifest := Manifest{
				Name:    name,
				Tag:     data.Name,
				Size:    data.FullSize,
				Created: parseTimeAgo(data.LastUpdated),
				Layers:  len(data.Images),
			}

			if data.Digest != "" {
				manifest.Digest = truncateDigest(data.Digest)
			}

			// Collect platforms
			platforms := make([]string, 0, len(data.Images))
			for _, img := range data.Images {
				platform := fmt.Sprintf("%s/%s", img.OS, img.Architecture)
				platforms = append(platforms, platform)
				// Use first image for primary arch/os
				if manifest.Architecture == "" {
					manifest.Architecture = img.Architecture
					manifest.OS = img.OS
					if img.Digest != "" {
						manifest.Digest = truncateDigest(img.Digest)
					}
				}
			}
			manifest.Platforms = platforms

			return output.Print(manifest)
		},
	}

	return cmd
}

func normalizeImageName(name string) string {
	// If no namespace, assume library (official images)
	if !strings.Contains(name, "/") {
		return "library/" + name
	}
	return name
}

func parseNameTag(nameTag string) (name, tag string) {
	if idx := strings.LastIndex(nameTag, ":"); idx != -1 {
		return nameTag[:idx], nameTag[idx+1:]
	}
	return nameTag, "latest"
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func truncateDigest(digest string) string {
	// Return short form like sha256:abc123
	if len(digest) > 19 {
		return digest[:19] + "..."
	}
	return digest
}

func parseTimeAgo(ts string) string {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		// Try without nano
		t, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return ""
		}
	}
	return timeAgo(t)
}

func timeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh", int(diff.Hours()))
	default:
		return fmt.Sprintf("%dd", int(diff.Hours()/24))
	}
}
