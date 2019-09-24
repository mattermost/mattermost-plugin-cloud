module github.com/mattermost/mattermost-plugin-starter-template

go 1.12

require (
	github.com/heroku/docker-registry-client v0.0.0-20190909225348-afc9e1acc3d5
	github.com/mattermost/mattermost-cloud v0.3.0
	github.com/mattermost/mattermost-server v5.12.0+incompatible
	github.com/pkg/errors v0.8.1
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.4.0
)

// Workaround for https://github.com/golang/go/issues/30831 and fallout.
replace github.com/golang/lint => github.com/golang/lint v0.0.0-20190227174305-8f45f776aaf1

replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999
