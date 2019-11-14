package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"

	flag "github.com/spf13/pflag"
)

var installationNameMatcher = regexp.MustCompile(`^[a-zA-Z0-9-]*$`)

func getCreateFlagSet() *flag.FlagSet {
	createFlagSet := flag.NewFlagSet("create", flag.ContinueOnError)
	createFlagSet.String("size", "miniSingleton", "Size of the Mattermost installation e.g. 'miniSingleton' or 'miniHA'")
	createFlagSet.String("version", "", "Mattermost version to run, e.g. '5.12.4'")
	createFlagSet.String("affinity", "multitenant", "Whether the installation is isolated in it's own cluster or shares ones. Can be 'isolated' or 'multitenant'")
	createFlagSet.String("license", "e20", "The enterprise license to use. Can be 'e10' or 'e20'")
	createFlagSet.String("storage", "cloud", "Specify the backing database stores. Can be 'cloud' to use Amazon RDS and S3 or 'local' to use the MySQL and Minio Operators inside the cluster")
	createFlagSet.Bool("test-data", false, "Set to pre-load the server with test data")

	return createFlagSet
}

// parseCreateArgs is responsible for reading in arguments and basic input validity checking
func parseCreateArgs(args []string, install *Installation) error {
	createFlagSet := getCreateFlagSet()
	err := createFlagSet.Parse(args)
	if err != nil {
		return err
	}

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

	if !validLicenseOption(install.License) {
		return fmt.Errorf("invalid license option %s, must be %s or %s", install.License, licenseOptionE10, licenseOptionE20)
	}

	install.StorageType, err = createFlagSet.GetString("storage")
	if err != nil {
		return err
	}

	if !validStorageOption(install.StorageType) {
		return fmt.Errorf("invalid storage option %s; must be %s or %s", install.StorageType, storageOptionAWS, storageOptionOperator)
	}

	install.TestData, err = createFlagSet.GetBool("test-data")
	if err != nil {
		return err
	}

	return nil
}

func (p *Plugin) runCreateCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 {
		return nil, true, fmt.Errorf("must provide an installation name")
	}

	install := &Installation{
		Name: args[0],
	}

	if install.Name == "" || strings.HasPrefix(install.Name, "--") {
		return nil, true, fmt.Errorf("must provide an installation name")
	}
	if !validInstallationName(install.Name) {
		return nil, true, fmt.Errorf("installation name %s is invalid: only letters, numbers, and hyphens are permitted", install.Name)
	}

	err := parseCreateArgs(args, install)
	if err != nil {
		return nil, true, err
	}

	config := p.getConfiguration()

	license := config.E20License
	if install.License == licenseOptionE10 {
		license = config.E10License
	}

	if install.Version != "" {
		var exists bool
		repository := "mattermost/mattermost-enterprise-edition"
		exists, err = p.dockerClient.ValidTag(install.Version, repository)
		if err != nil {
			p.API.LogError(errors.Wrapf(err, "unable to check if %s:%s exists", repository, install.Version).Error())
		}
		if !exists {
			return nil, true, fmt.Errorf("%s is not a valid docker tag for repository %s", install.Version, repository)
		}
	}

	// determine filestore and database type
	// TODO break this into a helper func if it ever gets more complex
	var filestore string
	var database string
	switch install.StorageType {
	case storageOptionAWS:
		filestore = cloud.InstallationFilestoreAwsS3
		database = cloud.InstallationDatabaseAwsRDS
	case storageOptionOperator:
		filestore = cloud.InstallationFilestoreMinioOperator
		database = cloud.InstallationDatabaseMysqlOperator
	default:
		// Nota bene: it shouldn't be possible to have an invalid storage type here, as long as validation on available types was properly performed in parseCreateArgs. Hitting this error probably means there was a regression!
		return nil, false, fmt.Errorf("storage type %s is not valid", install.StorageType)
	}

	req := &cloud.CreateInstallationRequest{
		Affinity:  install.Affinity,
		DNS:       fmt.Sprintf("%s.%s", install.Name, config.InstallationDNS),
		Database:  database,
		Filestore: filestore,
		License:   license,
		OwnerID:   extra.UserId,
		Size:      install.Size,
		Version:   install.Version,
	}

	cloudInstallation, err := p.cloudClient.CreateInstallation(req)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to create installation")
	}

	install.Installation = *cloudInstallation

	err = p.storeInstallation(install)
	if err != nil {
		return nil, false, err
	}

	cloudInstallation.License = "hidden"

	data, err := json.Marshal(cloudInstallation)
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Installation being created. You will receive a notification when it is ready. Use `/cloud list` to check on the status of your installations.\n\n"+jsonCodeBlock(prettyPrintJSON(string(data)))), false, nil
}

func validInstallationName(name string) bool {
	return installationNameMatcher.MatchString(name)
}
