package main

import (
	"encoding/json"
	"fmt"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

type listConfig struct {
	Shared bool
}

func getListFlagSet() *flag.FlagSet {
	flagSet := flag.NewFlagSet("list", flag.ContinueOnError)
	flagSet.Bool("shared-installations", false, "Lists shared installations instead of personal ones")

	return flagSet
}

func parseListFlagSet(args []string) (*listConfig, error) {
	flagSet := getListFlagSet()
	err := flagSet.Parse(args)
	if err != nil {
		return nil, errors.Wrap(err, "falied to parse flags")
	}

	config := &listConfig{}
	config.Shared, err = flagSet.GetBool("shared-installations")
	if err != nil {
		return nil, errors.Wrap(err, "falied to get shared-installations value")
	}

	return config, nil
}

func (p *Plugin) runListCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	config, err := parseListFlagSet(args)
	if err != nil {
		return nil, true, err
	}

	var installs []*Installation
	if config.Shared {
		installs, err = p.getUpdatedSharedInstallations(true)
		if err != nil {
			return nil, false, err
		}
	} else {
		installs, err = p.getUpdatedInstallsForUserWithoutSensitive(extra.UserId)
		if err != nil {
			return nil, false, err
		}
	}

	if len(installs) == 0 {
		return getCommandResponse(model.CommandResponseTypeEphemeral, "No installations found.", extra), false, nil
	}

	data, err := json.Marshal(installs)
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, jsonCodeBlock(prettyPrintJSON(string(data))), extra), false, nil
}

func (p *Plugin) getUpdatedInstallsForUserWithSensitive(userID string) ([]*Installation, error) {
	return p.getUpdatedInstallsForUser(userID, false)
}

func (p *Plugin) getUpdatedInstallsForUserWithoutSensitive(userID string) ([]*Installation, error) {
	return p.getUpdatedInstallsForUser(userID, true)
}

func (p *Plugin) getUpdatedInstallsForUser(userID string, hideSensitive bool) ([]*Installation, error) {
	pluginInstalls, err := p.getInstallationsForUser(userID)
	if err != nil {
		return nil, err
	}

	// Grab the cloud installations belonging to this user. Note that we are not
	// asking for deleted installations. This is done for performance reasons as
	// we can ask for deleted installations later if necesssary.
	cloudInstalls, err := p.cloudClient.GetInstallations(&cloud.GetInstallationsRequest{
		OwnerID:            userID,
		IncludeGroupConfig: true,
		Paging:             cloud.AllPagesNotDeleted(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get installations from cloud server")
	}

	var deleted bool
	for i, pluginInstall := range pluginInstalls {
		deleted, err = p.processInstallationUpdate(pluginInstall, cloudInstalls, hideSensitive)
		if err != nil {
			return nil, errors.Wrap(err, "unable to process installation")
		}
		if deleted {
			// Notify the user and also show the deleted installation in their
			// list one last time with a DELETED tag.
			pluginInstalls[i] = &Installation{
				Name: fmt.Sprintf("%s [ DELETED ]", pluginInstall.Name),
			}
		}
	}

	return pluginInstalls, nil
}

func (p *Plugin) processInstallationUpdate(pluginInstall *Installation, cloudInstalls []*cloud.InstallationDTO, hideSensitive bool) (bool, error) {
	for _, cloudInstall := range cloudInstalls {
		if pluginInstall.ID == cloudInstall.ID {
			pluginInstall.InstallationDTO = *cloudInstall
			if hideSensitive {
				pluginInstall.HideSensitiveFields()
			}
			return false, nil
		}
	}

	// No match could be made with the provided slice of cloud installations.
	// Let's verify that this installation was deleted.
	updatedInstall, err := p.cloudClient.GetInstallation(pluginInstall.ID,
		&cloud.GetInstallationRequest{
			IncludeGroupConfig: true,
		})
	if err != nil {
		return false, errors.Wrapf(err, "unable to get installation %s from cloud server", pluginInstall.ID)
	}
	if updatedInstall == nil {
		return false, fmt.Errorf("could not find installation %s", pluginInstall.ID)
	}

	pluginInstall.Installation = updatedInstall.Installation
	if hideSensitive {
		pluginInstall.HideSensitiveFields()
	}

	if updatedInstall.State != cloud.InstallationStateDeleted {
		// This is strange as the installation should have been retrieved in the
		// original cloud server query.
		// Handle this by logging and returning the installation as normal.
		p.API.LogWarn(fmt.Sprintf("Cloud installation %s with name %s was not returned on the original cloud server query", pluginInstall.ID, pluginInstall.Name))
		return false, nil
	}

	// The installation was deleted on the cloud server so remove it from the KV
	// store to sync state and notify the user.
	p.API.LogWarn(fmt.Sprintf("Removing deleted installation %s with name %s from the KV store", pluginInstall.ID, pluginInstall.Name))
	err = p.deleteInstallation(pluginInstall.ID)
	if err != nil {
		return true, errors.Wrapf(err, "unable to delete installation %s in the KV store", pluginInstall.ID)
	}

	return true, nil
}
