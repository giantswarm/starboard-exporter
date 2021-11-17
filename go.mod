module github.com/giantswarm/starboard-exporter

go 1.16

require (
	github.com/aquasecurity/starboard v0.13.0
	github.com/go-logr/logr v0.4.0
	k8s.io/apimachinery v0.22.3
	k8s.io/client-go v0.22.3
	sigs.k8s.io/controller-runtime v0.10.3
)

replace (
	github.com/containerd/containerd => github.com/containerd/containerd v1.5.7
	github.com/coreos/etcd => github.com/coreos/etcd v3.3.27+incompatible
	github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt v3.2.2+incompatible
)
