package main

import (
	"fmt"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
)

const (
	clusterTableHeader = `
| Cluster | Size | State |
| -- | -- | -- |
`

	installationTableHeader = `
| Installation | DNS | Size | Version | State |
| -- | -- | -- | -- | -- |
`
)

// The status command is primarily intended to help the team administrating the
// cloud infrastructure, so we don't publish the command in the help info.
func (p *Plugin) runStatusCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	clusters, err := p.cloudClient.GetClusters(&cloud.GetClustersRequest{
		Page:           0,
		PerPage:        100,
		IncludeDeleted: false,
	})
	if err != nil {
		return nil, false, err
	}

	installations, err := p.cloudClient.GetInstallations(&cloud.GetInstallationsRequest{
		Page:           0,
		PerPage:        100,
		IncludeDeleted: false,
	})
	if err != nil {
		return nil, false, err
	}

	status := clusterTableHeader
	for _, cluster := range clusters {
		status += fmt.Sprintf("| %s | %s | %s |", cluster.ID, cluster.Size, cluster.State)
	}

	status += "\n"
	status += installationTableHeader
	for _, installation := range installations {
		status += fmt.Sprintf("| %s | %s | %s | %s | %s |", installation.ID, installation.DNS, installation.Size, installation.Version, installation.State)
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, status), false, nil
}
