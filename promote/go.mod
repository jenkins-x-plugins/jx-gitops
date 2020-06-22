module github.com/jenkins-x/jxl-promote

require (
	github.com/apache/thrift v0.12.0 // indirect
	github.com/gophercloud/gophercloud v0.1.0 // indirect
	github.com/jenkins-x/jx v0.0.0-20200501072805-de27d267fa83
	github.com/jenkins-x/jx/v2 v2.1.78 // indirect
	github.com/openzipkin/zipkin-go v0.1.6 // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.4 // indirect
	github.com/rifflock/lfshook v0.0.0-20180920164130-b9218ef580f5 // indirect
	k8s.io/metrics v0.0.0-20190704050707-780c337c9cbd // indirect
)

exclude github.com/jenkins-x/jx/v2/pkg/prow v0.0.0-20190912224545-e8f82ee218ba

replace github.com/golang/lint => golang.org/x/lint v0.0.0-20180702182130-06c8688daad7

replace github.com/heptio/sonobuoy => github.com/jenkins-x/sonobuoy v0.11.7-0.20190318120422-253758214767

replace k8s.io/api => k8s.io/api v0.0.0-20190528110122-9ad12a4af326

replace k8s.io/metrics => k8s.io/metrics v0.0.0-20181128195641-3954d62a524d

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221084156-01f179d85dbc

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190528110200-4f3abb12cae2

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190528110544-fa58353d80f3

replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999

replace github.com/sirupsen/logrus => github.com/jtnord/logrus v1.4.2-0.20190423161236-606ffcaf8f5d

replace github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v21.1.0+incompatible

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v10.15.5+incompatible

replace github.com/banzaicloud/bank-vaults => github.com/banzaicloud/bank-vaults v0.0.0-20190508130850-5673d28c46bd

go 1.12
