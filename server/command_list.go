package main

import (
	"encoding/json"
	"fmt"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

type listConfig struct {
	Shared bool
}

type installationRefreshOptions struct {
	hideSensitive  bool
	cleanupDeleted bool
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

	if config.Shared {
		installs, sharedErr := p.getUpdatedSharedInstallations(false)
		if sharedErr != nil {
			return nil, false, sharedErr
		}
		return renderInstallationsList(installs, extra)
	}

	installs, err := p.getUpdatedInstallsForUserWithSensitive(extra.UserId)
	if err != nil {
		return nil, false, err
	}

	return renderInstallationsList(installs, extra)
}

func renderInstallationsList(installs []*Installation, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(installs) == 0 {
		return getCommandResponse(model.CommandResponseTypeEphemeral, "No installations found.", extra), false, nil
	}

	data, err := marshalInstallationsList(sanitizeInstallationCopies(installs))
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, jsonCodeBlock(prettyPrintJSON(string(data))), extra), false, nil
}

func marshalInstallationsList(installs []*Installation) ([]byte, error) {
	data, err := json.Marshal(installs)
	if err != nil {
		return nil, err
	}

	var rawInstalls []map[string]interface{}
	if err = json.Unmarshal(data, &rawInstalls); err != nil {
		return nil, err
	}
	for _, install := range rawInstalls {
		delete(install, "License")
		delete(install, "MattermostEnv")
		delete(install, "PriorityEnv")
	}

	return json.Marshal(rawInstalls)
}

func (p *Plugin) getUpdatedInstallsForUserWithSensitive(userID string) ([]*Installation, error) {
	return p.getUpdatedInstallsForUser(userID, installationRefreshOptions{cleanupDeleted: true})
}

func (p *Plugin) getUpdatedInstallsForUserWithoutSensitive(userID string) ([]*Installation, error) {
	return p.getUpdatedInstallsForUser(userID, installationRefreshOptions{hideSensitive: true, cleanupDeleted: true})
}

func (p *Plugin) getRefreshedInstallsForUser(userID string, cleanupDeleted bool) ([]*Installation, error) {
	return p.getUpdatedInstallsForUser(userID, installationRefreshOptions{cleanupDeleted: cleanupDeleted})
}

func (p *Plugin) getUpdatedInstallsForUser(userID string, options installationRefreshOptions) ([]*Installation, error) {
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
		originalID := pluginInstall.ID
		originalOwnerID := pluginInstall.OwnerID
		originalCreateAt := pluginInstall.CreateAt
		deleted, err = p.processInstallationUpdate(pluginInstall, cloudInstalls, options.cleanupDeleted)
		if err != nil {
			return nil, errors.Wrap(err, "unable to process installation")
		}
		if deleted {
			// Notify the user and also show the deleted installation in their
			// list one last time with a DELETED tag.
			pluginInstalls[i] = deletedInstallationPlaceholder(pluginInstall, originalID, originalOwnerID, originalCreateAt)
		}
	}

	if options.hideSensitive {
		return sanitizeInstallationCopies(pluginInstalls), nil
	}

	return pluginInstalls, nil
}

func deletedInstallationPlaceholder(source *Installation, id, ownerID string, createAt int64) *Installation {
	if source == nil {
		return nil
	}
	if id == "" {
		id = source.ID
	}
	if ownerID == "" {
		ownerID = source.OwnerID
	}
	if createAt == 0 {
		createAt = source.CreateAt
	}

	return &Installation{
		Name: fmt.Sprintf("%s [ DELETED ]", source.Name),
		InstallationDTO: cloud.InstallationDTO{
			Installation: &cloud.Installation{
				ID:       id,
				OwnerID:  ownerID,
				State:    cloud.InstallationStateDeleted,
				CreateAt: createAt,
			},
		},
		Tag:                source.Tag,
		TestData:           source.TestData,
		Shared:             source.Shared,
		AllowSharedUpdates: source.AllowSharedUpdates,
	}
}

func (p *Plugin) processInstallationUpdate(pluginInstall *Installation, cloudInstalls []*cloud.InstallationDTO, cleanupDeleted bool) (bool, error) {
	for _, cloudInstall := range cloudInstalls {
		if pluginInstall.ID == cloudInstall.ID {
			pluginInstall.InstallationDTO = *cloudInstall
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

	pluginInstall.InstallationDTO = *updatedInstall

	if updatedInstall.State != cloud.InstallationStateDeleted {
		// This is strange as the installation should have been retrieved in the
		// original cloud server query.
		// Handle this by logging and returning the installation as normal.
		p.API.LogWarn(fmt.Sprintf("Cloud installation %s with name %s was not returned on the original cloud server query", pluginInstall.ID, pluginInstall.Name))
		return false, nil
	}

	if !cleanupDeleted {
		return true, nil
	}

	// The installation was deleted on the cloud server so remove it from the KV
	// store to sync state and notify the user.
	p.API.LogWarn(fmt.Sprintf("Removing deleted installation %s with name %s from the KV store", pluginInstall.ID, pluginInstall.Name))
	if err = p.deleteInstallation(pluginInstall.ID); err != nil {
		return true, errors.Wrapf(err, "unable to delete installation %s in the KV store", pluginInstall.ID)
	}

	return true, nil
}
