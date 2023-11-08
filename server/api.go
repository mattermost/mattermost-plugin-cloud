package main

import (
	"encoding/json"
	"net/http"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
)

const serviceEnvironmentEnvVarKey = "MM_SERVICEENVIRONMENT"

// InstallationWebWrapper embeds the standard plugin installation object with
// some extra ephemeral fields used in the webapp.
type InstallationWebWrapper struct {
	*Installation
	CreateAtDate        string
	ServiceEnvironment  string
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
	i.HideSensitiveFields()

	return &InstallationWebWrapper{
		Installation:        i,
		CreateAtDate:        cloud.DateStringFromMillis(i.CreateAt),
		ServiceEnvironment:  serviceEnvironment,
		InstallationLogsURL: installationLogsURL,
		ProvisionerLogsURL:  provisionerLogsURL,
	}, nil
}

// ServeHTTP handles HTTP requests to the plugin.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
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
	case "/api/v1/deletion-lock":
		p.handleDeletionLock(w, r)
	case "/api/v1/deletion-unlock":
		p.handleDeletionUnlock(w, r)
	case "/api/v1/config":
		p.handleGetConfig(w, r)
	default:
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

	req := &CloudUserRequest{}
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

func (p *Plugin) handleDeletionLock(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	req := &CloudDeletionLockRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.InstallationID == "" {
		if err != nil {
			p.API.LogError(errors.Wrap(err, "Unable to decode cloud deletion lock request").Error())
		}

		http.Error(w, "Please provide a JSON object with a non-blank installation_id field", http.StatusBadRequest)
		return
	}

	// Lock for deletion needs to fetch all installations and ensure ownership to validate
	// users aren't locking more than one, so there's no need to check ownership here.
	err = p.lockForDeletion(req.InstallationID, userID)
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to lock for deletion").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	j, err := json.Marshal(req)
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to marshal cloud deletion lock request").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(j)
}

func (p *Plugin) handleDeletionUnlock(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	req := &CloudDeletionLockRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.InstallationID == "" {
		if err != nil {
			p.API.LogError(errors.Wrap(err, "Unable to decode cloud deletion unlock request").Error())
		}

		http.Error(w, "Please provide a JSON object with a non-blank installation_id field", http.StatusBadRequest)
		return
	}

	// Unlock for deletion needs to fetch all installations and ensure ownership to validate
	// users aren't unlocking more than one, so there's no need to check ownership here.
	err = p.unlockForDeletion(req.InstallationID, userID)
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to unlock for deletion").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	j, err := json.Marshal(req)
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to marshal cloud deletion unlock request").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(j)
}

func (p *Plugin) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	data, err := json.Marshal(config.ToConfigResponse())
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to marshal config").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

func getInstallationServiceEnvironment(installation *Installation) string {
	if v, ok := installation.PriorityEnv[serviceEnvironmentEnvVarKey]; ok {
		return v.Value
	}
	if v, ok := installation.MattermostEnv[serviceEnvironmentEnvVarKey]; ok {
		return v.Value
	}

	return "production"
}
