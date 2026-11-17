#!/usr/bin/env bash
# Configures the dev stack and installs the plugin. Safe to run again: every create step ignores "already exists".
set -euo pipefail

cd "$(dirname "$0")/.."

: "${COMPOSE_PROJECT_NAME:?Run this script through make: make dev-start}"
: "${DEV_URL_MATTERMOST:?Run this script through make: make dev-start}"
MM_URL=$DEV_URL_MATTERMOST
PLUGIN_ID=com.mattermost.cloud
ADMIN_USER=${DEV_ADMIN_USERNAME:-admin}
ADMIN_PASS=${DEV_ADMIN_PASSWORD:-Sysadmin123!}
ADMIN_EMAIL=${DEV_ADMIN_EMAIL:-admin@example.com}
TEAM_NAME=${DEV_TEAM_NAME:-dev}

dc() { docker compose "$@"; }
mmctl() { dc exec -T mattermost mmctl --local "$@"; }

# ensure <description> <already-exists-pattern> <command...>
ensure() {
	local what=$1 pattern=$2 out
	shift 2
	if out=$("$@" 2>&1); then
		echo "  created: $what"
	elif grep -qiE "$pattern" <<<"$out"; then
		echo "  exists:  $what"
	else
		echo "  FAILED:  $what" >&2
		echo "$out" >&2
		return 1
	fi
}

wait_for() {
	local what=$1 timeout=$2
	shift 2
	local deadline=$((SECONDS + timeout))
	until "$@" >/dev/null 2>&1; do
		if ((SECONDS >= deadline)); then
			echo "Timed out waiting for $what" >&2
			exit 1
		fi
		sleep 2
	done
}

# Prints the contents of the license file that the variable <name> points to, or an empty string.
license() {
	local path=${!1:-}
	if [ -n "$path" ]; then
		cat "$path" || {
			echo "Cannot read $1=$path" >&2
			return 1
		}
	fi
}

echo "==> Starting services"
dc up -d --remove-orphans

echo "==> Waiting for Mattermost"
wait_for "Mattermost" 300 curl -fs "$MM_URL/api/v4/system/ping"

echo "==> Mattermost accounts"
ensure "user $ADMIN_USER" "already exists|exists" \
	mmctl user create --email "$ADMIN_EMAIL" --username "$ADMIN_USER" --password "$ADMIN_PASS" --system-admin --email-verified
ensure "team $TEAM_NAME" "already exists|exists" \
	mmctl team create --name "$TEAM_NAME" --display-name "Dev" --email "$ADMIN_EMAIL"
mmctl team users add "$TEAM_NAME" "$ADMIN_USER" >/dev/null

# The plugin does not activate without a provisioning server URL. The stack has none, so the default is a
# server that runs on the host. Licenses are needed only to create installations.
echo "==> Plugin settings"
e10_license=$(license DEV_E10_LICENSE_FILE)
e20_license=$(license DEV_E20_LICENSE_FILE)
professional_license=$(license DEV_PROFESSIONAL_LICENSE_FILE)
enterprise_license=$(license DEV_ENTERPRISE_LICENSE_FILE)
enterprise_advanced_license=$(license DEV_ENTERPRISE_ADVANCED_LICENSE_FILE)
config_file=$(mktemp)
trap 'rm -f "$config_file"' EXIT
jq -n \
	--arg id "$PLUGIN_ID" \
	--arg url "${DEV_PROVISIONER_URL:-http://host.docker.internal:8075}" \
	--arg secret "${DEV_WEBHOOK_SECRET:-dev-webhook-secret}" \
	--arg dns "${DEV_INSTALLATION_DNS:-dev.cloud.mattermost.com}" \
	--arg e10 "$e10_license" \
	--arg e20 "$e20_license" \
	--arg professional "$professional_license" \
	--arg enterprise "$enterprise_license" \
	--arg enterpriseadvanced "$enterprise_advanced_license" \
	'{PluginSettings: {Plugins: {($id): {
		provisioningserverurl: $url,
		provisioningserverwebhooksecret: $secret,
		installationdns: $dns,
		e10license: $e10,
		e20license: $e20,
		professionallicense: $professional,
		enterpriselicense: $enterprise,
		enterpriseadvancedlicense: $enterpriseadvanced
	}}}} | .PluginSettings.Plugins[$id] |= with_entries(select(.value != ""))' >"$config_file"

# The Mattermost image has no shell, so the patch file is copied in instead of piped.
chmod 644 "$config_file"
dc cp "$config_file" mattermost:/tmp/cloud-dev-config.json
mmctl config patch /tmp/cloud-dev-config.json >/dev/null

echo "==> Deploying plugin"
make dev-deploy

cat <<EOF

Ready.

  Stack       $COMPOSE_PROJECT_NAME
  Mattermost  $MM_URL
  Login       $ADMIN_USER / $ADMIN_PASS
  Team        $TEAM_NAME
  Plugin      $PLUGIN_ID
EOF
