package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
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

	scope := InstallationScopeMine
	if includeShared {
		scope = InstallationScopeUpdatable
	}
	_, err = p.restartInstallationForUser(extra.UserId, InstallationRef{Name: name}, scope)
	if err != nil {
		if strings.Contains(err.Error(), "no installation with the name") {
			return nil, true, err
		}
		return nil, false, err
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, fmt.Sprintf("Installation %s restarting now.", name), extra), false, nil
}
