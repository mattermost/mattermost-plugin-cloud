package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"

	flag "github.com/spf13/pflag"
)

const (
	defaultMultiTenantAnnotation = "multi-tenant"
)

var dockerRepoWhitelist = []string{
	"mattermost/mattermost-enterprise-edition",
	"mattermost/mm-ee-test",
	"mattermost/mm-ee-cloud",
	"mattermost/mm-te",
	"mattermost/mattermost-team-edition",
}
var installationNameMatcher = regexp.MustCompile(`^[a-zA-Z0-9-]*$`)

type latestMattermostVersionCache struct {
	version   string
	timestamp time.Time
}

func getCreateFlagSet() *flag.FlagSet {
	createFlagSet := flag.NewFlagSet("create", flag.ContinueOnError)
	createFlagSet.String("size", "miniSingleton", "Size of the Mattermost installation e.g. 'miniSingleton' or 'miniHA'")
	createFlagSet.String("version", "latest", "Mattermost version to run, e.g. '5.12.4'")
	createFlagSet.String("affinity", cloud.InstallationAffinityMultiTenant, "Whether the installation is isolated in it's own cluster or shares ones. Can be 'isolated' or 'multitenant'")
	createFlagSet.String("license", licenseOptionE20, "The enterprise license to use. Can be 'e10', 'e20', or 'te'")
	createFlagSet.String("filestore", cloud.InstallationFilestoreBifrost, "Specify the backing file store. Can be 'aws-multitenant-s3' (S3 Shared Bucket), 'aws-s3' (S3 Bucket), 'operator' (Minio Operator inside the cluster. Default 'aws-multi-tenant-s3' for E20, and 'aws-s3' for E10 and E0/TE.")
	createFlagSet.String("database", cloud.InstallationDatabaseMultiTenantRDSPostgresPGBouncer, "Specify the backing database. Can be 'perseus' (RDS Postgres with perseus proxy connections), 'aws-multitenant-rds-postgres-pgbouncer' (RDS Postgres with pgbouncer proxy connections), 'aws-multitenant-rds-postgres' (RDS Postgres Shared), 'aws-multitenant-rds' (RDS MySQL Shared), 'aws-rds-postgres' (RDS Postgres), 'aws-rds' (RDS MySQL), 'mysql-operator' (MySQL Operator inside the cluster)")
	createFlagSet.Bool("test-data", false, "Set to pre-load the server with test data")
	createFlagSet.String("image", defaultImage, fmt.Sprintf("Docker image repository. Can be %s", strings.Join(dockerRepoWhitelist, ", ")))
	createFlagSet.StringSlice("env", []string{}, "Environment variables in form: ENV1=test,ENV2=test")
	return createFlagSet
}

// parseCreateArgs is responsible for reading in arguments and basic input validation
func (p *Plugin) parseCreateArgs(args []string, install *Installation) error {
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

	if install.Version == "latest" {
		install.Version, err = p.githubLatestVersion()
		if err != nil {
			return errors.Wrap(err, "failed to determine latest tag for requested version 'latest'")
		}
		if install.Version == "" {
			return errors.New("failed to determine latest tag for requested version 'latest': got empty version")
		}
	}
	install.Tag = install.Version

	install.Affinity, err = createFlagSet.GetString("affinity")
	if err != nil {
		return err
	}

	if !cloud.IsSupportedAffinity(install.Affinity) {
		return errors.Errorf("invalid affinity option %s, must be %s or %s", install.Affinity, cloud.InstallationAffinityIsolated, cloud.InstallationAffinityMultiTenant)
	}

	install.License, err = createFlagSet.GetString("license")
	if err != nil {
		return err
	}

	if !validLicenseOption(install.License) {
		return errors.Errorf("invalid license option %s, must be %s, %s or %s", install.License, licenseOptionE10, licenseOptionE20, licenseOptionTE)
	}

	install.Image, err = createFlagSet.GetString("image")
	if err != nil {
		return err
	}

	if !validImageName(install.Image) {
		return errors.Errorf("invalid image name %s, valid options are %s", install.Image, strings.Join(dockerRepoWhitelist, ", "))
	}

	install.Database, err = createFlagSet.GetString("database")
	if err != nil {
		return err
	}

	if !cloud.IsSupportedDatabase(install.Database) {
		return errors.Errorf("invalid database option %s; valid options are: %s, %s, %s, %s, %s, %s, %s",
			install.Database,
			cloud.InstallationDatabasePerseus,
			cloud.InstallationDatabaseMultiTenantRDSPostgresPGBouncer,
			cloud.InstallationDatabaseMysqlOperator,
			cloud.InstallationDatabaseSingleTenantRDSMySQL,
			cloud.InstallationDatabaseSingleTenantRDSPostgres,
			cloud.InstallationDatabaseMultiTenantRDSMySQL,
			cloud.InstallationDatabaseMultiTenantRDSPostgres,
		)
	}

	install.Filestore, err = createFlagSet.GetString("filestore")
	if err != nil {
		return err
	}

	// the filestore has a different default depending upon the target installation type
	if install.Filestore == "" {
		if install.License == licenseOptionE20 {
			install.Filestore = cloud.InstallationFilestoreMultiTenantAwsS3
		} else {
			install.Filestore = cloud.InstallationFilestoreAwsS3
		}
	}

	if !cloud.IsSupportedFilestore(install.Filestore) {
		return errors.Errorf("invalid filestore option %s; must be %s, %s, or %s",
			install.Filestore,
			cloud.InstallationFilestoreMinioOperator,
			cloud.InstallationFilestoreAwsS3,
			cloud.InstallationFilestoreMultiTenantAwsS3,
		)
	}

	if install.Filestore == cloud.InstallationFilestoreMultiTenantAwsS3 && install.License != licenseOptionE20 {
		return errors.Errorf("filestore option %s requires license option %s", cloud.InstallationFilestoreMultiTenantAwsS3, licenseOptionE20)
	}

	install.TestData, err = createFlagSet.GetBool("test-data")
	if err != nil {
		return err
	}

	envVars, err := createFlagSet.GetStringSlice("env")
	if err != nil {
		return err
	}
	envVarMap, err := parseEnvVarInput(envVars, nil)
	if err != nil {
		return err
	}
	install.Installation.PriorityEnv = envVarMap

	return nil
}

func (p *Plugin) runCreateCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 {
		return nil, true, errors.New("must provide an installation name")
	}

	install := &Installation{
		Name: standardizeName(args[0]),
		InstallationDTO: cloud.InstallationDTO{
			Installation: &cloud.Installation{},
		},
	}

	if install.Name == "" || strings.HasPrefix(install.Name, "--") {
		return nil, true, errors.New("must provide an installation name")
	}

	if !validInstallationName(install.Name) {
		return nil, true, errors.Errorf("installation name %s is invalid: only letters, numbers, and hyphens are permitted", install.Name)
	}

	exists, err := p.installationWithNameExists(install.Name)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to determine if installation name is already taken")
	}
	if exists {
		return nil, true, errors.Errorf("Installation name %s already exists. Names are case insensitive and must be unique so you must choose a new name and try again", install.Name)
	}

	err = p.parseCreateArgs(args, install)
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

	err = validVersionOption(install.Version)
	if err != nil {
		return nil, true, errors.Wrap(err, "Invalid version number")
	}

	validTag, err := p.dockerClient.ValidTag(install.Version, install.Image)
	if err != nil {
		p.API.LogError(errors.Wrapf(err, "unable to check if %s:%s exists", install.Image, install.Version).Error())
	}
	if !validTag {
		return nil, true, errors.Errorf("%s is not a valid docker tag for repository %s", install.Version, install.Image)
	}

	var digest string
	digest, err = p.dockerClient.GetDigestForTag(install.Version, install.Image)
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to find a manifest digest for version %s", install.Version)
	}
	install.Version = digest

	req := &cloud.CreateInstallationRequest{
		Name:        install.Name,
		OwnerID:     extra.UserId,
		GroupID:     config.GroupID,
		Affinity:    install.Affinity,
		DNSNames:    []string{fmt.Sprintf("%s.%s", install.Name, config.InstallationDNS)},
		Database:    install.Database,
		Filestore:   install.Filestore,
		PriorityEnv: install.PriorityEnv,
		License:     license,
		Size:        install.Size,
		Version:     install.Version,
		Image:       install.Image,
		Annotations: []string{defaultMultiTenantAnnotation},
	}

	cloudInstallation, err := p.cloudClient.CreateInstallation(req)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to create installation")
	}

	install.Installation = cloudInstallation.Installation

	err = p.storeInstallation(install)
	if err != nil {
		return nil, false, err
	}

	install.HideSensitiveFields()

	return getCommandResponse(model.CommandResponseTypeEphemeral, "Installation being created. You will receive a notification when it is ready. Use `/cloud list` to check on the status of your installations.\n\n"+jsonCodeBlock(install.ToPrettyJSON()), extra), false, nil
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

func validImageName(imageName string) bool {
	for _, image := range dockerRepoWhitelist {
		if image == imageName {
			return true
		}
	}
	return false
}

type githubReleaseMetadata struct {
	TagName string `json:"tag_name"`
}

func (p *Plugin) githubLatestVersion() (string, error) {

	// avoids Github rate limiting for unauthenticated requests
	if p.latestMattermostVersion != nil &&
		p.latestMattermostVersion.version != "" &&
		p.latestMattermostVersion.timestamp.After(time.Now().Add(time.Minute*time.Duration(-5))) {

		return p.latestMattermostVersion.version, nil
	}

	// else version is more than five minutes old or doesn't exist, so get it from Github
	// use the releases endpoint and not releases/latest to avoid getting a dot release
	resp, err := http.Get("https://api.github.com/repos/mattermost/mattermost-server/releases")
	if err != nil {
		return "", errors.Wrap(err, "failed to find latest release from GitHub")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("got unexpected status code %d while determining latest release from GitHub", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response body")
	}

	grm := []githubReleaseMetadata{}
	err = json.Unmarshal(body, &grm)
	if err != nil {
		return "", errors.Wrap(err, "failed to unmarshal JSON from GitHub to determine latest release")
	}

	var (
		latestTag        string
		latestTagVersion semver.Version
	)

	for _, release := range grm {
		if release.TagName == "" {
			continue
		}
		currentTag := strings.TrimPrefix(release.TagName, "v")
		currentTagVersion, err := semver.Parse(currentTag)
		if err != nil {
			p.API.LogError(err.Error())
			continue
		}

		if latestTag == "" || currentTagVersion.GE(latestTagVersion) {
			latestTag = currentTag
			latestTagVersion = currentTagVersion
			continue
		}
	}

	if latestTag == "" {
		return "", errors.New("failed to determine latest version of Mattermost")
	}

	p.latestMattermostVersion =
		&latestMattermostVersionCache{
			timestamp: time.Now(),
			version:   latestTag,
		}

	return latestTag, nil
}

func parseEnvVarInput(rawInput []string, clearEnvs []string) (cloud.EnvVarMap, error) {
	if len(rawInput) == 0 && len(clearEnvs) == 0 {
		return nil, nil
	}

	envVarMap := make(cloud.EnvVarMap)

	for _, env := range rawInput {
		// Split the input once by "=" to allow for multiple "="s to be in the
		// value. Expect there to still be one key and value.
		kv := strings.SplitN(env, "=", 2)
		if len(kv) != 2 || len(kv[0]) == 0 {
			return nil, errors.Errorf("%s is not in a valid env format; expecting KEY_NAME=VALUE", env)
		}

		if _, ok := envVarMap[kv[0]]; ok {
			return nil, errors.Errorf("env var %s was defined more than once", kv[0])
		}

		envVarMap[kv[0]] = cloud.EnvVar{Value: kv[1]}
	}

	// Clearing envs take precedence over setting them
	for _, env := range clearEnvs {
		envVarMap[env] = cloud.EnvVar{}
	}

	return envVarMap, nil
}
