package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

func getUpdateFlagSet() *flag.FlagSet {
	updateFlagSet := flag.NewFlagSet("update", flag.ContinueOnError)
	updateFlagSet.String("version", "", "Mattermost version to run, e.g. '9.1.0'")
	updateFlagSet.String("license", "", "The Enterprise license to use. Can be 'enterprise-advanced', 'enterprise', 'professional', 'e20', 'e10', or 'te'")
	updateFlagSet.String("size", "", "Size of the Mattermost installation e.g. 'miniSingleton' or 'miniHA'")
	updateFlagSet.String("image", "", fmt.Sprintf("Docker image repository, can be %s", strings.Join(dockerRepoWhitelist, ", ")))
	updateFlagSet.StringSlice("env", []string{}, "Environment variables in form: ENV1=test,ENV2=test")
	updateFlagSet.StringSlice("clear-env", []string{}, "List of custom environment variables to erase, for example: ENV1,ENV2")
	updateFlagSet.Bool("shared-installation", false, "Set this to true when attempting to update a shared installation")

	return updateFlagSet
}

// runUpdateCommand requests an update and returns the response, an
// error, and a boolean set to true if a non-nil error is returned due
// to user error, and false if the error was caused by something else.
func (p *Plugin) runUpdateCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	input, shared, err := updateInstallationInputFromArgs(args)
	if err != nil {
		return nil, true, err
	}

	scope := InstallationScopeMine
	if shared {
		scope = InstallationScopeUpdatable
	}
	result, err := p.updateInstallationForUser(extra.UserId, InstallationRef{Name: name}, input, scope)
	if err != nil {
		if isUpdateUserError(err) {
			return nil, true, err
		}
		return nil, false, err
	}

	if shared {
		// Send a message to the installation owner to let them know an update
		// occurred. Only log an error if there is an issue getting the update.
		// requester details, but still try to send the message.
		username := "A user"
		updateRequester, err := p.API.GetUser(extra.UserId)
		if err != nil {
			p.API.LogError(errors.Wrap(err, "failed to get update request user details").Error())
		} else {
			username = fmt.Sprintf("@%s", updateRequester.Username)
		}
		p.PostBotDM(result.Installation.OwnerID, fmt.Sprintf("%s has updated an installation you have shared. The following command was run: `%s`", username, extra.Command))
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, fmt.Sprintf("Update of installation %s has begun. You will receive a notification when it is ready. Use /cloud list to check on the status of your installations.", name), extra), false, nil
}

func updateInstallationInputFromArgs(args []string) (UpdateInstallationInput, bool, error) {
	updateFlagSet := getUpdateFlagSet()
	if err := updateFlagSet.Parse(args); err != nil {
		return UpdateInstallationInput{}, false, err
	}

	input := UpdateInstallationInput{}
	var err error
	input.Version, err = updateFlagSet.GetString("version")
	if err != nil {
		return UpdateInstallationInput{}, false, err
	}
	input.License, err = updateFlagSet.GetString("license")
	if err != nil {
		return UpdateInstallationInput{}, false, err
	}
	input.Size, err = updateFlagSet.GetString("size")
	if err != nil {
		return UpdateInstallationInput{}, false, err
	}
	input.Image, err = updateFlagSet.GetString("image")
	if err != nil {
		return UpdateInstallationInput{}, false, err
	}

	envVars, err := updateFlagSet.GetStringSlice("env")
	if err != nil {
		return UpdateInstallationInput{}, false, err
	}
	input.ClearEnv, err = updateFlagSet.GetStringSlice("clear-env")
	if err != nil {
		return UpdateInstallationInput{}, false, err
	}
	envVarMap, err := parseEnvVarInput(envVars, input.ClearEnv)
	if err != nil {
		return UpdateInstallationInput{}, false, err
	}
	input.SetEnv = map[string]string{}
	for key, env := range envVarMap {
		if env.HasValue() {
			input.SetEnv[key] = env.Value
		}
	}

	shared, err := updateFlagSet.GetBool("shared-installation")
	if err != nil {
		return UpdateInstallationInput{}, false, err
	}

	return input, shared, nil
}

func isUpdateUserError(err error) bool {
	errText := err.Error()
	return strings.Contains(errText, "must specify at least one option") ||
		strings.Contains(errText, "Invalid size:") ||
		strings.Contains(errText, "invalid license option") ||
		strings.Contains(errText, "invalid image name") ||
		strings.Contains(errText, "valid env format") ||
		strings.Contains(errText, "defined more than once") ||
		strings.Contains(errText, "no installation with the name") ||
		strings.Contains(errText, "is not a valid docker tag")
}
