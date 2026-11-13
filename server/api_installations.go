package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
)

const (
	maxInstallationRequestBytes = 1 << 20

	// websocketEventInstallationUpdated is delivered to webapp clients as
	// "custom_<plugin id>_installation_updated".
	websocketEventInstallationUpdated = "installation_updated"
)

// InstallationEvent is the websocket payload sent when an installation changes.
// It holds identifiers and state only, so it is safe to send to every user for
// a shared installation.
type InstallationEvent struct {
	InstallationID string `json:"installation_id"`
	OwnerID        string `json:"owner_id"`
	State          string `json:"state"`
	Shared         bool   `json:"shared"`
}

// InstallationCreateRequest is the request type to create an installation.
type InstallationCreateRequest struct {
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	Size      string            `json:"size"`
	License   string            `json:"license"`
	Affinity  string            `json:"affinity"`
	Database  string            `json:"database"`
	Filestore string            `json:"filestore"`
	Image     string            `json:"image"`
	TestData  bool              `json:"test_data"`
	Env       map[string]string `json:"env"`
}

// InstallationUpdateRequest is the request type to update an installation.
// Empty fields stay unchanged.
type InstallationUpdateRequest struct {
	InstallationID string            `json:"installation_id"`
	Version        string            `json:"version"`
	License        string            `json:"license"`
	Size           string            `json:"size"`
	Image          string            `json:"image"`
	SetEnv         map[string]string `json:"set_env"`
	ClearEnv       []string          `json:"clear_env"`
}

// InstallationDeleteRequest is the request type to delete an installation.
type InstallationDeleteRequest struct {
	InstallationID string `json:"installation_id"`
	ConfirmName    string `json:"confirm_name"`
}

// InstallationShareRequest is the request type to change the sharing settings
// of an installation.
type InstallationShareRequest struct {
	InstallationID string `json:"installation_id"`
	Shared         bool   `json:"shared"`
	AllowUpdates   bool   `json:"allow_updates"`
}

func (p *Plugin) handleInstallationAction(w http.ResponseWriter, r *http.Request, action string) {
	switch action {
	case "create":
		p.handleInstallationCreate(w, r)
	case "update":
		p.handleInstallationUpdate(w, r)
	case "restart":
		p.handleInstallationSimpleAction(w, r, func(userID, id string) (InstallationActionResult, error) {
			return p.restartInstallationForUser(userID, InstallationRef{ID: id}, InstallationScopeUpdatable)
		})
	case "hibernate":
		p.handleInstallationSimpleAction(w, r, func(userID, id string) (InstallationActionResult, error) {
			return p.hibernateInstallationForUser(userID, InstallationRef{ID: id})
		})
	case "wake":
		p.handleInstallationSimpleAction(w, r, func(userID, id string) (InstallationActionResult, error) {
			return p.wakeInstallationForUser(userID, InstallationRef{ID: id})
		})
	case "delete":
		p.handleInstallationDelete(w, r)
	case "share":
		p.handleInstallationShare(w, r)
	default:
		http.NotFound(w, r)
	}
}

// authorizeInstallationRequest validates the method and the user, and decodes
// the JSON body into dst. It writes the error response and returns false when
// the request must not continue.
func (p *Plugin) authorizeInstallationRequest(w http.ResponseWriter, r *http.Request, dst any) (string, bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return "", false
	}

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" || !p.authorizedPluginUser(userID) {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return "", false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxInstallationRequestBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to decode installation request").Error())
		http.Error(w, "Please provide a valid JSON object", http.StatusBadRequest)
		return "", false
	}

	return userID, true
}

func (p *Plugin) handleInstallationCreate(w http.ResponseWriter, r *http.Request) {
	var req InstallationCreateRequest
	userID, ok := p.authorizeInstallationRequest(w, r, &req)
	if !ok {
		return
	}

	install, err := p.createInstallationForUser(userID, CreateInstallationInput(req))
	if err != nil {
		p.writeInstallationError(w, "create installation", err)
		return
	}

	summary, err := installationSummary(sanitizeInstallationCopy(install), false)
	if err != nil {
		p.writeInstallationError(w, "summarize installation", err)
		return
	}

	p.publishInstallationEvent(installationEventFor(install, install.State))
	p.writeInstallationJSON(w, http.StatusCreated, InstallationActionResult{Installation: summary, Status: "create_requested"})
}

func (p *Plugin) handleInstallationUpdate(w http.ResponseWriter, r *http.Request) {
	var req InstallationUpdateRequest
	userID, ok := p.authorizeInstallationRequest(w, r, &req)
	if !ok {
		return
	}
	if req.InstallationID == "" {
		http.Error(w, "Please provide a non-blank installation_id field", http.StatusBadRequest)
		return
	}

	result, err := p.updateInstallationForUser(userID, InstallationRef{ID: req.InstallationID}, UpdateInstallationInput{
		Version:  req.Version,
		License:  req.License,
		Size:     req.Size,
		Image:    req.Image,
		SetEnv:   req.SetEnv,
		ClearEnv: req.ClearEnv,
	}, InstallationScopeUpdatable)
	if err != nil {
		p.writeInstallationError(w, "update installation", err)
		return
	}

	if result.Installation.OwnerID != "" && result.Installation.OwnerID != userID {
		p.notifyOwnerOfSharedUpdate(userID, result)
	}

	p.publishInstallationEvent(summaryEvent(result.Installation))
	p.writeInstallationJSON(w, http.StatusOK, result)
}

func (p *Plugin) handleInstallationSimpleAction(w http.ResponseWriter, r *http.Request, run func(userID, installationID string) (InstallationActionResult, error)) {
	var req CloudDeletionLockRequest
	userID, ok := p.authorizeInstallationRequest(w, r, &req)
	if !ok {
		return
	}
	if req.InstallationID == "" {
		http.Error(w, "Please provide a non-blank installation_id field", http.StatusBadRequest)
		return
	}

	result, err := run(userID, req.InstallationID)
	if err != nil {
		p.writeInstallationError(w, "run installation action", err)
		return
	}

	p.publishInstallationEvent(summaryEvent(result.Installation))
	p.writeInstallationJSON(w, http.StatusOK, result)
}

func (p *Plugin) handleInstallationDelete(w http.ResponseWriter, r *http.Request) {
	var req InstallationDeleteRequest
	userID, ok := p.authorizeInstallationRequest(w, r, &req)
	if !ok {
		return
	}
	if req.InstallationID == "" {
		http.Error(w, "Please provide a non-blank installation_id field", http.StatusBadRequest)
		return
	}

	result, err := p.deleteInstallationForUser(userID, InstallationRef{ID: req.InstallationID}, req.ConfirmName)
	if err != nil {
		p.writeInstallationError(w, "delete installation", err)
		return
	}

	event := summaryEvent(result.Installation)
	event.State = "deleted"
	p.publishInstallationEvent(event)
	p.writeInstallationJSON(w, http.StatusOK, result)
}

func (p *Plugin) handleInstallationShare(w http.ResponseWriter, r *http.Request) {
	var req InstallationShareRequest
	userID, ok := p.authorizeInstallationRequest(w, r, &req)
	if !ok {
		return
	}
	if req.InstallationID == "" {
		http.Error(w, "Please provide a non-blank installation_id field", http.StatusBadRequest)
		return
	}

	result, err := p.setInstallationSharingForUser(userID, InstallationRef{ID: req.InstallationID}, req.Shared, req.AllowUpdates)
	if err != nil {
		p.writeInstallationError(w, "update installation sharing", err)
		return
	}

	event := summaryEvent(result.Installation)
	p.publishInstallationEvent(event)
	if !event.Shared {
		p.publishInstallationEventToOthers(event)
	}
	p.writeInstallationJSON(w, http.StatusOK, result)
}

// notifyOwnerOfSharedUpdate lets the owner of a shared installation know that
// another user changed it.
func (p *Plugin) notifyOwnerOfSharedUpdate(requesterID string, result InstallationActionResult) {
	username := "A user"
	requester, appErr := p.API.GetUser(requesterID)
	if appErr != nil {
		p.API.LogError(errors.Wrap(appErr, "failed to get update request user details").Error())
	} else {
		username = fmt.Sprintf("@%s", requester.Username)
	}

	p.PostBotDM(result.Installation.OwnerID, fmt.Sprintf("%s has updated an installation you have shared. The installation is %s and the changed fields are: %s.", username, result.Installation.Name, strings.Join(result.ChangedFields, ", ")))
}

func (p *Plugin) writeInstallationJSON(w http.ResponseWriter, status int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		p.API.LogError(errors.Wrap(err, "Unable to marshal installation response").Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	w.Write(data)
}

// writeInstallationError returns the message of a user error as a bad request.
// Any other error is logged and returned as an internal server error.
func (p *Plugin) writeInstallationError(w http.ResponseWriter, operation string, err error) {
	if isInstallationUserError(err) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p.API.LogError(errors.Wrapf(err, "Unable to %s", operation).Error())
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

func isInstallationUserError(err error) bool {
	if isCreateUserError(err) || isUpdateUserError(err) {
		return true
	}

	errText := err.Error()
	for _, fragment := range []string{
		"no installation with the",
		"must provide an installation",
		"to hibernate",
		"to wake up",
		"does not match installation name",
		"you may only have at most",
		"is read-only",
	} {
		if strings.Contains(errText, fragment) {
			return true
		}
	}

	return false
}

func installationEventFor(install *Installation, state string) InstallationEvent {
	event := InstallationEvent{
		State:  state,
		Shared: install.Shared,
	}
	if install.Installation != nil {
		event.InstallationID = install.ID
		event.OwnerID = install.OwnerID
	}

	return event
}

func summaryEvent(summary InstallationSummary) InstallationEvent {
	return InstallationEvent{
		InstallationID: summary.ID,
		OwnerID:        summary.OwnerID,
		State:          summary.State,
		Shared:         summary.Shared,
	}
}

// publishInstallationEvent sends the event to the owner. A shared installation
// also goes to all other connected users, which is why the payload holds no
// installation details.
func (p *Plugin) publishInstallationEvent(event InstallationEvent) {
	if event.InstallationID == "" {
		return
	}

	if event.OwnerID != "" {
		p.API.PublishWebSocketEvent(websocketEventInstallationUpdated, event.payload(), &model.WebsocketBroadcast{UserId: event.OwnerID})
	}
	if event.Shared {
		p.publishInstallationEventToOthers(event)
	}
}

// publishInstallationEventToOthers sends the event to every connected user
// except the owner. It lets users who lost access to an installation, for
// example after the owner stopped sharing it, drop it from their lists.
func (p *Plugin) publishInstallationEventToOthers(event InstallationEvent) {
	if event.InstallationID == "" {
		return
	}

	p.API.PublishWebSocketEvent(websocketEventInstallationUpdated, event.payload(), &model.WebsocketBroadcast{OmitUsers: map[string]bool{event.OwnerID: true}})
}

func (e InstallationEvent) payload() map[string]any {
	return map[string]any{
		"installation_id": e.InstallationID,
		"owner_id":        e.OwnerID,
		"state":           e.State,
		"shared":          e.Shared,
	}
}

// publishInstallationWebhookEvent forwards a provisioner state change to the
// webapp. Installations that the plugin does not track are ignored.
func (p *Plugin) publishInstallationWebhookEvent(installationID, newState string) {
	installs, _, err := p.getInstallations()
	if err != nil {
		p.API.LogError(errors.Wrap(err, "unable to load installations to publish websocket event").Error())
		return
	}

	for _, install := range installs {
		if install.Installation != nil && install.ID == installationID {
			p.publishInstallationEvent(installationEventFor(install, newState))
			return
		}
	}
}
