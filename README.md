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

- After the PR is merged, generate the release notes. Here will show the commands but if need more information please refer to the [internal docs](https://app.gitbook.com/@mattermost-private/s/internal-documentation/cloud/cloud/releases/rel-notes)

```bash
$ GO111MODULE=on go get k8s.io/release/cmd/release-notes
$ export GITHUB_TOKEN=<personal_github_api_token>
$ release-notes --github-org mattermost \
  --github-repo mattermost-plugin-cloud \
  --start-sha START_SHA \ # the GIT SHA from the previous release
  --end-sha END_SHA \ # the GIT SHA from the current release
  --debug \
  --output ./relnote.md \
  --required-author ""
```

This generates the release notes in the `relnote.md` you can apply any changes.

- Update your local repository and create a tag with the same version that you used in the previous step and add the `v` as prefix, for example:

```bash
$ git fetch origin --prune
$ git pull origin master
$ git tag vX.Y.Z
$ git push origin vX.Y.Z
```

This will kick the CI job to build/package and release the plugin in GitHub.

- Create a new Release by clicking and editing the Tag that is created in GitHub and paste the release notes that was generated in the step before.
- After the CI job is complete the package will be available in the Release as well.
