package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

func (p *Plugin) getHelp() string {
	help := `Available Commands:

create [name] [flags]
	Creates a Mattermost installation.
	Flags:
%s
	example: /cloud create myinstallation --license e10 --test-data

list
	Lists the Mattermost installations created by you.
%s

import [DNS]
	Imports installation using DNS value.

update [name] [flags]
	Update a Mattermost installation.
	Flags:
%s
	example: /cloud update myinstallation --version 7.8.1

share [name] [flags]
	Share a Mattermost installation with other plugin users.
	Flags:
%s
	example: /cloud share myinstallation --allow-updates=true

unshare [name] [flags]
	Remove the shared setting from an installation that is already shared.

	example: /cloud unshare myinstallation

restart [name]
	Restarts the servers in a Mattermost installation.

hibernate [name]
	Hibernates a Mattermost installation.

wake-up [name]
	Wakes a Mattermost installation up.

mmcli [name] [mattermost-subcommand]
	Runs Mattermost CLI commands on an installation.

	example: /cloud mmcli myinstallation version
		(equivalent to running 'mattermost version' on myinstallation)

mmctl [name] [mmctl-subcommand]
	Runs mmctl commands on an installation.

	example: /cloud mmctl myinstallation config get ServiceSettings.SiteURL
		(equivalent to running 'mmctl config get ServiceSettings.SiteURL' on myinstallation)

delete [name]
	Deletes a Mattermost installation.

info
	Shows basic cloud plugin information.
`
	return codeBlock(fmt.Sprintf(
		help,
		p.getCreateFlagSet().FlagUsages(),
		getListFlagSet().FlagUsages(),
		getUpdateFlagSet().FlagUsages(),
		getShareFlagSet().FlagUsages(),
	))
}

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "cloud",
		DisplayName:      "Mattermost Private Cloud",
		Description:      "This command allows spinning up and down Mattermost installations using Mattermost Private Cloud.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: create, list, update, mmcli, mmctl, delete, share, unshare, restart, hibernate, wake-up, info, import",
		AutoCompleteHint: "[command]",
		AutocompleteData: &model.AutocompleteData{
			Trigger: "cloud",
			RoleID: "system_user",
			SubCommands: []*model.AutocompleteData{
				{
					Trigger: "create",
					HelpText: "Creates a Mattermost installation",
					Arguments: []*model.AutocompleteArg{
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "name",
								Pattern: "^[a-zA-Z0-9-]+$",
							},
							HelpText: "Name of the installation",
							Required: true,
						},
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "license",
							},
							Name: "license",
							HelpText: "The Mattermost license to use. Can be 'enterprise', 'professional', 'e20', 'e10', or 'te' (default \"enterprise\")",
							Required: false,
						},
						{
							Name: "test-data",
							HelpText: "Set to pre-load the server with test data",
							Required: false,
						},
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "affinity",
								Pattern: "^(isolated|multitenant)$",
							},
							Name: "affinity",
							HelpText: "Whether the installation is isolated in it's own cluster or shares ones. Can be 'isolated' or 'multitenant' (default \"multitenant\")",
							Required: false,
						},
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "database",
							},
							Name: "database",
							HelpText: "Specify the backing database. Can be 'aws-multitenant-rds-postgres-pgbouncer' (RDS Postgres with pgbouncer proxy connections), 'aws-rds' (RDS MySQL). (default \"aws-multitenant-rds-postgres-pgbouncer\")",
							Required: false,
						},
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "env",
							},
							Name: "env",
							HelpText: "Environment variables in form: ENV1=test,ENV2=test",
							Required: false,
						},
						{
							Name: "image",
							HelpText: "Docker image repository, can be mattermost/mattermost-enterprise-edition, mattermost/mm-ee-cloud, mattermost/mm-te, mattermost/mattermost-team-edition, mattermostdevelopment/mm-ee-test, mattermostdevelopment/mm-te-test, mattermostdevelopment/mattermost-enterprise-edition, mattermostdevelopment/mattermost-team-edition",
							Required: false,
						},
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "filestore",
							},
							Name: "filestore",
							HelpText: "Specify the backing file store. Can be 'bifrost' (S3 Shared Bucket), 'aws-multitenant-s3' (S3 Shared Bucket), 'aws-s3' (S3 Bucket). (default \"bifrost\")",
							Required: false,
						},
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "image",
							},
							Name: "image",
							HelpText: "Docker image repository. Can be mattermost/mattermost-enterprise-edition, mattermost/mm-ee-cloud, mattermost/mm-te, mattermost/mattermost-team-edition, mattermostdevelopment/mm-ee-test, mattermostdevelopment/mm-te-test, mattermostdevelopment/mattermost-enterprise-edition, mattermostdevelopment/mattermost-team-edition (default \"mattermost/mattermost-enterprise-edition\")",
							Required: false,
						},
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "size",
							},
							Name: "size",
							HelpText: "Size of the Mattermost installation e.g. 'miniSingleton' or 'miniHA' (default \"miniSingleton\")",
							Required: false,
						},
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "version",
							},
							Name: "version",
							HelpText: "Mattermost version to run, e.g. '9.1.0' (default \"latest\")",
							Required: false,
						},
					},
				},
				{
					Trigger: "list",
					HelpText: "Lists your Mattermost installations",
					Arguments: []*model.AutocompleteArg{
						{
							Name: "shared-installations",
							HelpText: "Lists shared installations instead of personal ones",
							Required: false,
						},
					},
				},
				{
					Trigger: "update",
					HelpText: "Update a Mattermost installation",
					Arguments: []*model.AutocompleteArg{
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "name",
								Pattern: "^[a-zA-Z0-9-]+$",
							},
							HelpText: "Name of the installation to update",
							Required: true,
						},
						{
							Name: "version",
							HelpText: "Mattermost version to run, e.g. '9.1.0'",
							Required: false,
						},
						{
							Name: "license",
							HelpText: "The enterprise license to use. Can be 'enterprise', 'professional', 'e20', 'e10', or 'te'",
							Required: false,
						},
						{
							Name: "clear-env",
							HelpText: "List of custom environment variables to erase, for example: ENV1,ENV2",
							Required: false,
						},
						{
							Name: "env",
							HelpText: "Environment variables in form: ENV1=test,ENV2=test",
							Required: false,
						},
						{
							Name: "shared-installation",
							HelpText: "Set this to true when attempting to update a shared installation",
							Required: false,
						},
						{
							Name: "size",
							HelpText: "Size of the Mattermost installation e.g. 'miniSingleton' or 'miniHA'",
							Required: false,
						},
					},
				},
				{
					Trigger: "share",
					HelpText: "Share a Mattermost installation",
					Arguments: []*model.AutocompleteArg{
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "name",
								Pattern: "^[a-zA-Z0-9-]+$",
							},
							HelpText: "Name of the installation to share",
							Required: true,
						},
						{
							Name: "allow-updates",
							HelpText: "Allow other plugin users to update the installation configuration",
							Required: false,
						},
					},
				},
				{
					Trigger: "unshare",
					HelpText: "Remove sharing from an installation",
					Arguments: []*model.AutocompleteArg{
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "name",
								Pattern: "^[a-zA-Z0-9-]+$",
							},
							HelpText: "Name of the installation to unshare",
							Required: true,
						},
					},
				},
				{
					Trigger: "restart",
					HelpText: "Restart a Mattermost installation",
					Arguments: []*model.AutocompleteArg{
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "name",
								Pattern: "^[a-zA-Z0-9-]+$",
							},
							HelpText: "Name of the installation to restart",
							Required: true,
						},
					},
				},
				{
					Trigger: "hibernate",
					HelpText: "Hibernate a Mattermost installation",
					Arguments: []*model.AutocompleteArg{
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "name",
								Pattern: "^[a-zA-Z0-9-]+$",
							},
							HelpText: "Name of the installation to hibernate",
							Required: true,
						},
					},
				},
				{
					Trigger: "wake-up",
					HelpText: "Wake up a hibernated installation",
					Arguments: []*model.AutocompleteArg{
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "name",
								Pattern: "^[a-zA-Z0-9-]+$",
							},
							HelpText: "Name of the installation to wake up",
							Required: true,
						},
					},
				},
				{
					Trigger: "delete",
					HelpText: "Delete a Mattermost installation",
					Arguments: []*model.AutocompleteArg{
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "name",
								Pattern: "^[a-zA-Z0-9-]+$",
							},
							HelpText: "Name of the installation to delete",
							Required: true,
						},
					},
				},
				{
					Trigger: "info",
					HelpText: "Show cloud plugin information",
				},
				{
					Trigger: "import",
					HelpText: "Import an existing installation",
					Arguments: []*model.AutocompleteArg{
						{
							Name: "DNS",
							HelpText: "DNS value of the installation to import",
							Required: true,
						},
					},
				},
				{
					Trigger: "mmcli",
					HelpText: "Runs Mattermost CLI commands on an installation",
					Arguments: []*model.AutocompleteArg{
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "name",
								Pattern: "^[a-zA-Z0-9-]+$",
							},
							HelpText: "Name of the installation to run CLI commands on",
							Required: true,
						},
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "mattermost-subcommand",
							},
							HelpText: "The Mattermost CLI subcommand to run",
							Required: true,
						},
					},
				},
				{
					Trigger: "mmctl",
					HelpText: "Runs mmctl commands on an installation",
					Arguments: []*model.AutocompleteArg{
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "name",
								Pattern: "^[a-zA-Z0-9-]+$",
							},
							HelpText: "Name of the installation to run mmctl commands on",
							Required: true,
						},
						{
							Type: model.AutocompleteArgTypeText,
							Data: &model.AutocompleteTextArg{
								Hint: "mmctl-subcommand",
							},
							HelpText: "The mmctl subcommand to run",
							Required: true,
						},
					},
				},
			},
		},
	}
}

func getCommandResponse(responseType, text string, args *model.CommandArgs) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: responseType,
		Text:         "Command invoked: `" + args.Command + "`\n\n" + text,
		Username:     "cloud",
		IconURL:      fmt.Sprintf("/plugins/%s/profile.png", manifest.ID),
	}
}

// ExecuteCommand executes a given command and returns a command response.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if !p.authorizedPluginUser(args.UserId) {
		return getCommandResponse(model.CommandResponseTypeEphemeral, "Permission denied. Please talk to your system administrator to get access.", args), nil
	}

	stringArgs := strings.Split(args.Command, " ")

	if len(stringArgs) < 2 {
		return getCommandResponse(model.CommandResponseTypeEphemeral, p.getHelp(), args), nil
	}

	command := stringArgs[1]

	var handler func([]string, *model.CommandArgs) (*model.CommandResponse, bool, error)

	switch command {
	case "create":
		handler = p.runCreateCommand
	case "mmcli":
		handler = p.runMattermostCLICommand
	case "mmctl":
		handler = p.runMmctlCommand
	case "list":
		handler = p.runListCommand
	case "upgrade":
		handler = p.runUpgradeHelperCommand
	case "update":
		handler = p.runUpdateCommand
	case "restart":
		handler = p.runRestartCommand
	case "hibernate":
		handler = p.runHibernateCommand
	case "wake-up":
		handler = p.runWakeUpCommand
	case "delete":
		handler = p.runDeleteCommand
	case "status":
		handler = p.runStatusCommand
	case "info":
		handler = p.runInfoCommand
	case "import":
		handler = p.runImportCommand
	case "share":
		handler = p.runShareInstallationCommand
	case "unshare":
		handler = p.runUnshareInstallationCommand
	case "deletion-lock":
		handler = p.runDeletionLockCommand
	case "deletion-unlock":
		handler = p.runDeletionUnlockCommand
	}

	if handler == nil {
		return getCommandResponse(model.CommandResponseTypeEphemeral, p.getHelp(), args), nil
	}

	resp, isUserError, err := handler(stringArgs[2:], args)

	if err != nil {
		if isUserError {
			return getCommandResponse(model.CommandResponseTypeEphemeral, fmt.Sprintf("__Error: %s__\n\nRun `/cloud help` for usage instructions.", err.Error()), args), nil
		}
		p.API.LogError(err.Error())
		return getCommandResponse(model.CommandResponseTypeEphemeral, "An unknown error occurred. Please talk to your resident cloud team for help.", args), nil
	}

	return resp, nil
}

func (p *Plugin) runInfoCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	resp := fmt.Sprintf("Mattermost Cloud plugin version: %s, "+
		"[%s](https://github.com/mattermost/mattermost-plugin-cloud/commit/%s), built %s\n",
		manifest.Version, BuildHashShort, BuildHash, BuildDate)

	return getCommandResponse(model.CommandResponseTypeEphemeral, resp, extra), false, nil
}

// Helper function to prevent user confusion.
// TODO: remove this at a later date.
func (p *Plugin) runUpgradeHelperCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	return getCommandResponse(
		model.CommandResponseTypeEphemeral,
		"`/cloud upgrade` has been deprecated. Use `/cloud update` instead.",
		extra,
	), false, nil
}

// authorizedPluginUser returns if a given userID is authorized to use the plugin
// commands with the current plugin configuration.
func (p *Plugin) authorizedPluginUser(userID string) bool {
	config := p.getConfiguration()

	if config.AllowedEmailDomain == "" {
		return true
	}

	user, err := p.API.GetUser(userID)
	if err != nil {
		p.API.LogError("Failed to get user", "error", err)
		return false
	}
	if !strings.HasSuffix(user.Email, "@"+config.AllowedEmailDomain) {
		return false
	}

	return true
}
