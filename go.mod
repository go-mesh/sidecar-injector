module github.com/go-mesh/sidecar-injector

require (
	github.com/Sirupsen/logrus v1.0.6
	github.com/docker/distribution v2.6.0-rc.1.0.20180913220339-b089e9168825+incompatible // indirect
	github.com/ghodss/yaml v1.0.1-0.20180820084758-c7ce16629ff4
	github.com/go-chassis/go-chassis v0.8.2
	github.com/howeyc/fsnotify v0.9.1-0.20151003194602-f0c08ee9c607
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect

	k8s.io/api v0.0.0-20180913155108-f456898a08e4
	k8s.io/apiextensions-apiserver v0.0.0-20180913001544-d65f4428c04f // indirect
	k8s.io/apimachinery v0.0.0-20180904193909-def12e63c512

	k8s.io/apiserver v0.0.0-20180914001516-67c892841170 // indirect
	k8s.io/client-go v8.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20180731170545-e3762e86a74c // indirect
	k8s.io/kubernetes v1.11.3
)

replace (
	github.com/kubernetes/apimachinery => ../k8s.io/apimachinery
	github.com/kubernetes/client-go => ../k8s.io/client-go
	github.com/kubernetes/kubernetes/ => ../k8s.io/kubernetes
)
