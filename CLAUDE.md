# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Mattermost plugin that wraps the [mattermost-cloud](https://github.com/mattermost/mattermost-cloud) provisioner. It lets Mattermost users spin up, manage, share, and destroy Mattermost installations on Kubernetes via the `/cloud` slash command and an RHS webapp UI. The plugin is a thin orchestration layer: it stores per-user installation records in the plugin KV store and proxies real work to a remote provisioning server.

## Commands

All builds go through the Mattermost plugin Makefile (`build/setup.mk` is shared scaffolding from the plugin starter template — don't edit it).

- `make` / `make all` — `check-style test dist` (full local validation).
- `make dist` — build server binaries (linux-amd64, darwin-amd64, darwin-arm64), build webapp, bundle as `dist/<plugin-id>-<version>.tar.gz`.
- `make deploy` — build+bundle, then push to a running Mattermost server using `build/bin/pluginctl`. Requires `MM_SERVICESETTINGS_SITEURL` + `MM_ADMIN_USERNAME` + `MM_ADMIN_PASSWORD` (or falls back to copying into a sibling `mattermost-server/` directory).
- `make debug-deploy` — same as `deploy` but webapp is built with `--mode=none` for readable JS.
- `make test` — runs `go test -race ./server/...` then `npm run test` in `webapp/`.
- `make coverage` — server coverage report (opens HTML).
- `make check-style` — `gofmt`, `go vet`, and pinned `golangci-lint` on the server, including `govet` shadow analysis; `eslint` on the webapp.
- `make apply` — propagate `plugin.json` version/id into `server/manifest.go` and `webapp/src/manifest.js`. Run this after bumping `version` in `plugin.json`.

### Single-test invocations
- One Go test: `cd server && go test -run TestName ./...`
- Single Go file's tests: `cd server && go test -race -run TestName$ .`
- Single webapp test file: `cd webapp && npx jest path/to/file.test.js`

### Release
Tags are cut from `master` (or `release/*`) using `make patch | minor | major | patch-rc | minor-rc | major-rc`. Each target validates the branch is up-to-date with `origin`, prompts for confirmation, then pushes a signed annotated tag. CI in `mattermost/actions-workflows` builds and publishes the release. Before bumping, update `plugin.json` version and `make apply`.

## Architecture

### Two halves, one bundle
- **`server/`** — Go plugin loaded by the Mattermost server. Single `main` package. Entry point is `server/main.go` → `Plugin` struct in `server/plugin.go`.
- **`webapp/`** — React/Redux frontend bundled as `webapp/dist/main.js`. Registers RHS, sidebar, channel header button, and App Bar components against the Mattermost webapp plugin registry (`webapp/src/index.js`).

### Plugin lifecycle (server)
`Plugin.OnActivate` (`server/plugin.go`) ensures a `cloud` bot user exists, loads its profile image and the App Bar icon, initializes the cloud + docker clients, and registers the `/cloud` slash command. `OnConfigurationChange` (`server/configuration.go`) reloads settings, rebuilds the docker client, and re-creates the cloud client. Configuration is read under `configurationLock` and the struct is treated as immutable — callers `Clone()` before mutating.

### Cloud client selection
`Plugin.setCloudClient` picks between three transports based on plugin settings:
1. OAuth client credentials (`ProvisioningServerClientID/Secret/TokenEndpoint`)
2. API gateway token (`ProvisioningServerAuthToken` → `x-api-key` header)
3. Plain HTTP

All three implement the `CloudClient` interface declared in `server/plugin.go`. Tests substitute a fake — search for `mockCloudClient` and `MockedClient`.

### Slash command dispatch
`server/command.go` defines `getCommand()` and an `ExecuteCommand` switch. Each subcommand has its own file: `command_create.go`, `command_delete.go`, `command_list.go`, `command_update.go`, `command_share.go`, `command_hibernate.go`, `command_wakeup.go`, `command_restart.go`, `command_import.go`, `command_mmctl.go`, `command_cli.go` (mmcli), `command_status.go`, `command_debug_packet.go`, `command_deletion_lock.go`. Each subcommand file ships with a `_test.go` neighbor — keep that convention when adding commands. Flags are defined with `spf13/pflag` and surfaced via `AutocompleteData` for the webapp UI.

### Installation storage
There is no database. Installations are persisted as a single JSON-encoded slice under KV key `installs` (`server/installation.go: StoreInstallsKey`). All mutations go through `storeInstallation` / `updateInstallation` / `deleteInstallation`, which use `KVCompareAndSet` in a 3-retry loop with linear backoff to handle concurrent writes. Always go through these helpers — never call `KVSet` on `installs` directly.

The local `Installation` struct embeds `cloud.InstallationDTO` and adds plugin-specific fields (`Name`, `Tag`, `TestData`, `Shared`, `AllowSharedUpdates`). `OwnerID` (from the embedded DTO) is the authority for ownership; `Shared` + `AllowSharedUpdates` gate cross-user updates.

### HTTP surface
`Plugin.ServeHTTP` in `server/api.go` routes plugin HTTP requests:
- `/webhook` — provisioner callbacks, authenticated via the `X-MM-Cloud-Plugin-Auth-Token` header against `ProvisioningServerWebhookSecret` (constant-time compare). Webhook handling is in `server/webhook.go`; events are processed asynchronously and may post to configured cluster/installation alert channels.
- `/api/v1/userinstalls`, `/api/v1/sharedinstalls` — feed the RHS UI; require `Mattermost-User-ID`. `InstallationWebWrapper` (`server/api.go`) is the response shape — it calls `HideSensitiveFields()` (`License`, `MattermostEnv`) before serializing.
- `/api/v1/deletion-lock`, `/api/v1/deletion-unlock` — toggle deletion lock for an installation (max per user from `DeletionLockInstallationsAllowedPerPerson`).
- `/api/v1/config` — sanitized config for the webapp via `ConfigResponse`.
- `/profile.png` — bot avatar.

### MCP readiness
There is no persistent MCP implementation, `server/mcp/` package, `server/mcp_adapter.go`, Agents dependency, or `modelcontextprotocol/go-sdk` dependency. Add MCP code in a dedicated implementation PR, and keep any dependency or routing changes explicit.

### License + image whitelists
`server/configuration.go` declares `validLicenseOptions` and `dockerRepoWhitelist`. Adding a new license tier (e.g., the recent `enterprise-advanced`) requires: adding the `licenseOption*` constant, appending to `validLicenseOptions`, adding the new field to `configuration` and `plugin.json` settings schema, and wiring `getLicenseValue`. Adding a new Mattermost docker image requires appending to `dockerRepoWhitelist` (and optionally `dockerRepoTestImages` if it needs special handling).

## Conventions

- The repo uses the Mattermost public server module (`github.com/mattermost/mattermost/server/public`); keep that baseline aligned across the root and build modules when upgrading Mattermost APIs.
- Tests live next to the file they cover (`foo.go` ↔ `foo_test.go`) and use stdlib `testing` + `testify`. Cloud client is mocked, not the KV store; the harness uses `plugintest.API`.
- Server `Logf`/log calls go through `p.API.LogError`/`LogDebug`/`LogWarn` — `fmt.Println` and friends will be dropped.
- Webapp tests use Jest with `jsdom`. Run with `npm test` in `webapp/`.
- `gofmt -s`, `go vet`, and pinned `golangci-lint` are enforced by `make check-style`.
- `.nvmrc` pins the Node version for the webapp build.
