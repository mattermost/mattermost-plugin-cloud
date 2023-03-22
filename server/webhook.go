package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	installationLogsURLTmpl = "https://grafana.internal.mattermost.com/explore?orgId=1&left=%7B%22datasource%22:%22PFB2D5CACEC34D62E%22,%22queries%22:%5B%7B%22refId%22:%22A%22,%22datasource%22:%7B%22type%22:%22loki%22,%22uid%22:%22PFB2D5CACEC34D62E%22%7D,%22editorMode%22:%22code%22,%22expr%22:%22%7Bapp%3D%5C%22mattermost%5C%22,%20namespace%3D%5C%22{{.ID}}%5C%22%7D%22,%22queryType%22:%22range%22%7D%5D,%22range%22:%7B%22from%22:%22now-1h%22,%22to%22:%22now%22%7D%7D"
	provisionerLogsURLTmpl  = "https://grafana.internal.mattermost.com/explore?orgId=1&left=%7B%22datasource%22:%22PFB2D5CACEC34D62E%22,%22queries%22:%5B%7B%22refId%22:%22A%22,%22datasource%22:%7B%22type%22:%22loki%22,%22uid%22:%22PFB2D5CACEC34D62E%22%7D,%22editorMode%22:%22code%22,%22expr%22:%22%7Bnamespace%3D%5C%22mattermost-cloud-test%5C%22,%20component%3D%5C%22provisioner%5C%22%7D%20%7C%3D%20%60{{.ID}}%60%22,%22queryType%22:%22range%22%7D%5D,%22range%22:%7B%22from%22:%22now-3h%22,%22to%22:%22now%22%7D%7D"
)

// getStringFromTemplate returns a string from a template and data provided.
func getStringFromTemplate(tmpl string, data any) (string, error) {
	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", errors.Wrap(err, "error parsing template")
	}

	var result bytes.Buffer
	err = t.Execute(&result, data)
	if err != nil {
		return "", errors.Wrap(err, "error executing template")
	}

	return result.String(), nil
}

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

	if payload.NewState != cloud.InstallationStateStable &&
		payload.NewState != cloud.InstallationStateHibernating {
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
		p.API.LogError(fmt.Sprintf("failed to find installation %s", install.ID))
		return
	}
	install.Installation = installation.Installation

	if payload.NewState == cloud.InstallationStateHibernating {
		p.PostBotDM(install.OwnerID, fmt.Sprintf("Installation %s has been hibernated", install.Name))
		return
	}

	var dnsRecord string
	if len(install.DNSRecords) > 0 {
		dnsRecord = install.DNSRecords[0].DomainName
	}

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

		installationLogsURL, err := getStringFromTemplate(installationLogsURLTmpl, install)
		if err != nil {
			p.API.LogError(err.Error(), "installation", install.Name)
			return
		}

		provisionerLogsURL, err := getStringFromTemplate(provisionerLogsURLTmpl, install)
		if err != nil {
			p.API.LogError(err.Error(), "installation", install.Name)
			return
		}

		message := fmt.Sprintf(`
Installation %s is ready!

Access at: https://%s

Login with:

| Username | Password | Note |
| -- | -- | -- |
| %s | %s | Admin user |
| %s | %s | Regular user |

Grafana logs for this installation:

- [Installation logs](%s)
- [Provisioner logs](%s)

Installation details:
%s
`, install.Name, dnsRecord, defaultAdminUsername, defaultAdminPassword, defaultUserUsername, defaultUserPassword, installationLogsURL, provisionerLogsURL, jsonCodeBlock(install.ToPrettyJSON()))

		p.PostBotDM(install.OwnerID, message)
	}
}

func (p *Plugin) handleClusterWebhook(payload *cloud.WebhookPayload) error {
	if !p.configuration.ClusterWebhookAlertsEnable {
		return nil
	}

	if payload.Type != cloud.TypeCluster {
		return fmt.Errorf("unable to process payload type %s in 'handleClusterWebhook'", payload.Type)
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
		return fmt.Errorf("unable to process payload type %s in 'handleInstallationWebhook'", payload.Type)
	}

	message := fmt.Sprintf(`
[ Cloud Webhook ] Installation
---
ID: %s
DNS: %s
ClusterID: %s
State: from %s to %s
`, inlineCode(payload.ID),
		inlineCode(payload.ExtraData["DNS"]),
		inlineCode(payload.ExtraData["ClusterID"]),
		inlineCode(payload.OldState), inlineCode(payload.NewState))

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
