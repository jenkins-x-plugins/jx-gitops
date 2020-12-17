module github.com/jenkins-x/jx-gitops

require (
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/davecgh/go-spew v1.1.1
	github.com/fatih/color v1.10.0 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v0.3.0 // indirect
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/google/go-cmp v0.5.2
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/h2non/gock v1.0.9
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.11
	github.com/jenkins-x/go-scm v1.5.199
	github.com/jenkins-x/jx-api/v4 v4.0.14
	github.com/jenkins-x/jx-helpers/v3 v3.0.39
	github.com/jenkins-x/jx-kube-client/v3 v3.0.1
	github.com/jenkins-x/jx-logging/v3 v3.0.2
	github.com/jenkins-x/lighthouse v0.0.885
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/roboll/helmfile v0.135.1-0.20201213020320-54eb73b4239a
	github.com/rollout/rox-go v0.0.0-20181220111955-29ddae74a8c4
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/vrischmann/envconfig v1.3.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	golang.org/x/net v0.0.0-20201201195509-5d6afe98e0b7 // indirect
	golang.org/x/sys v0.0.0-20201201145000-ef89a241ccb3 // indirect
	golang.org/x/text v0.3.4 // indirect
	gopkg.in/validator.v2 v2.0.0-20200605151824-2b28d334fa05
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/helm v2.16.10+incompatible
	k8s.io/klog/v2 v2.4.0 // indirect
	sigs.k8s.io/kustomize/api v0.4.1
	sigs.k8s.io/kustomize/kyaml v0.6.1
	sigs.k8s.io/structured-merge-diff/v4 v4.0.2 // indirect
	sigs.k8s.io/yaml v1.2.0

)

replace (
	github.com/jenkins-x/lighthouse => github.com/rawlingsj/lighthouse v0.0.0-20201005083317-4d21277f7992

	// fix yaml comment parsing issue
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776

	k8s.io/client-go => k8s.io/client-go v0.19.2

	// fix yaml comment parsing issue
	sigs.k8s.io/kustomize/kyaml => sigs.k8s.io/kustomize/kyaml v0.6.1
	sigs.k8s.io/yaml => sigs.k8s.io/yaml v1.2.0
)

go 1.15
