package main

import (
	"fmt"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

func (p *Plugin) runRestartCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.New("must provide an installation name")
	}

	name := standardizeName(args[0])

	installs, _, err := p.getInstallations()
	if err != nil {
		return nil, false, err
	}

	var installToRestart *Installation
	for _, install := range installs {
		if install.OwnerID == extra.UserId && standardizeName(install.Name) == name {
			installToRestart = install
			break
		}
	}

	if installToRestart == nil {
		return nil, true, errors.Errorf("no installation with the name %s found", name)
	}

	// We can force a restart by changing any environment variable, so we will
	// set something arbitrary with the current date and time.
	patch := &cloud.PatchInstallationRequest{MattermostEnv: cloud.EnvVarMap{
		"CLOUD_PLUGIN_RESTART": cloud.EnvVar{Value: cloud.DateTimeStringFromMillis(cloud.GetMillis())},
	}}
	_, err = p.cloudClient.UpdateInstallation(installToRestart.ID, patch)
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, fmt.Sprintf("Installation %s restarting now.", name), extra), false, nil
}
