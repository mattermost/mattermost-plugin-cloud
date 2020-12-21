package main

import (
	"fmt"

	"github.com/mattermost/mattermost-server/model"
)

// PostBotDM posts a DM as the cloud bot user.
func (p *Plugin) PostBotDM(userID, message string) error {
	channel, appError := p.API.GetDirectChannel(userID, p.BotUserID)
	if appError != nil {
		return appError
	}
	if channel == nil {
		return fmt.Errorf("could not get direct channel for bot and user_id=%s", userID)
	}

	_, appError = p.API.CreatePost(&model.Post{
		UserId:    p.BotUserID,
		ChannelId: channel.Id,
		Message:   message,
	})

	return appError
}

// PostUniqueBotDM will post a DM to a user with userID like
// PostBotDM, but if a unique pendingPostID is specified, the caller
// may assume that any additional calls with the same ID will be
// dropped upon receipt by Mattermost
func (p *Plugin) PostUniqueBotDM(userID, pendingPostID, message string) error {
	channel, appError := p.API.GetDirectChannel(userID, p.BotUserID)
	if appError != nil {
		return appError
	}
	if channel == nil {
		return fmt.Errorf("could not get direct channel for bot and user_id=%s", userID)
	}

	_, appError = p.API.CreatePost(&model.Post{
		PendingPostId: pendingPostID,
		UserId:        p.BotUserID,
		ChannelId:     channel.Id,
		Message:       message,
	})

	return appError
}

// PostToChannelByIDAsBot posts a message to the provided channel.
func (p *Plugin) PostToChannelByIDAsBot(channelID, message string) error {
	_, appError := p.API.CreatePost(&model.Post{
		UserId:    p.BotUserID,
		ChannelId: channelID,
		Message:   message,
	})
	if appError != nil {
		return appError
	}

	return nil
}
