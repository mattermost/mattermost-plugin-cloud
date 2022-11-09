module github.com/mattermost/mattermost-plugin-cloud

go 1.15

require (
	github.com/blang/semver/v4 v4.0.0
	github.com/docker/distribution v2.7.1+incompatible
	github.com/heroku/docker-registry-client v0.0.0-20190909225348-afc9e1acc3d5
	github.com/mattermost/mattermost-cloud v0.53.2-0.20220314101214-00a3172ee2c7
	github.com/mattermost/mattermost-server/v6 v6.0.2
	github.com/mholt/archiver/v3 v3.5.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.1
)

replace k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
