package main

import (
	"github.com/heroku/docker-registry-client/registry"
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
