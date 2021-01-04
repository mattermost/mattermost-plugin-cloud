package main

import (
	"fmt"
	"net/http"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/heroku/docker-registry-client/registry"
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

// ValidTag returns if a given tag exists for the given repository.
func (dc *DockerClient) ValidTag(desiredTag, repository string) (bool, error) {
	hub, err := registry.New(dc.registryURL, dc.username, dc.password)
	if err != nil {
		return false, err
	}

	tags, err := hub.Tags(repository)
	if err != nil {
		return false, err
	}

	for _, tag := range tags {
		if tag == desiredTag {
			return true, nil
		}
	}

	return false, nil
}

// GetDigestForTag fetches the digest for the image. Sadly, this
// functionality is not present in the Heroku docker client, which
// will only get digests for v1 manifests, which contain the wrong
// digest sum
func (dc *DockerClient) GetDigestForTag(desiredTag, repository string) (string, error) {
	resource := fmt.Sprintf("%s/v2/%s/manifests/%s", dc.registryURL, repository, desiredTag)
	tt := &registry.TokenTransport{
		Transport: http.DefaultTransport,
		Username:  dc.username,
		Password:  dc.password,
	}

	req, err := http.NewRequest("HEAD", resource, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Accept", schema2.MediaTypeManifest)
	resp, err := tt.RoundTrip(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to HEAD manifest registry endpoint")
	}

	digestHeader, ok := resp.Header["Docker-Content-Digest"]
	if !ok {
		return "", errors.New("image digest header was missing")
	} else if len(digestHeader) < 1 {
		return "", errors.New("image digest header was empty")
	}

	return digestHeader[0], nil
}
