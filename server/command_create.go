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
	createFlagSet.String("license", "e20", "The enterprise license to use. Can be 'e10', 'e20', or 'te'")
	createFlagSet.String("filestore", "aws-s3", "Specify the backing file store. Can be 'aws-s3' to use Amazon S3 or 'operator' to use the Minio Operator inside the cluster")
	createFlagSet.String("database", "aws-rds", "Specify the backing database. Can be 'aws-rds' to use Amazon RDS or 'operator' to use the MySQL Operator inside the cluster")
	createFlagSet.Bool("test-data", false, "Set to pre-load the server with test data")

	return createFlagSet
}

// parseCreateArgs is responsible for reading in arguments and basic input validation
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

	if !cloud.IsSupportedAffinity(install.Affinity) {
		return fmt.Errorf("invalid affinity option %s, must be %s or %s", install.Affinity, cloud.InstallationAffinityIsolated, cloud.InstallationAffinityMultiTenant)
	}

	install.License, err = createFlagSet.GetString("license")
	if err != nil {
		return err
	}

	if !validLicenseOption(install.License) {
		return fmt.Errorf("invalid license option %s, must be %s, %s or %s", install.License, licenseOptionE10, licenseOptionE20, licenseOptionTE)
	}

	install.Database, err = createFlagSet.GetString("database")
	if err != nil {
		return err
	}

	if !validDatabaseOption(install.Database) {
		return fmt.Errorf("invalid database option %s; must be %s or %s", install.Database, databaseOptionRDS, databaseOptionOperator)
	}

	install.Filestore, err = createFlagSet.GetString("filestore")
	if err != nil {
		return err
	}

	if !validFilestoreOption(install.Filestore) {
		return fmt.Errorf("invalid filestore option %s; must be %s or %s", install.Filestore, filestoreOptionS3, filestoreOptionOperator)
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
		Name: standardizeName(args[0]),
	}

	if install.Name == "" || strings.HasPrefix(install.Name, "--") {
		return nil, true, fmt.Errorf("must provide an installation name")
	}

	if !validInstallationName(install.Name) {
		return nil, true, fmt.Errorf("installation name %s is invalid: only letters, numbers, and hyphens are permitted", install.Name)
	}

	exists, err := p.installationWithNameExists(install.Name)
	if err != nil || exists {
		if err != nil {
			return nil, false, err
		}
		return nil, true, fmt.Errorf("Installation name %s already exists. Names are case insensitive and must be unique so you must choose a new name and try again", install.Name)
	}

	err = parseCreateArgs(args, install)
	if err != nil {
		return nil, true, err
	}

	config := p.getConfiguration()

	license := ""
	if install.License == licenseOptionE10 {
		license = config.E10License
	} else if install.License == licenseOptionE20 {
		license = config.E20License
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

	database := ""
	if install.Database == databaseOptionRDS {
		database = cloud.InstallationDatabaseAwsRDS
	} else if install.Database == databaseOptionOperator {
		database = cloud.InstallationDatabaseMysqlOperator
	}

	if len(database) == 0 {
		return nil, false, fmt.Errorf("could not determine database type; provided database type was %s", install.Database)
	}

	var filestore string
	if install.Filestore == filestoreOptionS3 {
		filestore = cloud.InstallationFilestoreAwsS3
	} else if install.Filestore == filestoreOptionOperator {
		filestore = cloud.InstallationFilestoreMinioOperator
	}

	if len(filestore) == 0 {
		return nil, false, fmt.Errorf("could not determine filestore type; provided filestore type was %s", install.Filestore)
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

// installationWithNameExists returns true when there already exists an installation with name "name"
func (p *Plugin) installationWithNameExists(name string) (bool, error) {
	existing, _, err := p.getInstallations()
	if err != nil {
		return false, errors.Wrap(err, "trouble looking up existing installations")
	}

	for _, i := range existing {
		// FIXME standardizing these here really shouldn't be necessary if everything is stored in the correct format, but better safe than sorry until we can find a better approach
		if standardizeName(name) == standardizeName(i.Name) {
			return true, nil
		}
	}

	return false, nil
}
