module github.com/openyurtio/openyurt

go 1.13

replace (
	google.golang.org/grpc v1.27.0 => google.golang.org/grpc v1.26.0
	k8s.io/api => k8s.io/api v0.16.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.10-beta.0
	k8s.io/client-go => k8s.io/client-go v0.16.9
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client => sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.15
)

require (
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	google.golang.org/grpc v1.27.0
	k8s.io/api v0.17.1
	k8s.io/apimachinery v0.17.1
	k8s.io/client-go v0.17.1
	k8s.io/klog/v2 v2.5.0
	sigs.k8s.io/apiserver-network-proxy v0.0.15
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.0
	yunion.io/x/log v0.0.0-20201210064738-43181789dc74 // indirect
	yunion.io/x/pkg v0.0.0-20210218105412-13a69f60034c
)
