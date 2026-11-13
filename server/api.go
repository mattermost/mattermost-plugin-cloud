package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/pkg/errors"
)

const serviceEnvironmentEnvVarKey = "MM_SERVICEENVIRONMENT"

// InstallationWebWrapper embeds the standard plugin installation object with
// some extra ephemeral fields used in the webapp.
type InstallationWebWrapper struct {
	*Installation
	CreateAtDate        string
	ServiceEnvironment  string
	EnvVarNames         []string
	InstallationLogsURL string
	ProvisionerLogsURL  string
}

// CreateInstallationWebWrapper creates a new InstallationWebWrapper from a
// standard Installation.
func CreateInstallationWebWrapper(i *Installation) (*InstallationWebWrapper, error) {
	installationLogsURL, err := getStringFromTemplate(installationLogsURLTmpl, i)
	if err != nil {
		return nil, err
	}

	provisionerLogsURL, err := getStringFromTemplate(provisionerLogsURLTmpl, i)
	if err != nil {
		return nil, err
	}

	serviceEnvironment := getInstallationServiceEnvironment(i)
	sanitizedInstall := sanitizeInstallationCopy(i)

	return &InstallationWebWrapper{
		Installation:        sanitizedInstall,
		CreateAtDate:        cloud.DateStringFromMillis(sanitizedInstall.CreateAt),
		ServiceEnvironment:  serviceEnvironment,
		EnvVarNames:         getInstallationEnvVarNames(i),
		InstallationLogsURL: installationLogsURL,
		ProvisionerLogsURL:  provisionerLogsURL,
	}, nil
}

// ServeHTTP handles HTTP requests to the plugin.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	if p.serveMCPIfMatch(w, r) {
		return
	}

	config := p.getConfiguration()

	if err := config.IsValid(); err != nil {
		http.Error(w, "This plugin is not configured.", http.StatusNotImplemented)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch path := r.URL.Path; path {
	case "/webhook":
		p.handleWebhook(w, r)
	case "/profile.png":
		p.handleProfileImage(w, r)
	case "/api/v1/userinstalls":
		p.handleUserInstalls(w, r)
	case "/api/v1/sharedinstalls":
		p.handleSharedInstalls(w, r)
	case "/api/v1/deletion-lock":
		p.handleDeletionLock(w, r)
	case "/api/v1/deletion-unlock":
		p.handleDeletionUnlock(w, r)
	case "/api/v1/config":
		p.handleGetConfig(w, r)
	default:
		if action, ok := strings.CutPrefix(path, "/api/v1/installations/"); ok {
			p.handleInstallationAction(w, r, action)
			return
		}
		http.NotFound(w, r)
	}
}

// CloudUserRequest is the request type to obtain installs for a given user.
type CloudUserRequest struct {
	UserID string `json:"user_id"`
}

// CloudDeletionLockRequest is the request type to lock or unlock an installation for deletion.
type CloudDeletionLockRequest struct {
	InstallationID string `json:"installation_id"`
}

func (p *Plugin) handleUserInstalls(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	var req CloudUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.UserID == "" {
		if err != nil {
			p.API.LogError(errors.Wrap(err, "Unable to decode cloud user request").Error())
		}

		http.Error(w, "Please provide a JSON object with a non-blank user_id field", http.StatusBadRequest)
		return
	}

	installsForUser, err := p.getUpdatedInstallsForUserWithSensitive(req.UserID)
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to getUpdatedInstallsForUserWithSensitive").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var webInstalls []*InstallationWebWrapper
	for _, install := range installsForUser {
		webInstall, wrapErr := CreateInstallationWebWrapper(install)
		if wrapErr != nil {
			p.API.LogError(errors.Wrapf(wrapErr, "Unable to CreateInstallationWebWrapper for %s", install.Name).Error())
			continue
		}
		webInstalls = append(webInstalls, webInstall)
	}

	data, err := json.Marshal(webInstalls)
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to marshal webInstalls").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

func (p *Plugin) handleSharedInstalls(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}
	// Let's require users to also have plugin permissions to view the shared
	// list. It should be fine to allow for all logged in users to see the list,
	// but let's start with this.
	if !p.authorizedPluginUser(userID) {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	sharedInstalls, err := p.getUpdatedSharedInstallations(false)
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to getUpdatedSharedInstallations").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var webInstalls []*InstallationWebWrapper
	for _, install := range sharedInstalls {
		webInstall, wrapErr := CreateInstallationWebWrapper(install)
		if wrapErr != nil {
			p.API.LogError(errors.Wrapf(wrapErr, "Unable to CreateInstallationWebWrapper for %s", install.Name).Error())
			continue
		}
		webInstalls = append(webInstalls, webInstall)
	}

	data, err := json.Marshal(webInstalls)
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to marshal webInstalls").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

func (p *Plugin) handleDeletionLock(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	var req CloudDeletionLockRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.InstallationID == "" {
		if err != nil {
			p.API.LogError(errors.Wrap(err, "Unable to decode cloud deletion lock request").Error())
		}

		http.Error(w, "Please provide a JSON object with a non-blank installation_id field", http.StatusBadRequest)
		return
	}

	result, err := p.setDeletionLockForUser(userID, InstallationRef{ID: req.InstallationID}, true)
	if err != nil {
		p.writeInstallationError(w, "lock for deletion", err)
		return
	}

	j, err := json.Marshal(req)
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to marshal cloud deletion lock request").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	p.publishInstallationEvent(summaryEvent(result.Installation))
	w.WriteHeader(http.StatusCreated)
	w.Write(j)
}

func (p *Plugin) handleDeletionUnlock(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	var req CloudDeletionLockRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.InstallationID == "" {
		if err != nil {
			p.API.LogError(errors.Wrap(err, "Unable to decode cloud deletion unlock request").Error())
		}

		http.Error(w, "Please provide a JSON object with a non-blank installation_id field", http.StatusBadRequest)
		return
	}

	result, err := p.setDeletionLockForUser(userID, InstallationRef{ID: req.InstallationID}, false)
	if err != nil {
		p.writeInstallationError(w, "unlock for deletion", err)
		return
	}

	j, err := json.Marshal(req)
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to marshal cloud deletion unlock request").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	p.publishInstallationEvent(summaryEvent(result.Installation))
	w.WriteHeader(http.StatusCreated)
	w.Write(j)
}

func (p *Plugin) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	config := p.getConfiguration()

	data, err := json.Marshal(config.ToConfigResponse())
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to marshal config").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// getInstallationEnvVarNames returns the sorted names of the environment
// variables set on the installation, from both MattermostEnv and PriorityEnv.
// Values are never returned because they can hold secrets. The service
// environment has its own field, so it is left out of the list.
func getInstallationEnvVarNames(installation *Installation) []string {
	names := []string{}
	if installation == nil || installation.Installation == nil {
		return names
	}
	seen := map[string]struct{}{}
	for _, envMap := range []cloud.EnvVarMap{installation.MattermostEnv, installation.PriorityEnv} {
		for name := range envMap {
			if _, ok := seen[name]; ok || name == serviceEnvironmentEnvVarKey {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func getInstallationServiceEnvironment(installation *Installation) string {
	if installation == nil || installation.Installation == nil {
		return "production"
	}
	if v, ok := installation.PriorityEnv[serviceEnvironmentEnvVarKey]; ok {
		return v.Value
	}
	if v, ok := installation.MattermostEnv[serviceEnvironmentEnvVarKey]; ok {
		return v.Value
	}

	return "production"
}
