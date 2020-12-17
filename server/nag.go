package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-server/model"
)

// nagUsersSensibly sends a DM to any users whose installations have
// been around for at least a day, but only if it's 12:30PM to 1:30PM
// local time for that user
func (p *Plugin) nagUsersSensibly() error {
	installs, _, err := p.getInstallations()
	if err != nil {
		return err
	}

	userInstalls := make(map[string][]*Installation)
	nowMillis := time.Now().Unix() * 1000
	year, month, day := time.Now().Date()
	for _, install := range installs {
		if nowMillis-install.CreateAt < (time.Hour.Milliseconds() * 24) {
			continue
		}

		user, err := p.API.GetUser(install.OwnerID)
		if err != nil {
			p.API.LogError(fmt.Sprintf(
				"found an installation %s claimed by %s but failed to look up that user due to error: %s",
				install.ID, install.OwnerID, err.Error()))
			continue
		}
		userInstalls[user.Id] = append(userInstalls[user.Id], install)
	}

	for userID, installs := range userInstalls {
		user, apperr := p.API.GetUser(userID)
		if apperr != nil {
			p.API.LogError("failed to look up user %s due to error: %s", userID, apperr.Error())
		}
		tzText := user.GetPreferredTimezone()
		tz, err := time.LoadLocation(tzText)
		if err != nil {
			p.API.LogError(fmt.Sprintf(
				"failed to parse timezone string %s for user %s", tzText, userID))
		}
		noonInTz, err := time.ParseInLocation("3:04pm 1 2 2006", fmt.Sprintf("1:00pm %d %d %d", month, day, year), tz)
		if err != nil {
			p.API.LogError(fmt.Sprintf(
				"failed to create Time object for target time in timezone %s due to error: %s", tz, err.Error()))
		}
		if !time.Now().Round(time.Hour).Equal(noonInTz) {
			continue
		}
		p.nagUser(user, installs)
	}
	return nil
}

// nagUser DMs a user to remind them to delete old resources defined
// by the slice of installations provided
func (p *Plugin) nagUser(user *model.User, installations []*Installation) {
	message := fmt.Sprintf("Hello %s,\n\nIt looks like you have one or more Cloud instances which have been running for some time.\nPlease review your list of running Cloud instances and be sure to clean up any you're not using anymore!\n\nThese are the instances that have been around for awhile:\n\n|Name|Creation Time|\n|:----|:----|\n", user.FirstName)
	for _, installation := range installations {
		tz, err := time.LoadLocation(user.GetPreferredTimezone())
		if err != nil {
			p.API.LogError(fmt.Sprintf(
				"failed to parse timezone string %s for user %s", user.GetPreferredTimezone(), user.Id))
		}

		installationLink := fmt.Sprintf("[%s](%s)", installation.Name, installation.DNS)
		creationTimestamp := time.Unix(installation.CreateAt/1000, (installation.CreateAt%1000)*1000).In(tz).
			Format("Mon Jan 2 15:04:05 -0700 MST 2006")

		message = fmt.Sprintf("%s|%s|%s|\n",
			message,
			installationLink,
			creationTimestamp,
		)
	}
	p.PostBotDM(user.Id, message)
}

// periodicallyNagUsersAboutOldInstallations starts a new thread and
// sleeps until the beginning of the next hour, then calls
// nagUsersSensibly() and calls it each hour after that, which will
// nag users about old Installations
func (p *Plugin) periodicallyNagUsersAboutOldInstallations() {
	go func() {
		for {
			time.Sleep(untilNext(time.Hour))
			p.nagUsersSensibly()
		}
	}()
}

// untilNext returns the duration until the beginning of the next
// slice of time, e.g. minute or hour
func untilNext(duration time.Duration) time.Duration {
	now := time.Now()
	t := time.Now().Round(duration)
	if t.Before(now) {
		t = now.Add(duration).Round(duration)
	}
	return time.Until(t)
}
