package main

type MockedDockerClient struct {
	tagExists      bool
	digest         string
	validTagCalls  []dockerClientCall
	getDigestCalls []dockerClientCall
}

type dockerClientCall struct {
	tag        string
	repository string
}

func (mc *MockedDockerClient) ValidTag(desiredTag, repository string) (bool, error) {
	mc.validTagCalls = append(mc.validTagCalls, dockerClientCall{tag: desiredTag, repository: repository})
	return mc.tagExists, nil
}

func (mc *MockedDockerClient) GetDigestForTag(desiredTag, repository string) (string, error) {
	mc.getDigestCalls = append(mc.getDigestCalls, dockerClientCall{tag: desiredTag, repository: repository})
	if mc.digest != "" {
		return mc.digest, nil
	}
	return desiredTag, nil
}
