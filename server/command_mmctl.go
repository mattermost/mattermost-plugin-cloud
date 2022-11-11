package main

import (
	"fmt"
	"strings"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

func (p *Plugin) runMmctlCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 {
		return nil, true, errors.New("must provide an installation name")
	}

	name := standardizeName(args[0])
	if name == "" {
		return nil, true, errors.New("must provide an installation name")
	}

	subcommand := args[1:]
	if len(subcommand) == 0 {
		return nil, true, errors.New("must provide a mmctl command")
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

	p.API.SendEphemeralPost(extra.UserId, &model.Post{
		UserId:    p.BotUserID,
		ChannelId: extra.ChannelId,
		Message:   fmt.Sprintf("Running the command `mmctl %s` on `%s` now. Please wait as this may take a while.", strings.Join(subcommand, " "), installToExec.Name),
	})

	output, err := p.execMmctl(installToExec.ID, subcommand)
	if err != nil {
		return nil, false, err
	}

	resp := fmt.Sprintf("Installation: %s\n\nCommand: mmctl %s\n\nResponse:\n%s",
		installToExec.Name,
		strings.Join(subcommand, " "),
		codeBlock(string(output)),
	)

	return getCommandResponse(model.CommandResponseTypeEphemeral, resp, extra), false, nil
}

func (p *Plugin) execMmctl(installationID string, subcommand []string) ([]byte, error) {
	clusterInstallations, err := p.cloudClient.GetClusterInstallations(&cloud.GetClusterInstallationsRequest{
		InstallationID: installationID,
		Paging:         cloud.AllPagesNotDeleted(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get cluster installations")
	}

	if len(clusterInstallations) == 0 {
		return nil, fmt.Errorf("no cluster installations found for installation %s", installationID)
	}

	subcommand = append(subcommand, "--local")
	output, err := p.cloudClient.ExecClusterInstallationCLI(clusterInstallations[0].ID, "mmctl", subcommand)
	if err != nil && err.Error() == "failed with status code 504" {
		// TODO: make this not gross.
		// Return an error type that can be checked or allow us to pass in
		// something with a timeout that we can control.
		p.API.LogWarn(errors.Wrapf(err, "Command /mmctl %s didn't complete before the connection was closed", strings.Join(subcommand, " ")).Error())
		return []byte(fmt.Sprintf("Command /mmctl %s didn't complete before the connection was closed. It will continue running until it is completed.", strings.Join(subcommand, " "))), nil
	} else if err != nil {
		return nil, err
	}

	return output, nil
}
