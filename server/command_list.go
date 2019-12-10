package main

import (
	"encoding/json"
	"fmt"

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
	pluginInstalls, err := p.getInstallationsForUser(userID)
	if err != nil {
		return nil, err
	}

	var cloudInstall *cloud.Installation
	for k, pluginInstall := range pluginInstalls {
		cloudInstall, err = p.cloudClient.GetInstallation(pluginInstall.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get updated installation %s", pluginInstall.ID)
		}

		if cloudInstall == nil {
			// If it was never an installation in cloud server and it was never marked as deleted
			// in KV store, we return en error.
			if pluginInstall.DeleteAt == 0 {
				return nil, fmt.Errorf("no records of installation ID %s", pluginInstall.ID)
			}

			// If installation was marked as deleted, remove it from the KV store.
			err = p.deleteInstallation(pluginInstall.ID)
			if err != nil {
				p.API.LogError(err.Error(), pluginInstall.ID)
				continue
			}

			// Notify the user that installation was removed.
			p.PostBotDM(userID, fmt.Sprintf("Cloud installation ID %s has been removed from your Mattermost app.", pluginInstall.ID))

			// Remove item from installs.
			i := len(pluginInstalls) - 1
			pluginInstalls[i] = pluginInstalls[k]
			pluginInstalls[k] = pluginInstalls[i]
			pluginInstalls = pluginInstalls[:i]
			continue
		}

		pluginInstall.Installation = *cloudInstall
		pluginInstall.License = "hidden"
	}

	return pluginInstalls, nil
}
