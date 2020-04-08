package main

import (
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-plugin-starter-template/server/mocks"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	suite.Suite

	mockedCloudClient   *mocks.MockCloudClient
	mockedDockerClient  *mocks.MockDockerClientInterface
	mockedPluginAPI     *mocks.MockAPI
	mockedPluginHelpers *mocks.MockHelpers

	ctrl *gomock.Controller

	plugin *Plugin
}

func (d *ServerTestSuite) SetupTest() {
	d.mockedCloudClient = mocks.NewMockCloudClient(d.ctrl)
	d.mockedDockerClient = mocks.NewMockDockerClientInterface(d.ctrl)
	d.mockedPluginAPI = mocks.NewMockAPI(d.ctrl)
	d.mockedPluginHelpers = mocks.NewMockHelpers(d.ctrl)

	d.plugin.cloudClient = d.mockedCloudClient
	d.plugin.dockerClient = d.mockedDockerClient
	d.plugin.MattermostPlugin = plugin.MattermostPlugin{
		API:     d.mockedPluginAPI,
		Helpers: d.mockedPluginHelpers,
	}
}

func (d *ServerTestSuite) TearDown() {
	d.ctrl.Finish()
}

func NewAWSTestSuite(t *testing.T) *ServerTestSuite {
	return &ServerTestSuite{
		ctrl: gomock.NewController(t),
		plugin: &Plugin{
			BotUserID: "bot-123-id",

			configurationLock: sync.RWMutex{},

			configuration: &configuration{
				ProvisioningServerURL:       "test-server:8586",
				ProvisioningServerAuthToken: "provisioner-auth-token-123",
				InstallationDNS:             "mattermost.com",
				AllowedEmailDomain:          "admin@mattermost.com",

				E10License: "e10-lic-123",
				E20License: "e20-lic-123",

				EmailSettings: "email.settings",

				ClusterWebhookAlertsEnable:    true,
				ClusterWebhookAlertsChannelID: "webhook-id",

				InstallationWebhookAlertsEnable:    true,
				InstallationWebhookAlertsChannelID: "intall-webhook-id",
			},
		},
	}
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, NewAWSTestSuite(t))
}
