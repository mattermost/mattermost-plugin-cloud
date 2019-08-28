package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
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
	default:
		http.NotFound(w, r)
	}
}

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := cloud.WebhookPayloadFromReader(r.Body)
	if err != nil {
		p.API.LogError(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	str, _ := payload.ToJSON()
	p.API.LogDebug(str)

	if payload.Type != "installation" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if payload.OldState != cloud.InstallationStateCreationRequested || payload.NewState != cloud.InstallationStateStable {
		w.WriteHeader(http.StatusOK)
		return
	}

	install, err := p.getInstallation(payload.ID)
	if err != nil {
		p.API.LogError(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if install == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	err = p.setupInstallation(install)
	if err != nil {
		p.API.LogError(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	message := fmt.Sprintf(`
Installation %s Ready!

Access at: https://%s

Login with:

| Username | Password |
| -- | -- |
| %s | %s |

Installation details:
%s
`, install.Name, install.DNS, DefaultAdminUsername, DefaultAdminPassword, install.ToPrettyJSON())

	p.PostBotDM(install.OwnerID, message)

	w.WriteHeader(http.StatusOK)
}

func (p *Plugin) handleProfileImage(w http.ResponseWriter, r *http.Request) {
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		p.API.LogError("Unable to get bundle path, err=" + err.Error())
		return
	}

	img, err := os.Open(filepath.Join(bundlePath, "assets", "profile.png"))
	if err != nil {
		http.NotFound(w, r)
		p.API.LogError("Unable to read profile image, err=" + err.Error())
		return
	}
	defer img.Close()

	w.Header().Set("Content-Type", "image/png")
	io.Copy(w, img)
}
