package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigurationIsValid(t *testing.T) {
	baseConfiguration := configuration{
		ProvisioningServerURL:           "https://provisioner.url.com",
		InstallationDNS:                 "test.com",
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
