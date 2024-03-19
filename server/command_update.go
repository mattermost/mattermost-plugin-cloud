package main

import (
	"fmt"
	"strings"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

func getUpdateFlagSet() *flag.FlagSet {
	updateFlagSet := flag.NewFlagSet("update", flag.ContinueOnError)
	updateFlagSet.String("version", "", "Mattermost version to run, e.g. '9.1.0'")
	updateFlagSet.String("license", "", "The enterprise license to use. Can be 'enterprise', 'professional', 'e20', 'e10', or 'te'")
	updateFlagSet.String("size", "", "Size of the Mattermost installation e.g. 'miniSingleton' or 'miniHA'")
	updateFlagSet.String("image", "", fmt.Sprintf("Docker image repository, can be %s", strings.Join(dockerRepoWhitelist, ", ")))
	updateFlagSet.StringSlice("env", []string{}, "Environment variables in form: ENV1=test,ENV2=test")
	updateFlagSet.StringSlice("clear-env", []string{}, "List of custom environment variables to erase, for example: ENV1,ENV2")
	updateFlagSet.Bool("shared-installation", false, "Set this to true when attempting to update a shared installation")

	return updateFlagSet
}

func buildPatchInstallationRequestFromArgs(args []string) (*cloud.PatchInstallationRequest, bool, error) {
	updateFlagSet := getUpdateFlagSet()
	err := updateFlagSet.Parse(args)
	if err != nil {
		return nil, false, err
	}

	version, err := updateFlagSet.GetString("version")
	if err != nil {
		return nil, false, err
	}
	license, err := updateFlagSet.GetString("license")
	if err != nil {
		return nil, false, err
	}
	size, err := updateFlagSet.GetString("size")
	if err != nil {
		return nil, false, err
	}
	if size != "" && !Contains(validInstallationSizes, size) {
		return nil, false, fmt.Errorf("Invalid size: %s", size)
	}

	image, err := updateFlagSet.GetString("image")
	if err != nil {
		return nil, false, err
	}
	envVars, err := updateFlagSet.GetStringSlice("env")
	if err != nil {
		return nil, false, err
	}
	envClear, err := updateFlagSet.GetStringSlice("clear-env")
	if err != nil {
		return nil, false, err
	}
	if version == "" && license == "" && size == "" && image == "" && len(envVars) == 0 && len(envClear) == 0 {
		return nil, false, errors.New("must specify at least one option: version, license, image, size, env, clear-env")
	}
	if license != "" && !validLicenseOption(license) {
		return nil, false, errors.Errorf("invalid license option %s, valid options are %s", license, strings.Join(validLicenseOptions, ", "))
	}
	if image != "" && !validImageName(image) {
		return nil, false, errors.Errorf("invalid image name %s, valid options are %s", image, strings.Join(dockerRepoWhitelist, ", "))
	}

	envVarMap, err := parseEnvVarInput(envVars, envClear)
	if err != nil {
		return nil, false, err
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

	shared, err := updateFlagSet.GetBool("shared-installation")
	if err != nil {
		return nil, false, err
	}

	return request, shared, nil
}

// runUpdateCommand requests an update and returns the response, an
// error, and a boolean set to true if a non-nil error is returned due
// to user error, and false if the error was caused by something else.
func (p *Plugin) runUpdateCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	request, shared, err := buildPatchInstallationRequestFromArgs(args)
	if err != nil {
		return nil, true, err
	}
	var installToUpdate *Installation

	installs, err := p.getUpdatableInstallationsForUser(extra.UserId, shared)
	if err != nil {
		return nil, false, err
	}

	// Find the installation by name
	for _, install := range installs {
		if install.Name == name {
			installToUpdate = install
			break
		}
	}

	if installToUpdate == nil {
		return nil, true, errors.Errorf("no installation with the name %s found", name)
	}

	if request.Version != nil || request.Image != nil {
		dockerTag := installToUpdate.Version
		dockerRepository := installToUpdate.Image

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
		installToUpdate.Tag = dockerTag
		request.Version = &digest
	}

	// Obtain the new image value if there is one to properly apply a license.
	image := installToUpdate.Image
	if request.Image != nil {
		image = *request.Image
	}
	if request.License != nil {
		// Translate the license option.
		licenseValue := p.getLicenseValue(*request.License, image)
		request.License = &licenseValue
	}

	_, err = p.cloudClient.UpdateInstallation(installToUpdate.ID, request)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to update installation")
	}

	err = p.updateInstallation(installToUpdate)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to store updated installation metadata")
	}

	if shared {
		// Send a message to the installation owner to let them know an update
		// occured. Only log an error if there is an issue getting the update
		// requester details, but still try to send the message.
		username := "A user"
		updateRquester, err := p.API.GetUser(extra.UserId)
		if err != nil {
			p.API.LogError(errors.Wrap(err, "failed to get update request user details").Error())
		} else {
			username = fmt.Sprintf("@%s", updateRquester.Username)
		}
		p.PostBotDM(installToUpdate.OwnerID, fmt.Sprintf("%s has updated an installation you have shared. The following command was run: `%s`", username, extra.Command))
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, fmt.Sprintf("Update of installation %s has begun. You will receive a notification when it is ready. Use /cloud list to check on the status of your installations.", name), extra), false, nil
}
