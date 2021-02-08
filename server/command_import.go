package main

import (
	"encoding/json"
	"strings"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

func (p *Plugin) runImportCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 {
		return nil, true, errors.New("must provide an installation ID")
	}
	installationID := standardizeName(args[0])

	installwithID, err := p.getInstallWithDNS(installationID, extra.UserId)
	if err != nil {
		return nil, false, err
	}

	dataInstall, err := json.Marshal(installwithID)
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, jsonCodeBlock(prettyPrintJSON(string(dataInstall)))), false, nil

}
func (p *Plugin) getInstallWithDNS(DNS string, userID string) (*Installation, error) {
	cloudInstall, err := p.cloudClient.GetInstallationByDNS(DNS, nil)
	if err != nil {
		return nil, err
	}

	splitDNS := strings.Split(cloudInstall.DNS, ".")
	if len(splitDNS) < 2 {
		return nil, errors.New("failed to parse DNS value")
	}
	name := splitDNS[0]

	if cloudInstall.OwnerID != userID {
		cloudInstall.OwnerID = userID
	}

	p.cloudClient.UpdateInstallation(cloudInstall.ID, &cloud.PatchInstallationRequest{
		OwnerID: &cloudInstall.OwnerID,
	})

	pluginInstall := &Installation{
		Name: name,
	}
	pluginInstall.Installation = *cloudInstall.Installation

	err = p.storeInstallation(pluginInstall)
	if err != nil {
		return nil, err
	}
	pluginInstall.HideSensitiveFields()

	return pluginInstall, nil

}
