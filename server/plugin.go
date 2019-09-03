package main

import (
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	cloudClient CloudClient

	BotUserID string

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// CloudClient is the interface for managing cloud installations.
type CloudClient interface {
	CreateInstallation(request *cloud.CreateInstallationRequest) (*cloud.Installation, error)
	GetInstallation(installationID string) (*cloud.Installation, error)
	DeleteInstallation(installationID string) error
}

// OnActivate runs when the plugin activates and ensures the plugin is properly
// configured.
func (p *Plugin) OnActivate() error {
	config := p.getConfiguration()
	if err := config.IsValid(); err != nil {
		return err
	}

	botID, err := p.Helpers.EnsureBot(&model.Bot{
		Username:    "cloud",
		DisplayName: "Cloud",
		Description: "Created by the Mattermost Private Cloud plugin.",
	})
	if err != nil {
		return errors.Wrap(err, "failed to ensure github bot")
	}
	p.BotUserID = botID

	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return errors.Wrap(err, "couldn't get bundle path")
	}

	profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "profile.png"))
	if err != nil {
		return errors.Wrap(err, "couldn't read profile image")
	}

	appErr := p.API.SetProfileImage(botID, profileImage)
	if appErr != nil {
		return errors.Wrap(appErr, "couldn't set profile image")
	}

	p.cloudClient = cloud.NewClient(config.ProvisioningServerURL)
	return p.API.RegisterCommand(getCommand())
}
