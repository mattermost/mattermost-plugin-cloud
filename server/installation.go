package main

import (
	"encoding/json"
	"fmt"
	"time"

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
	Name string
	cloud.InstallationDTO
	TestData bool
	Tag      string
}

// ToPrettyJSON will return a JSON string installation with indentation and new lines
func (i *Installation) ToPrettyJSON() string {
	b, err := json.MarshalIndent(i, "", "\t")
	if err != nil {
		return ""
	}
	return string(b)
}

// HideSensitiveFields hides installation fields that could contain sensitive
// information.
func (i *Installation) HideSensitiveFields() {
	i.License = "hidden"
	i.MattermostEnv = nil
}

func (p *Plugin) storeInstallation(install *Installation) error {
	for i := 0; i < StoreInstallRetries; i++ {
		// Use the retry count value to build an increasing backoff that has no
		// delay on the first attempt.
		time.Sleep(time.Duration(i) * time.Second)

		installs, originalJSONInstalls, err := p.getInstallations()
		if err != nil {
			p.API.LogWarn(errors.Wrap(err, "unable to get installations").Error())
			continue
		}

		installs = append(installs, install)

		newJSONInstalls, err := json.Marshal(installs)
		if err != nil {
			p.API.LogWarn(errors.Wrap(err, "unable to marshal installations").Error())
			continue
		}

		ok, appErr := p.API.KVCompareAndSet(StoreInstallsKey, originalJSONInstalls, newJSONInstalls)
		if appErr != nil {
			p.API.LogWarn(errors.Wrap(appErr, "unable to store install").Error())
			continue
		}

		// If err is nil but ok is false, then something else updated the installs between the get and set above
		// so we need to try again, otherwise we can return
		if ok {
			return nil
		}
		p.API.LogWarn("unable to store installs due to another process making an update first")
	}

	return fmt.Errorf("failed %d times to store installation %s", StoreInstallRetries, install.ID)
}

func (p *Plugin) updateInstallation(install *Installation) error {
	for i := 0; i < StoreInstallRetries; i++ {
		// Use the retry count value to build an increasing backoff that has no
		// delay on the first attempt.
		time.Sleep(time.Duration(i) * time.Second)

		installs, originalJSONInstalls, err := p.getInstallations()
		if err != nil {
			p.API.LogWarn(errors.Wrap(err, "unable to get installations").Error())
			continue
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

		newJSONInstalls, err := json.Marshal(installs)
		if err != nil {
			p.API.LogWarn(errors.Wrap(err, "unable to marshal installations").Error())
			continue
		}

		ok, appErr := p.API.KVCompareAndSet(StoreInstallsKey, originalJSONInstalls, newJSONInstalls)
		if appErr != nil {
			p.API.LogWarn(errors.Wrap(appErr, "unable to store install").Error())
			continue
		}

		// If err is nil but ok is false, then something else updated the installs between the get and set above
		// so we need to try again, otherwise we can return
		if ok {
			return nil
		}
		p.API.LogWarn("unable to store installs due to another process making an update first")
	}

	return fmt.Errorf("failed %d times to store updated installation %s", StoreInstallRetries, install.ID)
}

func (p *Plugin) deleteInstallation(installationID string) error {
	for i := 0; i < StoreInstallRetries; i++ {
		// Use the retry count value to build an increasing backoff that has no
		// delay on the first attempt.
		time.Sleep(time.Duration(i) * time.Second)

		installs, originalJSONInstalls, err := p.getInstallations()
		if err != nil {
			p.API.LogWarn(errors.Wrap(err, "unable to get installations").Error())
			continue
		}

		indexToDelete := -1
		for index, install := range installs {
			if install.ID == installationID {
				indexToDelete = index
			}
		}

		installs = append(installs[:indexToDelete], installs[indexToDelete+1:]...)

		newJSONInstalls, err := json.Marshal(installs)
		if err != nil {
			p.API.LogWarn(errors.Wrap(err, "unable to marshal installations").Error())
			continue
		}

		ok, appErr := p.API.KVCompareAndSet(StoreInstallsKey, originalJSONInstalls, newJSONInstalls)
		if appErr != nil {
			p.API.LogWarn(errors.Wrap(appErr, "unable to store install").Error())
			continue
		}

		// If err is nil but ok is false, then something else updated the installs between the get and set above
		// so we need to try again, otherwise we can break
		if ok {
			return nil
		}
		p.API.LogWarn("unable to store installs due to another process making an update first")
	}

	return fmt.Errorf("failed %d times to delete installation %s", StoreInstallRetries, installationID)
}

// getInstallations fetches existing installs from the KV store and returns a slice of pointers to the Installations (unmarshalled from JSON), the original JSON as a byte slice, and any errors
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
			// Retrieve the information we need from the installation directly from the provisioner
			cloudInstall, err := p.cloudClient.GetInstallation(install.ID, &cloud.GetInstallationRequest{})
			if err != nil {
				return nil, err
			}

			install.DNSRecords = cloudInstall.DNSRecords

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
