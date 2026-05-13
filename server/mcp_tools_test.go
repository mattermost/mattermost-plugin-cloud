package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	mcpListInstallationsToolName      = "com_mattermost_cloud__list_installations"
	mcpGetInstallationToolName        = "com_mattermost_cloud__get_installation"
	mcpCreateInstallationToolName     = "com_mattermost_cloud__create_installation"
	mcpUpdateInstallationToolName     = "com_mattermost_cloud__update_installation"
	mcpRestartInstallationToolName    = "com_mattermost_cloud__restart_installation"
	mcpHibernateInstallationToolName  = "com_mattermost_cloud__hibernate_installation"
	mcpWakeInstallationToolName       = "com_mattermost_cloud__wake_installation"
	mcpSetInstallationSharingToolName = "com_mattermost_cloud__set_installation_sharing"
	mcpSetDeletionLockToolName        = "com_mattermost_cloud__set_deletion_lock"
	mcpDeleteInstallationToolName     = "com_mattermost_cloud__delete_installation"
	mcpCloudStatusToolName            = "com_mattermost_cloud__cloud_status"
)

func TestMCPToolsRegistration(t *testing.T) {
	plugin, _, _ := newMCPToolsTestPlugin(t, nil)
	session, cleanup := connectMCPToolsClient(t, plugin, "owner")
	defer cleanup()

	result, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	require.NoError(t, err)
	require.Len(t, result.Tools, 11)

	tools := map[string]*mcp.Tool{}
	for _, tool := range result.Tools {
		tools[tool.Name] = tool
	}

	listTool := tools[mcpListInstallationsToolName]
	require.NotNil(t, listTool)
	assert.Equal(t, "List Cloud Installations", listTool.Title)
	require.NotNil(t, listTool.Annotations)
	assert.True(t, listTool.Annotations.ReadOnlyHint)
	require.NotNil(t, listTool.Annotations.DestructiveHint)
	assert.False(t, *listTool.Annotations.DestructiveHint)
	require.NotNil(t, listTool.Annotations.OpenWorldHint)
	assert.False(t, *listTool.Annotations.OpenWorldHint)
	assertMCPInputSchemaProperties(t, listTool, "scope", "refresh", "include_log_urls")

	getTool := tools[mcpGetInstallationToolName]
	require.NotNil(t, getTool)
	assert.Equal(t, "Get Cloud Installation", getTool.Title)
	require.NotNil(t, getTool.Annotations)
	assert.True(t, getTool.Annotations.ReadOnlyHint)
	require.NotNil(t, getTool.Annotations.DestructiveHint)
	assert.False(t, *getTool.Annotations.DestructiveHint)
	require.NotNil(t, getTool.Annotations.OpenWorldHint)
	assert.False(t, *getTool.Annotations.OpenWorldHint)
	assertMCPInputSchemaProperties(t, getTool, "installation_id", "name", "scope", "refresh", "include_log_urls")

	lifecycleTools := map[string][]string{
		mcpCreateInstallationToolName:     {"name", "version", "size", "license", "affinity", "database", "filestore", "image", "test_data", "env"},
		mcpUpdateInstallationToolName:     {"installation_id", "name", "scope", "version", "image", "license", "size", "set_env", "clear_env"},
		mcpRestartInstallationToolName:    {"installation_id", "name", "scope"},
		mcpHibernateInstallationToolName:  {"installation_id", "name"},
		mcpWakeInstallationToolName:       {"installation_id", "name"},
		mcpSetInstallationSharingToolName: {"installation_id", "name", "shared", "allow_updates"},
		mcpSetDeletionLockToolName:        {"installation_id", "name", "locked"},
		mcpDeleteInstallationToolName:     {"installation_id", "name", "confirm_name"},
	}
	for toolName, schemaProperties := range lifecycleTools {
		tool := tools[toolName]
		require.NotNil(t, tool, "missing tool %s", toolName)
		require.NotNil(t, tool.Annotations)
		assert.False(t, tool.Annotations.ReadOnlyHint)
		require.NotNil(t, tool.Annotations.DestructiveHint)
		assert.Equal(t, toolName == mcpDeleteInstallationToolName, *tool.Annotations.DestructiveHint)
		require.NotNil(t, tool.Annotations.OpenWorldHint)
		assert.False(t, *tool.Annotations.OpenWorldHint)
		assertMCPInputSchemaProperties(t, tool, schemaProperties...)
	}

	assertMCPInputSchemaRequired(t, tools[mcpDeleteInstallationToolName], "confirm_name")
}

func TestListInstallationsMCP(t *testing.T) {
	t.Run("authorizes caller from plugin user context", func(t *testing.T) {
		plugin, _, api := newMCPToolsTestPlugin(t, []*Installation{serviceTestInstall("owned-id", "Owned", "owner")})

		missingUserSession, missingUserCleanup := connectMCPToolsClient(t, plugin, "")
		defer missingUserCleanup()
		result, err := callMCPTool(t, missingUserSession, mcpListInstallationsToolName, map[string]any{"refresh": false})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "no Mattermost user ID")

		plugin.configuration.AllowedEmailDomain = "mattermost.com"
		api.On("GetUser", "owner").Return(&model.User{Id: "owner", Email: "owner@example.com"}, nil).Once()

		deniedSession, deniedCleanup := connectMCPToolsClient(t, plugin, "owner")
		defer deniedCleanup()
		result, err = callMCPTool(t, deniedSession, mcpListInstallationsToolName, map[string]any{"refresh": false})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "not authorized")
	})

	t.Run("lists scoped installs with defaults and redacted output", func(t *testing.T) {
		owned := serviceTestInstall("owned-id", "Owned", "owner")
		owned.License = "secret-license"
		owned.MattermostEnv = cloud.EnvVarMap{"SECRET": cloud.EnvVar{Value: "secret-env-value"}}
		owned.PriorityEnv = cloud.EnvVarMap{"PRIORITY_SECRET": cloud.EnvVar{Value: "priority-secret-value"}}
		privateOther := serviceTestInstall("private-other-id", "PrivateOther", "other")
		sharedRead := serviceTestInstall("shared-read-id", "SharedRead", "other")
		sharedRead.Shared = true
		sharedUpdate := serviceTestInstall("shared-update-id", "SharedUpdate", "other")
		sharedUpdate.Shared = true
		sharedUpdate.AllowSharedUpdates = true

		plugin, _, _ := newMCPToolsTestPlugin(t, []*Installation{owned, privateOther, sharedRead, sharedUpdate})
		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpListInstallationsToolName, map[string]any{"refresh": false})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output := decodeMCPStructuredOutput[ListInstallationsMCPOutput](t, result)
		require.Equal(t, 1, output.Count)
		assert.Equal(t, "owned-id", output.Installations[0].ID)
		assert.Empty(t, output.Installations[0].InstallationLogsURL)
		assertMCPResultRedacted(t, result, "secret-license", "secret-env-value", "priority-secret-value", "MattermostEnv", "PriorityEnv")

		result, err = callMCPTool(t, session, mcpListInstallationsToolName, map[string]any{"scope": "shared", "refresh": false})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output = decodeMCPStructuredOutput[ListInstallationsMCPOutput](t, result)
		assert.Equal(t, 2, output.Count)
		assert.ElementsMatch(t, []string{"shared-read-id", "shared-update-id"}, installationSummaryIDs(output.Installations))

		result, err = callMCPTool(t, session, mcpListInstallationsToolName, map[string]any{"scope": "updatable", "refresh": false})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output = decodeMCPStructuredOutput[ListInstallationsMCPOutput](t, result)
		assert.Equal(t, 2, output.Count)
		assert.ElementsMatch(t, []string{"owned-id", "shared-update-id"}, installationSummaryIDs(output.Installations))

		result, err = callMCPTool(t, session, mcpListInstallationsToolName, map[string]any{"refresh": false, "include_log_urls": true})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output = decodeMCPStructuredOutput[ListInstallationsMCPOutput](t, result)
		require.Len(t, output.Installations, 1)
		assert.NotEmpty(t, output.Installations[0].InstallationLogsURL)
		assert.NotEmpty(t, output.Installations[0].ProvisionerLogsURL)
	})

	t.Run("invalid scope is a tool error", func(t *testing.T) {
		plugin, _, _ := newMCPToolsTestPlugin(t, nil)
		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpListInstallationsToolName, map[string]any{"scope": "bogus", "refresh": false})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "unknown installation scope bogus")
	})

	t.Run("refresh does not clean up deleted installs", func(t *testing.T) {
		deleted := serviceTestInstall("deleted-id", "Deleted", "owner")
		plugin, cloudClient, api := newMCPToolsTestPlugin(t, []*Installation{deleted})
		cloudClient.overrideGetInstallationDTO = &cloud.InstallationDTO{Installation: &cloud.Installation{
			ID:      "deleted-id",
			OwnerID: "owner",
			State:   cloud.InstallationStateDeleted,
		}}
		cloudClient.mockedCloudInstallationsDTO = nil
		api.On("KVCompareAndSet").Unset()

		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()
		result, err := callMCPTool(t, session, mcpListInstallationsToolName, map[string]any{"include_log_urls": true})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output := decodeMCPStructuredOutput[ListInstallationsMCPOutput](t, result)
		require.Equal(t, 1, output.Count)
		assert.Equal(t, "deleted-id", output.Installations[0].ID)
		assert.Equal(t, "owner", output.Installations[0].OwnerID)
		assert.Equal(t, cloud.InstallationStateDeleted, output.Installations[0].State)
		assert.Contains(t, output.Installations[0].Name, "DELETED")
		assert.NotEmpty(t, output.Installations[0].InstallationLogsURL)
		assert.NotEmpty(t, output.Installations[0].ProvisionerLogsURL)
		api.AssertNotCalled(t, "KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything)
	})
}

func TestGetInstallationMCP(t *testing.T) {
	t.Run("resolves by ID and name with scoped visibility", func(t *testing.T) {
		owned := serviceTestInstall("owned-id", "OwnedInstall", "owner")
		sharedRead := serviceTestInstall("shared-read-id", "SharedRead", "other")
		sharedRead.Shared = true
		sharedUpdate := serviceTestInstall("shared-update-id", "SharedUpdate", "other")
		sharedUpdate.Shared = true
		sharedUpdate.AllowSharedUpdates = true

		plugin, _, _ := newMCPToolsTestPlugin(t, []*Installation{owned, sharedRead, sharedUpdate})
		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpGetInstallationToolName, map[string]any{"installation_id": "owned-id", "refresh": false})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output := decodeMCPStructuredOutput[GetInstallationMCPOutput](t, result)
		assert.Equal(t, "owned-id", output.Installation.ID)
		assert.NotEmpty(t, output.Installation.InstallationLogsURL)
		assert.NotEmpty(t, output.Installation.ProvisionerLogsURL)

		result, err = callMCPTool(t, session, mcpGetInstallationToolName, map[string]any{"name": "OWNEDINSTALL", "refresh": false, "include_log_urls": false})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output = decodeMCPStructuredOutput[GetInstallationMCPOutput](t, result)
		assert.Equal(t, "owned-id", output.Installation.ID)
		assert.Empty(t, output.Installation.InstallationLogsURL)

		result, err = callMCPTool(t, session, mcpGetInstallationToolName, map[string]any{"name": "sharedread", "scope": "shared", "refresh": false})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output = decodeMCPStructuredOutput[GetInstallationMCPOutput](t, result)
		assert.Equal(t, "shared-read-id", output.Installation.ID)

		result, err = callMCPTool(t, session, mcpGetInstallationToolName, map[string]any{"name": "sharedupdate", "scope": "updatable", "refresh": false})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output = decodeMCPStructuredOutput[GetInstallationMCPOutput](t, result)
		assert.Equal(t, "shared-update-id", output.Installation.ID)

		result, err = callMCPTool(t, session, mcpGetInstallationToolName, map[string]any{"name": "sharedread", "scope": "updatable", "refresh": false})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "no installation with the name sharedread found")
	})

	t.Run("validates reference and scope errors", func(t *testing.T) {
		plugin, _, _ := newMCPToolsTestPlugin(t, []*Installation{serviceTestInstall("owned-id", "Owned", "owner")})
		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()

		for _, args := range []map[string]any{
			{},
			{"installation_id": "owned-id", "name": "owned"},
		} {
			result, err := callMCPTool(t, session, mcpGetInstallationToolName, args)
			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, mcpToolText(t, result), "must provide exactly one installation id or name")
		}

		result, err := callMCPTool(t, session, mcpGetInstallationToolName, map[string]any{"installation_id": "owned-id", "scope": "bogus"})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "unknown installation scope bogus")

		result, err = callMCPTool(t, session, mcpGetInstallationToolName, map[string]any{"installation_id": "missing-id", "refresh": false})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "no installation with the id missing-id found")
	})

	t.Run("refresh returns refreshed state without cleanup side effects", func(t *testing.T) {
		install := serviceTestInstall("refresh-id", "RefreshMe", "owner")
		plugin, cloudClient, api := newMCPToolsTestPlugin(t, []*Installation{install})
		refreshed := serviceTestInstall("refresh-id", "RefreshMe", "owner")
		refreshed.State = cloud.InstallationStateUpdateInProgress
		cloudClient.mockedCloudInstallationsDTO = serviceDTOs(refreshed)
		api.On("KVCompareAndSet").Unset()

		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()
		result, err := callMCPTool(t, session, mcpGetInstallationToolName, map[string]any{"installation_id": "refresh-id"})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output := decodeMCPStructuredOutput[GetInstallationMCPOutput](t, result)
		assert.Equal(t, cloud.InstallationStateUpdateInProgress, output.Installation.State)
		api.AssertNotCalled(t, "KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything)
	})

	t.Run("refresh resolves deleted install by ID", func(t *testing.T) {
		deleted := serviceTestInstall("deleted-id", "Deleted", "owner")
		plugin, cloudClient, api := newMCPToolsTestPlugin(t, []*Installation{deleted})
		cloudClient.overrideGetInstallationDTO = &cloud.InstallationDTO{Installation: &cloud.Installation{
			ID:      "deleted-id",
			OwnerID: "owner",
			State:   cloud.InstallationStateDeleted,
		}}
		cloudClient.mockedCloudInstallationsDTO = nil
		api.On("KVCompareAndSet").Unset()

		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()
		result, err := callMCPTool(t, session, mcpGetInstallationToolName, map[string]any{"installation_id": "deleted-id"})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output := decodeMCPStructuredOutput[GetInstallationMCPOutput](t, result)
		assert.Equal(t, "deleted-id", output.Installation.ID)
		assert.Equal(t, "owner", output.Installation.OwnerID)
		assert.Equal(t, cloud.InstallationStateDeleted, output.Installation.State)
		assert.Contains(t, output.Installation.Name, "DELETED")
		assert.NotEmpty(t, output.Installation.InstallationLogsURL)
		assert.NotEmpty(t, output.Installation.ProvisionerLogsURL)
		api.AssertNotCalled(t, "KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything)
	})

	t.Run("redacts sensitive output", func(t *testing.T) {
		install := serviceTestInstall("secret-id", "Secret", "owner")
		install.License = "secret-license"
		install.MattermostEnv = cloud.EnvVarMap{"SECRET": cloud.EnvVar{Value: "secret-env-value"}}
		install.PriorityEnv = cloud.EnvVarMap{"PRIORITY_SECRET": cloud.EnvVar{Value: "priority-secret-value"}}
		plugin, _, _ := newMCPToolsTestPlugin(t, []*Installation{install})

		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()
		result, err := callMCPTool(t, session, mcpGetInstallationToolName, map[string]any{"installation_id": "secret-id", "refresh": false})
		require.NoError(t, err)
		require.False(t, result.IsError)
		assertMCPResultRedacted(t, result, "secret-license", "secret-env-value", "priority-secret-value", "MattermostEnv", "PriorityEnv")
	})
}

func newMCPToolsTestPlugin(t *testing.T, installs []*Installation) (*Plugin, *MockClient, *plugintest.API) {
	t.Helper()

	plugin, cloudClient, api := newServiceTestPlugin(t, installs)
	cloudClient.mockedCloudInstallationsDTO = serviceDTOs(installs...)
	require.NoError(t, plugin.ensureMCPServer())

	return plugin, cloudClient, api
}

func connectMCPToolsClient(t *testing.T, plugin *Plugin, userID string) (*mcp.ClientSession, func()) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Mattermost-Plugin-ID", "mattermost-ai")
		if userID != "" {
			r.Header.Set("X-Mattermost-UserID", userID)
		}
		plugin.ServeHTTP(nil, w, r)
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-tools-test", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             server.URL + mcpBasePath,
		DisableStandaloneSSE: true,
	}, nil)
	cancel()
	require.NoError(t, err)

	return session, func() {
		require.NoError(t, session.Close())
		server.Close()
	}
}

func callMCPTool(t *testing.T, session *mcp.ClientSession, name string, arguments any) (*mcp.CallToolResult, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
}

func decodeMCPStructuredOutput[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()

	data, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	var output T
	require.NoError(t, json.Unmarshal(data, &output))
	return output
}

func assertMCPInputSchemaProperties(t *testing.T, tool *mcp.Tool, properties ...string) {
	t.Helper()

	data, err := json.Marshal(tool.InputSchema)
	require.NoError(t, err)
	var schema map[string]any
	require.NoError(t, json.Unmarshal(data, &schema))
	schemaProperties, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "input schema properties missing from %s", tool.Name)
	for _, property := range properties {
		assert.Contains(t, schemaProperties, property)
	}
}

func assertMCPInputSchemaRequired(t *testing.T, tool *mcp.Tool, requiredProperties ...string) {
	t.Helper()

	data, err := json.Marshal(tool.InputSchema)
	require.NoError(t, err)
	var schema map[string]any
	require.NoError(t, json.Unmarshal(data, &schema))
	required, ok := schema["required"].([]any)
	require.True(t, ok, "input schema required fields missing from %s", tool.Name)
	for _, property := range requiredProperties {
		assert.Contains(t, required, property)
	}
}

func assertMCPResultRedacted(t *testing.T, result *mcp.CallToolResult, forbidden ...string) {
	t.Helper()

	data, err := json.Marshal(result)
	require.NoError(t, err)
	for _, value := range forbidden {
		assert.NotContains(t, string(data), value)
	}
}

func mcpToolText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	data, err := json.Marshal(result.Content)
	require.NoError(t, err)
	return string(data)
}

func installationSummaryIDs(summaries []InstallationSummary) []string {
	ids := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		ids = append(ids, summary.ID)
	}
	return ids
}
