package main

import (
	"fmt"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

// runWakeUpCommand wakes up the provided installation.
func (p *Plugin) runWakeUpCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	installs, err := p.getUpdatedInstallsForUser(extra.UserId)
	if err != nil {
		return nil, false, err
	}

	var installToWakeUp *Installation
	for _, install := range installs {
		if install.OwnerID == extra.UserId && install.Name == name {
			installToWakeUp = install
			break
		}
	}

	if installToWakeUp == nil {
		return nil, true, errors.Errorf("no installation with the name %s found", name)
	}
	if installToWakeUp.State != cloud.InstallationStateHibernating {
		return nil, true, errors.Errorf("installation state is currently %s and must be %s to wake up", installToWakeUp.State, cloud.InstallationStateHibernating)
	}

	_, err = p.cloudClient.WakeupInstallation(installToWakeUp.ID, &cloud.PatchInstallationRequest{})
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, fmt.Sprintf("Installation %s is waking up. You will receive a notification when it is updated. Use /cloud list to check on the status of your installations.", name)), false, nil
}
