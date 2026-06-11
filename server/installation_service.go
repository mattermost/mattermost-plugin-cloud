package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

type InstallationScope string

const (
	InstallationScopeMine      InstallationScope = "mine"
	InstallationScopeShared    InstallationScope = "shared"
	InstallationScopeUpdatable InstallationScope = "updatable"
)

type InstallationRef struct {
	ID   string
	Name string
}

type ListInstallationsInput struct {
	Scope          InstallationScope
	Refresh        bool
	IncludeLogURLs bool
}

type CreateInstallationInput struct {
	Name      string
	Version   string
	Size      string
	License   string
	Affinity  string
	Database  string
	Filestore string
	Image     string
	TestData  bool
	Env       map[string]string
}

type UpdateInstallationInput struct {
	Version  string
	License  string
	Size     string
	Image    string
	SetEnv   map[string]string
	ClearEnv []string
}

type InstallationSummary struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	DNS                 string `json:"dns,omitempty"`
	State               string `json:"state"`
	OwnerID             string `json:"owner_id"`
	Version             string `json:"version"`
	VersionTag          string `json:"version_tag,omitempty"`
	Image               string `json:"image,omitempty"`
	Size                string `json:"size,omitempty"`
	Database            string `json:"database,omitempty"`
	Filestore           string `json:"filestore,omitempty"`
	Affinity            string `json:"affinity,omitempty"`
	TestData            bool   `json:"test_data"`
	Shared              bool   `json:"shared"`
	AllowSharedUpdates  bool   `json:"allow_shared_updates"`
	DeletionLocked      bool   `json:"deletion_locked"`
	CreateAt            int64  `json:"create_at,omitempty"`
	ServiceEnvironment  string `json:"service_environment,omitempty"`
	InstallationLogsURL string `json:"installation_logs_url,omitempty"`
	ProvisionerLogsURL  string `json:"provisioner_logs_url,omitempty"`
}

type InstallationActionResult struct {
	Installation   InstallationSummary `json:"installation"`
	Status         string              `json:"status"`
	ChangedFields  []string            `json:"changed_fields,omitempty"`
	ChangedEnvKeys []string            `json:"changed_env_keys,omitempty"`
	ClearedEnvKeys []string            `json:"cleared_env_keys,omitempty"`
	Message        string              `json:"message,omitempty"`
}

// sanitizeInstallationCopy returns a copy of install with sensitive fields
// hidden. HideSensitiveFields only reassigns fields, so a shallow copy of the
// outer wrapper plus the embedded *cloud.Installation is enough to keep the
// caller's pointer untouched.
func sanitizeInstallationCopy(install *Installation) *Installation {
	if install == nil {
		return nil
	}

	sanitized := *install
	if install.Installation != nil {
		innerCopy := *install.Installation
		sanitized.Installation = &innerCopy
	}
	sanitized.HideSensitiveFields()
	return &sanitized
}

func sanitizeInstallationCopies(installs []*Installation) []*Installation {
	sanitized := make([]*Installation, 0, len(installs))
	for _, install := range installs {
		if sanitizedInstall := sanitizeInstallationCopy(install); sanitizedInstall != nil {
			sanitized = append(sanitized, sanitizedInstall)
		}
	}
	return sanitized
}

func installationSummary(install *Installation, includeLogURLs bool) (InstallationSummary, error) {
	if install == nil {
		return InstallationSummary{}, errors.New("installation must not be nil")
	}

	summary := InstallationSummary{
		Name:               install.Name,
		TestData:           install.TestData,
		Shared:             install.Shared,
		AllowSharedUpdates: install.AllowSharedUpdates,
		ServiceEnvironment: getInstallationServiceEnvironment(install),
	}

	if install.Installation != nil {
		summary.ID = install.ID
		summary.State = install.State
		summary.OwnerID = install.OwnerID
		summary.Version = install.Version
		summary.VersionTag = install.Tag
		summary.Image = install.Image
		summary.Size = install.Size
		summary.Database = install.Database
		summary.Filestore = install.Filestore
		summary.Affinity = install.Affinity
		summary.DeletionLocked = install.DeletionLocked
		summary.CreateAt = install.CreateAt
	}

	if len(install.DNSRecords) > 0 && install.DNSRecords[0] != nil {
		summary.DNS = install.DNSRecords[0].DomainName
	}

	if includeLogURLs && summary.ID != "" {
		logURLData := struct {
			ID string
		}{ID: summary.ID}

		installationLogsURL, err := getStringFromTemplate(installationLogsURLTmpl, logURLData)
		if err != nil {
			return InstallationSummary{}, err
		}
		provisionerLogsURL, err := getStringFromTemplate(provisionerLogsURLTmpl, logURLData)
		if err != nil {
			return InstallationSummary{}, err
		}
		summary.InstallationLogsURL = installationLogsURL
		summary.ProvisionerLogsURL = provisionerLogsURL
	}

	return summary, nil
}

func (p *Plugin) findInstallationForUser(userID string, ref InstallationRef, scope InstallationScope) (*Installation, error) {
	if err := ref.validate(); err != nil {
		return nil, err
	}

	installs, _, err := p.getInstallations()
	if err != nil {
		return nil, err
	}

	return findInstallationInSlice(userID, ref, defaultInstallationScope(scope), installs)
}

func (p *Plugin) listInstallationsForUser(userID string, input ListInstallationsInput) ([]*Installation, error) {
	scope := defaultInstallationScope(input.Scope)
	if err := validateInstallationScope(scope); err != nil {
		return nil, err
	}

	if !input.Refresh {
		installs, _, err := p.getInstallations()
		if err != nil {
			return nil, err
		}
		return filterInstallationsForScope(userID, scope, installs), nil
	}

	switch scope {
	case InstallationScopeMine:
		return p.getRefreshedInstallsForUser(userID, false)
	case InstallationScopeShared:
		return p.getUpdatedSharedInstallations(false)
	case InstallationScopeUpdatable:
		ownedInstalls, err := p.getRefreshedInstallsForUser(userID, false)
		if err != nil {
			return nil, err
		}
		sharedInstalls, err := p.getUpdatedSharedInstallations(false)
		if err != nil {
			return nil, err
		}
		return dedupeInstallations(append(ownedInstalls, filterInstallationsForScope(userID, InstallationScopeUpdatable, sharedInstalls)...)), nil
	default:
		return nil, errors.Errorf("unknown installation scope %s", scope)
	}
}

func (p *Plugin) createInstallationForUser(userID string, input CreateInstallationInput) (*Installation, error) {
	install, err := p.buildCreateInstallation(userID, input)
	if err != nil {
		return nil, err
	}

	validTag, err := p.dockerClient.ValidTag(install.Version, install.Image)
	if err != nil {
		p.API.LogError(errors.Wrapf(err, "unable to check if %s:%s exists", install.Image, install.Version).Error())
	}
	if !validTag {
		return nil, errors.Errorf("%s is not a valid docker tag for repository %s", install.Version, install.Image)
	}

	digest, err := p.dockerClient.GetDigestForTag(install.Version, install.Image)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find a manifest digest for version %s", install.Version)
	}

	install.Version = digest
	config := p.getConfiguration()
	req := &cloud.CreateInstallationRequest{
		Name:        install.Name,
		OwnerID:     userID,
		GroupID:     config.GroupID,
		Affinity:    install.Affinity,
		DNSNames:    []string{fmt.Sprintf("%s.%s", install.Name, config.InstallationDNS)},
		Database:    install.Database,
		Filestore:   install.Filestore,
		PriorityEnv: install.PriorityEnv,
		License:     p.getLicenseValue(install.License),
		Size:        install.Size,
		Version:     install.Version,
		Image:       install.Image,
		Annotations: []string{defaultMultiTenantAnnotation},
	}

	if config.ScheduledDeletionHours != "" && config.ScheduledDeletionHours != "0" {
		hours, hoursErr := strconv.Atoi(config.ScheduledDeletionHours)
		if hoursErr == nil && hours > 0 {
			req.ScheduledDeletionTime = time.Now().Add(time.Duration(hours) * time.Hour).UnixMilli()
		}
	}

	cloudInstallation, err := p.cloudClient.CreateInstallation(req)
	if err != nil {
		if strings.Contains(err.Error(), "409") {
			return nil, errors.Errorf("Installation name %s already exists. **NOTE**: installation names are reserved for 24 hours after deletion in order to support restoration. Please try a new name, wait 24 hours, or contact the Cloud Platform team for support.", install.Name)
		}
		return nil, errors.Wrap(err, "failed to create installation")
	}

	install.Installation = cloudInstallation.Installation
	if err = p.storeInstallation(install); err != nil {
		return nil, err
	}

	return install, nil
}

func (p *Plugin) updateInstallationForUser(userID string, ref InstallationRef, input UpdateInstallationInput, scope InstallationScope) (InstallationActionResult, error) {
	scope = defaultInstallationScope(scope)
	if scope == InstallationScopeShared {
		return InstallationActionResult{}, errors.New("shared scope is read-only for updates")
	}

	installToUpdate, err := p.findInstallationForUser(userID, ref, scope)
	if err != nil {
		return InstallationActionResult{}, err
	}

	request, changedFields, setEnvKeys, clearEnvKeys, requestedTag, err := p.buildUpdateInstallationRequest(installToUpdate, input)
	if err != nil {
		return InstallationActionResult{}, err
	}

	if _, err = p.cloudClient.UpdateInstallation(installToUpdate.ID, request); err != nil {
		return InstallationActionResult{}, errors.Wrap(err, "failed to update installation")
	}

	if requestedTag != "" {
		installToUpdate.Tag = requestedTag
	}
	if input.Image != "" {
		installToUpdate.Image = input.Image
	}
	if input.Size != "" {
		installToUpdate.Size = input.Size
	}

	if err = p.updateInstallation(installToUpdate); err != nil {
		return InstallationActionResult{}, errors.Wrap(err, "failed to store updated installation metadata")
	}

	summary, err := installationSummary(installToUpdate, false)
	if err != nil {
		return InstallationActionResult{}, err
	}

	return InstallationActionResult{
		Installation:   summary,
		Status:         "update_requested",
		ChangedFields:  changedFields,
		ChangedEnvKeys: setEnvKeys,
		ClearedEnvKeys: clearEnvKeys,
	}, nil
}

func (p *Plugin) restartInstallationForUser(userID string, ref InstallationRef, scope InstallationScope) (InstallationActionResult, error) {
	scope = defaultInstallationScope(scope)
	if scope == InstallationScopeShared {
		return InstallationActionResult{}, errors.New("shared scope is read-only for restarts")
	}

	installToRestart, err := p.findInstallationForUser(userID, ref, scope)
	if err != nil {
		return InstallationActionResult{}, err
	}

	patch := &cloud.PatchInstallationRequest{MattermostEnv: cloud.EnvVarMap{
		"CLOUD_PLUGIN_RESTART": cloud.EnvVar{Value: cloud.DateTimeStringFromMillis(cloud.GetMillis())},
	}}
	if _, err = p.cloudClient.UpdateInstallation(installToRestart.ID, patch); err != nil {
		return InstallationActionResult{}, err
	}

	summary, err := installationSummary(installToRestart, false)
	if err != nil {
		return InstallationActionResult{}, err
	}
	return InstallationActionResult{
		Installation:   summary,
		Status:         "restart_requested",
		ChangedEnvKeys: []string{"CLOUD_PLUGIN_RESTART"},
	}, nil
}

func (p *Plugin) hibernateInstallationForUser(userID string, ref InstallationRef) (InstallationActionResult, error) {
	installToHibernate, err := p.findRefInRefreshedList(userID, ref, InstallationScopeMine)
	if err != nil {
		return InstallationActionResult{}, err
	}
	if installToHibernate.State != cloud.InstallationStateStable {
		return InstallationActionResult{}, errors.Errorf("installation state is currently %s and must be %s to hibernate", installToHibernate.State, cloud.InstallationStateStable)
	}

	if _, err = p.cloudClient.HibernateInstallation(installToHibernate.ID); err != nil {
		return InstallationActionResult{}, err
	}

	summary, err := installationSummary(installToHibernate, false)
	if err != nil {
		return InstallationActionResult{}, err
	}
	return InstallationActionResult{Installation: summary, Status: "hibernate_requested"}, nil
}

func (p *Plugin) wakeInstallationForUser(userID string, ref InstallationRef) (InstallationActionResult, error) {
	installToWake, err := p.findRefInRefreshedList(userID, ref, InstallationScopeMine)
	if err != nil {
		return InstallationActionResult{}, err
	}
	if installToWake.State != cloud.InstallationStateHibernating {
		return InstallationActionResult{}, errors.Errorf("installation state is currently %s and must be %s to wake up", installToWake.State, cloud.InstallationStateHibernating)
	}

	if _, err = p.cloudClient.WakeupInstallation(installToWake.ID, &cloud.PatchInstallationRequest{}); err != nil {
		return InstallationActionResult{}, err
	}

	summary, err := installationSummary(installToWake, false)
	if err != nil {
		return InstallationActionResult{}, err
	}
	return InstallationActionResult{Installation: summary, Status: "wake_requested"}, nil
}

func (p *Plugin) setInstallationSharingForUser(userID string, ref InstallationRef, shared bool, allowUpdates bool) (InstallationActionResult, error) {
	install, err := p.findRefInRefreshedList(userID, ref, InstallationScopeMine)
	if err != nil {
		return InstallationActionResult{}, err
	}

	install.Shared = shared
	install.AllowSharedUpdates = allowUpdates
	if !shared {
		install.AllowSharedUpdates = false
	}

	if err = p.updateInstallation(install); err != nil {
		return InstallationActionResult{}, err
	}

	summary, err := installationSummary(install, false)
	if err != nil {
		return InstallationActionResult{}, err
	}
	return InstallationActionResult{
		Installation:  summary,
		Status:        "sharing_updated",
		ChangedFields: []string{"shared", "allow_shared_updates"},
	}, nil
}

func (p *Plugin) setDeletionLockForUser(userID string, ref InstallationRef, locked bool) (InstallationActionResult, error) {
	if ref.ID == "" && ref.Name == "" {
		return InstallationActionResult{}, errors.New("must provide an installation ID or name")
	}

	installs, err := p.listInstallationsForUser(userID, ListInstallationsInput{Scope: InstallationScopeMine, Refresh: true})
	if err != nil {
		return InstallationActionResult{}, err
	}
	if len(installs) == 0 {
		return InstallationActionResult{}, errors.New("no installations found for the given User ID")
	}

	ref.Name = standardizeName(ref.Name)
	var target *Installation
	lockedCount := 0
	for _, install := range installs {
		if install.OwnerID != userID {
			continue
		}
		if install.DeletionLocked {
			lockedCount++
		}
		if ref.matches(install) {
			target = install
		}
	}

	if target == nil {
		if locked {
			return InstallationActionResult{}, errors.New("installation to be locked not found")
		}
		return InstallationActionResult{}, errors.New("installation to be unlocked not found")
	}

	if locked {
		maxLockedInstallations, limitErr := strconv.Atoi(p.getConfiguration().DeletionLockInstallationsAllowedPerPerson)
		if limitErr != nil {
			return InstallationActionResult{}, errors.New("invalid value for DeletionLockInstallationsAllowedPerPerson")
		}
		if !target.DeletionLocked && maxLockedInstallations <= lockedCount {
			return InstallationActionResult{}, fmt.Errorf("you may only have at most %d installations locked for deletion at a time", maxLockedInstallations)
		}
		if err = p.cloudClient.LockDeletionLockForInstallation(target.ID); err != nil {
			return InstallationActionResult{}, err
		}
		target.DeletionLocked = true
	} else {
		if err = p.cloudClient.UnlockDeletionLockForInstallation(target.ID); err != nil {
			return InstallationActionResult{}, err
		}
		target.DeletionLocked = false
	}

	if err = p.updateInstallation(target); err != nil {
		return InstallationActionResult{}, errors.Wrap(err, "failed to persist deletion lock state")
	}

	summary, err := installationSummary(target, false)
	if err != nil {
		return InstallationActionResult{}, err
	}
	return InstallationActionResult{
		Installation:  summary,
		Status:        "deletion_lock_updated",
		ChangedFields: []string{"deletion_locked"},
	}, nil
}

func (p *Plugin) deleteInstallationForUser(userID string, ref InstallationRef, confirmName string) (InstallationActionResult, error) {
	installToDelete, err := p.findInstallationForUser(userID, ref, InstallationScopeMine)
	if err != nil {
		return InstallationActionResult{}, err
	}

	if standardizeName(confirmName) != standardizeName(installToDelete.Name) {
		return InstallationActionResult{}, errors.Errorf("confirmation name %s does not match installation name %s", confirmName, installToDelete.Name)
	}

	summary, err := installationSummary(installToDelete, false)
	if err != nil {
		return InstallationActionResult{}, err
	}

	if err = p.cloudClient.DeleteInstallation(installToDelete.ID); err != nil {
		return InstallationActionResult{}, err
	}
	if err = p.deleteInstallation(installToDelete.ID); err != nil {
		return InstallationActionResult{}, err
	}

	return InstallationActionResult{Installation: summary, Status: "delete_requested"}, nil
}

func (p *Plugin) buildCreateInstallation(userID string, input CreateInstallationInput) (*Installation, error) {
	config := p.getConfiguration()
	install := &Installation{
		Name: standardizeName(input.Name),
		InstallationDTO: cloud.InstallationDTO{
			Installation: &cloud.Installation{},
		},
	}

	if install.Name == "" || strings.HasPrefix(install.Name, "--") {
		return nil, errors.New("must provide an installation name")
	}
	if !validInstallationName(install.Name) {
		return nil, errors.Errorf("installation name %s is invalid: only letters, numbers, and hyphens are permitted", install.Name)
	}

	exists, err := p.installationWithNameExists(install.Name)
	if err != nil {
		return nil, errors.Wrap(err, "unable to determine if installation name is already taken")
	}
	if exists {
		return nil, errors.Errorf("Installation name %s already exists. **NOTE**: installation names are reserved for 24 hours after deletion in order to support restoration. Please try a new name, wait 24 hours, or contact the Cloud Platform team for support.", install.Name)
	}

	install.Size = defaultString(input.Size, "miniSingleton")
	if install.Size != "" && !Contains(validInstallationSizes, install.Size) {
		return nil, fmt.Errorf("Invalid size: %s", install.Size)
	}

	install.Version = defaultString(input.Version, "latest")
	if install.Version == "latest" {
		install.Version, err = p.githubLatestVersion()
		if err != nil {
			return nil, errors.Wrap(err, "failed to determine latest tag for requested version 'latest'")
		}
		if install.Version == "" {
			return nil, errors.New("failed to determine latest tag for requested version 'latest': got empty version")
		}
	}
	install.Tag = install.Version
	if err = validVersionOption(install.Version); err != nil {
		return nil, errors.Wrap(err, "Invalid version number")
	}

	install.Affinity = defaultString(input.Affinity, cloud.InstallationAffinityMultiTenant)
	if !cloud.IsSupportedAffinity(install.Affinity) {
		return nil, errors.Errorf("invalid affinity option %s, must be %s or %s", install.Affinity, cloud.InstallationAffinityIsolated, cloud.InstallationAffinityMultiTenant)
	}

	install.License = defaultString(input.License, licenseOptionEnterprise)
	if !validLicenseOption(install.License) {
		return nil, errors.Errorf("invalid license option %s, valid options are %s", install.License, strings.Join(validLicenseOptions, ", "))
	}

	install.Image = defaultString(input.Image, defaultImage)
	if !validImageName(install.Image) {
		return nil, errors.Errorf("invalid image name %s, valid options are %s", install.Image, strings.Join(dockerRepoWhitelist, ", "))
	}

	install.Database = input.Database
	if install.Database == "" {
		install.Database = config.DefaultDatabase
	}
	if install.Database == "" {
		install.Database = cloud.InstallationDatabaseMultiTenantRDSPostgresPGBouncer
	}
	if !cloud.IsSupportedDatabase(install.Database) {
		return nil, errors.Errorf("invalid database option %s; valid options are: %s, %s, %s",
			install.Database,
			cloud.InstallationDatabasePerseus,
			cloud.InstallationDatabaseMultiTenantRDSPostgresPGBouncer,
			cloud.InstallationDatabaseMysqlOperator,
		)
	}

	install.Filestore = input.Filestore
	if install.Filestore == "" {
		install.Filestore = config.DefaultFilestore
	}
	if install.Filestore == "" {
		install.Filestore = cloud.InstallationFilestoreBifrost
	}
	if !cloud.IsSupportedFilestore(install.Filestore) {
		return nil, errors.Errorf("invalid filestore option %s; must be %s, %s, %s, or %s",
			install.Filestore,
			cloud.InstallationFilestoreBifrost,
			cloud.InstallationFilestoreMinioOperator,
			cloud.InstallationFilestoreAwsS3,
			cloud.InstallationFilestoreMultiTenantAwsS3,
		)
	}
	if install.Filestore == cloud.InstallationFilestoreMultiTenantAwsS3 && install.License != licenseOptionEnterprise && install.License != licenseOptionE20 && install.License != licenseOptionEnterpriseAdvanced {
		return nil, errors.Errorf("filestore option %s requires license option %s or %s or %s", cloud.InstallationFilestoreMultiTenantAwsS3, licenseOptionEnterprise, licenseOptionE20, licenseOptionEnterpriseAdvanced)
	}

	install.TestData = input.TestData
	install.PriorityEnv = envMapFromInput(input.Env)
	install.OwnerID = userID

	return install, nil
}

func (p *Plugin) buildUpdateInstallationRequest(install *Installation, input UpdateInstallationInput) (*cloud.PatchInstallationRequest, []string, []string, []string, string, error) {
	if input.Version == "" && input.License == "" && input.Size == "" && input.Image == "" && len(input.SetEnv) == 0 && len(input.ClearEnv) == 0 {
		return nil, nil, nil, nil, "", errors.New("must specify at least one option: version, license, image, size, env, clear-env")
	}

	if input.Size != "" && !Contains(validInstallationSizes, input.Size) {
		return nil, nil, nil, nil, "", fmt.Errorf("Invalid size: %s", input.Size)
	}
	if input.License != "" && !validLicenseOption(input.License) {
		return nil, nil, nil, nil, "", errors.Errorf("invalid license option %s, valid options are %s", input.License, strings.Join(validLicenseOptions, ", "))
	}
	if input.Image != "" && !validImageName(input.Image) {
		return nil, nil, nil, nil, "", errors.Errorf("invalid image name %s, valid options are %s", input.Image, strings.Join(dockerRepoWhitelist, ", "))
	}

	request := &cloud.PatchInstallationRequest{}
	changedFields := []string{}

	if input.Size != "" {
		request.Size = &input.Size
		changedFields = append(changedFields, "size")
	}
	if input.License != "" {
		licenseValue := p.getLicenseValue(input.License)
		request.License = &licenseValue
		changedFields = append(changedFields, "license")
	}
	if input.Image != "" {
		request.Image = &input.Image
		changedFields = append(changedFields, "image")
	}

	requestedTag := ""
	if input.Version != "" || input.Image != "" {
		dockerTag := defaultString(install.Tag, install.Version)
		dockerRepository := install.Image
		if input.Version != "" {
			dockerTag = input.Version
			changedFields = append(changedFields, "version")
		}
		if input.Image != "" {
			dockerRepository = input.Image
		}
		exists, err := p.dockerClient.ValidTag(dockerTag, dockerRepository)
		if err != nil {
			p.API.LogError(errors.Wrapf(err, "unable to check if %s:%s exists", dockerRepository, dockerTag).Error())
		}
		if !exists {
			return nil, nil, nil, nil, "", errors.Errorf("%s is not a valid docker tag for repository %s", dockerTag, dockerRepository)
		}
		digest, err := p.dockerClient.GetDigestForTag(dockerTag, dockerRepository)
		if err != nil {
			return nil, nil, nil, nil, "", errors.Wrapf(err, "failed to find a manifest digest for version %s", dockerTag)
		}
		request.Version = &digest
		requestedTag = dockerTag
	}

	setEnvKeys := sortedStringMapKeys(input.SetEnv)
	clearEnvKeys := append([]string{}, input.ClearEnv...)
	sort.Strings(clearEnvKeys)
	if len(input.SetEnv) > 0 || len(input.ClearEnv) > 0 {
		request.PriorityEnv = envMapFromInput(input.SetEnv)
		if request.PriorityEnv == nil {
			request.PriorityEnv = make(cloud.EnvVarMap, len(input.ClearEnv))
		}
		for _, key := range input.ClearEnv {
			request.PriorityEnv[key] = cloud.EnvVar{}
		}
		changedFields = append(changedFields, "env")
	}

	sort.Strings(changedFields)
	return request, changedFields, setEnvKeys, clearEnvKeys, requestedTag, nil
}

func (p *Plugin) findRefInRefreshedList(userID string, ref InstallationRef, scope InstallationScope) (*Installation, error) {
	if err := ref.validate(); err != nil {
		return nil, err
	}
	installs, err := p.listInstallationsForUser(userID, ListInstallationsInput{Scope: scope, Refresh: true})
	if err != nil {
		return nil, err
	}
	return findInstallationInSlice(userID, ref, scope, installs)
}

func (ref InstallationRef) validate() error {
	if (ref.ID == "" && ref.Name == "") || (ref.ID != "" && ref.Name != "") {
		return errors.New("must provide exactly one installation id or name")
	}
	return nil
}

func (ref InstallationRef) matches(install *Installation) bool {
	if install == nil {
		return false
	}
	if ref.ID != "" {
		return install.ID == ref.ID
	}
	return standardizeName(install.Name) == standardizeName(ref.Name)
}

func findInstallationInSlice(userID string, ref InstallationRef, scope InstallationScope, installs []*Installation) (*Installation, error) {
	scope = defaultInstallationScope(scope)
	if err := validateInstallationScope(scope); err != nil {
		return nil, err
	}

	ref.Name = standardizeName(ref.Name)
	for _, install := range installs {
		if !ref.matches(install) {
			continue
		}
		if installationInScope(userID, scope, install) {
			return install, nil
		}
	}

	if ref.Name != "" {
		return nil, errors.Errorf("no installation with the name %s found", ref.Name)
	}
	return nil, errors.Errorf("no installation with the id %s found", ref.ID)
}

func filterInstallationsForScope(userID string, scope InstallationScope, installs []*Installation) []*Installation {
	filtered := []*Installation{}
	for _, install := range installs {
		if installationInScope(userID, scope, install) {
			filtered = append(filtered, install)
		}
	}
	return filtered
}

func installationInScope(userID string, scope InstallationScope, install *Installation) bool {
	if install == nil {
		return false
	}

	switch scope {
	case InstallationScopeMine:
		return install.OwnerID == userID
	case InstallationScopeShared:
		return install.Shared
	case InstallationScopeUpdatable:
		return install.OwnerID == userID || (install.Shared && install.AllowSharedUpdates)
	default:
		return false
	}
}

func defaultInstallationScope(scope InstallationScope) InstallationScope {
	if scope == "" {
		return InstallationScopeMine
	}
	return scope
}

func validateInstallationScope(scope InstallationScope) error {
	switch scope {
	case InstallationScopeMine, InstallationScopeShared, InstallationScopeUpdatable:
		return nil
	default:
		return errors.Errorf("unknown installation scope %s", scope)
	}
}

func dedupeInstallations(installs []*Installation) []*Installation {
	seen := map[string]bool{}
	deduped := []*Installation{}
	for _, install := range installs {
		if install == nil || seen[install.ID] {
			continue
		}
		seen[install.ID] = true
		deduped = append(deduped, install)
	}
	return deduped
}

func defaultString(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}

func envMapFromInput(input map[string]string) cloud.EnvVarMap {
	if len(input) == 0 {
		return nil
	}

	env := make(cloud.EnvVarMap, len(input))
	for key, value := range input {
		env[key] = cloud.EnvVar{Value: value}
	}
	return env
}

func sortedStringMapKeys(input map[string]string) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
