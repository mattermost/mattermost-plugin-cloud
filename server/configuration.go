package main

import (
	"net/url"
	"reflect"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	licenseOptionEnterprise   = "enterprise"
	licenseOptionProfessional = "professional"
	licenseOptionE20          = "e20"
	licenseOptionE10          = "e10"
	licenseOptionTE           = "te"

	imageEE          = "mattermost/mattermost-enterprise-edition"
	imageEECloud     = "mattermost/mm-ee-cloud"
	imageTE          = "mattermost/mm-te"
	imageTeamEdition = "mattermost/mattermost-team-edition"
	imageEETest      = "mattermostdevelopment/mm-ee-test"
	imageTETest      = "mattermostdevelopment/mm-te-test"

	defaultImage = "mattermost/mattermost-enterprise-edition"
)

var validLicenseOptions = []string{
	licenseOptionEnterprise,
	licenseOptionProfessional,
	licenseOptionE20,
	licenseOptionE10,
	licenseOptionTE,
}

// dockerRepoWhitelist is the full list of valid docker repositories which
// Mattermost servers can be created from.
var dockerRepoWhitelist = []string{
	imageEE,
	imageEECloud,
	imageTE,
	imageTeamEdition,
	imageEETest,
	imageTETest,
}

// dockerRepoTestImages are repositories that contain artifacts used primarily
// for internal testing. They may require configuration overrides such as
// special licenses.
var dockerRepoTestImages = []string{
	imageEETest,
}

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.
type configuration struct {
	ProvisioningServerURL                     string
	ProvisioningServerAuthToken               string
	InstallationDNS                           string
	AllowedEmailDomain                        string
	DeletionLockInstallationsAllowedPerPerson int

	// License
	E10License              string
	E20License              string
	EnterpriseLicense       string
	ProfessionalLicense     string
	TestEnterpriseLicense   string
	TestProfessionalLicense string

	// Groups
	GroupID string

	// Email
	// Note: email settings are only used when group configuration is empty.
	EmailSettings string

	// Webhook Alerts
	ClusterWebhookAlertsEnable    bool
	ClusterWebhookAlertsChannelID string

	InstallationWebhookAlertsEnable    bool
	InstallationWebhookAlertsChannelID string

	DefaultDatabase  string
	DefaultFilestore string
}

// Clone shallow copies the configuration. Your implementation may require a deep copy if
// your configuration has reference types.
func (c *configuration) Clone() *configuration {
	var clone = *c
	return &clone
}

func (c *configuration) IsValid() error {
	if len(c.ProvisioningServerURL) == 0 {
		return errors.New("must specify ProvisioningServerURL")
	}
	_, err := url.Parse(c.ProvisioningServerURL)
	if err != nil {
		return errors.Wrap(err, "invalid ProvisioningServerURL")
	}

	if len(c.InstallationDNS) == 0 {
		return errors.New("must specify InstallationDNS")
	}

	if len(c.GroupID) != 0 && len(c.GroupID) != 26 {
		return errors.Errorf("group IDs are 26 characters long, the provided ID was %d", len(c.GroupID))
	}

	if c.ClusterWebhookAlertsEnable {
		if len(c.ClusterWebhookAlertsChannelID) == 0 {
			return errors.Errorf("must specify a cluster alerts channel ID when cluster alerts are enabled")
		}
	}

	if c.InstallationWebhookAlertsEnable {
		if len(c.InstallationWebhookAlertsChannelID) == 0 {
			return errors.Errorf("must specify an installation alerts channel ID when installation alerts are enabled")
		}
	}

	return nil
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		// Ignore assignment if the configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	var configuration = new(configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	if p.configuration != nil {
		p.setCloudClient()
	}

	p.setConfiguration(configuration)

	return nil
}

func (p *Plugin) setCloudClient() {
	configuration := p.getConfiguration()

	if configuration.ProvisioningServerAuthToken == "" {
		p.cloudClient = cloud.NewClient(configuration.ProvisioningServerURL)
		return
	}

	authHeaders := map[string]string{"x-api-key": configuration.ProvisioningServerAuthToken}
	p.cloudClient = cloud.NewClientWithHeaders(configuration.ProvisioningServerURL, authHeaders)
}

func (p *Plugin) getLicenseValue(licenseOption, image string) string {
	config := p.getConfiguration()

	switch licenseOption {
	case licenseOptionEnterprise:
		for _, ti := range dockerRepoTestImages {
			if ti == image {
				return config.TestEnterpriseLicense
			}
		}
		return config.EnterpriseLicense
	case licenseOptionProfessional:
		for _, ti := range dockerRepoTestImages {
			if ti == image {
				return config.TestProfessionalLicense
			}
		}
		return config.ProfessionalLicense
	case licenseOptionE20:
		return config.E20License
	case licenseOptionE10:
		return config.E10License
	}

	return ""
}
