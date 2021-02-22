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

	go p.processWebhookEvent(payload)

	w.WriteHeader(http.StatusOK)
}

func (p *Plugin) processWebhookEvent(payload *cloud.WebhookPayload) {
	str, err := payload.ToJSON()
	if err != nil {
		p.API.LogError(err.Error())
		return
	}
	p.API.LogDebug(str)

	switch payload.Type {
	case cloud.TypeCluster:
		err = p.handleClusterWebhook(payload)
		if err != nil {
			p.API.LogError(err.Error())
		}

		return
	case cloud.TypeInstallation:
		err = p.handleInstallationWebhook(payload)
		if err != nil {
			p.API.LogError(err.Error())
		}

		// Don't return so that any installation finalization can be processed.
	default:
		return
	}

	if payload.NewState != cloud.InstallationStateStable {
		return
	}

	install, err := p.getInstallation(payload.ID)
	if err != nil {
		p.API.LogError(err.Error(), "installation", install.Name)
		return
	}
	if install == nil {
		return
	}

	installation, err := p.cloudClient.GetInstallation(payload.ID,
		&cloud.GetInstallationRequest{
			IncludeGroupConfig:          true,
			IncludeGroupConfigOverrides: false,
		})
	if err != nil {
		p.API.LogError(err.Error(), "installation", install.Name)
		return
	}
	if installation == nil {
		p.API.LogError(fmt.Sprintf("could not find installation %s", install.ID))
	}
	install.Installation = *installation.Installation

	switch payload.OldState {
	case cloud.InstallationStateUpdateRequested,
		cloud.InstallationStateUpdateInProgress,
		cloud.InstallationStateUpdateFailed:

		install.HideSensitiveFields()

		message := fmt.Sprintf(`
Installation %s has been updated!

Installation details:
%s
`, install.Name, jsonCodeBlock(install.ToPrettyJSON()))

		p.PostBotDM(install.OwnerID, message)

	case cloud.InstallationStateCreationRequested,
		cloud.InstallationStateCreationPreProvisioning,
		cloud.InstallationStateCreationInProgress,
		cloud.InstallationStateCreationDNS,
		cloud.InstallationStateCreationNoCompatibleClusters,
		cloud.InstallationStateCreationFailed,
		cloud.InstallationStateCreationFinalTasks:

		err = p.setupInstallation(install)
		if err != nil {
			p.API.LogError(err.Error(), "installation", install.Name)
			return
		}

		install.HideSensitiveFields()

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

func (p *Plugin) handleClusterWebhook(payload *cloud.WebhookPayload) error {
	if !p.configuration.ClusterWebhookAlertsEnable {
		return nil
	}

	if payload.Type != cloud.TypeCluster {
		return fmt.Errorf("Unable to process payload type %s in 'handleClusterWebhook'", payload.Type)
	}

	message := fmt.Sprintf(`
[ Cloud Webhook ] Cluster
---
ID: %s
State: from %s to %s
`, inlineCode(payload.ID), inlineCode(payload.OldState), inlineCode(payload.NewState))

	return p.PostToChannelByIDAsBot(p.configuration.ClusterWebhookAlertsChannelID, message)
}

func (p *Plugin) handleInstallationWebhook(payload *cloud.WebhookPayload) error {
	if !p.configuration.InstallationWebhookAlertsEnable {
		return nil
	}

	if payload.Type != cloud.TypeInstallation {
		return fmt.Errorf("Unable to process payload type %s in 'handleInstallationWebhook'", payload.Type)
	}

	message := fmt.Sprintf(`
[ Cloud Webhook ] Installation
---
ID: %s
DNS: %s
ClusterID: %s
State: from %s to %s
`, inlineCode(payload.ID), inlineCode(payload.ExtraData["DNS"]), inlineCode(payload.ExtraData["ClusterID"]), inlineCode(payload.OldState), inlineCode(payload.NewState))

	return p.PostToChannelByIDAsBot(p.configuration.InstallationWebhookAlertsChannelID, message)
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
