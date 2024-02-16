package main

import (
	"fmt"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

func getRestartFlagSet() *flag.FlagSet {
	restartFlagSet := flag.NewFlagSet("restart", flag.ContinueOnError)
	restartFlagSet.Bool("shared-installation", false, "Set this to true when attempting to restart a shared installation")

	return restartFlagSet
}

func (p *Plugin) runRestartCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.New("must provide an installation name")
	}

	restartFlagSet := getRestartFlagSet()
	err := restartFlagSet.Parse(args)
	if err != nil {
		return nil, true, err
	}

	includeShared, err := restartFlagSet.GetBool("shared-installation")
	if err != nil {
		return nil, false, err
	}

	name := standardizeName(args[0])

	installs, err := p.getUpdatableInstallationsForUser(extra.UserId, includeShared)
	if err != nil {
		return nil, false, err
	}

	var installToRestart *Installation
	for _, install := range installs {
		if standardizeName(install.Name) == name {
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
