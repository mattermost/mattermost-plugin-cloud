package main

import (
	"fmt"

	"github.com/mattermost/mattermost-server/model"
)

func (p *Plugin) runDeleteCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, fmt.Errorf("must provide an installation name")
	}

	name := args[0]

	installs, _, err := p.getInstallations()
	if err != nil {
		return nil, false, err
	}

	var installToDelete *Installation
	for _, install := range installs {
		if install.OwnerID == extra.UserId && install.Name == name {
			installToDelete = install
			break
		}
	}

	if installToDelete == nil {
		return nil, true, fmt.Errorf("no installation with the name %s found", name)
	}

	// Delete the installation before removing it from the database in case we
	// encounter an error.
	err = p.cloudClient.DeleteInstallation(installToDelete.ID)
	if err != nil {
		return nil, false, err
	}

	err = p.deleteInstallation(installToDelete.ID)
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Installation %s deleted.", name)), false, nil
}
