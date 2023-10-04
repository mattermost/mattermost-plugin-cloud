package main

import (
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockClient struct {
	mockedCloudClustersDTO          []*cloud.ClusterDTO
	mockedCloudInstallationsDTO     []*cloud.InstallationDTO
	mockedCloudClusterInstallations []*cloud.ClusterInstallation

	overrideGetInstallationDTO *cloud.InstallationDTO
	returnNilDNSInstalation    bool
	returnDNSErrorOverride     error

	// Stores latest CreateInstallationRequest passed to mock
	creationRequest *cloud.CreateInstallationRequest
	// Stores latest PatchInstallationRequest passed to mock
	patchRequest *cloud.PatchInstallationRequest

	err error
}

func (mc *MockClient) ExecClusterInstallationCLI(clusterInstallationID, command string, subcommand []string) ([]byte, error) {
	return []byte{}, nil
}

func (mc *MockClient) GetClusters(request *cloud.GetClustersRequest) ([]*cloud.ClusterDTO, error) {
	return mc.mockedCloudClustersDTO, mc.err
}

func (mc *MockClient) CreateInstallation(request *cloud.CreateInstallationRequest) (*cloud.InstallationDTO, error) {
	mc.creationRequest = request
	return &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid"}}, nil
}

func (mc *MockClient) GetInstallation(installataionID string, request *cloud.GetInstallationRequest) (*cloud.InstallationDTO, error) {
	if mc.overrideGetInstallationDTO != nil {
		return mc.overrideGetInstallationDTO, nil
	}

	return &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid", OwnerID: "joramid", State: cloud.InstallationStateStable}}, nil
}

func (mc *MockClient) GetInstallationByDNS(DNS string, request *cloud.GetInstallationRequest) (*cloud.InstallationDTO, error) {
	if mc.returnNilDNSInstalation {
		return nil, nil
	}
	if mc.returnDNSErrorOverride != nil {
		return nil, mc.returnDNSErrorOverride
	}
	if mc.overrideGetInstallationDTO != nil {
		return mc.overrideGetInstallationDTO, nil
	}

	return &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid"}, DNS: DNS}, nil
}

func (mc *MockClient) GetInstallations(request *cloud.GetInstallationsRequest) ([]*cloud.InstallationDTO, error) {
	return mc.mockedCloudInstallationsDTO, mc.err
}

func (mc *MockClient) UpdateInstallation(installationID string, request *cloud.PatchInstallationRequest) (*cloud.InstallationDTO, error) {
	mc.patchRequest = request
	return &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid", OwnerID: "joramid"}}, nil
}

func (mc *MockClient) HibernateInstallation(installationID string) (*cloud.InstallationDTO, error) {
	return &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid", OwnerID: "joramid"}}, nil
}

func (mc *MockClient) WakeupInstallation(installationID string, request *cloud.PatchInstallationRequest) (*cloud.InstallationDTO, error) {
	return &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid", OwnerID: "joramid"}}, nil
}

func (mc *MockClient) LockDeletionLockForInstallation(installationID string) error {
	return nil
}

func (mc *MockClient) UnlockDeletionLockForInstallation(installationID string) error {
	return nil
}

func (mc *MockClient) DeleteInstallation(installationID string) error {
	return nil
}

func (mc *MockClient) GetClusterInstallations(request *cloud.GetClusterInstallationsRequest) ([]*cloud.ClusterInstallation, error) {
	return mc.mockedCloudClusterInstallations, nil
}

func (mc *MockClient) RunMattermostCLICommandOnClusterInstallation(clusterInstallationID string, subcommand []string) ([]byte, error) {
	return []byte("mocked command output"), nil
}

func (mc *MockClient) RunMmctlCommandOnClusterInstallation(clusterInstallationID string, subcommand []string) ([]byte, error) {
	return []byte("mocked mmctl command output"), nil
}

func (mc *MockClient) GetGroup(groupID string) (*cloud.GroupDTO, error) {
	return &cloud.GroupDTO{Group: &cloud.Group{ID: groupID, Name: "test-group"}}, nil
}

func TestInfoCommand(t *testing.T) {
	mockedCloudClient := &MockClient{}
	plugin := Plugin{cloudClient: mockedCloudClient}

	t.Run("success", func(t *testing.T) {
		resp, isUserError, err := plugin.runInfoCommand([]string{""}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, manifest.Version)
	})
}

func TestUpgradeHelperCommand(t *testing.T) {
	mockedCloudClient := &MockClient{}
	plugin := Plugin{cloudClient: mockedCloudClient}

	t.Run("success", func(t *testing.T) {
		resp, isUserError, err := plugin.runUpgradeHelperCommand([]string{""}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "`/cloud upgrade` has been deprecated. Use `/cloud update` instead.")
	})
}

func TestValidInstallationName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"abc", true},
		{"abc123", true},
		{"abcABC123", true},
		{"123", true},
		{"A1", true},
		{"A1-", true},
		{"A1-abc", true},
		{"realllllllllllllllllylongname123123123123123", true},
		{"bad.", false},
		{"bad\\", false},
		{"bad/", false},
		{"bad,", false},
		{"bad:", false},
		{"bad;", false},
		{"bad_", false},
		{"123.,", false},
		{".", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.valid, validInstallationName(test.name))
		})
	}
}
