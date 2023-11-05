package main

import (
	"fmt"
	"strconv"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

func (p *Plugin) lockForDeletion(installationID string, userID string) error {
	if installationID == "" {
		return errors.New("installationID must not be empty")
	}
	installations, err := p.getUpdatedInstallsForUserWithoutSensitive(userID)
	if err != nil {
		return err
	}

	if len(installations) == 0 {
		return errors.New("no installations found for the given User ID")
	}

	maxLockedInstallations, err := strconv.Atoi(p.getConfiguration().DeletionLockInstallationsAllowedPerPerson)
	if err != nil {
		return errors.New("invalid value for DeletionLockInstallationsAllowedPerPerson")
	}

	numExistingLockedInstallations := 0
	var installationToLock *Installation
	for _, install := range installations {
		if install.OwnerID == userID && install.ID == installationID {
			installationToLock = install
			break
		}
		if install.OwnerID == userID && install.DeletionLocked {
			numExistingLockedInstallations++
		}
	}

	if maxLockedInstallations <= numExistingLockedInstallations {
		return fmt.Errorf("you may only have at most %d installations locked for deletion at a time", maxLockedInstallations)
	}

	if installationToLock == nil {
		return errors.New("installation to be locked not found")
	}

	err = p.cloudClient.LockDeletionLockForInstallation(installationToLock.ID)
	return err
}

func (p *Plugin) unlockForDeletion(installationID string, userID string) error {
	if installationID == "" {
		return errors.New("installationID must not be empty")
	}

	installations, err := p.getUpdatedInstallsForUserWithoutSensitive(userID)
	if err != nil {
		return err
	}

	if len(installations) == 0 {
		return errors.New("no installations found for the given User ID")
	}

	var installationToLock *Installation
	for _, install := range installations {
		if install.OwnerID == userID && install.ID == installationID {
			installationToLock = install
			break
		}
	}

	if installationToLock == nil {
		return errors.New("installation to be unlocked not found")
	}

	err = p.cloudClient.UnlockDeletionLockForInstallation(installationToLock.ID)
	return err
}

func (p *Plugin) runDeletionLockCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	installations, err := p.getUpdatedInstallsForUserWithoutSensitive(extra.UserId)
	if err != nil {
		return nil, true, err
	}
	var installationIDToLock string
	for _, installation := range installations {
		if installation.OwnerID == extra.UserId && installation.Name == name {
			installationIDToLock = installation.ID
			break
		}
	}

	if installationIDToLock == "" {
		return nil, true, errors.Errorf("no installation with the name %s found", name)
	}

	err = p.lockForDeletion(installationIDToLock, extra.UserId)
	if err != nil {
		return getCommandResponse(model.CommandResponseTypeEphemeral, err.Error(), extra), false, err
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, "Deletion lock has been applied, your workspace will be preserved.", extra), false, nil
}

func (p *Plugin) runDeletionUnlockCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	installs, err := p.getUpdatedInstallsForUserWithoutSensitive(extra.UserId)
	if err != nil {
		return nil, false, err
	}

	var installationIDToUnlock string
	for _, install := range installs {
		if install.OwnerID == extra.UserId && install.Name == name {
			installationIDToUnlock = install.ID
			break
		}
	}

	if installationIDToUnlock == "" {
		return nil, true, errors.Errorf("no installation with the name %s found", name)
	}

	err = p.unlockForDeletion(installationIDToUnlock, extra.UserId)
	if err != nil {
		return getCommandResponse(model.CommandResponseTypeEphemeral, err.Error(), extra), false, err
	}

	return getCommandResponse(model.CommandResponseTypeEphemeral, "Deletion lock has been removed, your workspace can now be deleted", extra), false, nil
}
