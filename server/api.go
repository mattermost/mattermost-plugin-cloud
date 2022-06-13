package main

import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
)

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
	default:
		http.NotFound(w, r)
	}
}

// CloudUserRequest is the request type to obtain installs for a given user.
type CloudUserRequest struct {
	UserID string `json:"user_id"`
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

	installsForUser, err := p.getUpdatedInstallsForUser(req.UserID)
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to getUpdatedInstallsForUser").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(installsForUser)
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to marshal installsForUser").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write(data)
}
