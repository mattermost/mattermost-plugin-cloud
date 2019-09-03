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
		return err
	}

	err = p.createAndLoginAdminUser(client)
	if err != nil {
		return err
	}

	err = p.setupInstallationConfiguration(client, install)
	if err != nil {
		return err
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
			p.API.LogError(resp.Error.Error())
		}
		time.Sleep(time.Second * 10)
	}
	return errors.New("unable to ping installation")
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
	p.configureSAML(client, config, install, pluginConfig)
	p.configureLDAP(config, install, pluginConfig)
	p.configureOAuth(config, install, pluginConfig)

	config.ServiceSettings.EnableDeveloper = NewBool(true)
	config.TeamSettings.EnableOpenServer = NewBool(true)
	config.PluginSettings.EnableUploads = NewBool(true)

	_, resp = client.UpdateConfig(config)
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (p *Plugin) configureEmail(config *model.Config, pluginConfig *configuration) {
	err := json.Unmarshal([]byte(pluginConfig.EmailSettings), &config.EmailSettings)
	if err != nil {
		p.API.LogError("unable to unmarshal email settings err=%s", err.Error())
	}
}

func (p *Plugin) configureSAML(client *model.Client4, config *model.Config, install *Installation, pluginConfig *configuration) {
	samlSettings := ""
	idpCert := ""
	privateKey := ""
	publicCert := ""

	switch install.SAML {
	case samlOptionADFS:
		samlSettings = pluginConfig.SAMLSettingsADFS
		idpCert = pluginConfig.IDPCertADFS
		privateKey = pluginConfig.PrivateKeyADFS
		publicCert = pluginConfig.PublicCertADFS
	case samlOptionOneLogin:
		samlSettings = pluginConfig.SAMLSettingsOneLogin
		idpCert = pluginConfig.IDPCertOneLogin
		privateKey = pluginConfig.PrivateKeyOneLogin
		publicCert = pluginConfig.PublicCertOneLogin
	case samlOptionOkta:
		samlSettings = pluginConfig.SAMLSettingsOkta
		idpCert = pluginConfig.IDPCertOkta
		privateKey = pluginConfig.PrivateKeyOkta
		publicCert = pluginConfig.PublicCertOkta
	}

	if samlSettings == "" {
		return
	}

	err := json.Unmarshal([]byte(samlSettings), &config.SamlSettings)
	if err != nil {
		p.API.LogError("unable to unmarshal saml settings err=%s", err.Error())
		return
	}

	config.SamlSettings.AssertionConsumerServiceURL = NewString(fmt.Sprintf("https://%s/login/sso/saml", install.DNS))
	config.SamlSettings.IdpCertificateFile = NewString("idp.crt")
	config.SamlSettings.PrivateKeyFile = NewString("private.key")
	config.SamlSettings.PublicCertificateFile = NewString("public.crt")

	if idpCert != "" {
		_, resp := client.UploadSamlIdpCertificate([]byte(idpCert), "idp.crt")
		if resp.Error != nil {
			p.API.LogError("unable to upload IDP cert err=%s", resp.Error.Error())
			return
		}
	}

	if privateKey != "" {
		_, resp := client.UploadSamlPrivateCertificate([]byte(privateKey), "private.key")
		if resp.Error != nil {
			p.API.LogError("unable to upload private key err=%s", resp.Error.Error())
			return
		}
	}

	if publicCert != "" {
		_, resp := client.UploadSamlPublicCertificate([]byte(publicCert), "public.crt")
		if resp.Error != nil {
			p.API.LogError("unable to upload public cert err=%s", resp.Error.Error())
			return
		}
	}
}

func (p *Plugin) configureLDAP(config *model.Config, install *Installation, pluginConfig *configuration) {
	if !install.LDAP {
		return
	}
	err := json.Unmarshal([]byte(pluginConfig.LDAPSettings), &config.LdapSettings)
	if err != nil {
		p.API.LogError("unable to unmarshal ldap settings err=%s", err.Error())
	}
}

func (p *Plugin) configureOAuth(config *model.Config, install *Installation, pluginConfig *configuration) {
	if install.OAuth == oAuthOptionGitLab && pluginConfig.OAuthGitLabSettings != "" {
		err := json.Unmarshal([]byte(pluginConfig.OAuthGitLabSettings), &config.GitLabSettings)
		if err != nil {
			p.API.LogError("unable to unmarshal gitlab settings err=%s", err.Error())
		}
	}

	if install.OAuth == oAuthOptionGoogle && pluginConfig.OAuthGoogleSettings != "" {
		err := json.Unmarshal([]byte(pluginConfig.OAuthGoogleSettings), &config.GoogleSettings)
		if err != nil {
			p.API.LogError("unable to unmarshal google settings err=%s", err.Error())
		}
	}

	if install.OAuth == oAuthOptionOffice365 && pluginConfig.OAuthOffice365Settings != "" {
		err := json.Unmarshal([]byte(pluginConfig.OAuthOffice365Settings), &config.Office365Settings)
		if err != nil {
			p.API.LogError("unable to unmarshal office365 settings err=%s", err.Error())
		}
	}
}
