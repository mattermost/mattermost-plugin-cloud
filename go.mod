module github.com/mattermost/mattermost-plugin-starter-template

go 1.12

require (
	github.com/emicklei/go-restful v2.10.0+incompatible // indirect
	github.com/go-openapi/jsonreference v0.19.3 // indirect
	github.com/go-openapi/spec v0.19.3 // indirect
	github.com/gogo/protobuf v1.3.0 // indirect
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/heroku/docker-registry-client v0.0.0-20190909225348-afc9e1acc3d5
	github.com/mailru/easyjson v0.7.0 // indirect
	github.com/mattermost/mattermost-cloud v0.6.2-0.20191018130017-04565f8ae5b1
	github.com/mattermost/mattermost-operator v0.7.0 // indirect
	github.com/mattermost/mattermost-server v1.4.1-0.20190926112648-af3ffeed1a4a
	github.com/pkg/errors v0.8.1
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.4.0
	golang.org/x/crypto v0.0.0-20190926114937-fa1a29108794 // indirect
	golang.org/x/net v0.0.0-20190926025831-c00fd9afed17 // indirect
	k8s.io/api v0.0.0-20190923155552-eac758366a00 // indirect
	k8s.io/kube-openapi v0.0.0-20190918143330-0270cf2f1c1d // indirect
)

// Workaround for https://github.com/golang/go/issues/30831 and fallout.
replace github.com/golang/lint => github.com/golang/lint v0.0.0-20190227174305-8f45f776aaf1

replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999
