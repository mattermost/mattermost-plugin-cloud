package main

type MockedDockerClient struct {
	tagExists bool
}

func (mc *MockedDockerClient) ValidTag(desiredTag, repository string) (bool, error) {
	return mc.tagExists, nil
}
