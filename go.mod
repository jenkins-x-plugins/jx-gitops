module github.com/jenkins-x/jx-gitops

require (
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/davecgh/go-spew v1.1.1
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/google/go-cmp v0.4.1
	github.com/h2non/gock v1.0.9
	github.com/jenkins-x/gen-crd-api-reference-docs v0.1.6 // indirect
	github.com/jenkins-x/go-scm v1.5.164
	github.com/jenkins-x/jx-api v0.0.18
	github.com/jenkins-x/jx-helpers v1.0.59
	github.com/jenkins-x/jx-kube-client v0.0.8
	github.com/jenkins-x/jx-logging v0.0.11
	github.com/jenkins-x/lighthouse v0.0.812
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/roboll/helmfile v0.125.7
	github.com/rollout/rox-go v0.0.0-20181220111955-29ddae74a8c4
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	gopkg.in/validator.v2 v2.0.0-20200605151824-2b28d334fa05
	k8s.io/api v0.18.1
	k8s.io/apimachinery v0.18.1
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/helm v2.16.10+incompatible
	sigs.k8s.io/kustomize/api v0.4.1
	sigs.k8s.io/kustomize/kyaml v0.6.1
	sigs.k8s.io/yaml v1.2.0

)

replace (
	// fix yaml comment parsing issue
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776

	k8s.io/api => k8s.io/api v0.17.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.6
	k8s.io/kubernetes => k8s.io/kubernetes v1.14.7

	// fix yaml comment parsing issue
	sigs.k8s.io/kustomize/kyaml => sigs.k8s.io/kustomize/kyaml v0.6.1
	sigs.k8s.io/yaml => sigs.k8s.io/yaml v1.2.0
)

go 1.13
