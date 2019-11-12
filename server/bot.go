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

// PostToChannelByNameForTeamNameAsBot posts a message to the provided team + channel as the bot.
func (p *Plugin) PostToChannelByNameForTeamNameAsBot(teamName, channelName, message string) error {
	channel, appError := p.API.GetChannelByNameForTeamName(teamName, channelName, false)
	if appError != nil {
		return appError
	}
	if channel == nil {
		return fmt.Errorf("channel %s for team %s is nil", channelName, teamName)
	}

	_, appError = p.API.CreatePost(&model.Post{
		UserId:    p.BotUserID,
		ChannelId: channel.Id,
		Message:   message,
	})
	if appError != nil {
		return appError
	}

	return nil
}
