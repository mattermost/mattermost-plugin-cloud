package main

type MockedDockerClient struct {
	tagExists bool
}

func (mc *MockedDockerClient) ValidTag(desiredTag, repository string) (bool, error) {
	return mc.tagExists, nil
}

func (mc *MockedDockerClient) GetDigestForTag(desiredTag, repository string) (string, error) {
	return desiredTag, nil
}
