package main

import (
	"encoding/json"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	// StoreInstallRetries is the number of retries to use when storing installs fails on a race
	StoreInstallRetries = 3
	// StoreInstallsKey is the key used to store installs in the plugin KV store
	StoreInstallsKey = "installs"
)

// Installation extends the cloud struct of the same name to add additional configuration options
type Installation struct {
	cloud.Installation
	Name     string
	TestData bool
}

// ToPrettyJSON will return a JSON string installation with indentation and new lines
func (i *Installation) ToPrettyJSON() string {
	b, err := json.MarshalIndent(i, "", "\t")
	if err != nil {
		return ""
	}
	return string(b)
}

func (p *Plugin) storeInstallation(install *Installation) error {
	for i := 0; i < StoreInstallRetries; i++ {
		installs, originalJSONInstalls, err := p.getInstallations()
		if err != nil {
			return err
		}

		installs = append(installs, install)

		newJSONInstalls, jsonErr := json.Marshal(installs)
		if jsonErr != nil {
			return jsonErr
		}

		ok, appErr := p.API.KVCompareAndSet(StoreInstallsKey, originalJSONInstalls, newJSONInstalls)
		if appErr != nil {
			return err
		}

		// If err is nil but ok is false, then something else updated the installs between the get and set above
		// so we need to try again, otherwise we can return
		if ok {
			return nil
		}
	}

	return errors.New("unable to store installation")
}

func (p *Plugin) updateInstallation(install *Installation) error {
	for i := 0; i < StoreInstallRetries; i++ {
		installs, originalJSONInstalls, err := p.getInstallations()
		if err != nil {
			return err
		}

		found := false
		for index, existingInstall := range installs {
			if existingInstall.ID == install.ID {
				found = true
				installs[index] = install
				break
			}
		}

		if !found {
			return errors.New("installation does not exist")
		}

		newJSONInstalls, jsonErr := json.Marshal(installs)
		if jsonErr != nil {
			return jsonErr
		}

		ok, appErr := p.API.KVCompareAndSet(StoreInstallsKey, originalJSONInstalls, newJSONInstalls)
		if appErr != nil {
			return err
		}

		// If err is nil but ok is false, then something else updated the installs between the get and set above
		// so we need to try again, otherwise we can return
		if ok {
			return nil
		}
	}

	return errors.New("unable to update installation")
}

func (p *Plugin) deleteInstallation(installationID string) error {
	for i := 0; i < StoreInstallRetries; i++ {
		installs, originalJSONInstalls, err := p.getInstallations()
		if err != nil {
			return err
		}

		indexToDelete := -1
		for index, install := range installs {
			if install.ID == installationID {
				indexToDelete = index
			}
		}

		installs = append(installs[:indexToDelete], installs[indexToDelete+1:]...)

		newJSONInstalls, jsonErr := json.Marshal(installs)
		if jsonErr != nil {
			return jsonErr
		}

		ok, appErr := p.API.KVCompareAndSet(StoreInstallsKey, originalJSONInstalls, newJSONInstalls)
		if appErr != nil {
			return err
		}

		// If err is nil but ok is false, then something else updated the installs between the get and set above
		// so we need to try again, otherwise we can break
		if ok {
			break
		}
	}

	return nil
}

func (p *Plugin) getInstallations() ([]*Installation, []byte, error) {
	originalJSONInstalls, err := p.API.KVGet(StoreInstallsKey)
	if err != nil {
		return nil, nil, err
	}

	if originalJSONInstalls == nil {
		return []*Installation{}, originalJSONInstalls, nil
	}

	var installs []*Installation
	jsonErr := json.Unmarshal(originalJSONInstalls, &installs)
	if jsonErr != nil {
		return nil, nil, jsonErr
	}

	return installs, originalJSONInstalls, nil
}

func (p *Plugin) getInstallation(installationID string) (*Installation, error) {
	installs, _, err := p.getInstallations()
	if err != nil {
		return nil, err
	}

	for _, install := range installs {
		if install.ID == installationID {
			return install, nil
		}
	}

	return nil, nil
}

func (p *Plugin) getInstallationsForUser(userID string) ([]*Installation, error) {
	installs, _, err := p.getInstallations()
	if err != nil {
		return nil, err
	}

	installsForUser := []*Installation{}
	for _, install := range installs {
		if install.OwnerID == userID {
			installsForUser = append(installsForUser, install)
		}
	}

	return installsForUser, nil
}
