package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// These tests cover installations the plugin still tracks in its KV store but
// the provisioning server no longer knows about, e.g. after a move to a new
// provisioning server or a deletion on the previous one.

func TestGetUpdatedInstallsForUserSkipsInstallationsNotFound(t *testing.T) {
	plugin := Plugin{
		cloudClient: &MockClient{
			returnNilGetInstallation: true,
			mockedCloudInstallationsDTO: []*cloud.InstallationDTO{
				{Installation: &cloud.Installation{ID: "id1", OwnerID: "owner 1", State: cloud.InstallationStateStable}},
				{Installation: &cloud.Installation{ID: "id2", OwnerID: "owner 1", State: cloud.InstallationStateStable}},
				{Installation: &cloud.Installation{ID: "id4", OwnerID: "owner 1", State: cloud.InstallationStateStable}},
				{Installation: &cloud.Installation{ID: "id5", OwnerID: "owner 1", State: cloud.InstallationStateStable}},
			},
		},
	}

	_, installationBytes, err := getFakePluginInstallations()
	require.NoError(t, err)

	// No KVCompareAndSet/KVSet expectation: the KV entry for the missing
	// installation must not be removed, and the mock fails the test if it is.
	api := &plugintest.API{}
	api.On("KVGet", mock.AnythingOfType("string")).Return(installationBytes, nil)
	api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
	plugin.SetAPI(api)

	for name, refresh := range map[string]func(string) ([]*Installation, error){
		"with sensitive":    plugin.getUpdatedInstallsForUserWithSensitive,
		"without sensitive": plugin.getUpdatedInstallsForUserWithoutSensitive,
	} {
		t.Run(name, func(t *testing.T) {
			installs, err := refresh("owner 1")
			require.NoError(t, err)

			var ids []string
			for _, install := range installs {
				ids = append(ids, install.ID)
			}
			assert.Equal(t, []string{"id1", "id2", "id4", "id5"}, ids)
		})
	}

	api.AssertCalled(t, "LogWarn", "Skipping installation id3 with name installation-three: not found on the provisioning server")
}

func TestGetUpdatedInstallsForUserOtherErrorsStillFail(t *testing.T) {
	// A real provisioner error must still fail the list; only "not found" is skipped.
	plugin := Plugin{cloudClient: &erroringGetInstallationClient{MockClient: &MockClient{}}}

	_, installationBytes, err := getFakePluginInstallations()
	require.NoError(t, err)
	api := &plugintest.API{}
	api.On("KVGet", mock.AnythingOfType("string")).Return(installationBytes, nil)
	plugin.SetAPI(api)

	_, err = plugin.getUpdatedInstallsForUserWithSensitive("owner 1")
	require.Error(t, err)
	assert.False(t, errors.Is(err, errInstallationNotFound))
}

type erroringGetInstallationClient struct {
	*MockClient
}

func (c *erroringGetInstallationClient) GetInstallation(string, *cloud.GetInstallationRequest) (*cloud.InstallationDTO, error) {
	return nil, errors.New("provisioner unavailable")
}

func TestGetUpdatedSharedInstallationsSkipsInstallationsNotFound(t *testing.T) {
	plugin := Plugin{cloudClient: &MockClient{returnNilGetInstallation: true}}

	shared := []*Installation{
		{Name: "shared-one", Shared: true, InstallationDTO: cloud.InstallationDTO{Installation: &cloud.Installation{ID: "id1", OwnerID: "owner 1"}}},
		{Name: "shared-two", Shared: true, InstallationDTO: cloud.InstallationDTO{Installation: &cloud.Installation{ID: "id2", OwnerID: "owner 2"}}},
	}
	b, err := json.Marshal(shared)
	require.NoError(t, err)

	api := &plugintest.API{}
	api.On("KVGet", mock.AnythingOfType("string")).Return(b, nil)
	api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
	plugin.SetAPI(api)

	installs, err := plugin.getUpdatedSharedInstallations(true)
	require.NoError(t, err)
	assert.Empty(t, installs)
}

func TestGetInstallationNotFoundOnProvisioner(t *testing.T) {
	plugin := Plugin{cloudClient: &MockClient{returnNilGetInstallation: true}}

	// No DNSRecords stored, so getInstallation asks the provisioning server.
	_, installationBytes, err := getFakePluginInstallations()
	require.NoError(t, err)
	api := &plugintest.API{}
	api.On("KVGet", mock.AnythingOfType("string")).Return(installationBytes, nil)
	plugin.SetAPI(api)

	require.NotPanics(t, func() {
		install, err := plugin.getInstallation("id3")
		require.NoError(t, err)
		assert.Nil(t, install)
	})
}

func TestProcessWebhookEventInstallationNotFound(t *testing.T) {
	payload := &cloud.WebhookPayload{
		Type:     cloud.TypeInstallation,
		ID:       "id3",
		NewState: cloud.InstallationStateHibernating,
		OldState: cloud.InstallationStateHibernationInProgress,
	}

	t.Run("not on the provisioning server", func(t *testing.T) {
		plugin := Plugin{
			cloudClient:   &MockClient{returnNilGetInstallation: true},
			configuration: &configuration{},
		}
		_, installationBytes, err := getFakePluginInstallations()
		require.NoError(t, err)
		// No CreatePost/GetDirectChannel expectation: no DM may be sent.
		api := &plugintest.API{}
		api.On("KVGet", mock.AnythingOfType("string")).Return(installationBytes, nil)
		api.On("LogDebug", mock.AnythingOfType("string")).Return(nil)
		api.On("PublishWebSocketEvent", websocketEventInstallationUpdated, mock.Anything, mock.Anything).Return()
		plugin.SetAPI(api)

		require.NotPanics(t, func() { plugin.processWebhookEvent(payload) })
	})

	t.Run("getInstallation error", func(t *testing.T) {
		// install is nil on this path; logging install.Name used to panic.
		plugin := Plugin{
			cloudClient:   &MockClient{},
			configuration: &configuration{},
		}
		api := &plugintest.API{}
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("not json"), nil)
		api.On("LogDebug", mock.AnythingOfType("string")).Return(nil)
		api.On("LogError", mock.AnythingOfType("string"), "installation", "id3").Return(nil)
		api.On("LogError", mock.AnythingOfType("string")).Return(nil)
		plugin.SetAPI(api)

		require.NotPanics(t, func() { plugin.processWebhookEvent(payload) })
		api.AssertCalled(t, "LogError", mock.AnythingOfType("string"), "installation", "id3")
	})
}

func TestOnConfigurationChangeUsesNewProvisioningServer(t *testing.T) {
	var oldHits, newHits atomic.Int32
	notFound := func(counter *atomic.Int32) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			counter.Add(1)
			w.WriteHeader(http.StatusNotFound)
		}))
	}
	oldServer := notFound(&oldHits)
	defer oldServer.Close()
	newServer := notFound(&newHits)
	defer newServer.Close()

	// An activated plugin talking to the old provisioning server.
	plugin := Plugin{configuration: &configuration{ProvisioningServerURL: oldServer.URL}}
	plugin.setCloudClient()

	api := &plugintest.API{}
	api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Run(func(args mock.Arguments) {
		args.Get(0).(*configuration).ProvisioningServerURL = newServer.URL
	}).Return(nil)
	plugin.SetAPI(api)

	require.NoError(t, plugin.OnConfigurationChange())
	assert.Equal(t, newServer.URL, plugin.getConfiguration().ProvisioningServerURL)

	_, err := plugin.getCloudClient().GetInstallation("someid", &cloud.GetInstallationRequest{})
	require.NoError(t, err)
	assert.Equal(t, int32(1), newHits.Load(), "client must use the new provisioning server URL")
	assert.Equal(t, int32(0), oldHits.Load(), "client must not use the previous provisioning server URL")
}

func TestCloudClientSwapDuringRequestsIsRaceFree(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	plugin := Plugin{configuration: &configuration{ProvisioningServerURL: server.URL}}
	plugin.setCloudClient()

	installs, err := json.Marshal([]*Installation{{
		Name:            "shared",
		Shared:          true,
		InstallationDTO: cloud.InstallationDTO{Installation: &cloud.Installation{ID: "id1"}},
	}})
	require.NoError(t, err)

	api := &plugintest.API{}
	api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Run(func(args mock.Arguments) {
		args.Get(0).(*configuration).ProvisioningServerURL = server.URL
	}).Return(nil)
	api.On("KVGet", StoreInstallsKey).Return(installs, nil)
	api.On("LogWarn", mock.AnythingOfType("string"))
	plugin.SetAPI(api)

	// A config change swaps the client while list requests use it. Run with
	// -race to catch unsynchronized access to the client. Only one goroutine
	// changes the config because the server calls OnConfigurationChange
	// serially.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 20 {
			assert.NoError(t, plugin.OnConfigurationChange())
		}
	}()
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				_, listErr := plugin.getUpdatedSharedInstallations(true)
				assert.NoError(t, listErr)
			}
		}()
	}
	wg.Wait()
}
