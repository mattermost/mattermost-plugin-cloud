package main

import (
	"fmt"
	"time"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

func (p *Plugin) runGetDebugPacketCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 {
		return nil, true, errors.New("must provide an installation name")
	}

	name := standardizeName(args[0])
	if name == "" {
		return nil, true, errors.New("must provide an installation name")
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
		Message:   fmt.Sprintf("Gathering debug data for `%s` now. Please wait as this may take a while.", installToExec.Name),
	})

	err = p.execGetDebugPacket(installToExec.ID, extra.UserId, name)
	if err != nil {
		return nil, false, err
	}

	resp := "Debug packet generated. Check your direct messages from the cloud bot."

	return getCommandResponse(model.CommandResponseTypeEphemeral, resp, extra), false, nil
}

func (p *Plugin) execGetDebugPacket(installationID, userID, installationName string) error {
	clusterInstallations, err := p.cloudClient.GetClusterInstallations(&cloud.GetClusterInstallationsRequest{
		InstallationID: installationID,
		Paging:         cloud.AllPagesNotDeleted(),
	})
	if err != nil {
		return errors.Wrap(err, "unable to get cluster installations")
	}

	if len(clusterInstallations) == 0 {
		return fmt.Errorf("no cluster installations found for installation %s", installationID)
	}

	fileBytes, err := p.cloudClient.ExecClusterInstallationPPROF(clusterInstallations[0].ID)
	if err != nil {
		return errors.Wrap(err, "failed to gather debug packet data")
	}
	if fileBytes == nil {
		return errors.Wrap(err, "no debug data returned")
	}

	botDMChannel, appErr := p.API.GetDirectChannel(userID, p.BotUserID)
	if appErr != nil {
		return err
	}
	if botDMChannel == nil {
		return fmt.Errorf("could not get direct channel for bot and user_id=%s", userID)
	}

	filename := fmt.Sprintf("%s.%d.debug.zip", installationName, time.Now().UnixMilli())
	fileInfo, appErr := p.API.UploadFile(fileBytes, botDMChannel.Id, filename)
	if appErr != nil {
		return errors.Wrap(err, "unable to upload debug file")
	}

	_, err = p.API.CreatePost(&model.Post{
		UserId:    p.BotUserID,
		ChannelId: botDMChannel.Id,
		Message:   fmt.Sprintf("Here is a debug packet for installation %s", installationName),
		FileIds:   []string{fileInfo.Id},
	})

	return nil
}
