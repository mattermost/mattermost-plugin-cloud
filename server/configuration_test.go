package main

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigurationIsValid(t *testing.T) {
	baseConfiguration := configuration{
		ProvisioningServerURL:           "https://provisioner.url.com",
		InstallationDNS:                 "test.com",
		GroupID:                         "",
		ClusterWebhookAlertsEnable:      false,
		InstallationWebhookAlertsEnable: false,
	}

	t.Run("valid", func(t *testing.T) {
		require.NoError(t, baseConfiguration.IsValid())
	})

	t.Run("no provisioner url", func(t *testing.T) {
		config := baseConfiguration
		config.ProvisioningServerURL = ""
		require.Error(t, config.IsValid())
	})

	t.Run("no intallation dns", func(t *testing.T) {
		config := baseConfiguration
		config.InstallationDNS = ""
		require.Error(t, config.IsValid())
	})

	t.Run("groups", func(t *testing.T) {
		t.Run("valid group ID length", func(t *testing.T) {
			config := baseConfiguration
			config.GroupID = model.NewID()
			require.NoError(t, config.IsValid())
		})

		t.Run("invalid group ID length", func(t *testing.T) {
			config := baseConfiguration
			config.GroupID = "tooshort"
			require.Error(t, config.IsValid())
		})
	})

	t.Run("cluster alerts", func(t *testing.T) {
		config := baseConfiguration
		config.ClusterWebhookAlertsEnable = true
		t.Run("no channel ID", func(t *testing.T) {
			require.Error(t, config.IsValid())
		})
		t.Run("valid", func(t *testing.T) {
			config.ClusterWebhookAlertsChannelID = "channel1"
			require.NoError(t, config.IsValid())
		})
	})

	t.Run("installation alerts", func(t *testing.T) {
		config := baseConfiguration
		config.InstallationWebhookAlertsEnable = true
		t.Run("no channel ID", func(t *testing.T) {
			require.Error(t, config.IsValid())
		})
		t.Run("valid", func(t *testing.T) {
			config.InstallationWebhookAlertsChannelID = "channel1"
			require.NoError(t, config.IsValid())
		})
	})
}

func TestGetLicenseValue(t *testing.T) {
	plugin := Plugin{
		configuration: &configuration{
			E10License:          "e10license",
			E20License:          "e20license",
			EnterpriseLicense:   "enterpriselicense",
			ProfessionalLicense: "professionallicense",
		},
	}

	t.Run("e20", func(t *testing.T) {
		t.Run("license", func(t *testing.T) {
			assert.Equal(t, "e20license", plugin.getLicenseValue(licenseOptionE20))
		})
	})

	t.Run("e10", func(t *testing.T) {
		t.Run("license", func(t *testing.T) {
			assert.Equal(t, "e10license", plugin.getLicenseValue(licenseOptionE10))
		})
	})

	t.Run("enterprise", func(t *testing.T) {
		t.Run("license", func(t *testing.T) {
			assert.Equal(t, "enterpriselicense", plugin.getLicenseValue(licenseOptionEnterprise))
		})
	})

	t.Run("professional", func(t *testing.T) {
		t.Run("license", func(t *testing.T) {
			assert.Equal(t, "professionallicense", plugin.getLicenseValue(licenseOptionProfessional))
		})
	})

	t.Run("no matching", func(t *testing.T) {
		assert.Equal(t, "", plugin.getLicenseValue("nope"))
	})
}
