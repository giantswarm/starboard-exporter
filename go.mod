module github.com/giantswarm/starboard-exporter

go 1.21

toolchain go1.21.4

require (
	github.com/aquasecurity/trivy-operator v0.19.0
	github.com/buraksezer/consistent v0.10.0
	github.com/cespare/xxhash/v2 v2.2.0
	github.com/go-logr/logr v1.4.1
	github.com/google/go-cmp v0.6.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.19.0
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.29.2
	k8s.io/apimachinery v0.29.2
	k8s.io/client-go v0.29.2
	sigs.k8s.io/controller-runtime v0.17.2
)

require (
	github.com/anchore/go-struct-converter v0.0.0-20230627203149-c72ef8859ca9 // indirect
	github.com/aquasecurity/defsec v0.94.1 // indirect
	github.com/aquasecurity/go-dep-parser v0.0.0-20240202105001-4f19ab402b0b // indirect
	github.com/aquasecurity/table v1.8.0 // indirect
	github.com/aquasecurity/tml v0.6.1 // indirect
	github.com/aquasecurity/trivy v0.49.1 // indirect
	github.com/aquasecurity/trivy-db v0.0.0-20231204103057-c212f374b52e // indirect
	github.com/aquasecurity/trivy-policies v0.8.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/evanphx/json-patch/v5 v5.8.0 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.20.1 // indirect
	github.com/go-openapi/jsonreference v0.20.3 // indirect
	github.com/go-openapi/swag v0.22.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/gnostic-models v0.6.9-0.20230804172637-c7be7c783f49 // indirect
	github.com/google/go-containerregistry v0.19.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/masahiro331/go-xfs-filesystem v0.0.0-20231205045356-1b22259a6c44 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/package-url/packageurl-go v0.1.2 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/samber/lo v1.39.0 // indirect
	github.com/spdx/tools-golang v0.5.4-0.20231108154018-0c0f394b5e1a // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/exp v0.0.0-20231206192017-f3f8817b8deb // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/oauth2 v0.16.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/term v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.29.2 // indirect
	k8s.io/component-base v0.29.2 // indirect
	k8s.io/klog/v2 v2.120.0 // indirect
	k8s.io/kube-openapi v0.0.0-20231206194836-bf4651e18aa8 // indirect
	k8s.io/utils v0.0.0-20231127182322-b307cd553661 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace github.com/go-git/go-git/v5 v5.10.1 => github.com/go-git/go-git/v5 v5.11.0

replace go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.42.0 => go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.47.0

replace github.com/moby/buildkit v0.12.3 => github.com/moby/buildkit v0.12.5
