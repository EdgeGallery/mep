module dns-server

go 1.14

replace k8s.io/client-go v2.0.0-alpha.0.0.20180817174322-745ca8300397+incompatible => github.com/kubernetes/client-go v0.0.0-20180817174322-745ca8300397

require (
	github.com/agiledragon/gomonkey v2.0.2+incompatible
	github.com/apache/servicecomb-service-center v0.0.0-20200414061342-d422a1f75fbd
	github.com/labstack/echo/v4 v4.1.16
	github.com/labstack/gommon v0.3.0 // indirect
	github.com/miekg/dns v1.1.29
	github.com/sirupsen/logrus v1.3.0
	github.com/stretchr/testify v1.4.0
	go.etcd.io/bbolt v1.3.4
	golang.org/x/sys v0.0.0-20200515095857-1151b9dac4a9 // indirect
)
