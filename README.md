# Mattermost Cloud Plugin

This plugin allows for the creation and management of Mattermost kubernetes installations directly from Mattermost. Commands are exposed that can be used to communicate with a remote [cloud server](https://github.com/mattermost/mattermost-cloud).

## Getting Started

1. Install the plugin.
2. Enter configuration information for your environment and enable the plugin.
3. From any channel or DM, run `/cloud -h` for information on using the plugin.

## Developing

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

