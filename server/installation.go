package main

import (
	"encoding/json"

	cloud "github.com/mattermost/mattermost-cloud/model"
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
	SAML     string
	LDAP     bool
	OAuth    string
	TestData bool
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
		// so we need to try again, otherwise we can break
		if ok {
			break
		}
	}

	return nil
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

	if len(originalJSONInstalls) == 0 {
		originalJSONInstalls = []byte("[]")
	}

	var installs []*Installation
	jsonErr := json.Unmarshal(originalJSONInstalls, &installs)
	if jsonErr != nil {
		return nil, nil, jsonErr
	}

	return installs, originalJSONInstalls, nil
}

func (p *Plugin) getInstallationsForUser(userID string) ([]*Installation, error) {
	installs, _ , err := p.getInstallations()
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
