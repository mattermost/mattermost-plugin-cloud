package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattermost/mattermost-plugin-agents/external/pluginmcp"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestEnsureMCPServerCreatesServerOnce(t *testing.T) {
	restoreMCPTestSeams(t)

	var configs []pluginmcp.Config
	mcpNewServer = func(api pluginmcp.PluginAPI, config pluginmcp.Config) *pluginmcp.Server {
		configs = append(configs, config)
		return pluginmcp.NewServer(api, config)
	}

	p := &Plugin{}
	require.NoError(t, p.ensureMCPServer())
	require.NoError(t, p.ensureMCPServer())

	require.Len(t, configs, 1)
	assert.Equal(t, manifest.ID, configs[0].PluginID)
	assert.Equal(t, mcpServerName, configs[0].Name)
	assert.Equal(t, mcpBasePath, configs[0].Path)
	assert.True(t, configs[0].ExposeExternal)
	assert.Equal(t, manifest.Version, configs[0].Version)
	assert.NotNil(t, p.currentMCPServer())
}

func TestEnsureMCPServerRequiresManifestData(t *testing.T) {
	restoreMCPTestSeams(t)

	originalManifest := manifest
	t.Cleanup(func() {
		manifest = originalManifest
	})

	manifest.ID = ""
	require.ErrorContains(t, (&Plugin{}).ensureMCPServer(), "plugin manifest id is required")

	manifest = originalManifest
	manifest.Version = ""
	require.ErrorContains(t, (&Plugin{}).ensureMCPServer(), "plugin manifest version is required")
}

func TestRegisterMCPServerBestEffort(t *testing.T) {
	restoreMCPTestSeams(t)

	t.Run("missing server logs warning", func(t *testing.T) {
		p, api := newMCPTestPlugin()
		api.On("LogWarn", "MCP registration unavailable; continuing plugin activation", "reason", "server not initialized").Once()

		p.registerMCPServerBestEffort()

		api.AssertExpectations(t)
	})

	t.Run("register error logs warning", func(t *testing.T) {
		p, api := newMCPTestPlugin()
		p.mcpServer = pluginmcp.NewServer(nil, testMCPConfig())
		mcpRegister = func(server *pluginmcp.Server) error {
			return errors.New("agents unavailable")
		}
		api.On("LogWarn", "MCP registration unavailable; continuing plugin activation", "err", "agents unavailable").Once()

		p.registerMCPServerBestEffort()

		api.AssertExpectations(t)
	})

	t.Run("success registers once", func(t *testing.T) {
		p, api := newMCPTestPlugin()
		p.mcpServer = pluginmcp.NewServer(nil, testMCPConfig())
		var calls int
		mcpRegister = func(server *pluginmcp.Server) error {
			calls++
			return nil
		}

		p.registerMCPServerBestEffort()

		assert.Equal(t, 1, calls)
		api.AssertExpectations(t)
	})
}

func TestUnregisterMCPServerBestEffort(t *testing.T) {
	restoreMCPTestSeams(t)

	t.Run("missing server is no-op", func(t *testing.T) {
		p, api := newMCPTestPlugin()

		p.unregisterMCPServerBestEffort()

		api.AssertExpectations(t)
	})

	t.Run("unregister error logs warning", func(t *testing.T) {
		p, api := newMCPTestPlugin()
		p.mcpServer = pluginmcp.NewServer(nil, testMCPConfig())
		mcpUnregister = func(server *pluginmcp.Server) error {
			return errors.New("agents unavailable")
		}
		api.On("LogWarn", "MCP unregister failed; continuing plugin shutdown", "err", "agents unavailable").Once()

		p.unregisterMCPServerBestEffort()

		assert.Nil(t, p.currentMCPServer())
		api.AssertExpectations(t)
	})

	t.Run("success unregisters once", func(t *testing.T) {
		p, api := newMCPTestPlugin()
		p.mcpServer = pluginmcp.NewServer(nil, testMCPConfig())
		var calls int
		mcpUnregister = func(server *pluginmcp.Server) error {
			calls++
			return nil
		}

		p.unregisterMCPServerBestEffort()

		assert.Equal(t, 1, calls)
		assert.Nil(t, p.currentMCPServer())
		api.AssertExpectations(t)
	})
}

func TestMCPServerLifecycleCreatesFreshServerAfterUnregister(t *testing.T) {
	restoreMCPTestSeams(t)

	p, api := newMCPTestPlugin()
	var servers []*pluginmcp.Server
	var registered []*pluginmcp.Server
	var unregistered []*pluginmcp.Server

	mcpNewServer = func(api pluginmcp.PluginAPI, config pluginmcp.Config) *pluginmcp.Server {
		server := pluginmcp.NewServer(api, config)
		servers = append(servers, server)
		return server
	}
	mcpRegister = func(server *pluginmcp.Server) error {
		registered = append(registered, server)
		return nil
	}
	mcpUnregister = func(server *pluginmcp.Server) error {
		unregistered = append(unregistered, server)
		return nil
	}

	require.NoError(t, p.ensureMCPServer())
	firstServer := p.currentMCPServer()
	p.registerMCPServerBestEffort()
	p.unregisterMCPServerBestEffort()
	require.Nil(t, p.currentMCPServer())

	require.NoError(t, p.ensureMCPServer())
	secondServer := p.currentMCPServer()
	p.registerMCPServerBestEffort()

	require.Len(t, servers, 2)
	require.NotNil(t, firstServer)
	require.NotNil(t, secondServer)
	assert.True(t, firstServer != secondServer)
	assert.Equal(t, []*pluginmcp.Server{firstServer, secondServer}, registered)
	assert.Equal(t, []*pluginmcp.Server{firstServer}, unregistered)
	api.AssertExpectations(t)
}

func TestServeMCPIfMatchRouting(t *testing.T) {
	t.Run("matches exact and child paths", func(t *testing.T) {
		for _, path := range []string{"/mcp", "/mcp/tools"} {
			p := &Plugin{}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, path, nil)

			assert.True(t, p.serveMCPIfMatch(rec, req))
			assert.Equal(t, http.StatusNotFound, rec.Code)
		}
	})

	t.Run("does not match partial prefixes", func(t *testing.T) {
		for _, path := range []string{"/mcpfoo", "/mcproxy", "/api/v1/config"} {
			p := &Plugin{}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, path, nil)

			assert.False(t, p.serveMCPIfMatch(rec, req))
		}
	})

	t.Run("delegates matching paths to MCP server", func(t *testing.T) {
		p := &Plugin{mcpServer: pluginmcp.NewServer(nil, testMCPConfig())}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)

		assert.True(t, p.serveMCPIfMatch(rec, req))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestServeHTTPRoutesMCPBeforeConfigValidation(t *testing.T) {
	p := &Plugin{mcpServer: pluginmcp.NewServer(nil, testMCPConfig())}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)

	p.ServeHTTP(nil, rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "forbidden")
	assert.NotContains(t, rec.Body.String(), "This plugin is not configured")
	assert.NotEqual(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestOnActivateRegistersCommandThenMCPBestEffort(t *testing.T) {
	restoreMCPTestSeams(t)

	bundlePath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(bundlePath, "assets"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(bundlePath, "public"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bundlePath, "assets", "profile.png"), []byte("profile"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(bundlePath, "public", "app-bar-icon.png"), []byte("icon"), 0o644))

	var events []string
	mcpNewServer = func(api pluginmcp.PluginAPI, config pluginmcp.Config) *pluginmcp.Server {
		events = append(events, "mcp-new")
		return pluginmcp.NewServer(api, config)
	}
	mcpRegister = func(server *pluginmcp.Server) error {
		events = append(events, "mcp-register")
		return nil
	}

	p, api := newMCPTestPlugin()
	p.configuration = &configuration{
		ProvisioningServerURL: "https://provisioner.example.com",
		InstallationDNS:       "example.com",
	}
	api.On("CreateBot", mock.AnythingOfType("*model.Bot")).Return(&model.Bot{UserId: "bot-id"}, nil).Once()
	api.On("GetBundlePath").Return(bundlePath, nil).Once()
	api.On("SetProfileImage", "bot-id", mock.Anything).Return(nil).Once()
	api.On("RegisterCommand", mock.AnythingOfType("*model.Command")).
		Run(func(args mock.Arguments) {
			events = append(events, "command")
		}).
		Return(nil).
		Once()

	require.NoError(t, p.OnActivate())

	assert.Equal(t, []string{"command", "mcp-new", "mcp-register"}, events)
	assert.NotNil(t, p.currentMCPServer())
	api.AssertExpectations(t)
}

func TestOnDeactivateUnregistersMCPBestEffort(t *testing.T) {
	restoreMCPTestSeams(t)

	p, api := newMCPTestPlugin()
	p.mcpServer = pluginmcp.NewServer(nil, testMCPConfig())
	var calls int
	mcpUnregister = func(server *pluginmcp.Server) error {
		calls++
		return nil
	}

	require.NoError(t, p.OnDeactivate())

	assert.Equal(t, 1, calls)
	assert.Nil(t, p.currentMCPServer())
	api.AssertExpectations(t)
}

func restoreMCPTestSeams(t *testing.T) {
	t.Helper()

	originalNewServer := mcpNewServer
	originalRegister := mcpRegister
	originalUnregister := mcpUnregister

	t.Cleanup(func() {
		mcpNewServer = originalNewServer
		mcpRegister = originalRegister
		mcpUnregister = originalUnregister
	})
}

func newMCPTestPlugin() (*Plugin, *plugintest.API) {
	p := &Plugin{}
	api := &plugintest.API{}
	p.SetAPI(api)
	return p, api
}

func testMCPConfig() pluginmcp.Config {
	return pluginmcp.Config{
		PluginID:       manifest.ID,
		Name:           mcpServerName,
		Path:           mcpBasePath,
		ExposeExternal: true,
		Version:        manifest.Version,
	}
}
