package main

import (
	"encoding/json"
	"fmt"
	"strings"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"

	flag "github.com/spf13/pflag"
)

const (
	SAMLOptionADFS     = "adfs"
	SAMLOptionOneLogin = "onelogin"
	SAMLOptionOkta     = "okta"

	LicenseOptionE10 = "e10"
	LicenseOptionE20 = "e20"

	OAuthOptionGitLab    = "gitlab"
	OAuthOptionGoogle    = "google"
	OAuthOptionOffice365 = "office365"
)

var createFlagSet *flag.FlagSet

func init() {
	createFlagSet = flag.NewFlagSet("create", flag.ContinueOnError)
	createFlagSet.String("size", "100users", "Size of the Mattermost installation e.g. '100users' or '1000users'")
	createFlagSet.String("version", "", "Mattermost version or Docker image e.g. '5.12.4' or 'mattermost/mattermost-enterprise-edition:5.12.5-rc1'")
	createFlagSet.String("affinity", "multitenant", "Whether the installation is isolated in it's own cluster or shares ones. Can be 'isolated' or 'multitenant'")
	createFlagSet.String("license", "e20", "The enterprise license to use. Can be 'e10' or 'e20'")
	createFlagSet.String("saml", "", "Set to 'onelogin', 'okta' or 'adfs' to configure SAML auth")
	createFlagSet.Bool("ldap", false, "Set to configure LDAP auth")
	createFlagSet.String("oauth", "", "Set to 'gitlab', 'google' or 'office365' to configure OAuth 2.0 auth")
	createFlagSet.Bool("test-data", false, "Set to pre-load the server with test data")
}

func parseCreateArgs(install *Installation) error {
	var err error
	install.Size, err = createFlagSet.GetString("size")
	if err != nil {
		return err
	}
	install.Version, err = createFlagSet.GetString("version")
	if err != nil {
		return err
	}
	install.Affinity, err = createFlagSet.GetString("affinity")
	if err != nil {
		return err
	}
	install.License, err = createFlagSet.GetString("license")
	if err != nil {
		return err
	}
	install.SAML, err = createFlagSet.GetString("saml")
	if err != nil {
		return err
	}
	install.LDAP, err = createFlagSet.GetBool("ldap")
	if err != nil {
		return err
	}
	install.OAuth, err = createFlagSet.GetString("oauth")
	if err != nil {
		return err
	}
	install.TestData, err = createFlagSet.GetBool("test-data")
	if err != nil {
		return err
	}
	return nil
}

func (p *Plugin) runCreateCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	install := &Installation{}

	if len(args) == 0 {
		return nil, true, fmt.Errorf("must provide an installation name")
	}

	install.Name = args[0]

	if install.Name == "" || strings.HasPrefix(install.Name, "--") {
		return nil, true, fmt.Errorf("must provide an installation name")
	}

	err := createFlagSet.Parse(args)
	if err != nil {
		return nil, false, err
	}

	err = parseCreateArgs(install)
	if err != nil {
		return nil, false, err
	}

	config := p.getConfiguration()

	license := config.E20License
	if install.License == LicenseOptionE10 {
		license = config.E10License
	}

	req := &cloud.CreateInstallationRequest{
		OwnerID:  extra.UserId,
		DNS:      fmt.Sprintf("%s.%s", install.Name, config.InstallationDNS),
		Version:  install.Version,
		Size:     install.Size,
		Affinity: install.Affinity,
		License:  license,
	}

	cloudInstallation, err := p.cloudClient.CreateInstallation(req)
	if err != nil {
		return nil, false, err
	}

	install.Installation = *cloudInstallation

	err = p.storeInstallation(install)
	if err != nil {
		return nil, false, err
	}

	data, err := json.Marshal(cloudInstallation)
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Installation being created. You will receive a notification when it is ready. Use `/cloud list` to check on the status of your installations.\n\n"+prettyPrintJSON(string(data))), false, nil
}
