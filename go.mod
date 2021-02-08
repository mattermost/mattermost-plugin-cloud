module github.com/mattermost/mattermost-plugin-starter-template

go 1.13

require (
	github.com/blang/semver/v4 v4.0.0
	github.com/docker/distribution v2.7.1+incompatible
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/heroku/docker-registry-client v0.0.0-20190909225348-afc9e1acc3d5
	github.com/mattermost/mattermost-cloud v0.39.1-0.20210210190544-89161840694f
	github.com/mattermost/mattermost-server v1.4.1-0.20190926112648-af3ffeed1a4a
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6 // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.14.1 // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	google.golang.org/genproto v0.0.0-20200117163144-32f20d992d24 // indirect
	google.golang.org/grpc v1.27.0 // indirect
	k8s.io/client-go v12.0.0+incompatible // indirect
)

// Pinned to kubernetes-1.17.7
replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	k8s.io/api => k8s.io/api v0.17.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.7
	k8s.io/apiserver => k8s.io/apiserver v0.17.7
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.7
	k8s.io/client-go => k8s.io/client-go v0.17.7
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.7
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.7
	k8s.io/code-generator => k8s.io/code-generator v0.17.7
	k8s.io/component-base => k8s.io/component-base v0.17.7
	k8s.io/cri-api => k8s.io/cri-api v0.17.7
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.7
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.7
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.7
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.7
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.7
	k8s.io/kubectl => k8s.io/kubectl v0.17.7
	k8s.io/kubelet => k8s.io/kubelet v0.17.7
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.7
	k8s.io/metrics => k8s.io/metrics v0.17.7
	k8s.io/node-api => k8s.io/node-api v0.17.7
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.7
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.17.7
	k8s.io/sample-controller => k8s.io/sample-controller v0.17.7
)
