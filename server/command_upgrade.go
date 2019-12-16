package main

import (
	"fmt"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

const (
	dockerRepository = "mattermost/mattermost-enterprise-edition"
)

func getUpgradeFlagSet() *flag.FlagSet {
	upgradeFlagSet := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	upgradeFlagSet.String("version", "", "Mattermost version to run, e.g. '5.12.4'")
	upgradeFlagSet.String("license", "", "The enterprise license to use. Can be 'e10', 'e20', or 'te'")

	return upgradeFlagSet
}

func parseUpgradeArgs(args []string) (string, string, error) {
	upgradeFlagSet := getUpgradeFlagSet()
	err := upgradeFlagSet.Parse(args)
	if err != nil {
		return "", "", err
	}

	version, err := upgradeFlagSet.GetString("version")
	if err != nil {
		return "", "", err
	}

	license, err := upgradeFlagSet.GetString("license")
	if err != nil {
		return "", "", err
	}
	if license != "" && !validLicenseOption(license) {
		return "", "", fmt.Errorf("invalid license option %s; must be %s, %s or %s", license, licenseOptionE10, licenseOptionE20, licenseOptionTE)
	}

	if version == "" && license == "" {
		return "", "", errors.New("must specify at least one option: license or version")
	}

	return version, license, nil
}

// runUpgradeCommand requests an upgrade and returns the response, an error, and a boolean set to true if a non-nil error is returned due to user error, and false if the error was caused by something else.
func (p *Plugin) runUpgradeCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, fmt.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	installs, _, err := p.getInstallations()
	if err != nil {
		return nil, false, err
	}

	var installToUpgrade *Installation
	for _, install := range installs {
		if install.OwnerID == extra.UserId && install.Name == name {
			installToUpgrade = install
			break
		}
	}

	if installToUpgrade == nil {
		return nil, true, fmt.Errorf("no installation with the name %s found", name)
	}

	version, newLicense, err := parseUpgradeArgs(args)
	if err != nil {
		return nil, true, err
	}

	// Use the current version if none was provided.
	if version == "" {
		version = installToUpgrade.Version
	}

	exists, err := p.dockerClient.ValidTag(version, dockerRepository)
	if err != nil {
		p.API.LogError(errors.Wrapf(err, "unable to check if %s:%s exists", dockerRepository, version).Error())
	}
	if !exists {
		return nil, true, fmt.Errorf("%s is not a valid docker tag for repository %s", version, dockerRepository)
	}

	config := p.getConfiguration()

	// Only change the license if a value was provided.
	license := installToUpgrade.License
	if newLicense != "" {
		license = config.E20License
		if newLicense == licenseOptionE10 {
			license = config.E10License
		} else if newLicense == licenseOptionTE {
			license = ""
		}
	}

	upgradeRequest := &cloud.UpgradeInstallationRequest{
		Version: version,
		License: license,
	}

	err = p.cloudClient.UpgradeInstallation(installToUpgrade.ID, upgradeRequest)
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Upgrade of installation %s has begun. You will receive a notification when it is ready. Use /cloud list to check on the status of your installations.", name)), false, nil
}
