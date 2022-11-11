package main

import (
	"fmt"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

// runHibernateCommand hibernates the provided installation.
func (p *Plugin) runHibernateCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	installs, err := p.getUpdatedInstallsForUser(extra.UserId)
	if err != nil {
		return nil, false, err
	}

	var installToHibernate *Installation
	for _, install := range installs {
		if install.OwnerID == extra.UserId && install.Name == name {
			installToHibernate = install
			break
		}
	}

	if installToHibernate == nil {
		return nil, true, errors.Errorf("no installation with the name %s found", name)
	}
	if installToHibernate.State != cloud.InstallationStateStable {
		return nil, true, errors.Errorf("installation state is currently %s and must be %s to hibernate", installToHibernate.State, cloud.InstallationStateStable)
	}

	_, err = p.cloudClient.HibernateInstallation(installToHibernate.ID)
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, fmt.Sprintf("Hibernation of installation %s has begun. You will receive a notification when it is hibernated. Use /cloud list to check on the status of your installations.", name), extra), false, nil
}
