module github.com/mattermost/mattermost-plugin-starter-template

go 1.12

require (
	github.com/braintree/manners v0.0.0-20160418043613-82a8879fc5fd // indirect
	github.com/cpanato/golang-jenkins v0.0.0-20181010175751-6a66fc16d07d // indirect
	github.com/google/go-github v17.0.0+incompatible
	github.com/mattermost/mattermost-cloud v0.3.0
	github.com/mattermost/mattermost-mattermod v0.0.0-20190718124140-f9ed1a92db14 // indirect
	github.com/mattermost/mattermost-server v5.12.0+incompatible
	github.com/pkg/errors v0.8.1
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.3.0
)

// Workaround for https://github.com/golang/go/issues/30831 and fallout.
replace github.com/golang/lint => github.com/golang/lint v0.0.0-20190227174305-8f45f776aaf1
