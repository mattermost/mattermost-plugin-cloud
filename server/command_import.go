package main

import (
	"encoding/json"
	"net/url"
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

	u, err := url.Parse(installationDNS)
	if err != nil {
		return nil, true, errors.Wrap(err, "error parsing url")
	}

	hostname := u.Hostname()

	if hostname == "" {
		hostname = installationDNS
	}

	splitDNS := strings.Split(hostname, ".")
	if len(splitDNS) < 2 {
		return nil, true, errors.Errorf("failed to parse DNS value: %s", hostname)
	}
	name := splitDNS[0]

	cloudInstall, err := p.cloudClient.GetInstallationByDNS(hostname, nil)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get installation by DNS")
	}
	if cloudInstall == nil {
		return nil, true, errors.New("no installation for the DNS provided")
	}

	installs, _, err := p.getInstallations()
	if err != nil {
		return nil, false, err
	}

	for _, install := range installs {
		if install.ID == cloudInstall.ID {
			return nil, true, errors.New("installation has already been imported to cloud plugin")
		}
	}

	if cloudInstall.OwnerID != extra.UserId {
		cloudInstall.OwnerID = extra.UserId
		_, err = p.cloudClient.UpdateInstallation(cloudInstall.ID, &cloud.PatchInstallationRequest{
			OwnerID: &cloudInstall.OwnerID,
		})
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to update installation")
		}
	}

	pluginInstall := &Installation{
		Name: name,
	}
	pluginInstall.Installation = *cloudInstall.Installation

	err = p.storeInstallation(pluginInstall)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to store updated installation")
	}
	pluginInstall.HideSensitiveFields()

	dataInstall, err := json.Marshal(pluginInstall)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to Marshal installation")
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Installation imported:\n\n"+jsonCodeBlock(prettyPrintJSON(string(dataInstall)))), false, nil

}
