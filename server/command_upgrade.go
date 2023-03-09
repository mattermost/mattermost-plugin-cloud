package main

import (
	"fmt"
	"strings"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

func getUpgradeFlagSet() *flag.FlagSet {
	upgradeFlagSet := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	upgradeFlagSet.String("version", "", "Mattermost version to run, e.g. '5.12.4'")
	upgradeFlagSet.String("license", "", "The enterprise license to use. Can be 'e10', 'e20', or 'te'")
	upgradeFlagSet.String("size", "", "Size of the Mattermost installation e.g. 'miniSingleton' or 'miniHA'")
	upgradeFlagSet.String("image", "", fmt.Sprintf("Docker image repository, can be %s", strings.Join(dockerRepoWhitelist, ", ")))
	upgradeFlagSet.StringSlice("env", []string{}, "Environment variables in form: ENV1=test,ENV2=test")
	upgradeFlagSet.StringSlice("clear-env", []string{}, "List of custom environment variables to erase, for example: ENV1,ENV2")

	return upgradeFlagSet
}

func buildPatchInstallationRequestFromArgs(args []string) (*cloud.PatchInstallationRequest, error) {
	upgradeFlagSet := getUpgradeFlagSet()
	err := upgradeFlagSet.Parse(args)
	if err != nil {
		return nil, err
	}

	version, err := upgradeFlagSet.GetString("version")
	if err != nil {
		return nil, err
	}
	license, err := upgradeFlagSet.GetString("license")
	if err != nil {
		return nil, err
	}
	size, err := upgradeFlagSet.GetString("size")
	if err != nil {
		return nil, err
	}
	if size != "" && !Contains(validInstallationSizes, size) {
		return nil, fmt.Errorf("Invalid size: %s", size)
	}

	image, err := upgradeFlagSet.GetString("image")
	if err != nil {
		return nil, err
	}
	envVars, err := upgradeFlagSet.GetStringSlice("env")
	if err != nil {
		return nil, err
	}
	envClear, err := upgradeFlagSet.GetStringSlice("clear-env")
	if err != nil {
		return nil, err
	}
	if version == "" && license == "" && size == "" && image == "" && len(envVars) == 0 && len(envClear) == 0 {
		return nil, errors.New("must specify at least one option: version, license, image, size, env, clear-env")
	}
	if license != "" && !validLicenseOption(license) {
		return nil, errors.Errorf("invalid license option %s, valid options are %s", license, strings.Join(validLicenseOptions, ", "))
	}
	if image != "" && !validImageName(image) {
		return nil, errors.Errorf("invalid image name %s, valid options are %s", image, strings.Join(dockerRepoWhitelist, ", "))
	}

	envVarMap, err := parseEnvVarInput(envVars, envClear)
	if err != nil {
		return nil, err
	}

	request := &cloud.PatchInstallationRequest{
		PriorityEnv: envVarMap,
	}
	if version != "" {
		request.Version = &version
	}
	if license != "" {
		request.License = &license
	}
	if size != "" {
		request.Size = &size
	}
	if image != "" {
		request.Image = &image
	}

	return request, nil
}

// runUpgradeCommand requests an upgrade and returns the response, an
// error, and a boolean set to true if a non-nil error is returned due
// to user error, and false if the error was caused by something else.
func (p *Plugin) runUpgradeCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
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
		return nil, true, errors.Errorf("no installation with the name %s found", name)
	}

	request, err := buildPatchInstallationRequestFromArgs(args)
	if err != nil {
		return nil, true, err
	}

	if request.Version != nil || request.Image != nil {
		dockerTag := installToUpgrade.Version
		dockerRepository := installToUpgrade.Image

		if request.Version != nil {
			dockerTag = *request.Version
		}
		if request.Image != nil {
			dockerRepository = *request.Image
		}
		// Check that new version exists.
		var exists bool
		exists, err = p.dockerClient.ValidTag(dockerTag, dockerRepository)
		if err != nil {
			p.API.LogError(errors.Wrapf(err, "unable to check if %s:%s exists", dockerRepository, dockerTag).Error())
		}
		if !exists {
			return nil, true, errors.Errorf("%s is not a valid docker tag for repository %s", dockerTag, dockerRepository)
		}
		var digest string
		digest, err = p.dockerClient.GetDigestForTag(dockerTag, dockerRepository)
		if err != nil {
			return nil, false, errors.Wrapf(err, "failed to find a manifest digest for version %s", dockerTag)
		}
		installToUpgrade.Tag = dockerTag
		request.Version = &digest
	}

	if request.License != nil {
		// Translate the license option.
		licenseValue := p.getLicenseValue(*request.License)
		request.License = &licenseValue
	}

	_, err = p.cloudClient.UpdateInstallation(installToUpgrade.ID, request)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to update installation")
	}

	err = p.updateInstallation(installToUpgrade)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to store updated installation metadata")
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, fmt.Sprintf("Upgrade of installation %s has begun. You will receive a notification when it is ready. Use /cloud list to check on the status of your installations.", name), extra), false, nil
}
