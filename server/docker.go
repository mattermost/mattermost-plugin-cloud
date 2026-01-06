package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

// fetchManifest fetches the manifest for a given tag and returns the response and status code.
// This is a shared helper for both ValidTag and GetDigestForTag.
func (dc *DockerClient) fetchManifest(desiredTag, repository, method string) (*http.Response, error) {
	if dc == nil {
		return nil, errors.New("docker client is not initialized")
	}
	if dc.registryURL == "" {
		return nil, errors.New("docker registry URL is not configured")
	}

	resource := fmt.Sprintf("%s/v2/%s/manifests/%s", dc.registryURL, repository, desiredTag)

	req, err := http.NewRequest(method, resource, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Accept", schema2.MediaTypeManifest)

	// Add authentication if provided
	if dc.username != "" && dc.password != "" {
		req.SetBasicAuth(dc.username, dc.password)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch manifest")
	}

	// If unauthorized, try with token authentication
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		token, err := dc.getAuthToken(repository)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get auth token")
		}

		req, err = http.NewRequest(method, resource, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create request")
		}
		req.Header.Set("Accept", schema2.MediaTypeManifest)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err = client.Do(req)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch manifest with token")
		}
	}

	return resp, nil
}

// ValidTag checks if a given tag exists for the given repository by querying the manifest endpoint.
func (dc *DockerClient) ValidTag(desiredTag, repository string) (bool, error) {
	resp, err := dc.fetchManifest(desiredTag, repository, "HEAD")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Tag exists if we get a 200 OK
	return resp.StatusCode == http.StatusOK, nil
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
	resp, err := dc.fetchManifest(desiredTag, repository, "HEAD")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

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
