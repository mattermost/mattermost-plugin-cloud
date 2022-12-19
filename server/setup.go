package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/blang/semver/v4"
	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

const (
	defaultAdminUsername = "sysadmin"
	defaultAdminPassword = "Sys@dmin123"
	defaultAdminEmail    = "success+sysadmin@simulator.amazonses.com"

	defaultUserUsername = "user"
	defaultUserPassword = defaultAdminPassword
	defaultUserEmail    = "success+user@simulator.amazonses.com"
)

func (p *Plugin) setupInstallation(install *Installation) error {
	if len(install.DNSRecords) == 0 {
		return fmt.Errorf("Installation %s doesn't have any DNSRecords", install.ID)
	}

	client := model.NewAPIv4Client(fmt.Sprintf("https://%s", install.DNSRecords[0].DomainName))
	if client == nil {
		return errors.New("got nil APIv4 Mattermost client for some reason")
	}

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

	// Create normal user
	err = p.createUser(client, defaultUserUsername, defaultUserPassword, defaultUserEmail)
	if err != nil {
		return errors.Wrap(err, "encountered an error creating installation user account")
	}

	return nil
}

func (p *Plugin) waitForDNS(client *model.Client4) error {
	for i := 0; i < 60; i++ {
		_, resp, err := client.GetPing()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		if err != nil {
			p.API.LogDebug(err.Error())
		}
		time.Sleep(time.Second * 10)
	}

	return errors.New("timed out waiting for installation DNS")
}

func (p *Plugin) createUser(client *model.Client4, username, password, email string) error {
	_, _, err := client.CreateUser(
		&model.User{
			Username: username,
			Password: password,
			Email:    email,
		},
	)
	return err
}

func (p *Plugin) createAndLoginAdminUser(client *model.Client4) error {
	err := p.createUser(client, defaultAdminUsername, defaultAdminPassword, defaultAdminEmail)
	if err != nil {
		return err
	}

	_, _, err = client.Login(defaultAdminUsername, defaultAdminPassword)
	if err != nil {
		return err
	}

	return nil
}

func (p *Plugin) setupInstallationConfiguration(client *model.Client4, install *Installation) error {
	config, resp, err := client.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get Mattermost config")
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("got unexpected status %d while getting Mattermost config")
	}

	pluginConfig := p.getConfiguration()

	if pluginConfig.GroupID == "" {
		// Set some basic config due to not being in a group.
		p.configureEmail(config, pluginConfig)

		config.ServiceSettings.EnableDeveloper = NewBool(true)
		config.TeamSettings.EnableOpenServer = NewBool(true)
		config.PluginSettings.EnableUploads = NewBool(true)

		_, _, err = client.UpdateConfig(config)
		if err != nil {
			return errors.Wrap(err, "unable to update installation config")
		}
	}

	err = p.createTestData(client, install)
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

func (p *Plugin) createTestData(client *model.Client4, install *Installation) error {
	if !install.TestData {
		return nil
	}

	mmReleaseVersion, err := semver.Parse(install.Tag)
	if err != nil {
		return errors.Wrapf(err, "failed to parse version %s", install.Tag)
	}

	if mmReleaseVersion.LT(semver.MustParse("6.0.0")) {
		_, err = p.execMattermostCLI(install.ID, []string{"sampledata"})
		if err != nil {
			// This probably won't complete before the AWS API Gateway timeout so
			// log and move on.
			p.API.LogWarn(errors.Wrapf(err, "Unable to finish generating test data for cloud installation %s", install.Name).Error())
		}

		// Test data generation overrides the sysadmin password, so need to reset
		_, err = p.execMattermostCLI(install.ID, []string{"user", "password", defaultAdminUsername, defaultAdminPassword})
		if err != nil {
			return errors.Wrap(err, "failed to reset sysadmin password back to the default")
		}

		return nil
	}

	clusterInstallations, err := p.cloudClient.GetClusterInstallations(
		&cloud.GetClusterInstallationsRequest{
			// any single CI will do, so only fetch one
			Paging:         cloud.AllPagesNotDeleted(),
			InstallationID: install.ID,
		})
	if err != nil {
		return errors.Wrap(err, "failed to get ClusterInstallations for Installation")
	}
	if len(clusterInstallations) != 1 {
		return errors.Errorf("got unexpected number of ClusterInstallations (%d)", len(clusterInstallations))
	}

	_, err = p.cloudClient.ExecClusterInstallationCLI(clusterInstallations[0].ID,
		"mmctl", []string{"--local", "sampledata"})
	if err != nil {
		// Gabe thinks this might not complete before the AWS API Gateway timeout so
		// log and move on the same as we do for versions previous to 6.0, seen earlier in this method.
		p.API.LogWarn(errors.Wrapf(err, "Unable to finish generating test data for cloud installation %s", install.Name).Error())
	}

	// Test data generation overrides the sysadmin password, so need to reset (using mmctl)
	_, err = p.cloudClient.ExecClusterInstallationCLI(clusterInstallations[0].ID,
		"mmctl", []string{"--local", "user", "change-password", defaultAdminUsername, "--password", defaultAdminPassword})
	if err != nil {
		return errors.Wrap(err, "failed to reset sysadmin password back to the default")
	}

	return nil
}
