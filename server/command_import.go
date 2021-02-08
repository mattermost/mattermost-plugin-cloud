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
		return nil, true, errors.New("must provide an installation DNS")
	}
	installationDNS := standardizeName(args[0])

	splitDNS := strings.Split(installationDNS, ".")
	if len(splitDNS) < 2 {
		return nil, true, errors.New("failed to parse DNS value")
	}
	name := splitDNS[0]

	cloudInstall, err := p.cloudClient.GetInstallationByDNS(installationDNS, nil)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get installation by DNS")
	}
	if cloudInstall == nil {
		return nil, true, errors.New("no installation for the DNS provided")
	}
	if cloudInstall.OwnerID != extra.UserId {
		cloudInstall.OwnerID = extra.UserId
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
		return nil, false, err
	}
	pluginInstall.HideSensitiveFields()

	dataInstall, err := json.Marshal(pluginInstall)
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Installation imported:\n\n"+jsonCodeBlock(prettyPrintJSON(string(dataInstall)))), false, nil

}
