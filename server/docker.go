package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/pkg/errors"
)

const dockerHubURL = "https://registry.hub.docker.com"

// DockerClient is a client for interacting with docker registries.
type DockerClient struct {
	registryURL string
	username    string
	password    string
}

// NewDockerClient returns a new docker client.
func NewDockerClient() *DockerClient {
	return &DockerClient{
		registryURL: dockerHubURL,
		username:    "",
		password:    "",
	}
}

// tagsResponse represents the JSON response from the Docker registry tags endpoint
type tagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// ValidTag returns if a given tag exists for the given repository.
// This implementation uses direct HTTP calls to avoid pagination issues in the heroku library.
func (dc *DockerClient) ValidTag(desiredTag, repository string) (bool, error) {
	if dc == nil {
		return false, errors.New("docker client is not initialized")
	}
	if dc.registryURL == "" {
		return false, errors.New("docker registry URL is not configured")
	}

	// Use direct HTTP API call to avoid pagination issues with heroku/docker-registry-client
	// We'll check batches of tags until we find the desired one or exhaust all tags
	pageSize := 1000
	lastTag := ""

	client := &http.Client{}

	for {
		// Build the URL for the tags list
		tagsURL := fmt.Sprintf("%s/v2/%s/tags/list?n=%d", dc.registryURL, repository, pageSize)
		if lastTag != "" {
			tagsURL = fmt.Sprintf("%s&last=%s", tagsURL, url.QueryEscape(lastTag))
		}

		req, err := http.NewRequest("GET", tagsURL, nil)
		if err != nil {
			return false, errors.Wrap(err, "failed to create request")
		}

		// Add authentication if provided
		if dc.username != "" && dc.password != "" {
			req.SetBasicAuth(dc.username, dc.password)
		}

		resp, err := client.Do(req)
		if err != nil {
			return false, errors.Wrap(err, "failed to fetch tags")
		}

		if resp.StatusCode == http.StatusUnauthorized {
			// Try to get an auth token for public repositories
			resp.Body.Close()
			var token string
			token, err = dc.getAuthToken(repository)
			if err != nil {
				return false, errors.Wrap(err, "failed to get auth token")
			}
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err = client.Do(req)
			if err != nil {
				return false, errors.Wrap(err, "failed to fetch tags with token")
			}
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return false, errors.Errorf("unexpected status code %d from registry", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return false, errors.Wrap(err, "failed to read response body")
		}

		var tagsResp tagsResponse
		if err := json.Unmarshal(body, &tagsResp); err != nil {
			return false, errors.Wrap(err, "failed to unmarshal tags response")
		}

		// Check if desired tag is in this batch
		for _, tag := range tagsResp.Tags {
			if tag == desiredTag {
				return true, nil
			}
		}

		// If we got fewer tags than requested, we've reached the end
		if len(tagsResp.Tags) < pageSize {
			break
		}

		// Update lastTag for next iteration
		lastTag = tagsResp.Tags[len(tagsResp.Tags)-1]
	}

	return false, nil
}

// getAuthToken retrieves an authentication token for accessing Docker Hub
func (dc *DockerClient) getAuthToken(repository string) (string, error) {
	authURL := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", repository)

	resp, err := http.Get(authURL)
	if err != nil {
		return "", errors.Wrap(err, "failed to get auth token")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("unexpected status code %d from auth endpoint", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read auth response")
	}

	var authResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &authResp); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal auth response")
	}

	return authResp.Token, nil
}

// GetDigestForTag fetches the digest for the image.
func (dc *DockerClient) GetDigestForTag(desiredTag, repository string) (string, error) {
	if dc == nil {
		return "", errors.New("docker client is not initialized")
	}
	if dc.registryURL == "" {
		return "", errors.New("docker registry URL is not configured")
	}

	resource := fmt.Sprintf("%s/v2/%s/manifests/%s", dc.registryURL, repository, desiredTag)

	req, err := http.NewRequest("HEAD", resource, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Accept", schema2.MediaTypeManifest)

	// Add authentication
	if dc.username != "" && dc.password != "" {
		req.SetBasicAuth(dc.username, dc.password)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to HEAD manifest registry endpoint")
	}
	defer resp.Body.Close()

	// If unauthorized, try with token authentication
	if resp.StatusCode == http.StatusUnauthorized {
		token, err := dc.getAuthToken(repository)
		if err != nil {
			return "", errors.Wrap(err, "failed to get auth token")
		}

		req, err = http.NewRequest("HEAD", resource, nil)
		if err != nil {
			return "", errors.Wrap(err, "failed to create request")
		}
		req.Header.Set("Accept", schema2.MediaTypeManifest)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err = client.Do(req)
		if err != nil {
			return "", errors.Wrap(err, "failed to HEAD manifest registry endpoint with token")
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("unexpected status code %d from registry", resp.StatusCode)
	}

	digestHeader, ok := resp.Header["Docker-Content-Digest"]
	if !ok {
		return "", errors.New("image digest header was missing")
	} else if len(digestHeader) < 1 {
		return "", errors.New("image digest header was empty")
	}

	return digestHeader[0], nil
}
