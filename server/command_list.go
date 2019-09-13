package main

import (
	"encoding/json"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

func (p *Plugin) runListCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	installsForUser, err := p.getUpdatedInstallsForUser(extra.UserId)
	if err != nil {
		return nil, false, err
	}

	if len(installsForUser) == 0 {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "No installations found."), false, nil
	}

	data, err := json.Marshal(installsForUser)
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, jsonCodeBlock(prettyPrintJSON(string(data)))), false, nil
}

func (p *Plugin) getUpdatedInstallsForUser(userID string) ([]*Installation, error) {
	installsForUser, err := p.getInstallationsForUser(userID)
	if err != nil {
		return nil, err
	}

	var updatedInstall *cloud.Installation
	for _, install := range installsForUser {
		updatedInstall, err = p.cloudClient.GetInstallation(install.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get updated installation %s", install.ID)
		}

		install.Installation = *updatedInstall
	}

	return installsForUser, nil
}
