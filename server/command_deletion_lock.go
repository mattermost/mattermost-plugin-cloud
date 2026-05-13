package main

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
)

func (p *Plugin) lockForDeletion(installationID string, userID string) error {
	if installationID == "" {
		return errors.New("installationID must not be empty")
	}
	_, err := p.setDeletionLockForUser(userID, InstallationRef{ID: installationID}, true)
	return err
}

func (p *Plugin) unlockForDeletion(installationID string, userID string) error {
	if installationID == "" {
		return errors.New("installationID must not be empty")
	}

	_, err := p.setDeletionLockForUser(userID, InstallationRef{ID: installationID}, false)
	return err
}

func (p *Plugin) runDeletionLockCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	_, err := p.setDeletionLockForUser(extra.UserId, InstallationRef{Name: name}, true)
	if err != nil {
		if isDeletionLockMissingTargetError(err) {
			return nil, true, errors.Errorf("no installation with the name %s found", name)
		}
		return getCommandResponse(model.CommandResponseTypeEphemeral, err.Error(), extra), false, err
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, "Deletion lock has been applied, your workspace will be preserved.", extra), false, nil
}

func (p *Plugin) runDeletionUnlockCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	_, err := p.setDeletionLockForUser(extra.UserId, InstallationRef{Name: name}, false)
	if err != nil {
		if isDeletionLockMissingTargetError(err) {
			return nil, true, errors.Errorf("no installation with the name %s found", name)
		}
		return getCommandResponse(model.CommandResponseTypeEphemeral, err.Error(), extra), false, err
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, "Deletion lock has been removed, your workspace can now be deleted", extra), false, nil
}

func isDeletionLockMissingTargetError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return message == "no installations found for the given User ID" ||
		message == "installation to be locked not found" ||
		message == "installation to be unlocked not found"
}
