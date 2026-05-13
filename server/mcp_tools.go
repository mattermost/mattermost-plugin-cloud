package main

import (
	"context"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-plugin-agents/external/pluginmcp"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

type ListInstallationsMCPInput struct {
	Scope          string `json:"scope,omitempty" jsonschema:"Visibility scope: mine, shared, or updatable. Defaults to mine."`
	Refresh        *bool  `json:"refresh,omitempty" jsonschema:"When true, refresh installation state from the provisioner before returning results. Defaults to true."`
	IncludeLogURLs *bool  `json:"include_log_urls,omitempty" jsonschema:"When true, include installation and provisioner log URLs. Defaults to false."`
}

type ListInstallationsMCPOutput struct {
	Installations []InstallationSummary `json:"installations" jsonschema:"Installations visible to the caller"`
	Count         int                   `json:"count" jsonschema:"Number of installations returned"`
}

type GetInstallationMCPInput struct {
	InstallationID string `json:"installation_id,omitempty" jsonschema:"Stable installation ID. Provide exactly one of installation_id or name."`
	Name           string `json:"name,omitempty" jsonschema:"Human-friendly installation name. Provide exactly one of installation_id or name."`
	Scope          string `json:"scope,omitempty" jsonschema:"Visibility scope: mine, shared, or updatable. Defaults to mine."`
	Refresh        *bool  `json:"refresh,omitempty" jsonschema:"When true, refresh installation state from the provisioner before returning results. Defaults to true."`
	IncludeLogURLs *bool  `json:"include_log_urls,omitempty" jsonschema:"When true, include installation and provisioner log URLs. Defaults to true."`
}

type GetInstallationMCPOutput struct {
	Installation InstallationSummary `json:"installation" jsonschema:"Installation detail visible to the caller"`
}

type CreateInstallationMCPInput struct {
	Name      string            `json:"name" jsonschema:"Required installation name."`
	Version   string            `json:"version,omitempty" jsonschema:"Mattermost version tag. Defaults to latest."`
	Size      string            `json:"size,omitempty" jsonschema:"Installation size, such as miniSingleton or miniHA. Defaults to miniSingleton."`
	License   string            `json:"license,omitempty" jsonschema:"License option: enterprise, enterprise-advanced, professional, e20, e10, or te. Defaults to enterprise."`
	Affinity  string            `json:"affinity,omitempty" jsonschema:"Cluster affinity, isolated or multitenant. Defaults to multitenant."`
	Database  string            `json:"database,omitempty" jsonschema:"Database backend. Defaults to plugin configuration."`
	Filestore string            `json:"filestore,omitempty" jsonschema:"Filestore backend. Defaults to plugin configuration."`
	Image     string            `json:"image,omitempty" jsonschema:"Docker image repository from the configured allowlist."`
	TestData  bool              `json:"test_data,omitempty" jsonschema:"Whether to pre-load test data."`
	Env       map[string]string `json:"env,omitempty" jsonschema:"Priority environment variables to set during creation. Values are never returned."`
}

type UpdateInstallationMCPInput struct {
	InstallationID string            `json:"installation_id,omitempty" jsonschema:"Stable installation ID. Provide exactly one of installation_id or name."`
	Name           string            `json:"name,omitempty" jsonschema:"Human-friendly installation name. Provide exactly one of installation_id or name."`
	Scope          string            `json:"scope,omitempty" jsonschema:"Update scope: mine or updatable. Defaults to mine."`
	Version        string            `json:"version,omitempty" jsonschema:"Mattermost version tag."`
	Image          string            `json:"image,omitempty" jsonschema:"Docker image repository from the configured allowlist."`
	License        string            `json:"license,omitempty" jsonschema:"License option: enterprise, enterprise-advanced, professional, e20, e10, or te."`
	Size           string            `json:"size,omitempty" jsonschema:"Installation size, such as miniSingleton or miniHA."`
	SetEnv         map[string]string `json:"set_env,omitempty" jsonschema:"Environment variables to set. Values are never returned."`
	ClearEnv       []string          `json:"clear_env,omitempty" jsonschema:"Environment variable keys to clear."`
}

type RestartInstallationMCPInput struct {
	InstallationID string `json:"installation_id,omitempty" jsonschema:"Stable installation ID. Provide exactly one of installation_id or name."`
	Name           string `json:"name,omitempty" jsonschema:"Human-friendly installation name. Provide exactly one of installation_id or name."`
	Scope          string `json:"scope,omitempty" jsonschema:"Restart scope: mine or updatable. Defaults to mine."`
}

type InstallationRefMCPInput struct {
	InstallationID string `json:"installation_id,omitempty" jsonschema:"Stable installation ID. Provide exactly one of installation_id or name."`
	Name           string `json:"name,omitempty" jsonschema:"Human-friendly installation name. Provide exactly one of installation_id or name."`
}

type SetInstallationSharingMCPInput struct {
	InstallationID string `json:"installation_id,omitempty" jsonschema:"Stable installation ID. Provide exactly one of installation_id or name."`
	Name           string `json:"name,omitempty" jsonschema:"Human-friendly installation name. Provide exactly one of installation_id or name."`
	Shared         bool   `json:"shared" jsonschema:"Whether the installation should be shared with other authorized plugin users."`
	AllowUpdates   bool   `json:"allow_updates,omitempty" jsonschema:"Whether shared users may update and restart the installation. Ignored when shared is false."`
}

type SetDeletionLockMCPInput struct {
	InstallationID string `json:"installation_id,omitempty" jsonschema:"Stable installation ID. Provide exactly one of installation_id or name."`
	Name           string `json:"name,omitempty" jsonschema:"Human-friendly installation name. Provide exactly one of installation_id or name."`
	Locked         bool   `json:"locked" jsonschema:"Whether deletion should be locked."`
}

type DeleteInstallationMCPInput struct {
	InstallationID string `json:"installation_id,omitempty" jsonschema:"Stable installation ID. Provide exactly one of installation_id or name."`
	Name           string `json:"name,omitempty" jsonschema:"Human-friendly installation name. Provide exactly one of installation_id or name."`
	ConfirmName    string `json:"confirm_name" jsonschema:"Required installation name confirmation. Must match the target installation name."`
}

type CloudStatusMCPInput struct {
	IncludeClusters bool `json:"include_clusters,omitempty" jsonschema:"When true, include cluster status summaries. Defaults to false."`
}

type CloudStatusMCPOutput struct {
	Installations     []CloudStatusInstallationSummary `json:"installations" jsonschema:"Global Cloud installation summaries"`
	InstallationCount int                              `json:"installation_count" jsonschema:"Number of installations returned"`
	Clusters          []CloudStatusClusterSummary      `json:"clusters,omitempty" jsonschema:"Global Cloud cluster summaries"`
	ClusterCount      int                              `json:"cluster_count,omitempty" jsonschema:"Number of clusters returned"`
}

type CloudStatusInstallationSummary struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name,omitempty"`
	DNS                   string   `json:"dns,omitempty"`
	State                 string   `json:"state"`
	OwnerID               string   `json:"owner_id"`
	Version               string   `json:"version,omitempty"`
	Size                  string   `json:"size,omitempty"`
	Database              string   `json:"database,omitempty"`
	Filestore             string   `json:"filestore,omitempty"`
	CreateAt              int64    `json:"create_at,omitempty"`
	DeleteAt              int64    `json:"delete_at,omitempty"`
	DeletionLocked        bool     `json:"deletion_locked"`
	ScheduledDeletionTime int64    `json:"scheduled_deletion_time,omitempty"`
	GroupID               string   `json:"group_id,omitempty"`
	ClusterIDs            []string `json:"cluster_ids,omitempty"`
}

type CloudStatusClusterSummary struct {
	ID                 string `json:"id"`
	Name               string `json:"name,omitempty"`
	State              string `json:"state"`
	Provider           string `json:"provider,omitempty"`
	Provisioner        string `json:"provisioner,omitempty"`
	AllowInstallations bool   `json:"allow_installations"`
	CreateAt           int64  `json:"create_at,omitempty"`
	DeleteAt           int64  `json:"delete_at,omitempty"`
	APISecurityLock    bool   `json:"api_security_lock"`
}

type InstallationActionMCPOutput struct {
	Result InstallationActionResult `json:"result" jsonschema:"Lifecycle action result"`
}

func (p *Plugin) registerMCPTools(server *pluginmcp.Server) {
	readOnly := true
	notReadOnly := false
	notDestructive := false
	destructive := true
	closedWorld := false

	pluginmcp.AddTool(server, &mcp.Tool{
		Name:        "list_installations",
		Title:       "List Cloud Installations",
		Description: "List Cloud-managed Mattermost installations visible to the calling user.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    readOnly,
			DestructiveHint: &notDestructive,
			OpenWorldHint:   &closedWorld,
			Title:           "List Cloud Installations",
		},
	}, p.listInstallationsMCPHandler)

	pluginmcp.AddTool(server, &mcp.Tool{
		Name:        "get_installation",
		Title:       "Get Cloud Installation",
		Description: "Get one Cloud-managed Mattermost installation visible to the calling user by ID or name.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    readOnly,
			DestructiveHint: &notDestructive,
			OpenWorldHint:   &closedWorld,
			Title:           "Get Cloud Installation",
		},
	}, p.getInstallationMCPHandler)

	pluginmcp.AddTool(server, &mcp.Tool{
		Name:        "create_installation",
		Title:       "Create Cloud Installation",
		Description: "Create a new Cloud-managed Mattermost installation owned by the calling user.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    notReadOnly,
			DestructiveHint: &notDestructive,
			OpenWorldHint:   &closedWorld,
			Title:           "Create Cloud Installation",
		},
	}, p.createInstallationMCPHandler)

	pluginmcp.AddTool(server, &mcp.Tool{
		Name:        "update_installation",
		Title:       "Update Cloud Installation",
		Description: "Update an owned Cloud installation or an explicitly updatable shared installation.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    notReadOnly,
			DestructiveHint: &notDestructive,
			OpenWorldHint:   &closedWorld,
			Title:           "Update Cloud Installation",
		},
	}, p.updateInstallationMCPHandler)

	pluginmcp.AddTool(server, &mcp.Tool{
		Name:        "restart_installation",
		Title:       "Restart Cloud Installation",
		Description: "Restart an owned Cloud installation or an explicitly updatable shared installation.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    notReadOnly,
			DestructiveHint: &notDestructive,
			OpenWorldHint:   &closedWorld,
			Title:           "Restart Cloud Installation",
		},
	}, p.restartInstallationMCPHandler)

	pluginmcp.AddTool(server, &mcp.Tool{
		Name:        "hibernate_installation",
		Title:       "Hibernate Cloud Installation",
		Description: "Hibernate an owned stable Cloud installation.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    notReadOnly,
			DestructiveHint: &notDestructive,
			OpenWorldHint:   &closedWorld,
			Title:           "Hibernate Cloud Installation",
		},
	}, p.hibernateInstallationMCPHandler)

	pluginmcp.AddTool(server, &mcp.Tool{
		Name:        "wake_installation",
		Title:       "Wake Cloud Installation",
		Description: "Wake an owned hibernating Cloud installation.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    notReadOnly,
			DestructiveHint: &notDestructive,
			OpenWorldHint:   &closedWorld,
			Title:           "Wake Cloud Installation",
		},
	}, p.wakeInstallationMCPHandler)

	pluginmcp.AddTool(server, &mcp.Tool{
		Name:        "set_installation_sharing",
		Title:       "Set Cloud Installation Sharing",
		Description: "Set sharing and shared-update access for an owned Cloud installation.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    notReadOnly,
			DestructiveHint: &notDestructive,
			OpenWorldHint:   &closedWorld,
			Title:           "Set Cloud Installation Sharing",
		},
	}, p.setInstallationSharingMCPHandler)

	pluginmcp.AddTool(server, &mcp.Tool{
		Name:        "set_deletion_lock",
		Title:       "Set Cloud Installation Deletion Lock",
		Description: "Lock or unlock deletion for an owned Cloud installation.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    notReadOnly,
			DestructiveHint: &notDestructive,
			OpenWorldHint:   &closedWorld,
			Title:           "Set Cloud Installation Deletion Lock",
		},
	}, p.setDeletionLockMCPHandler)

	pluginmcp.AddTool(server, &mcp.Tool{
		Name:        "delete_installation",
		Title:       "Delete Cloud Installation",
		Description: "Delete an owned Cloud installation after explicit name confirmation.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    notReadOnly,
			DestructiveHint: &destructive,
			OpenWorldHint:   &closedWorld,
			Title:           "Delete Cloud Installation",
		},
	}, p.deleteInstallationMCPHandler)

	pluginmcp.AddTool(server, &mcp.Tool{
		Name:        "cloud_status",
		Title:       "Get Cloud Status",
		Description: "Get global Cloud installation status, optionally including clusters. Requires Mattermost system admin permission.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    readOnly,
			DestructiveHint: &notDestructive,
			OpenWorldHint:   &closedWorld,
			Title:           "Get Cloud Status",
		},
	}, p.cloudStatusMCPHandler)
}

func (p *Plugin) listInstallationsMCPHandler(ctx context.Context, _ *mcp.CallToolRequest, input ListInstallationsMCPInput) (*mcp.CallToolResult, ListInstallationsMCPOutput, error) {
	userID, err := p.requireMCPUser(ctx)
	if err != nil {
		return nil, ListInstallationsMCPOutput{}, err
	}

	scope := mcpScope(input.Scope)
	if err = validateInstallationScope(scope); err != nil {
		return nil, ListInstallationsMCPOutput{}, err
	}

	includeLogURLs := boolDefault(input.IncludeLogURLs, false)
	installs, err := p.listInstallationsForUser(userID, ListInstallationsInput{
		Scope:          scope,
		Refresh:        boolDefault(input.Refresh, true),
		IncludeLogURLs: includeLogURLs,
	})
	if err != nil {
		return nil, ListInstallationsMCPOutput{}, err
	}

	output := ListInstallationsMCPOutput{
		Installations: make([]InstallationSummary, 0, len(installs)),
	}
	for _, install := range installs {
		summary, summaryErr := installationSummary(install, includeLogURLs)
		if summaryErr != nil {
			return nil, ListInstallationsMCPOutput{}, summaryErr
		}
		output.Installations = append(output.Installations, summary)
	}
	output.Count = len(output.Installations)

	return nil, output, nil
}

func (p *Plugin) getInstallationMCPHandler(ctx context.Context, _ *mcp.CallToolRequest, input GetInstallationMCPInput) (*mcp.CallToolResult, GetInstallationMCPOutput, error) {
	userID, err := p.requireMCPUser(ctx)
	if err != nil {
		return nil, GetInstallationMCPOutput{}, err
	}

	ref := mcpRef(input.InstallationID, input.Name)
	if err = ref.validate(); err != nil {
		return nil, GetInstallationMCPOutput{}, err
	}

	scope := mcpScope(input.Scope)
	if err = validateInstallationScope(scope); err != nil {
		return nil, GetInstallationMCPOutput{}, err
	}

	var install *Installation
	if boolDefault(input.Refresh, true) {
		installs, listErr := p.listInstallationsForUser(userID, ListInstallationsInput{Scope: scope, Refresh: true})
		if listErr != nil {
			return nil, GetInstallationMCPOutput{}, listErr
		}
		install, err = findInstallationInSlice(userID, ref, scope, installs)
	} else {
		install, err = p.findInstallationForUser(userID, ref, scope)
	}
	if err != nil {
		return nil, GetInstallationMCPOutput{}, err
	}

	summary, err := installationSummary(install, boolDefault(input.IncludeLogURLs, true))
	if err != nil {
		return nil, GetInstallationMCPOutput{}, err
	}

	return nil, GetInstallationMCPOutput{Installation: summary}, nil
}

func (p *Plugin) createInstallationMCPHandler(ctx context.Context, _ *mcp.CallToolRequest, input CreateInstallationMCPInput) (*mcp.CallToolResult, InstallationActionMCPOutput, error) {
	userID, err := p.requireMCPUser(ctx)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	install, err := p.createInstallationForUser(userID, CreateInstallationInput(input))
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	summary, err := installationSummary(install, false)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	return nil, mcpActionOutput(InstallationActionResult{
		Installation: summary,
		Status:       "creation_requested",
		Message:      "Installation creation requested. Use get_installation to poll status.",
	}), nil
}

func (p *Plugin) updateInstallationMCPHandler(ctx context.Context, _ *mcp.CallToolRequest, input UpdateInstallationMCPInput) (*mcp.CallToolResult, InstallationActionMCPOutput, error) {
	userID, err := p.requireMCPUser(ctx)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	ref, err := requireMCPRef(input.InstallationID, input.Name)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}
	scope := mcpScope(input.Scope)
	if err = validateMutationScope(scope); err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	result, err := p.updateInstallationForUser(userID, ref, UpdateInstallationInput{
		Version:  input.Version,
		License:  input.License,
		Size:     input.Size,
		Image:    input.Image,
		SetEnv:   input.SetEnv,
		ClearEnv: input.ClearEnv,
	}, scope)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	return nil, mcpActionOutput(result), nil
}

func (p *Plugin) restartInstallationMCPHandler(ctx context.Context, _ *mcp.CallToolRequest, input RestartInstallationMCPInput) (*mcp.CallToolResult, InstallationActionMCPOutput, error) {
	userID, err := p.requireMCPUser(ctx)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	ref, err := requireMCPRef(input.InstallationID, input.Name)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}
	scope := mcpScope(input.Scope)
	if err = validateMutationScope(scope); err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	result, err := p.restartInstallationForUser(userID, ref, scope)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	return nil, mcpActionOutput(result), nil
}

func (p *Plugin) hibernateInstallationMCPHandler(ctx context.Context, _ *mcp.CallToolRequest, input InstallationRefMCPInput) (*mcp.CallToolResult, InstallationActionMCPOutput, error) {
	userID, err := p.requireMCPUser(ctx)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	ref, err := requireMCPRef(input.InstallationID, input.Name)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	result, err := p.hibernateInstallationForUser(userID, ref)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	return nil, mcpActionOutput(result), nil
}

func (p *Plugin) wakeInstallationMCPHandler(ctx context.Context, _ *mcp.CallToolRequest, input InstallationRefMCPInput) (*mcp.CallToolResult, InstallationActionMCPOutput, error) {
	userID, err := p.requireMCPUser(ctx)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	ref, err := requireMCPRef(input.InstallationID, input.Name)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	result, err := p.wakeInstallationForUser(userID, ref)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	return nil, mcpActionOutput(result), nil
}

func (p *Plugin) setInstallationSharingMCPHandler(ctx context.Context, _ *mcp.CallToolRequest, input SetInstallationSharingMCPInput) (*mcp.CallToolResult, InstallationActionMCPOutput, error) {
	userID, err := p.requireMCPUser(ctx)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	ref, err := requireMCPRef(input.InstallationID, input.Name)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	result, err := p.setInstallationSharingForUser(userID, ref, input.Shared, input.AllowUpdates)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	return nil, mcpActionOutput(result), nil
}

func (p *Plugin) setDeletionLockMCPHandler(ctx context.Context, _ *mcp.CallToolRequest, input SetDeletionLockMCPInput) (*mcp.CallToolResult, InstallationActionMCPOutput, error) {
	userID, err := p.requireMCPUser(ctx)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	ref, err := requireMCPRef(input.InstallationID, input.Name)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	result, err := p.setDeletionLockForUser(userID, ref, input.Locked)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	return nil, mcpActionOutput(result), nil
}

func (p *Plugin) deleteInstallationMCPHandler(ctx context.Context, _ *mcp.CallToolRequest, input DeleteInstallationMCPInput) (*mcp.CallToolResult, InstallationActionMCPOutput, error) {
	userID, err := p.requireMCPUser(ctx)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	ref, err := requireMCPRef(input.InstallationID, input.Name)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}
	if input.ConfirmName == "" {
		return nil, InstallationActionMCPOutput{}, errors.New("confirm_name is required")
	}

	result, err := p.deleteInstallationForUser(userID, ref, input.ConfirmName)
	if err != nil {
		return nil, InstallationActionMCPOutput{}, err
	}

	return nil, mcpActionOutput(result), nil
}

func (p *Plugin) cloudStatusMCPHandler(ctx context.Context, _ *mcp.CallToolRequest, input CloudStatusMCPInput) (*mcp.CallToolResult, CloudStatusMCPOutput, error) {
	if _, err := p.requireAdminMCPUser(ctx); err != nil {
		return nil, CloudStatusMCPOutput{}, err
	}

	installations, err := p.cloudClient.GetInstallations(&cloud.GetInstallationsRequest{
		Paging: cloud.AllPagesNotDeleted(),
	})
	if err != nil {
		return nil, CloudStatusMCPOutput{}, err
	}

	output := CloudStatusMCPOutput{
		Installations: make([]CloudStatusInstallationSummary, 0, len(installations)),
	}
	for _, installation := range installations {
		output.Installations = append(output.Installations, cloudStatusInstallationSummary(installation))
	}
	output.InstallationCount = len(output.Installations)

	if !input.IncludeClusters {
		return nil, output, nil
	}

	clusters, err := p.cloudClient.GetClusters(&cloud.GetClustersRequest{
		Paging: cloud.AllPagesNotDeleted(),
	})
	if err != nil {
		return nil, CloudStatusMCPOutput{}, err
	}

	output.Clusters = make([]CloudStatusClusterSummary, 0, len(clusters))
	for _, cluster := range clusters {
		output.Clusters = append(output.Clusters, cloudStatusClusterSummary(cluster))
	}
	output.ClusterCount = len(output.Clusters)

	return nil, output, nil
}

func (p *Plugin) requireMCPUser(ctx context.Context) (string, error) {
	userID := pluginmcp.GetUserID(ctx)
	if userID == "" {
		return "", errors.New("no Mattermost user ID in MCP tool context")
	}
	if !p.authorizedPluginUser(userID) {
		return "", errors.New("user is not authorized to use the Cloud plugin")
	}
	return userID, nil
}

func (p *Plugin) requireAdminMCPUser(ctx context.Context) (string, error) {
	userID, err := p.requireMCPUser(ctx)
	if err != nil {
		return "", err
	}
	if !p.API.HasPermissionTo(userID, model.PermissionManageSystem) {
		return "", errors.New("user is not authorized to use Cloud admin tools")
	}
	return userID, nil
}

func boolDefault(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

func mcpScope(value string) InstallationScope {
	return defaultInstallationScope(InstallationScope(value))
}

func mcpRef(id, name string) InstallationRef {
	return InstallationRef{ID: id, Name: name}
}

func mcpActionOutput(result InstallationActionResult) InstallationActionMCPOutput {
	return InstallationActionMCPOutput{Result: result}
}

func validateMutationScope(scope InstallationScope) error {
	if err := validateInstallationScope(scope); err != nil {
		return err
	}
	if scope == InstallationScopeShared {
		return errors.New("shared scope is read-only for this tool")
	}
	return nil
}

func requireMCPRef(id, name string) (InstallationRef, error) {
	ref := mcpRef(id, name)
	if err := ref.validate(); err != nil {
		return InstallationRef{}, err
	}
	return ref, nil
}

func cloudStatusInstallationSummary(dto *cloud.InstallationDTO) CloudStatusInstallationSummary {
	if dto == nil || dto.Installation == nil {
		return CloudStatusInstallationSummary{}
	}

	clusterIDs := make([]string, 0, len(dto.ClusterIDs))
	for _, clusterID := range dto.ClusterIDs {
		if clusterID != nil {
			clusterIDs = append(clusterIDs, *clusterID)
		}
	}

	return CloudStatusInstallationSummary{
		ID:                    dto.ID,
		Name:                  dto.Name,
		DNS:                   firstDNSRecord(dto),
		State:                 dto.State,
		OwnerID:               dto.OwnerID,
		Version:               dto.Version,
		Size:                  dto.Size,
		Database:              dto.Database,
		Filestore:             dto.Filestore,
		CreateAt:              dto.CreateAt,
		DeleteAt:              dto.DeleteAt,
		DeletionLocked:        dto.DeletionLocked,
		ScheduledDeletionTime: dto.ScheduledDeletionTime,
		GroupID:               stringValue(dto.GroupID),
		ClusterIDs:            clusterIDs,
	}
}

func cloudStatusClusterSummary(dto *cloud.ClusterDTO) CloudStatusClusterSummary {
	if dto == nil || dto.Cluster == nil {
		return CloudStatusClusterSummary{}
	}

	return CloudStatusClusterSummary{
		ID:                 dto.ID,
		Name:               dto.Name,
		State:              dto.State,
		Provider:           dto.Provider,
		Provisioner:        dto.Provisioner,
		AllowInstallations: dto.AllowInstallations,
		CreateAt:           dto.CreateAt,
		DeleteAt:           dto.DeleteAt,
		APISecurityLock:    dto.APISecurityLock,
	}
}

func firstDNSRecord(dto *cloud.InstallationDTO) string {
	if dto == nil || len(dto.DNSRecords) == 0 || dto.DNSRecords[0] == nil {
		return ""
	}
	return dto.DNSRecords[0].DomainName
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
