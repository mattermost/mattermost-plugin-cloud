# Targets of the dev stack in local/. The Makefile includes this file.
#
# The stack is Mattermost 12 with its database and this plugin installed.
# Each worktree gets its own stack: its own Compose project, volumes and host address.
# The host reaches the stack in one of two modes:
#   orbstack  The Docker engine is OrbStack. The host uses the name mattermost.<project>.orb.local.
#             The stack publishes no host port.
#   ports     Any other engine. The stack publishes one port on 127.0.0.1.
# Make picks the mode. Force it with: make dev-start DEV_HOST_MODE=ports
#
#   make dev-start   Start the stack, configure it, build and install the plugin.
#   make dev-deploy  Build and install the plugin again after a code change.
#   make dev-info    Show the name, the address and the state of the stack.
#   make dev-clean   Delete the containers, volumes and networks of the stack.
#
# Settings (give the same values to every make dev-* command of one stack):
#   DEV_NAME             Stack name. Default: the name of the worktree directory.
#   DEV_HOST_MODE        orbstack or ports. Default: orbstack when the Docker engine is OrbStack.
#   DEV_PORT_MATTERMOST  Host port of the ports mode. Default: from the project name.
#   DEV_MATTERMOST_IMAGE Mattermost image. Default: mattermost/mattermost-enterprise-edition:12.0
#   DEV_ADMIN_USERNAME, DEV_ADMIN_PASSWORD, DEV_ADMIN_EMAIL, DEV_TEAM_NAME  Admin account and team.
#
# Plugin settings. The plugin does not activate without a provisioning server URL, and the stack has no
# provisioning server. The default is a server on the host.
#   DEV_PROVISIONER_URL  URL of a provisioning server that Mattermost can reach.
#                        Default: http://host.docker.internal:8075
#   DEV_WEBHOOK_SECRET, DEV_INSTALLATION_DNS
#   DEV_E10_LICENSE_FILE, DEV_E20_LICENSE_FILE, DEV_PROFESSIONAL_LICENSE_FILE,
#   DEV_ENTERPRISE_LICENSE_FILE, DEV_ENTERPRISE_ADVANCED_LICENSE_FILE  Paths of license files.

.PHONY: dev-notice

ifneq ($(filter dev-%,$(MAKECMDGOALS)),)

DEV_SLUG := $(shell printf '%s' '$(notdir $(CURDIR))' | tr 'A-Z' 'a-z' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//' | cut -c1-28 | sed -E 's/-+$$//')
DEV_NAME ?= $(DEV_SLUG)
COMPOSE_PROJECT_NAME ?= mmcloud-$(DEV_NAME)
export COMPOSE_PROJECT_NAME

DEV_ENGINE := $(shell docker info --format '{{.OperatingSystem}}' 2>/dev/null)
ifndef DEV_HOST_MODE
ifneq ($(findstring OrbStack,$(DEV_ENGINE)),)
DEV_HOST_MODE := orbstack
else
DEV_HOST_MODE := ports
DEV_MODE_AUTO := 1
endif
endif
export DEV_HOST_MODE

DEV_PORT_MATTERMOST ?= $(shell printf '%s' '$(COMPOSE_PROJECT_NAME)' | cksum | awk '{print 20000 + $$1 % 10000}')
export DEV_PORT_MATTERMOST

DEV_ADMIN_USERNAME ?= admin
DEV_ADMIN_PASSWORD ?= Sysadmin123!
export DEV_ADMIN_USERNAME DEV_ADMIN_PASSWORD

ifeq ($(DEV_HOST_MODE),orbstack)
export DEV_URL_MATTERMOST := http://mattermost.$(COMPOSE_PROJECT_NAME).orb.local:8065
export COMPOSE_FILE := local/docker-compose.yml
else ifeq ($(DEV_HOST_MODE),ports)
export DEV_URL_MATTERMOST := http://127.0.0.1:$(DEV_PORT_MATTERMOST)
export COMPOSE_FILE := local/docker-compose.yml:local/docker-compose.ports.yml
else
$(error DEV_HOST_MODE must be orbstack or ports, not "$(DEV_HOST_MODE)")
endif

# Settings that build/custom.mk can define. The scripts and Compose read them from the environment.
export DEV_PROVISIONER_URL DEV_WEBHOOK_SECRET DEV_INSTALLATION_DNS DEV_MATTERMOST_IMAGE DEV_ADMIN_EMAIL DEV_TEAM_NAME
export DEV_E10_LICENSE_FILE DEV_E20_LICENSE_FILE DEV_PROFESSIONAL_LICENSE_FILE DEV_ENTERPRISE_LICENSE_FILE DEV_ENTERPRISE_ADVANCED_LICENSE_FILE

export PLUGIN_ID
DEV_COMPOSE = docker compose

endif

## Prints the host access mode, and a warning when the mode is not OrbStack.
dev-notice:
	@echo "==> Dev stack $(COMPOSE_PROJECT_NAME), host access: $(DEV_HOST_MODE)"
ifneq ($(DEV_MODE_AUTO),)
	@echo "WARNING: The Docker engine is not OrbStack. The stack publishes port $(DEV_PORT_MATTERMOST) on 127.0.0.1." >&2
	@echo "WARNING: Stop other programs that use this port, or set DEV_PORT_MATTERMOST=<port>." >&2
endif
ifeq ($(DEV_HOST_MODE)$(findstring OrbStack,$(DEV_ENGINE)),orbstack)
	@echo "WARNING: DEV_HOST_MODE=orbstack is set, but the Docker engine is not OrbStack. The *.orb.local names do not resolve." >&2
endif

## Creates the dev stack, configures it and installs the plugin.
.PHONY: dev-start
dev-start: dev-notice
	./local/bootstrap.sh

## Builds the plugin bundle and deploys it to the dev Mattermost.
# COPYFILE_DISABLE stops macOS tar from adding ._ files that the Linux server counts as extra entries.
.PHONY: dev-deploy
dev-deploy: dev-notice
	COPYFILE_DISABLE=1 \
	MM_SERVICESETTINGS_SITEURL=$(DEV_URL_MATTERMOST) \
	MM_ADMIN_USERNAME='$(DEV_ADMIN_USERNAME)' \
	MM_ADMIN_PASSWORD='$(DEV_ADMIN_PASSWORD)' \
	$(MAKE) deploy

## Prints the name, the address and the state of the dev stack of this worktree.
.PHONY: dev-info
dev-info: dev-notice
	@echo "project:    $(COMPOSE_PROJECT_NAME)"
	@echo "mattermost: $(DEV_URL_MATTERMOST)"
	@echo "login:      $(DEV_ADMIN_USERNAME) / $(DEV_ADMIN_PASSWORD)"
	@echo "plugin:     $(PLUGIN_ID)"
	@echo ""
	-@$(DEV_COMPOSE) ps --format 'table {{.Service}}\t{{.Status}}'
	@echo ""
	-@$(DEV_COMPOSE) exec -T mattermost mmctl --local plugin list

## Deletes the dev containers, volumes and networks of this worktree.
.PHONY: dev-clean
dev-clean:
	$(DEV_COMPOSE) down --volumes --remove-orphans
