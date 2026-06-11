package main

import (
	"encoding/json"
	"testing"
	"time"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInstallationServiceSanitizeAndSummary(t *testing.T) {
	install := serviceTestInstall("install-id", "OwnerInstall", "owner")
	install.License = "super-secret-license"
	install.MattermostEnv = cloud.EnvVarMap{
		serviceEnvironmentEnvVarKey: cloud.EnvVar{Value: "dev"},
		"SECRET":                    cloud.EnvVar{Value: "secret-value"},
	}
	install.PriorityEnv = cloud.EnvVarMap{
		serviceEnvironmentEnvVarKey: cloud.EnvVar{Value: "staging"},
		"SAFE":                      cloud.EnvVar{Value: "safe-value"},
	}
	install.DNSRecords = []*cloud.InstallationDNS{{DomainName: "first.example.com"}, {DomainName: "second.example.com"}}

	sanitized := sanitizeInstallationCopy(install)
	require.NotNil(t, sanitized)
	assert.Equal(t, "hidden", sanitized.License)
	assert.Nil(t, sanitized.MattermostEnv)
	assert.Nil(t, sanitized.PriorityEnv)
	assert.Equal(t, "super-secret-license", install.License)
	assert.Equal(t, "secret-value", install.MattermostEnv["SECRET"].Value)
	assert.Equal(t, "safe-value", install.PriorityEnv["SAFE"].Value)

	summary, err := installationSummary(install, true)
	require.NoError(t, err)
	assert.Equal(t, "first.example.com", summary.DNS)
	assert.Equal(t, "staging", summary.ServiceEnvironment)
	assert.NotEmpty(t, summary.InstallationLogsURL)
	assert.NotEmpty(t, summary.ProvisionerLogsURL)

	summaryJSON, err := json.Marshal(summary)
	require.NoError(t, err)
	assert.NotContains(t, string(summaryJSON), "super-secret-license")
	assert.NotContains(t, string(summaryJSON), "secret-value")
	assert.NotContains(t, string(summaryJSON), "safe-value")

	wrapper, err := CreateInstallationWebWrapper(install)
	require.NoError(t, err)
	wrapperJSON, err := json.Marshal(wrapper)
	require.NoError(t, err)
	assert.Contains(t, string(wrapperJSON), "staging")
	assert.NotContains(t, string(wrapperJSON), "super-secret-license")
	assert.NotContains(t, string(wrapperJSON), "secret-value")
	assert.NotContains(t, string(wrapperJSON), "safe-value")

	placeholderSummary, err := installationSummary(&Installation{Name: "Deleted [ DELETED ]"}, true)
	require.NoError(t, err)
	assert.Equal(t, "Deleted [ DELETED ]", placeholderSummary.Name)
	assert.Empty(t, placeholderSummary.InstallationLogsURL)
	assert.Empty(t, placeholderSummary.ProvisionerLogsURL)
}

func TestInstallationServiceFindAndListScopes(t *testing.T) {
	installs := []*Installation{
		serviceTestInstall("owned-id", "OwnedInstall", "user-1"),
		serviceTestInstall("private-other-id", "PrivateOther", "user-2"),
		serviceTestInstall("shared-read-id", "SharedRead", "user-2"),
		serviceTestInstall("shared-update-id", "SharedUpdate", "user-2"),
	}
	installs[2].Shared = true
	installs[3].Shared = true
	installs[3].AllowSharedUpdates = true

	plugin, _, _ := newServiceTestPlugin(t, installs)

	found, err := plugin.findInstallationForUser("user-1", InstallationRef{ID: "owned-id"}, InstallationScopeMine)
	require.NoError(t, err)
	assert.Equal(t, "OwnedInstall", found.Name)

	found, err = plugin.findInstallationForUser("user-1", InstallationRef{Name: "OWNEDINSTALL"}, InstallationScopeMine)
	require.NoError(t, err)
	assert.Equal(t, "owned-id", found.ID)

	_, err = plugin.findInstallationForUser("user-1", InstallationRef{ID: "private-other-id"}, InstallationScopeMine)
	require.EqualError(t, err, "no installation with the id private-other-id found")

	found, err = plugin.findInstallationForUser("user-1", InstallationRef{Name: "sharedread"}, InstallationScopeShared)
	require.NoError(t, err)
	assert.Equal(t, "shared-read-id", found.ID)

	_, err = plugin.findInstallationForUser("user-1", InstallationRef{Name: "sharedread"}, InstallationScopeUpdatable)
	require.EqualError(t, err, "no installation with the name sharedread found")

	found, err = plugin.findInstallationForUser("user-1", InstallationRef{Name: "sharedupdate"}, InstallationScopeUpdatable)
	require.NoError(t, err)
	assert.Equal(t, "shared-update-id", found.ID)

	_, err = plugin.findInstallationForUser("user-1", InstallationRef{}, InstallationScopeMine)
	require.EqualError(t, err, "must provide exactly one installation id or name")

	_, err = plugin.findInstallationForUser("user-1", InstallationRef{ID: "owned-id", Name: "owned"}, InstallationScopeMine)
	require.EqualError(t, err, "must provide exactly one installation id or name")

	_, err = plugin.findInstallationForUser("user-1", InstallationRef{ID: "owned-id"}, InstallationScope("bogus"))
	require.EqualError(t, err, "unknown installation scope bogus")

	owned, err := plugin.listInstallationsForUser("user-1", ListInstallationsInput{Refresh: false})
	require.NoError(t, err)
	require.Len(t, owned, 1)
	assert.Equal(t, "owned-id", owned[0].ID)

	shared, err := plugin.listInstallationsForUser("user-1", ListInstallationsInput{Scope: InstallationScopeShared, Refresh: false})
	require.NoError(t, err)
	require.Len(t, shared, 2)

	updatable, err := plugin.listInstallationsForUser("user-1", ListInstallationsInput{Scope: InstallationScopeUpdatable, Refresh: false})
	require.NoError(t, err)
	require.Len(t, updatable, 2)
	assert.ElementsMatch(t, []string{"owned-id", "shared-update-id"}, []string{updatable[0].ID, updatable[1].ID})
}

func TestInstallationServiceRefreshWithoutCleanup(t *testing.T) {
	deleted := serviceTestInstall("deleted-id", "Deleted", "owner")
	plugin, cloudClient, api := newServiceTestPlugin(t, []*Installation{deleted})
	cloudClient.overrideGetInstallationDTO = &cloud.InstallationDTO{Installation: &cloud.Installation{
		ID:      "deleted-id",
		OwnerID: "owner",
		State:   cloud.InstallationStateDeleted,
	}}
	api.On("KVCompareAndSet").Unset()

	installs, err := plugin.listInstallationsForUser("owner", ListInstallationsInput{Scope: InstallationScopeMine, Refresh: true})
	require.NoError(t, err)
	require.Len(t, installs, 1)
	assert.Equal(t, "deleted-id", installs[0].ID)
	assert.Equal(t, "owner", installs[0].OwnerID)
	assert.Equal(t, cloud.InstallationStateDeleted, installs[0].State)
	assert.Contains(t, installs[0].Name, "DELETED")
	api.AssertNotCalled(t, "KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything)
}

func TestInstallationServiceCreate(t *testing.T) {
	t.Run("validates inputs", func(t *testing.T) {
		tests := []struct {
			name        string
			input       CreateInstallationInput
			errContains string
			installs    []*Installation
		}{
			{"missing name", CreateInstallationInput{Name: ""}, "must provide an installation name", nil},
			{"invalid name", CreateInstallationInput{Name: "bad_name"}, "installation name bad_name is invalid", nil},
			{"duplicate name", CreateInstallationInput{Name: "taken"}, "Installation name taken already exists", []*Installation{serviceTestInstall("taken-id", "taken", "owner")}},
			{"invalid size", CreateInstallationInput{Name: "new", Size: "huge"}, "Invalid size: huge", nil},
			{"invalid affinity", CreateInstallationInput{Name: "new", Affinity: "banana"}, "invalid affinity option banana", nil},
			{"invalid license", CreateInstallationInput{Name: "new", License: "e30"}, "invalid license option e30", nil},
			{"invalid image", CreateInstallationInput{Name: "new", Image: "mattermost/unknown"}, "invalid image name", nil},
			{"invalid database", CreateInstallationInput{Name: "new", Database: "sqlite"}, "invalid database option sqlite", nil},
			{"invalid filestore", CreateInstallationInput{Name: "new", Filestore: "local"}, "invalid filestore option local", nil},
			{"multitenant s3 license", CreateInstallationInput{Name: "new", Filestore: cloud.InstallationFilestoreMultiTenantAwsS3, License: licenseOptionProfessional}, "requires license option", nil},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				plugin, _, _ := newServiceTestPlugin(t, test.installs)
				_, err := plugin.createInstallationForUser("owner", test.input)
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.errContains)
			})
		}
	})

	t.Run("stores created install with digest request and requested tag", func(t *testing.T) {
		plugin, cloudClient, _ := newServiceTestPlugin(t, nil)
		plugin.dockerClient = &MockedDockerClient{tagExists: true, digest: "sha256:digest"}

		install, err := plugin.createInstallationForUser("owner", CreateInstallationInput{
			Name:    "Example",
			Version: "9.5.0",
			License: licenseOptionEnterprise,
			Env:     map[string]string{"ENV1": "value1"},
		})
		require.NoError(t, err)
		require.NotNil(t, cloudClient.creationRequest)
		assert.Equal(t, "example", cloudClient.creationRequest.Name)
		assert.Equal(t, "sha256:digest", cloudClient.creationRequest.Version)
		assert.Equal(t, "9.5.0", install.Tag)
		assert.Equal(t, cloud.EnvVarMap{"ENV1": cloud.EnvVar{Value: "value1"}}, cloudClient.creationRequest.PriorityEnv)
	})
}

func TestInstallationServiceUpdate(t *testing.T) {
	t.Run("rejects empty and inaccessible shared updates", func(t *testing.T) {
		installs := []*Installation{serviceTestInstall("shared-id", "Shared", "owner")}
		installs[0].Shared = true
		plugin, _, _ := newServiceTestPlugin(t, installs)

		_, err := plugin.updateInstallationForUser("owner", InstallationRef{Name: "shared"}, UpdateInstallationInput{}, InstallationScopeMine)
		require.EqualError(t, err, "must specify at least one option: version, license, image, size, env, clear-env")

		_, err = plugin.updateInstallationForUser("other", InstallationRef{Name: "shared"}, UpdateInstallationInput{Size: "miniHA"}, InstallationScopeUpdatable)
		require.EqualError(t, err, "no installation with the name shared found")
	})

	t.Run("converts license and docker tag without returning env values", func(t *testing.T) {
		installs := []*Installation{serviceTestInstall("install-id", "Install", "owner")}
		plugin, cloudClient, _ := newServiceTestPlugin(t, installs)
		plugin.configuration.E20License = "license-value"
		plugin.dockerClient = &MockedDockerClient{tagExists: true, digest: "sha256:new"}

		result, err := plugin.updateInstallationForUser("owner", InstallationRef{Name: "install"}, UpdateInstallationInput{
			Version:  "9.5.0",
			License:  licenseOptionE20,
			Image:    imageTE,
			Size:     "miniHA",
			SetEnv:   map[string]string{"SECRET_ENV": "secret-value"},
			ClearEnv: []string{"OLD_ENV"},
		}, InstallationScopeMine)
		require.NoError(t, err)

		require.NotNil(t, cloudClient.patchRequest)
		require.NotNil(t, cloudClient.patchRequest.Version)
		require.NotNil(t, cloudClient.patchRequest.License)
		assert.Equal(t, "sha256:new", *cloudClient.patchRequest.Version)
		assert.Equal(t, "license-value", *cloudClient.patchRequest.License)
		assert.Equal(t, cloud.EnvVarMap{
			"SECRET_ENV": cloud.EnvVar{Value: "secret-value"},
			"OLD_ENV":    cloud.EnvVar{},
		}, cloudClient.patchRequest.PriorityEnv)
		assert.ElementsMatch(t, []string{"env", "image", "license", "size", "version"}, result.ChangedFields)
		assert.Equal(t, []string{"SECRET_ENV"}, result.ChangedEnvKeys)
		assert.Equal(t, []string{"OLD_ENV"}, result.ClearedEnvKeys)

		resultJSON, err := json.Marshal(result)
		require.NoError(t, err)
		assert.NotContains(t, string(resultJSON), "secret-value")
	})

	t.Run("clear-env only without set-env", func(t *testing.T) {
		installs := []*Installation{serviceTestInstall("install-id", "Install", "owner")}
		plugin, cloudClient, _ := newServiceTestPlugin(t, installs)

		result, err := plugin.updateInstallationForUser("owner", InstallationRef{Name: "install"}, UpdateInstallationInput{
			ClearEnv: []string{"OLD_ENV"},
		}, InstallationScopeMine)
		require.NoError(t, err)

		require.NotNil(t, cloudClient.patchRequest)
		assert.Equal(t, cloud.EnvVarMap{"OLD_ENV": cloud.EnvVar{}}, cloudClient.patchRequest.PriorityEnv)
		assert.Equal(t, []string{"env"}, result.ChangedFields)
		assert.Empty(t, result.ChangedEnvKeys)
		assert.Equal(t, []string{"OLD_ENV"}, result.ClearedEnvKeys)
	})

	t.Run("image-only update uses stored tag instead of stored digest", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		install.Version = "sha256:stored-digest"
		install.Tag = "9.5.0"
		plugin, cloudClient, _ := newServiceTestPlugin(t, []*Installation{install})
		dockerClient := &MockedDockerClient{tagExists: true, digest: "sha256:new-image-digest"}
		plugin.dockerClient = dockerClient

		result, err := plugin.updateInstallationForUser("owner", InstallationRef{Name: "install"}, UpdateInstallationInput{
			Image: imageTE,
		}, InstallationScopeMine)
		require.NoError(t, err)

		require.NotNil(t, cloudClient.patchRequest)
		require.NotNil(t, cloudClient.patchRequest.Version)
		assert.Equal(t, "sha256:new-image-digest", *cloudClient.patchRequest.Version)
		assert.Equal(t, "9.5.0", result.Installation.VersionTag)
		assert.Equal(t, []dockerClientCall{{tag: "9.5.0", repository: imageTE}}, dockerClient.validTagCalls)
		assert.Equal(t, []dockerClientCall{{tag: "9.5.0", repository: imageTE}}, dockerClient.getDigestCalls)
	})

	t.Run("shared update after redacted read preserves sensitive persisted fields", func(t *testing.T) {
		shared := serviceTestInstall("shared-id", "Shared", "owner")
		shared.Shared = true
		shared.AllowSharedUpdates = true
		shared.License = "secret-license"
		shared.MattermostEnv = cloud.EnvVarMap{"SECRET": cloud.EnvVar{Value: "secret-value"}}
		shared.PriorityEnv = cloud.EnvVarMap{"PRIORITY": cloud.EnvVar{Value: "priority-value"}}
		plugin, cloudClient, api := newServiceTestPlugin(t, []*Installation{shared})
		cloudClient.overrideGetInstallationDTO = &cloud.InstallationDTO{
			Installation: shared.Clone(),
		}
		cloudClient.mockedCloudInstallationsDTO = serviceDTOs(shared)

		redacted, err := plugin.getUpdatedSharedInstallations(true)
		require.NoError(t, err)
		require.Len(t, redacted, 1)
		assert.Equal(t, "hidden", redacted[0].License)
		assert.Nil(t, redacted[0].MattermostEnv)
		assert.Nil(t, redacted[0].PriorityEnv)

		var persisted []*Installation
		api.On("KVCompareAndSet").Unset()
		api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				require.NoError(t, json.Unmarshal(args.Get(2).([]byte), &persisted))
			}).
			Return(true, nil)

		_, err = plugin.updateInstallationForUser("other", InstallationRef{Name: "shared"}, UpdateInstallationInput{Size: "miniHA"}, InstallationScopeUpdatable)
		require.NoError(t, err)
		require.Len(t, persisted, 1)
		assert.Equal(t, "secret-license", persisted[0].License)
		assert.Equal(t, "secret-value", persisted[0].MattermostEnv["SECRET"].Value)
		assert.Equal(t, "priority-value", persisted[0].PriorityEnv["PRIORITY"].Value)
	})
}

func TestInstallationServiceLifecycleActions(t *testing.T) {
	t.Run("restart patches env and respects shared update gate", func(t *testing.T) {
		installs := []*Installation{serviceTestInstall("shared-id", "Shared", "owner")}
		installs[0].Shared = true
		plugin, _, _ := newServiceTestPlugin(t, installs)

		_, err := plugin.restartInstallationForUser("other", InstallationRef{Name: "shared"}, InstallationScopeUpdatable)
		require.EqualError(t, err, "no installation with the name shared found")

		installs[0].AllowSharedUpdates = true
		plugin, cloudClient, _ := newServiceTestPlugin(t, installs)
		result, err := plugin.restartInstallationForUser("other", InstallationRef{Name: "shared"}, InstallationScopeUpdatable)
		require.NoError(t, err)
		assert.Equal(t, "restart_requested", result.Status)
		assert.Equal(t, "shared-id", cloudClient.patchInstallationID)
		assert.Contains(t, cloudClient.patchRequest.MattermostEnv, "CLOUD_PLUGIN_RESTART")
	})

	t.Run("hibernate and wake enforce state and owner", func(t *testing.T) {
		stable := serviceTestInstall("stable-id", "Stable", "owner")
		hibernating := serviceTestInstall("hibernating-id", "Hibernating", "owner")
		hibernating.State = cloud.InstallationStateHibernating
		plugin, cloudClient, _ := newServiceTestPlugin(t, []*Installation{stable, hibernating})
		cloudClient.mockedCloudInstallationsDTO = serviceDTOs(stable, hibernating)

		result, err := plugin.hibernateInstallationForUser("owner", InstallationRef{Name: "stable"})
		require.NoError(t, err)
		assert.Equal(t, "hibernate_requested", result.Status)
		assert.Equal(t, "stable-id", cloudClient.hibernatedInstallationID)

		_, err = plugin.hibernateInstallationForUser("other", InstallationRef{Name: "stable"})
		require.EqualError(t, err, "no installation with the name stable found")

		_, err = plugin.hibernateInstallationForUser("owner", InstallationRef{Name: "hibernating"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be stable to hibernate")

		result, err = plugin.wakeInstallationForUser("owner", InstallationRef{Name: "hibernating"})
		require.NoError(t, err)
		assert.Equal(t, "wake_requested", result.Status)
		assert.Equal(t, "hibernating-id", cloudClient.wokenInstallationID)

		_, err = plugin.wakeInstallationForUser("owner", InstallationRef{Name: "stable"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be hibernating to wake up")
	})
}

func TestInstallationServiceSharingDeletionLockAndDelete(t *testing.T) {
	t.Run("sharing updates and unsharing clears update permission", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		plugin, _, _ := newServiceTestPlugin(t, []*Installation{install})
		plugin.cloudClient.(*MockClient).mockedCloudInstallationsDTO = serviceDTOs(install)

		result, err := plugin.setInstallationSharingForUser("owner", InstallationRef{Name: "install"}, true, true)
		require.NoError(t, err)
		assert.True(t, result.Installation.Shared)
		assert.True(t, result.Installation.AllowSharedUpdates)

		result, err = plugin.setInstallationSharingForUser("owner", InstallationRef{Name: "install"}, false, true)
		require.NoError(t, err)
		assert.False(t, result.Installation.Shared)
		assert.False(t, result.Installation.AllowSharedUpdates)
	})

	t.Run("deletion lock counts all owned locked installs before enforcing limit", func(t *testing.T) {
		target := serviceTestInstall("target-id", "Target", "owner")
		locked := serviceTestInstall("locked-id", "Locked", "owner")
		locked.DeletionLocked = true
		plugin, cloudClient, _ := newServiceTestPlugin(t, []*Installation{target, locked})
		cloudClient.mockedCloudInstallationsDTO = serviceDTOs(target, locked)
		plugin.configuration.DeletionLockInstallationsAllowedPerPerson = "1"

		_, err := plugin.setDeletionLockForUser("owner", InstallationRef{Name: "target"}, true)
		require.EqualError(t, err, "you may only have at most 1 installations locked for deletion at a time")
		assert.Empty(t, cloudClient.lockedInstallationID)
	})

	t.Run("already locked target does not fail due to its own lock", func(t *testing.T) {
		target := serviceTestInstall("target-id", "Target", "owner")
		target.DeletionLocked = true
		plugin, cloudClient, _ := newServiceTestPlugin(t, []*Installation{target})
		cloudClient.mockedCloudInstallationsDTO = serviceDTOs(target)
		plugin.configuration.DeletionLockInstallationsAllowedPerPerson = "1"

		result, err := plugin.setDeletionLockForUser("owner", InstallationRef{Name: "target"}, true)
		require.NoError(t, err)
		assert.True(t, result.Installation.DeletionLocked)
		assert.Equal(t, "target-id", cloudClient.lockedInstallationID)
	})

	t.Run("deletion lock validates config target and owner", func(t *testing.T) {
		target := serviceTestInstall("target-id", "Target", "owner")
		plugin, _, _ := newServiceTestPlugin(t, []*Installation{target})
		plugin.cloudClient.(*MockClient).mockedCloudInstallationsDTO = serviceDTOs(target)

		_, err := plugin.setDeletionLockForUser("owner", InstallationRef{}, true)
		require.EqualError(t, err, "must provide an installation ID or name")

		_, err = plugin.setDeletionLockForUser("other", InstallationRef{Name: "target"}, true)
		require.EqualError(t, err, "no installations found for the given User ID")

		_, err = plugin.setDeletionLockForUser("owner", InstallationRef{Name: "missing"}, true)
		require.EqualError(t, err, "installation to be locked not found")

		plugin.configuration.DeletionLockInstallationsAllowedPerPerson = "invalid"
		_, err = plugin.setDeletionLockForUser("owner", InstallationRef{Name: "target"}, true)
		require.EqualError(t, err, "invalid value for DeletionLockInstallationsAllowedPerPerson")
	})

	t.Run("deletion lock and unlock call provisioner", func(t *testing.T) {
		target := serviceTestInstall("target-id", "Target", "owner")
		plugin, cloudClient, _ := newServiceTestPlugin(t, []*Installation{target})
		cloudClient.mockedCloudInstallationsDTO = serviceDTOs(target)

		result, err := plugin.setDeletionLockForUser("owner", InstallationRef{ID: "target-id"}, true)
		require.NoError(t, err)
		assert.Equal(t, "deletion_lock_updated", result.Status)
		assert.Equal(t, "target-id", cloudClient.lockedInstallationID)

		result, err = plugin.setDeletionLockForUser("owner", InstallationRef{ID: "target-id"}, false)
		require.NoError(t, err)
		assert.Equal(t, "target-id", cloudClient.unlockedInstallationID)
		assert.False(t, result.Installation.DeletionLocked)
	})

	t.Run("delete requires confirmation and calls provisioner before KV delete", func(t *testing.T) {
		target := serviceTestInstall("delete-id", "DeleteMe", "owner")
		plugin, cloudClient, api := newServiceTestPlugin(t, []*Installation{target})
		api.On("KVCompareAndSet").Unset()
		api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				assert.Equal(t, "delete-id", cloudClient.deletedInstallationID)
			}).
			Return(true, nil)

		_, err := plugin.deleteInstallationForUser("owner", InstallationRef{Name: "deleteme"}, "wrong")
		require.EqualError(t, err, "confirmation name wrong does not match installation name DeleteMe")
		assert.Empty(t, cloudClient.deletedInstallationID)

		result, err := plugin.deleteInstallationForUser("owner", InstallationRef{Name: "deleteme"}, "deleteme")
		require.NoError(t, err)
		assert.Equal(t, "delete_requested", result.Status)
		assert.Equal(t, "delete-id", cloudClient.deletedInstallationID)

		_, err = plugin.deleteInstallationForUser("owner", InstallationRef{ID: "missing-id"}, "deleteme")
		require.EqualError(t, err, "no installation with the id missing-id found")
	})
}

func newServiceTestPlugin(t *testing.T, installs []*Installation) (*Plugin, *MockClient, *plugintest.API) {
	t.Helper()

	installBytes, err := json.Marshal(installs)
	require.NoError(t, err)
	if installs == nil {
		installBytes = nil
	}

	cloudClient := &MockClient{}
	plugin := &Plugin{
		cloudClient:  cloudClient,
		dockerClient: &MockedDockerClient{tagExists: true},
		configuration: &configuration{
			InstallationDNS: "example.com",
			DeletionLockInstallationsAllowedPerPerson: "2",
			EnterpriseLicense:                         "enterprise-license",
			EnterpriseAdvancedLicense:                 "enterprise-advanced-license",
			ProfessionalLicense:                       "professional-license",
			E20License:                                "e20-license",
			E10License:                                "e10-license",
		},
		latestMattermostVersion: &latestMattermostVersionCache{version: "9.5.0", timestamp: time.Now()},
	}

	api := &plugintest.API{}
	api.On("KVGet", mock.AnythingOfType("string")).Return(installBytes, nil)
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
	api.On("LogError", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)
	plugin.SetAPI(api)

	return plugin, cloudClient, api
}

func serviceTestInstall(id, name, ownerID string) *Installation {
	return &Installation{
		Name: name,
		InstallationDTO: cloud.InstallationDTO{
			Installation: &cloud.Installation{
				ID:        id,
				OwnerID:   ownerID,
				Name:      name,
				Version:   "9.4.0",
				Image:     imageEE,
				Size:      "miniSingleton",
				Database:  cloud.InstallationDatabaseMultiTenantRDSPostgresPGBouncer,
				Filestore: cloud.InstallationFilestoreBifrost,
				Affinity:  cloud.InstallationAffinityMultiTenant,
				State:     cloud.InstallationStateStable,
				CreateAt:  1234,
			},
		},
		Tag: "9.4.0",
	}
}

func serviceDTOs(installs ...*Installation) []*cloud.InstallationDTO {
	dtos := make([]*cloud.InstallationDTO, 0, len(installs))
	for _, install := range installs {
		dtos = append(dtos, &cloud.InstallationDTO{
			Installation: install.Clone(),
			DNSRecords:   install.DNSRecords,
		})
	}
	return dtos
}
