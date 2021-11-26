module github.com/embercsi/ember-csi-operator

go 1.14

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/kubernetes-csi/external-snapshotter/client/v4 v4.2.0
	github.com/operator-framework/operator-lib v0.4.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	sigs.k8s.io/controller-runtime v0.8.2
)

replace k8s.io/api => k8s.io/api v0.19.7

replace k8s.io/apimachinery => k8s.io/apimachinery v0.19.7

replace k8s.io/client-go => k8s.io/client-go v0.19.7
