package main

import (
	"fmt"
	"strings"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
)

// runHibernateCommand hibernates the provided installation.
func (p *Plugin) runHibernateCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	_, err := p.hibernateInstallationForUser(extra.UserId, InstallationRef{Name: name})
	if err != nil {
		if strings.Contains(err.Error(), "no installation with the name") ||
			strings.Contains(err.Error(), fmt.Sprintf("must be %s to hibernate", cloud.InstallationStateStable)) {
			return nil, true, err
		}
		return nil, false, err
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, fmt.Sprintf("Hibernation of installation %s has begun. You will receive a notification when it is hibernated. Use /cloud list to check on the status of your installations.", name), extra), false, nil
}
