package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

func getHelp() string {
	help := `Available Commands:

create [name] [flags]
	Creates a Mattermost installation.
	Flags:
%s
	example: /cloud create myinstallation --license e10 --test-data

list
	Lists the Mattermost installations created by you.

upgrade [name] [flags]
	Upgades a Mattermost installaton.
	Flags:
%s
	example: /cloud upgrade myinstallation --version 5.13.2

mmcli [name] [mattermost-subcommand]
	Runs Mattermost CLI commands on an installation.

	example: /cloud mmcli myinstallation version
		(equivalent to running 'mattermost version' on myinstallation)

delete [name]
	Deletes a Mattermost installation.

info
	Shows basic cloud plugin information.
`
	return codeBlock(fmt.Sprintf(
		help,
		getCreateFlagSet().FlagUsages(),
		getUpgradeFlagSet().FlagUsages(),
	))
}

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "cloud",
		DisplayName:      "Mattermost Private Cloud",
		Description:      "This command allows spinning up and down Mattermost installations using Mattermost Private Cloud.",
		AutoComplete:     false,
		AutoCompleteDesc: "Available commands: create, list, upgrade, mmcli, delete",
		AutoCompleteHint: "[command]",
	}
}

func getCommandResponse(responseType, text string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: responseType,
		Text:         text,
		Username:     "cloud",
		IconURL:      fmt.Sprintf("/plugins/%s/profile.png", manifest.ID),
	}
}

// ExecuteCommand executes a given command and returns a command response.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	config := p.getConfiguration()

	if config.AllowedEmailDomain != "" {
		user, err := p.API.GetUser(args.UserId)
		if err != nil {
			return nil, err
		}

		if !strings.HasSuffix(user.Email, config.AllowedEmailDomain) {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Permission denied. Please talk to your system administrator to get access."), nil
		}
	}

	stringArgs := strings.Split(args.Command, " ")

	if len(stringArgs) < 2 {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, getHelp()), nil
	}

	command := stringArgs[1]

	var handler func([]string, *model.CommandArgs) (*model.CommandResponse, bool, error)

	switch command {
	case "create":
		handler = p.runCreateCommand
	case "mmcli":
		handler = p.runMattermostCLICommand
	case "list":
		handler = p.runListCommand
	case "upgrade":
		handler = p.runUpgradeCommand
	case "delete":
		handler = p.runDeleteCommand
	case "status":
		handler = p.runStatusCommand
	case "info":
		handler = p.runInfoCommand
	}

	if handler == nil {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, getHelp()), nil
	}

	resp, isUserError, err := handler(stringArgs[2:], args)

	if err != nil {
		if isUserError {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("__Error: %s__\n\nRun `/cloud help` for usage instructions.", err.Error())), nil
		}
		p.API.LogError(err.Error())
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "An unknown error occurred. Please talk to your resident cloud team for help."), nil
	}

	return resp, nil
}

func (p *Plugin) runInfoCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	resp := fmt.Sprintf("Mattermost Cloud plugin version: %s, "+
		"[%s](https://github.com/mattermost/mattermost-plugin-cloud/commit/%s), built %s\n",
		manifest.Version, BuildHashShort, BuildHash, BuildDate)

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, resp), false, nil
}
