module mepserver

go 1.14

replace (
	github.com/coreos/etcd v3.3.6+incompatible => github.com/coreos/etcd v3.3.13+incompatible
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b => github.com/go-chassis/glog v0.0.0-20180920075250-95a09b2413e9
	github.com/gorilla/websocket v1.2.0 => github.com/gorilla/websocket v1.4.1
	k8s.io/client-go v2.0.0-alpha.0.0.20180817174322-745ca8300397+incompatible => github.com/kubernetes/client-go v0.0.0-20180817174322-745ca8300397
)

require (
	github.com/agiledragon/gomonkey v2.0.1+incompatible
	github.com/apache/servicecomb-service-center v0.0.0-20191027084911-c2dc0caef706
	github.com/astaxie/beego v1.12.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-chassis/paas-lager v1.1.1 // indirect
	github.com/go-mesh/openlogging v1.0.1 // indirect
	github.com/go-playground/validator/v10 v10.4.1
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0 // indirect
	github.com/olivere/elastic/v7 v7.0.20
	github.com/satori/go.uuid v1.2.0
	github.com/shiena/ansicolor v0.0.0-20200904210342-c7312218db18 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a
)
