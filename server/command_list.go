package main

import (
	"encoding/json"
	"fmt"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
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

	cloudInstalls, err := p.cloudClient.GetInstallations(&cloud.GetInstallationsRequest{
		OwnerID:        userID,
		IncludeDeleted: true,
	})
	if err != nil {
		return nil, err
	}

	// TODO: This may become a performance issue as deleted installations in the database start
	// to grow. We could make it better by assuming that installation is not deleted. If an error
	// is returned when asked for it, the plugin makes a follow up call to confirm that it was
	// deleted. If deleted, the installation should be also removed from the plugin storage.
	for _, cloudInstall := range cloudInstalls {
		for j, pluginInstall := range pluginInstalls {
			if cloudInstall.ID == pluginInstall.ID {
				if cloudInstall.DeleteAt > 0 || cloudInstall.State == cloud.ClusterInstallationStateCreationFailed {
					err = p.deleteInstallation(pluginInstall.ID)
					if err != nil {
						p.API.LogError(err.Error(), pluginInstall.ID)
					} else {
						// Notify the user that installation was removed.
						p.PostBotDM(userID, fmt.Sprintf("Cloud installation ID %s has been removed from your Mattermost app.", pluginInstall.ID))
						// Remove key from the plugin installations in place.
						l := len(pluginInstalls) - 1
						pluginInstalls[j] = pluginInstalls[l]
						pluginInstalls = pluginInstalls[:l]
					}
				} else {
					pluginInstall.Installation = *cloudInstall
					pluginInstall.License = "hidden"
				}
				break
			}
		}
	}

	return pluginInstalls, nil
}
