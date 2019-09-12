package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	cloud "github.com/mattermost/mattermost-cloud/model"
)

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := cloud.WebhookPayloadFromReader(r.Body)
	if err != nil {
		p.API.LogError(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Respond to the cloud server that we have accepted the webhook event.
	w.WriteHeader(http.StatusOK)

	str, _ := payload.ToJSON()
	p.API.LogDebug(str)

	if payload.Type != "installation" {
		return
	}

	if payload.NewState != cloud.InstallationStateStable {
		return
	}

	install, err := p.getInstallation(payload.ID)
	if err != nil {
		p.API.LogError(err.Error())
		return
	}
	if install == nil {
		return
	}

	var installation *cloud.Installation

	switch payload.OldState {
	case cloud.InstallationStateUpgradeRequested,
		cloud.InstallationStateUpgradeInProgress,
		cloud.InstallationStateUpgradeFailed:
		installation, err = p.cloudClient.GetInstallation(payload.ID)
		if err != nil {
			p.API.LogError(err.Error())
			return
		}

		install.Installation = *installation

		message := fmt.Sprintf(`
Installation %s has been upgraded!

Installation details:
%s
`, install.Name, jsonCodeBlock(install.ToPrettyJSON()))

		p.PostBotDM(install.OwnerID, message)
	case cloud.InstallationStateCreationRequested,
		cloud.InstallationStateCreationDNS,
		cloud.InstallationStateCreationFailed:
		err = p.setupInstallation(install)
		if err != nil {
			p.API.LogError(err.Error())
			return
		}

		message := fmt.Sprintf(`
Installation %s is ready!

Access at: https://%s

Login with:

| Username | Password |
| -- | -- |
| %s | %s |

Installation details:
%s
`, install.Name, install.DNS, defaultAdminUsername, defaultAdminPassword, jsonCodeBlock(install.ToPrettyJSON()))

		p.PostBotDM(install.OwnerID, message)
	}
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
