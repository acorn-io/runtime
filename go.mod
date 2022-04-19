module github.com/ibuildthecloud/herd

go 1.18

replace (
	github.com/docker/docker => github.com/docker/docker v20.10.3-0.20220121014307-40bb9831756f+incompatible
	github.com/rancher/wrangler => github.com/ibuildthecloud/wrangler v0.0.0-20220405013541-7c01be7907d2
	k8s.io/client-go => k8s.io/client-go v0.23.0
)

require (
	cuelang.org/go v0.4.3-beta.1
	github.com/AlecAivazis/survey/v2 v2.3.4
	github.com/containerd/console v1.0.3
	github.com/containerd/containerd v1.6.0-rc.2
	github.com/google/go-containerregistry v0.8.1-0.20220110151055-a61fd0a8e2bb
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20220413173345-f1b065c6cb3d
	github.com/google/go-containerregistry/pkg/authn/kubernetes v0.0.0-20220413173345-f1b065c6cb3d
	github.com/goombaio/namegenerator v0.0.0-20181006234301-989e774b106e
	github.com/gorilla/websocket v1.4.2
	github.com/ibuildthecloud/baaah v0.0.0-20220419175848-b53f32e0abbc
	github.com/moby/buildkit v0.10.0-rc1.0.20220216012946-b36d860675c5
	github.com/opencontainers/image-spec v1.0.3-0.20220114050600-8b9d41f48198
	github.com/pkg/errors v0.9.1
	github.com/rancher/lasso v0.0.0-20220110204427-db9f3e569574
	github.com/rancher/wrangler v0.8.10
	github.com/rancher/wrangler-cli v0.0.0-20211112052728-f172e9bf59af
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.3.0
	github.com/stretchr/testify v1.7.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.23.4
	k8s.io/apimachinery v0.23.4
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/kubectl v0.0.0-20191219154910-1528d4eea6dd
	k8s.io/kubernetes v1.13.0
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
	sigs.k8s.io/controller-runtime v0.11.0
)

require (
	cloud.google.com/go/compute v1.5.0 // indirect
	github.com/Azure/azure-sdk-for-go v62.0.0+incompatible // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.24 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.18 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.11 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.5 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/Microsoft/go-winio v0.5.1 // indirect
	github.com/aws/aws-sdk-go-v2 v1.14.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.14.0 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.9.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.11.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.3.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.15.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.10.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.15.0 // indirect
	github.com/aws/smithy-go v1.11.0 // indirect
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.0.0-20220228164355-396b2034c795 // indirect
	github.com/chrismellard/docker-credential-acr-env v0.0.0-20220119192733-fe33c00cee21 // indirect
	github.com/cockroachdb/apd/v2 v2.0.1 // indirect
	github.com/containerd/continuity v0.2.2 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.11.0 // indirect
	github.com/containerd/typeurl v1.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/docker/cli v20.10.12+incompatible // indirect
	github.com/docker/distribution v2.8.0+incompatible // indirect
	github.com/docker/docker v20.10.12+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.6.4 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/emicklei/proto v1.6.15 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/logr v1.2.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gofrs/flock v0.8.1 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.3.0 // indirect
	github.com/golang/glog v1.0.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/klauspost/compress v1.14.2 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/sys/signal v0.6.0 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mpvl/unique v0.0.0-20150818121801-cbe035fff7de // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/protocolbuffers/txtpbfmt v0.0.0-20201118171849-f6a6b3f636fc // indirect
	github.com/rancher/lasso/controller-runtime v0.0.0-20220110205840-98715bdd6b5b // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/tonistiigi/fsutil v0.0.0-20220115021204-b19f7f9cb274 // indirect
	github.com/tonistiigi/units v0.0.0-20180711220420-6950e57a87ea // indirect
	github.com/tonistiigi/vt100 v0.0.0-20210615222946-8066bb97264f // indirect
	github.com/vbatts/tar-split v0.11.2 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.29.0 // indirect
	go.opentelemetry.io/otel v1.4.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.4.0 // indirect
	go.opentelemetry.io/otel/sdk v1.4.0 // indirect
	go.opentelemetry.io/otel/trace v1.4.0 // indirect
	go.opentelemetry.io/proto/otlp v0.12.0 // indirect
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/oauth2 v0.0.0-20220223155221-ee480838109b // indirect
	golang.org/x/sys v0.0.0-20220227234510-4e6760a101f9 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20220224211638-0e9765cccd65 // indirect
	golang.org/x/tools v0.1.8 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220222213610-43724f9ea8cf // indirect
	google.golang.org/grpc v1.44.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apiextensions-apiserver v0.23.0 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.40.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220124234850-424119656bbf // indirect
	sigs.k8s.io/controller-tools v0.8.0 // indirect
	sigs.k8s.io/json v0.0.0-20211208200746-9f7c6b3444d2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
