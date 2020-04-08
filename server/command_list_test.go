package main

import (
	"encoding/json"

	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-cloud/model"
	cloud "github.com/mattermost/mattermost-cloud/model"
)

func (d *ServerTestSuite) TestUpdatePluginInstalls() {
	pluginInstalls := []*Installation{
		{Name: "one"},
		{Name: "two"},
		{Name: "three"},
		{Name: "four"},
	}

	pluginInstalls = updatePluginInstalls(99, pluginInstalls)
	d.Assert().Equal(4, len(pluginInstalls))

	pluginInstalls = updatePluginInstalls(-1, pluginInstalls)
	d.Assert().Equal(4, len(pluginInstalls))

	pluginInstalls = updatePluginInstalls(3, pluginInstalls)
	d.Assert().Equal(3, len(pluginInstalls))
	d.Assert().Equal("one", pluginInstalls[0].Name)
	d.Assert().Equal("two", pluginInstalls[1].Name)
	d.Assert().Equal("three", pluginInstalls[2].Name)

	pluginInstalls = updatePluginInstalls(1, pluginInstalls)
	d.Assert().Equal(2, len(pluginInstalls))
	d.Assert().Equal("one", pluginInstalls[0].Name)
	d.Assert().Equal("three", pluginInstalls[1].Name)

	pluginInstalls = updatePluginInstalls(0, pluginInstalls)
	d.Assert().Equal(1, len(pluginInstalls))
	d.Assert().Equal("three", pluginInstalls[0].Name)

	pluginInstalls = updatePluginInstalls(0, pluginInstalls)
	d.Assert().Equal(0, len(pluginInstalls))

	pluginInstalls = updatePluginInstalls(0, pluginInstalls)
	d.Assert().Equal(0, len(pluginInstalls))
}

func (d *ServerTestSuite) TestGetUpdatedInstallsForUser() {
	_, installationBytes := d.getFakeCloudInstallations()
	pluginInstallations := []*model.Installation{
		&model.Installation{ID: "1", DeleteAt: 99999, OwnerID: "id-123"},
		&model.Installation{ID: "2", State: model.ClusterInstallationStateCreationFailed, OwnerID: "id-123"},
		&model.Installation{ID: "3", OwnerID: "id-123"},
		&model.Installation{ID: "4", OwnerID: "id-123"},
		&model.Installation{ID: "5", OwnerID: "id-123"},
	}

	d.mockedPluginAPI.EXPECT().KVGet(gomock.Any()).Return(installationBytes, nil).Times(1)
	d.mockedCloudClient.EXPECT().GetInstallations(gomock.Any()).Return(pluginInstallations, nil).Times(1)

	actualInstallations, err := d.plugin.getUpdatedInstallsForUser("id-123")
	d.Assert().NoError(err)
	d.Assert().Equal(3, len(actualInstallations))
}

func (d *ServerTestSuite) getFakeCloudInstallations() ([]*Installation, []byte) {
	installations := []*Installation{
		&Installation{Name: "installation-one", Installation: cloud.Installation{ID: "1", OwnerID: "id-123"}},
		&Installation{Name: "installation-two", Installation: cloud.Installation{ID: "2", OwnerID: "id-123"}},
		&Installation{Name: "installation-three", Installation: cloud.Installation{ID: "3", OwnerID: "id-123"}},
		&Installation{Name: "installation-four", Installation: cloud.Installation{ID: "4", OwnerID: "id-123"}},
		&Installation{Name: "installation-five", Installation: cloud.Installation{ID: "5", OwnerID: "id-123"}},
	}

	installationBytes, err := json.Marshal(installations)
	d.Assert().NoError(err)

	return installations, installationBytes
}
