package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

const (
	defaultAdminUsername = "sysadmin"
	defaultAdminPassword = "Sys@dmin123"
	defaultAdminEmail    = "success+sysadmin@simulator.amazonses.com"
)

func (p *Plugin) setupInstallation(install *Installation) error {
	client := model.NewAPIv4Client(fmt.Sprintf("https://%s", install.DNS))

	err := p.waitForDNS(client)
	if err != nil {
		return errors.Wrap(err, "encountered an error waiting for installation DNS")
	}

	err = p.createAndLoginAdminUser(client)
	if err != nil {
		return errors.Wrap(err, "encountered an error creating installation admin account")
	}

	err = p.setupInstallationConfiguration(client, install)
	if err != nil {
		return errors.Wrap(err, "encountered an error configuring the installation")
	}

	return nil
}

func (p *Plugin) waitForDNS(client *model.Client4) error {
	for i := 0; i < 20; i++ {
		_, resp := client.GetPing()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		if resp.Error != nil {
			p.API.LogDebug(resp.Error.Error())
		}
		time.Sleep(time.Second * 10)
	}

	return errors.New("timed out waiting for installation DNS")
}

func (p *Plugin) createAndLoginAdminUser(client *model.Client4) error {
	_, resp := client.CreateUser(&model.User{Username: defaultAdminUsername, Password: defaultAdminPassword, Email: defaultAdminEmail})
	if resp.Error != nil {
		return resp.Error
	}

	_, resp = client.Login(defaultAdminUsername, defaultAdminPassword)
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (p *Plugin) setupInstallationConfiguration(client *model.Client4, install *Installation) error {
	config, resp := client.GetConfig()
	if resp.Error != nil {
		return resp.Error
	}

	pluginConfig := p.getConfiguration()

	p.configureEmail(config, pluginConfig)

	config.ServiceSettings.EnableDeveloper = NewBool(true)
	config.TeamSettings.EnableOpenServer = NewBool(true)
	config.PluginSettings.EnableUploads = NewBool(true)

	_, resp = client.UpdateConfig(config)
	if resp.Error != nil {
		return errors.Wrap(resp.Error, "unable to update installation config")
	}

	err := p.createTestData(install)
	if err != nil {
		return errors.Wrap(err, "unable to generate installation sample data")
	}

	return nil
}

func (p *Plugin) configureEmail(config *model.Config, pluginConfig *configuration) {
	if pluginConfig.EmailSettings == "" {
		p.API.LogWarn("emailsettings is blank; skipping email configuration")
		return
	}

	err := json.Unmarshal([]byte(pluginConfig.EmailSettings), &config.EmailSettings)
	if err != nil {
		p.API.LogError(errors.Wrap(err, "unable to unmarshal email settings").Error())
	}
}

func (p *Plugin) createTestData(install *Installation) error {
	if !install.TestData {
		return nil
	}

	_, err := p.execMattermostCLI(install.ID, []string{"sampledata"})

	return err
}
