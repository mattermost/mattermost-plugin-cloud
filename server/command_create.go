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
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"

	flag "github.com/spf13/pflag"
)

const (
	defaultMultiTenantAnnotation = "multi-tenant"
)

var installationNameMatcher = regexp.MustCompile(`^[a-zA-Z0-9-]*$`)

var validInstallationSizes = []string{"miniSingleton", "miniHA"}

type latestMattermostVersionCache struct {
	version   string
	timestamp time.Time
}

func (p *Plugin) getCreateFlagSet() *flag.FlagSet {
	config := p.getConfiguration()
	defaultFileStore := config.DefaultFilestore
	if defaultFileStore == "" {
		defaultFileStore = cloud.InstallationFilestoreBifrost
	}
	defaultDatabase := config.DefaultDatabase
	if defaultDatabase == "" {
		defaultDatabase = cloud.InstallationDatabaseMultiTenantRDSPostgresPGBouncer
	}

	createFlagSet := flag.NewFlagSet("create", flag.ContinueOnError)
	createFlagSet.String("size", "miniSingleton", "Size of the Mattermost installation e.g. 'miniSingleton' or 'miniHA'")
	createFlagSet.String("version", "latest", "Mattermost version to run, e.g. '9.1.0'")
	createFlagSet.String("affinity", cloud.InstallationAffinityMultiTenant, "Whether the installation is isolated in it's own cluster or shares ones. Can be 'isolated' or 'multitenant'")
	createFlagSet.String("license", licenseOptionEnterprise, "The Mattermost license to use. Can be 'enterprise', 'enterprise-advanced', 'professional', 'e20', 'e10', or 'te'")
	createFlagSet.String("filestore", defaultFileStore, "Specify the backing file store. Can be 'bifrost' (S3 Shared Bucket), 'aws-multitenant-s3' (S3 Shared Bucket), 'aws-s3' (S3 Bucket).")
	createFlagSet.String("database", defaultDatabase, "Specify the backing database. Can be 'aws-multitenant-rds-postgres-pgbouncer' (RDS Postgres with pgbouncer proxy connections), 'aws-rds' (RDS MySQL).")
	createFlagSet.Bool("test-data", false, "Set to pre-load the server with test data")
	createFlagSet.String("image", defaultImage, fmt.Sprintf("Docker image repository. Can be %s", strings.Join(dockerRepoWhitelist, ", ")))
	createFlagSet.StringSlice("env", []string{}, "Environment variables in form: ENV1=test,ENV2=test")
	return createFlagSet
}

func (p *Plugin) runCreateCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 {
		return nil, true, errors.New("must provide an installation name")
	}

	input, err := p.createInstallationInputFromArgs(args)
	if err != nil {
		return nil, true, err
	}

	install, err := p.createInstallationForUser(extra.UserId, input)
	if err != nil {
		if isCreateUserError(err) {
			return nil, true, err
		}
		return nil, false, err
	}

	install = sanitizeInstallationCopy(install)

	return getCommandResponse(model.CommandResponseTypeEphemeral, "Installation being created. You will receive a notification when it is ready. Use `/cloud list` to check on the status of your installations.\n\n"+jsonCodeBlock(install.ToPrettyJSON()), extra), false, nil
}

func (p *Plugin) createInstallationInputFromArgs(args []string) (CreateInstallationInput, error) {
	if len(args) == 0 || args[0] == "" || strings.HasPrefix(args[0], "--") {
		return CreateInstallationInput{}, errors.New("must provide an installation name")
	}

	createFlagSet := p.getCreateFlagSet()
	if err := createFlagSet.Parse(args); err != nil {
		return CreateInstallationInput{}, err
	}

	envVars, err := createFlagSet.GetStringSlice("env")
	if err != nil {
		return CreateInstallationInput{}, err
	}
	envMap, err := parseEnvVarInput(envVars, nil)
	if err != nil {
		return CreateInstallationInput{}, err
	}

	input := CreateInstallationInput{Name: args[0], Env: map[string]string{}}
	input.Size, err = createFlagSet.GetString("size")
	if err != nil {
		return CreateInstallationInput{}, err
	}
	input.Version, err = createFlagSet.GetString("version")
	if err != nil {
		return CreateInstallationInput{}, err
	}
	input.Affinity, err = createFlagSet.GetString("affinity")
	if err != nil {
		return CreateInstallationInput{}, err
	}
	input.License, err = createFlagSet.GetString("license")
	if err != nil {
		return CreateInstallationInput{}, err
	}
	input.Image, err = createFlagSet.GetString("image")
	if err != nil {
		return CreateInstallationInput{}, err
	}
	input.Database, err = createFlagSet.GetString("database")
	if err != nil {
		return CreateInstallationInput{}, err
	}
	input.Filestore, err = createFlagSet.GetString("filestore")
	if err != nil {
		return CreateInstallationInput{}, err
	}
	input.TestData, err = createFlagSet.GetBool("test-data")
	if err != nil {
		return CreateInstallationInput{}, err
	}
	for key, env := range envMap {
		input.Env[key] = env.Value
	}

	return input, nil
}

func isCreateUserError(err error) bool {
	errText := err.Error()
	return strings.Contains(errText, "must provide an installation name") ||
		strings.Contains(errText, "is invalid: only letters, numbers, and hyphens are permitted") ||
		strings.Contains(errText, "already exists") ||
		strings.Contains(errText, "Invalid version number") ||
		strings.Contains(errText, "is not a valid docker tag") ||
		strings.Contains(errText, "Invalid size:") ||
		strings.Contains(errText, "invalid affinity option") ||
		strings.Contains(errText, "invalid license option") ||
		strings.Contains(errText, "invalid image name") ||
		strings.Contains(errText, "invalid database option") ||
		strings.Contains(errText, "invalid filestore option") ||
		strings.Contains(errText, "requires license option") ||
		strings.Contains(errText, "valid env format") ||
		strings.Contains(errText, "defined more than once")
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
