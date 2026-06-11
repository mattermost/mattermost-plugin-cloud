package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

type shareConfig struct {
	AllowUpdates bool
}

func getShareFlagSet() *flag.FlagSet {
	flagSet := flag.NewFlagSet("share", flag.ContinueOnError)
	flagSet.Bool("allow-updates", false, "Allow other plugin users to update the installation configuration")

	return flagSet
}

func parseShareFlagSet(args []string) (*shareConfig, error) {
	flagSet := getShareFlagSet()
	err := flagSet.Parse(args)
	if err != nil {
		return nil, errors.Wrap(err, "falied to parse flags")
	}

	config := &shareConfig{}
	config.AllowUpdates, err = flagSet.GetBool("allow-updates")
	if err != nil {
		return nil, errors.Wrap(err, "falied to get allow-updates value")
	}

	return config, nil
}

func (p *Plugin) runShareInstallationCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	config, err := parseShareFlagSet(args)
	if err != nil {
		return nil, true, err
	}

	_, err = p.setInstallationSharingForUser(extra.UserId, InstallationRef{Name: name}, true, config.AllowUpdates)
	if err != nil {
		if strings.Contains(err.Error(), "no installation with the name") {
			return nil, true, err
		}
		return getCommandResponse(model.CommandResponseTypeEphemeral, err.Error(), extra), false, err
	}

	sharedUpdateText := "Other plugin users are not permitted to update this installation."
	if config.AllowUpdates {
		sharedUpdateText = "Other plugin users will be allowed to update this installation."
	}
	return getCommandResponse(model.CommandResponseTypeEphemeral, fmt.Sprintf("Installation has been shared with other plugin users. %s", sharedUpdateText), extra), false, nil
}

func (p *Plugin) runUnshareInstallationCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	_, err := p.setInstallationSharingForUser(extra.UserId, InstallationRef{Name: name}, false, false)
	if err != nil {
		if strings.Contains(err.Error(), "no installation with the name") {
			return nil, true, err
		}
		return getCommandResponse(model.CommandResponseTypeEphemeral, err.Error(), extra), false, err
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, "Installation has been unshared.", extra), false, nil
}
