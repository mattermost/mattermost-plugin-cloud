package main

import (
	"fmt"
	"strings"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

func (p *Plugin) runMattermostCLICommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 {
		return nil, true, errors.New("must provide an installation name")
	}

	name := args[0]
	if name == "" {
		return nil, true, errors.New("must provide an installation name")
	}

	subcommand := args[1:]
	if len(subcommand) == 0 {
		return nil, true, errors.New("must provide an mattermost CLI command")
	}

	installsForUser, err := p.getInstallationsForUser(extra.UserId)
	if err != nil {
		return nil, false, err
	}

	var installToExec *Installation
	for _, install := range installsForUser {
		if install.OwnerID == extra.UserId && install.Name == name {
			installToExec = install
			break
		}
	}

	if installToExec == nil {
		return nil, true, fmt.Errorf("no installation with the name %s found", name)
	}

	output, err := p.execMattermostCLI(installToExec.ID, subcommand)
	if err != nil {
		return nil, false, err
	}

	resp := fmt.Sprintf("Installation: %s\n\nCommand: mattermost %s\n\nResponse:\n%s",
		installToExec.Name,
		strings.Join(subcommand, " "),
		codeBlock(string(output)),
	)

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, resp), false, nil
}

func (p *Plugin) execMattermostCLI(installationID string, subcommand []string) ([]byte, error) {
	clusterInstallations, err := p.cloudClient.GetClusterInstallations(&cloud.GetClusterInstallationsRequest{
		InstallationID: installationID,
		Page:           0,
		PerPage:        100,
		IncludeDeleted: false,
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get cluster installations")
	}

	if len(clusterInstallations) == 0 {
		return nil, fmt.Errorf("no cluster installations found for installation %s", installationID)
	}

	output, err := p.cloudClient.RunMattermostCLICommandOnClusterInstallation(clusterInstallations[0].ID, subcommand)
	if err != nil {
		return nil, err
	}

	return output, nil
}
