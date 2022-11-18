package main

import (
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusCommand(t *testing.T) {
	mockedCloudClient := &MockClient{}
	plugin := Plugin{cloudClient: mockedCloudClient}

	t.Run("no clusters or installations", func(t *testing.T) {
		t.Run("show clusters", func(t *testing.T) {
			resp, isUserError, err := plugin.runStatusCommand([]string{"--include-clusters=true"}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, clusterTableHeader)
			assert.Contains(t, resp.Text, installationTableHeader)
		})

		t.Run("don't show clusters", func(t *testing.T) {
			resp, isUserError, err := plugin.runStatusCommand([]string{""}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.NotContains(t, resp.Text, clusterTableHeader)
			assert.Contains(t, resp.Text, installationTableHeader)
		})
	})

	t.Run("clusters and installations", func(t *testing.T) {
		cluster1 := &cloud.Cluster{
			ID:    cloud.NewID(),
			State: cloud.ClusterStateStable,
		}
		mockedCloudClient.mockedCloudClustersDTO = []*cloud.ClusterDTO{{Cluster: cluster1}}

		installation1 := &cloud.Installation{
			ID:      cloud.NewID(),
			Size:    "superextralarge",
			Version: "v7.1.44",
			State:   cloud.InstallationStateCreationDNS,
		}
		installationDTO := []*cloud.InstallationDTO{{Installation: installation1, DNSRecords: []*cloud.InstallationDNS{{DomainName: "https://greatawesome.com"}}}}
		mockedCloudClient.mockedCloudInstallationsDTO = installationDTO

		t.Run("show clusters", func(t *testing.T) {
			resp, isUserError, err := plugin.runStatusCommand([]string{"--include-clusters=true"}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, clusterTableHeader)
			assert.Contains(t, resp.Text, installationTableHeader)
			assert.Contains(t, resp.Text, cluster1.ID)
			assert.Contains(t, resp.Text, cluster1.State)
			assert.Contains(t, resp.Text, installation1.ID)
			assert.Contains(t, resp.Text, installationDTO[0].DNSRecords[0].DomainName)
			assert.Contains(t, resp.Text, installation1.Size)
			assert.Contains(t, resp.Text, installation1.Version)
			assert.Contains(t, resp.Text, installation1.State)
		})

		t.Run("don't show clusters", func(t *testing.T) {
			resp, isUserError, err := plugin.runStatusCommand([]string{""}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.NotContains(t, resp.Text, clusterTableHeader)
			assert.Contains(t, resp.Text, installationTableHeader)
			assert.NotContains(t, resp.Text, cluster1.ID)
			assert.NotContains(t, resp.Text, cluster1.State)
			assert.Contains(t, resp.Text, installation1.ID)
			assert.Contains(t, resp.Text, installationDTO[0].DNSRecords[0].DomainName)
			assert.Contains(t, resp.Text, installation1.Size)
			assert.Contains(t, resp.Text, installation1.Version)
			assert.Contains(t, resp.Text, installation1.State)
		})
	})

	t.Run("error", func(t *testing.T) {
		mockedCloudClient.err = errors.New("an error was enountered")

		resp, isUserError, err := plugin.runStatusCommand([]string{"--include-clusters=true"}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.False(t, isUserError)
		assert.Nil(t, resp)
	})
}
