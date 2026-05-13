package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMCPAdminToolsRegistration(t *testing.T) {
	plugin, _, _ := newMCPToolsTestPlugin(t, nil)
	session, cleanup := connectMCPToolsClient(t, plugin, "admin")
	defer cleanup()

	result, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	require.NoError(t, err)
	require.Len(t, result.Tools, 11)

	tools := map[string]*mcp.Tool{}
	for _, tool := range result.Tools {
		tools[tool.Name] = tool
	}

	for _, toolName := range []string{
		mcpListInstallationsToolName,
		mcpGetInstallationToolName,
		mcpCreateInstallationToolName,
		mcpUpdateInstallationToolName,
		mcpRestartInstallationToolName,
		mcpHibernateInstallationToolName,
		mcpWakeInstallationToolName,
		mcpSetInstallationSharingToolName,
		mcpSetDeletionLockToolName,
		mcpDeleteInstallationToolName,
		mcpCloudStatusToolName,
	} {
		assert.Contains(t, tools, toolName)
	}

	statusTool := tools[mcpCloudStatusToolName]
	require.NotNil(t, statusTool)
	assert.Equal(t, "Get Cloud Status", statusTool.Title)
	require.NotNil(t, statusTool.Annotations)
	assert.True(t, statusTool.Annotations.ReadOnlyHint)
	require.NotNil(t, statusTool.Annotations.DestructiveHint)
	assert.False(t, *statusTool.Annotations.DestructiveHint)
	require.NotNil(t, statusTool.Annotations.OpenWorldHint)
	assert.False(t, *statusTool.Annotations.OpenWorldHint)
	assertMCPInputSchemaProperties(t, statusTool, "include_clusters")

	for _, deferredToolName := range []string{
		"com_mattermost_cloud__run_mmctl",
		"com_mattermost_cloud__run_mmcli",
		"com_mattermost_cloud__get_debug_packet",
		"com_mattermost_cloud__import_installation",
	} {
		assert.NotContains(t, tools, deferredToolName)
	}
}

func TestCloudStatusMCPAuthorization(t *testing.T) {
	t.Run("missing user context is denied", func(t *testing.T) {
		plugin, _, api := newMCPToolsTestPlugin(t, nil)
		session, cleanup := connectMCPToolsClient(t, plugin, "")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpCloudStatusToolName, map[string]any{})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "no Mattermost user ID")
		api.AssertNotCalled(t, "HasPermissionTo", mock.Anything, mock.Anything)
	})

	t.Run("domain-denied user is denied before admin permission check", func(t *testing.T) {
		plugin, _, api := newMCPToolsTestPlugin(t, nil)
		plugin.configuration.AllowedEmailDomain = "mattermost.com"
		api.On("GetUser", "owner").Return(&model.User{Id: "owner", Email: "owner@example.com"}, nil).Once()

		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpCloudStatusToolName, map[string]any{})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "user is not authorized to use the Cloud plugin")
		api.AssertNotCalled(t, "HasPermissionTo", mock.Anything, mock.Anything)
	})

	t.Run("authorized non-admin user is denied", func(t *testing.T) {
		plugin, _, api := newMCPToolsTestPlugin(t, nil)
		api.On("HasPermissionTo", "owner", model.PermissionManageSystem).Return(false).Once()

		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpCloudStatusToolName, map[string]any{})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "user is not authorized to use Cloud admin tools")
	})

	t.Run("system admin succeeds", func(t *testing.T) {
		plugin, _, api := newMCPToolsTestPlugin(t, nil)
		api.On("HasPermissionTo", "admin", model.PermissionManageSystem).Return(true).Once()

		session, cleanup := connectMCPToolsClient(t, plugin, "admin")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpCloudStatusToolName, map[string]any{})
		require.NoError(t, err)
		assert.False(t, result.IsError)
	})
}

func TestCloudStatusMCP(t *testing.T) {
	t.Run("returns sanitized installations without mutating state", func(t *testing.T) {
		plugin, cloudClient, api := newMCPToolsTestPlugin(t, nil)
		api.On("HasPermissionTo", "admin", model.PermissionManageSystem).Return(true).Once()
		cloudClient.mockedCloudInstallationsDTO = []*cloud.InstallationDTO{cloudStatusTestInstallation()}
		cloudClient.clusterErr = errors.New("clusters should not be fetched")

		session, cleanup := connectMCPToolsClient(t, plugin, "admin")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpCloudStatusToolName, map[string]any{})
		require.NoError(t, err)
		require.False(t, result.IsError)

		output := decodeMCPStructuredOutput[CloudStatusMCPOutput](t, result)
		require.Equal(t, 1, output.InstallationCount)
		require.Len(t, output.Installations, 1)
		assert.Equal(t, "status-install", output.Installations[0].ID)
		assert.Equal(t, "status.example.com", output.Installations[0].DNS)
		assert.Equal(t, "group-id", output.Installations[0].GroupID)
		assert.Equal(t, []string{"cluster-a", "cluster-b"}, output.Installations[0].ClusterIDs)
		assert.Empty(t, output.Clusters)
		assert.Zero(t, output.ClusterCount)
		assertJSONKeys(t, output.Installations[0],
			"id", "name", "dns", "state", "owner_id", "version", "size", "database", "filestore",
			"create_at", "delete_at", "deletion_locked", "scheduled_deletion_time", "group_id", "cluster_ids",
		)
		assertMCPResultRedacted(t, result,
			"secret-license", "secret-env-value", "priority-secret-value", "sensitive-annotation",
			"License", "MattermostEnv", "PriorityEnv", "Annotations", "LockAcquiredBy", "GroupOverrides",
		)
		api.AssertNotCalled(t, "KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything)
		assert.Empty(t, cloudClient.patchInstallationID)
		assert.Empty(t, cloudClient.deletedInstallationID)
		assert.Empty(t, cloudClient.lockedInstallationID)
		assert.Empty(t, cloudClient.unlockedInstallationID)
	})

	t.Run("includes sanitized clusters when requested", func(t *testing.T) {
		plugin, cloudClient, api := newMCPToolsTestPlugin(t, nil)
		api.On("HasPermissionTo", "admin", model.PermissionManageSystem).Return(true).Once()
		cloudClient.mockedCloudInstallationsDTO = []*cloud.InstallationDTO{cloudStatusTestInstallation()}
		cloudClient.mockedCloudClustersDTO = []*cloud.ClusterDTO{cloudStatusTestCluster()}

		session, cleanup := connectMCPToolsClient(t, plugin, "admin")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpCloudStatusToolName, map[string]any{"include_clusters": true})
		require.NoError(t, err)
		require.False(t, result.IsError)

		output := decodeMCPStructuredOutput[CloudStatusMCPOutput](t, result)
		require.Equal(t, 1, output.InstallationCount)
		require.Equal(t, 1, output.ClusterCount)
		require.Len(t, output.Clusters, 1)
		assert.Equal(t, "cluster-a", output.Clusters[0].ID)
		assert.Equal(t, "aws", output.Clusters[0].Provider)
		assert.Equal(t, "kops", output.Clusters[0].Provisioner)
		assertJSONKeys(t, output.Clusters[0],
			"id", "name", "state", "provider", "provisioner", "allow_installations", "create_at", "delete_at", "api_security_lock",
		)
		assertMCPResultRedacted(t, result,
			"secret-license", "secret-env-value", "priority-secret-value", "sensitive-annotation",
			"ProviderMetadataAWS", "ProvisionerMetadataKops", "Annotations", "LockAcquiredBy", "SchedulingLockAcquiredBy",
		)
	})

	t.Run("returns provisioner installation-list errors as tool errors", func(t *testing.T) {
		plugin, cloudClient, api := newMCPToolsTestPlugin(t, nil)
		api.On("HasPermissionTo", "admin", model.PermissionManageSystem).Return(true).Once()
		cloudClient.listErr = errors.New("installation list failed")

		session, cleanup := connectMCPToolsClient(t, plugin, "admin")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpCloudStatusToolName, map[string]any{})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "installation list failed")
	})

	t.Run("returns provisioner cluster-list errors only when requested", func(t *testing.T) {
		plugin, cloudClient, api := newMCPToolsTestPlugin(t, nil)
		api.On("HasPermissionTo", "admin", model.PermissionManageSystem).Return(true).Twice()
		cloudClient.mockedCloudInstallationsDTO = []*cloud.InstallationDTO{cloudStatusTestInstallation()}
		cloudClient.clusterErr = errors.New("cluster list failed")

		session, cleanup := connectMCPToolsClient(t, plugin, "admin")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpCloudStatusToolName, map[string]any{})
		require.NoError(t, err)
		assert.False(t, result.IsError)

		result, err = callMCPTool(t, session, mcpCloudStatusToolName, map[string]any{"include_clusters": true})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "cluster list failed")
	})
}

func cloudStatusTestInstallation() *cloud.InstallationDTO {
	groupID := "group-id"
	clusterA := "cluster-a"
	clusterB := "cluster-b"
	lockOwner := "lock-owner"

	return &cloud.InstallationDTO{
		Installation: &cloud.Installation{
			ID:                    "status-install",
			Name:                  "Status Install",
			OwnerID:               "owner",
			GroupID:               &groupID,
			State:                 cloud.InstallationStateStable,
			Version:               "9.5.0",
			Size:                  "miniSingleton",
			Database:              cloud.InstallationDatabaseMultiTenantRDSPostgresPGBouncer,
			Filestore:             cloud.InstallationFilestoreBifrost,
			License:               "secret-license",
			MattermostEnv:         cloud.EnvVarMap{"SECRET": cloud.EnvVar{Value: "secret-env-value"}},
			PriorityEnv:           cloud.EnvVarMap{"PRIORITY_SECRET": cloud.EnvVar{Value: "priority-secret-value"}},
			CreateAt:              1000,
			DeleteAt:              2000,
			DeletionLocked:        true,
			ScheduledDeletionTime: 3000,
			LockAcquiredBy:        &lockOwner,
			GroupOverrides:        map[string]string{"sensitive": "override"},
		},
		DNSRecords: []*cloud.InstallationDNS{{DomainName: "status.example.com"}},
		ClusterIDs: []*string{&clusterA, &clusterB},
		Annotations: []*cloud.Annotation{{
			ID: "sensitive-annotation",
		}},
	}
}

func cloudStatusTestCluster() *cloud.ClusterDTO {
	lockOwner := "lock-owner"

	return &cloud.ClusterDTO{
		Cluster: &cloud.Cluster{
			ID:                       "cluster-a",
			Name:                     "Cluster A",
			State:                    cloud.ClusterStateStable,
			Provider:                 "aws",
			ProviderMetadataAWS:      &cloud.AWSMetadata{},
			Provisioner:              "kops",
			ProvisionerMetadataKops:  &cloud.KopsMetadata{},
			AllowInstallations:       true,
			CreateAt:                 4000,
			DeleteAt:                 5000,
			APISecurityLock:          true,
			LockAcquiredBy:           &lockOwner,
			SchedulingLockAcquiredBy: &lockOwner,
		},
		Annotations: []*cloud.Annotation{{
			ID: "sensitive-annotation",
		}},
	}
}

func assertJSONKeys(t *testing.T, value any, expectedKeys ...string) {
	t.Helper()

	data, err := json.Marshal(value)
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(data, &fields))
	assert.ElementsMatch(t, expectedKeys, mapKeys(fields))
}

func mapKeys(fields map[string]any) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	return keys
}
