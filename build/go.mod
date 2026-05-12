module github.com/mattermost/mattermost-plugin-cloud/build

go 1.26.2

require (
	github.com/mattermost/mattermost/server/public v0.3.1-0.20260402155910-d9d71af83e3f
	github.com/pkg/errors v0.9.1
)

require (
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dyatlov/go-opengraph/opengraph v0.0.0-20220524092352-606d7b1e5f8a // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/francoispqt/gojay v1.2.13 // indirect
	github.com/go-asn1-ber/asn1-ber v1.5.7 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.7.0 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/mattermost/go-i18n v1.11.1-0.20211013152124-5c415071e404 // indirect
	github.com/mattermost/ldap v0.0.0-20231116144001-0f480c025956 // indirect
	github.com/mattermost/logr/v2 v2.0.22 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/oklog/run v1.2.0 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/tinylib/msgp v1.6.3 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/wiggin77/merror v1.0.5 // indirect
	github.com/wiggin77/srslog v1.0.1 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/mod v0.34.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260319201613-d00831a3d3e7 // indirect
	google.golang.org/grpc v1.79.3 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Hack to prevent the willf/bitset module from being upgraded to 1.2.0.
// They changed the module path from github.com/willf/bitset to
// github.com/bits-and-blooms/bitset and a couple of dependent repos are yet
// to update their module paths.
exclude (
	github.com/RoaringBitmap/roaring v0.7.0
	github.com/RoaringBitmap/roaring v0.7.1
	github.com/dyatlov/go-opengraph v0.0.0-20210112100619-dae8665a5b09
	github.com/willf/bitset v1.2.0
)
