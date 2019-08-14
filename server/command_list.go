package main

import (
	"encoding/json"

	"github.com/mattermost/mattermost-server/model"
)

func (p *Plugin) runListCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	installsForUser, err := p.getInstallationsForUser(extra.UserId)
	if err != nil {
		return nil, false, err
	}

	if len(installsForUser) == 0 {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "No installations found."), false, nil
	}

	for _, install := range installsForUser {
		updatedInstall, err := p.cloudClient.GetInstallation(install.ID)
		if err != nil {
			p.API.LogError("could not get updated installation %s", install.ID)
		}

		install.Installation = *updatedInstall
	}

	data, err := json.Marshal(installsForUser)
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, string(data)), false, nil
}
