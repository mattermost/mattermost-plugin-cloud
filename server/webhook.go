package main

import (
	"bytes"
	"crypto/subtle"
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
	installationLogsURLTmpl = `https://grafana.internal.mattermost.com/explore?orgId=1&left={"datasource":"PFB2D5CACEC34D62E","queries":[{"refId":"A","datasource":{"type":"loki","uid":"PFB2D5CACEC34D62E"},"editorMode":"code","expr":"{app=\"mattermost\", namespace=\"{{.ID}}\"}","queryType":"range"}],"range":{"from":"now-1h","to":"now"}}`
	provisionerLogsURLTmpl  = `https://grafana.internal.mattermost.com/explore?orgId=1&left={"datasource":"PFB2D5CACEC34D62E","queries":[{"refId":"A","datasource":{"type":"loki","uid":"PFB2D5CACEC34D62E"},"editorMode":"code","expr":"{namespace=\"mattermost-cloud-test\", component=\"provisioner\"} |= %60{{.ID}}%60","queryType":"range"}],"range":{"from":"now-3h","to":"now"}}`

	authHeaderKey = "X-MM-Cloud-Plugin-Auth-Token"
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

func (p *Plugin) authenticateWebhook(r *http.Request) error {
	token := r.Header.Get(authHeaderKey)

	if equal := subtle.ConstantTimeCompare([]byte(token), []byte(p.configuration.ProvisioningServerWebhookSecret)); equal != 1 {
		return errors.New("unauthorized")
	}

	return nil
}

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if err := p.authenticateWebhook(r); err != nil {
		p.API.LogError(errors.Wrap(err, "provisioner webhook authentication failed").Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

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
		payload.NewState != cloud.InstallationStateHibernating &&
		payload.NewState != cloud.InstallationStateDeletionPending &&
		payload.NewState != cloud.InstallationStateDeleted {
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
	install.HideSensitiveFields()

	if payload.NewState == cloud.InstallationStateHibernating {
		p.PostBotDM(install.OwnerID, fmt.Sprintf("Installation %s has been hibernated", install.Name))
		return
	}

	if payload.NewState == cloud.InstallationStateDeletionPending {
		if payload.ExtraData["actor_id"] == p.configuration.ProvisioningServerClientID {
			p.PostBotDM(install.OwnerID, fmt.Sprintf("Installation %s is pending final deletion. If this was a mistake, please contact the Cloud Platform team within 24 hours of this message, or your data will be lost forever.", install.Name))
			return
		}
		p.PostBotDM(install.OwnerID, fmt.Sprintf("Installation %s has automatically been moved to pending deletion state. If you believe this to be a mistake, please contact the Cloud Platform team for restoration. You have 24 hours to initiate before your data is lost forever.", install.Name))
		return
	}

	if payload.NewState == cloud.InstallationStateDeleted {
		p.PostBotDM(install.OwnerID, fmt.Sprintf("Installation %s has been deleted", install.Name))
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

		adminPassword := generateRandomPassword(defaultAdminUsername)
		userPassword := generateRandomPassword(defaultUserUsername)
		err = p.setupInstallation(install, adminPassword, userPassword)
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
`,
			install.Name,
			dnsRecord,
			inlineCode(defaultAdminUsername), inlineCode(adminPassword),
			inlineCode(defaultUserUsername), inlineCode(userPassword),
			installationLogsURL, provisionerLogsURL,
			jsonCodeBlock(install.ToPrettyJSON()),
		)

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
