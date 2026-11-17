# Mattermost Cloud Plugin

This plugin allows for the creation and management of Mattermost kubernetes installations directly from Mattermost. Commands are exposed that can be used to communicate with a remote [cloud server](https://github.com/mattermost/mattermost-cloud).

## Getting Started

1. Install the plugin.
2. Enter configuration information for your environment and enable the plugin.
3. From any channel or DM, run `/cloud -h` for information on using the plugin.

## Developing

### Local stack (Docker)

`make dev-start` starts Mattermost 12 and its database with Docker Compose, builds this plugin, and installs it. The stack has no provisioning server. The plugin connects to one on the host.

```
make dev-start
```

Each worktree gets its own stack. The Compose project is `mmcloud-<worktree directory name>`, so stacks of different worktrees do not clash.

| Command | Action |
| - | - |
| `make dev-start` | Create the stack, configure it, and install the plugin. |
| `make dev-deploy` | Build and install the plugin again after a code change. |
| `make dev-info` | Show the stack name, the address, the login, and the state. |
| `make dev-clean` | Delete the containers, volumes, and networks of the stack. |

Notes:

- **Address:** With OrbStack, open `http://mattermost.<project>.orb.local:8065`. With another Docker engine, the stack publishes one port on `127.0.0.1`. `make dev-info` prints the address.
- **Login:** `admin` / `Sysadmin123!`. The team is `dev`.
- **Provisioning server:** The stack has none, and the plugin does not activate without a provisioning server URL. The plugin uses `http://host.docker.internal:8075`, a server on your host. Set `DEV_PROVISIONER_URL` to use another URL. To create installations, set `DEV_WEBHOOK_SECRET` and the license files `DEV_E10_LICENSE_FILE`, `DEV_E20_LICENSE_FILE`, `DEV_PROFESSIONAL_LICENSE_FILE`, `DEV_ENTERPRISE_LICENSE_FILE` or `DEV_ENTERPRISE_ADVANCED_LICENSE_FILE`.
- **Image:** The Mattermost image is `linux/amd64` only. Docker emulates it on Apple silicon, so the first start is slow.
- **Settings:** `local/dev.mk` lists all variables.

### Manual setup

1. Run a Mattermost server and web app locally, see https://developers.mattermost.com/contribute/server/developer-setup/ and https://developers.mattermost.com/contribute/webapp/developer-setup/
2. Log in to create the system admin account
3. Run a cloud provisioning server locally, see https://github.com/mattermost/mattermost-cloud#developing
4. Create a webhook from your provisioning server to your Mattermost server with:
  ```
  cloud webhook create --owner cloud-plugin --url http://localhost:8065/plugins/com.mattermost.cloud/webhook
  ```
5. Set the following env variables:
  ```
  export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
  export MM_ADMIN_USERNAME=<your-sysadmin-username>
  export MM_ADMIN_PASSWORD=<your-sysadmin-password>
  ```
6. From the plugin directory, run `make deploy` to deploy the plugin to your local Mattermost server
7. Log in to the Mattermost server as the system admin, go to Plugins -> Mattermost Private Cloud and set the following settings:

  | Setting | Value |
  | - | - |
  | Provisioning Server URL | http://localhost:8075 |
  | Installation DNS | dev.cloud.mattermost.com |
  | Mattermost E10 License | `E10 license file contents` |
  | Mattermost E20 License | `E20 license file contents` |
8. Enable the plugin
The plugin is now usable. Make changes to your plugin and run `make deploy` again to run those changes on your local Mattermost server.


## Release

To release a new version of the `cloud plugin` please do the following steps:

- Bump the version to the desire version in `plugin.json` and then run `make apply`

- Create a PR with the changes, see [example](https://github.com/mattermost/mattermost-plugin-cloud/pull/52)

To trigger a release, follow these steps:

1. **For Patch Release:** Run the following command:
    ```
    make patch
    ```
   This will release a patch change.

2. **For Minor Release:** Run the following command:
    ```
    make minor
    ```
   This will release a minor change.

3. **For Major Release:** Run the following command:
    ```
    make major
    ```
   This will release a major change.

4. **For Patch Release Candidate (RC):** Run the following command:
    ```
    make patch-rc
    ```
   This will release a patch release candidate.

5. **For Minor Release Candidate (RC):** Run the following command:
    ```
    make minor-rc
    ```
   This will release a minor release candidate.

6. **For Major Release Candidate (RC):** Run the following command:
    ```
    make major-rc
    ```
   This will release a major release candidate.

