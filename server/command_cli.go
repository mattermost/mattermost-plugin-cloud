package main

import (
	"encoding/json"
	"fmt"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

func (p *Plugin) runCLICommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 {
		return nil, true, errors.New("must provide an installation name")
	}

	name := args[0]

	if name == "" {
		return nil, true, errors.New("must provide an installation name")
	}

	installsForUser, err := p.getInstallationsForUser(extra.UserId)
	if err != nil {
		return nil, false, err
	}

	if len(installsForUser) == 0 {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "No installations found."), false, nil
	}

	var updatedInstall *cloud.Installation
	for _, install := range installsForUser {
		updatedInstall, err = p.cloudClient.GetInstallation(install.ID)
		if err != nil {
			p.API.LogError(fmt.Sprintf("could not get updated installation %s", install.ID))
		}

		install.Installation = *updatedInstall
	}

	data, err := json.Marshal(installsForUser)
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, prettyPrintJSON(string(data))), false, nil
}
