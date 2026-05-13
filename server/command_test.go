package main

import (
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockClient struct {
	mockedCloudClustersDTO          []*cloud.ClusterDTO
	mockedCloudInstallationsDTO     []*cloud.InstallationDTO
	mockedCloudClusterInstallations []*cloud.ClusterInstallation

	overrideGetInstallationDTO *cloud.InstallationDTO
	returnNilDNSInstalation    bool
	returnDNSErrorOverride     error
	execDebugPacket            func(clusterInstallationID string) ([]byte, error)

	// Stores latest CreateInstallationRequest passed to mock
	creationRequest *cloud.CreateInstallationRequest
	// Stores latest PatchInstallationRequest passed to mock
	patchRequest             *cloud.PatchInstallationRequest
	patchInstallationID      string
	deletedInstallationID    string
	lockedInstallationID     string
	unlockedInstallationID   string
	hibernatedInstallationID string
	wokenInstallationID      string

	createErr    error
	updateErr    error
	deleteErr    error
	lockErr      error
	unlockErr    error
	hibernateErr error
	wakeErr      error
	listErr      error
	clusterErr   error
	err          error
}

func (mc *MockClient) ExecClusterInstallationCLI(clusterInstallationID, command string, subcommand []string) ([]byte, error) {
	return []byte{}, nil
}

func (mc *MockClient) GetClusters(request *cloud.GetClustersRequest) ([]*cloud.ClusterDTO, error) {
	if mc.clusterErr != nil {
		return nil, mc.clusterErr
	}
	return mc.mockedCloudClustersDTO, mc.err
}

func (mc *MockClient) CreateInstallation(request *cloud.CreateInstallationRequest) (*cloud.InstallationDTO, error) {
	mc.creationRequest = request
	if mc.createErr != nil {
		return nil, mc.createErr
	}
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
	if mc.listErr != nil {
		return nil, mc.listErr
	}
	return mc.mockedCloudInstallationsDTO, mc.err
}

func (mc *MockClient) UpdateInstallation(installationID string, request *cloud.PatchInstallationRequest) (*cloud.InstallationDTO, error) {
	mc.patchInstallationID = installationID
	mc.patchRequest = request
	if mc.updateErr != nil {
		return nil, mc.updateErr
	}
	return &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid", OwnerID: "joramid"}}, nil
}

func (mc *MockClient) HibernateInstallation(installationID string) (*cloud.InstallationDTO, error) {
	mc.hibernatedInstallationID = installationID
	if mc.hibernateErr != nil {
		return nil, mc.hibernateErr
	}
	return &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid", OwnerID: "joramid"}}, nil
}

func (mc *MockClient) WakeupInstallation(installationID string, request *cloud.PatchInstallationRequest) (*cloud.InstallationDTO, error) {
	mc.wokenInstallationID = installationID
	if mc.wakeErr != nil {
		return nil, mc.wakeErr
	}
	return &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid", OwnerID: "joramid"}}, nil
}

func (mc *MockClient) LockDeletionLockForInstallation(installationID string) error {
	mc.lockedInstallationID = installationID
	if mc.lockErr != nil {
		return mc.lockErr
	}
	return nil
}

func (mc *MockClient) UnlockDeletionLockForInstallation(installationID string) error {
	mc.unlockedInstallationID = installationID
	if mc.unlockErr != nil {
		return mc.unlockErr
	}
	return nil
}

func (mc *MockClient) DeleteInstallation(installationID string) error {
	mc.deletedInstallationID = installationID
	if mc.deleteErr != nil {
		return mc.deleteErr
	}
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

func (mc *MockClient) ExecClusterInstallationPPROF(clusterInstallationID string) ([]byte, error) {
	if mc.execDebugPacket != nil {
		return mc.execDebugPacket(clusterInstallationID)
	}

	return []byte("mocked debug packet output"), nil
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

func TestAuthorizedPluginUser(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		userEmail string
		config    configuration
		expected  bool
	}{
		{"no email config", "id1", "user1@mattermost.com", configuration{AllowedEmailDomain: ""}, true},
		{"valid email", "id1", "user1@mattermost.com", configuration{AllowedEmailDomain: "mattermost.com"}, true},
		{"invalid email", "id1", "user1@matterleast.com", configuration{AllowedEmailDomain: "mattermost.com"}, false},
		{"invalid email subdomain check", "id1", "user1@subdomain.mattermost.com", configuration{AllowedEmailDomain: "mattermost.com"}, false},
		{"invalid user ID", "id2", "user2@mattermost.com", configuration{AllowedEmailDomain: "mattermost.com"}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plugin := Plugin{configuration: &test.config}
			api := &plugintest.API{}
			api.On("GetUser", "id1").Return(&model.User{Email: test.userEmail}, nil)
			api.On("GetUser", mock.AnythingOfType("string")).Return(nil, model.NewAppError("", "", nil, "not found", 404))
			api.On("LogError", mock.AnythingOfType("string"), mock.Anything, mock.Anything)
			plugin.SetAPI(api)

			assert.Equal(t, test.expected, plugin.authorizedPluginUser(test.userID))
		})
	}
}
