package main

import (
	"encoding/json"

	cloud "github.com/mattermost/mattermost-cloud/model"
)

const (
	// StoreInstallRetries is the number of retries to use when storing a new install fails on a race
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
		originalJSONInstalls, err := p.API.KVGet(StoreInstallsKey)
		if err != nil {
			return err
		}

		var installs []*Installation
		jsonErr := json.Unmarshal(originalJSONInstalls, &installs)
		if jsonErr != nil {
			return jsonErr
		}

		installs = append(installs, install)

		newJSONInstalls, jsonErr := json.Marshal(installs)
		if jsonErr != nil {
			return jsonErr
		}

		ok, err := p.API.KVCompareAndSet(StoreInstallsKey, originalJSONInstalls, newJSONInstalls)
		if err != nil {
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
