module github.com/giantswarm/starboard-exporter

go 1.16

require (
	github.com/aquasecurity/starboard v0.14.0
	github.com/go-logr/logr v1.2.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	k8s.io/apimachinery v0.23.0
	k8s.io/client-go v0.23.0
	sigs.k8s.io/controller-runtime v0.11.0
)

replace (
	github.com/containerd/containerd => github.com/containerd/containerd v1.5.7
	github.com/coreos/etcd => github.com/coreos/etcd v3.3.27+incompatible
	github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt v3.2.2+incompatible
)
