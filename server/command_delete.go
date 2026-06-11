package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

func (p *Plugin) runDeleteCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, fmt.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	_, err := p.deleteInstallationForUser(extra.UserId, InstallationRef{Name: name}, name)
	if err != nil {
		if strings.Contains(err.Error(), "no installation with the name") {
			return nil, true, err
		}
		return nil, false, err
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, fmt.Sprintf("Installation %s deleted.", name), extra), false, nil
}
