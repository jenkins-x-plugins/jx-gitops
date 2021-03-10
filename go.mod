module github.com/jenkins-x/jx-gitops

require (
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/davecgh/go-spew v1.1.1
	github.com/fatih/color v1.10.0 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/google/go-cmp v0.5.4
	github.com/h2non/gock v1.0.9
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12
	github.com/jenkins-x/go-scm v1.5.229
	github.com/jenkins-x/jx-api/v4 v4.0.25
	github.com/jenkins-x/jx-helpers/v3 v3.0.86
	github.com/jenkins-x/jx-kube-client/v3 v3.0.2
	github.com/jenkins-x/jx-logging/v3 v3.0.3
	github.com/jenkins-x/lighthouse-client v0.0.63
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/roboll/helmfile v0.138.4
	github.com/rollout/rox-go v0.0.0-20181220111955-29ddae74a8c4
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/vrischmann/envconfig v1.3.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	gopkg.in/validator.v2 v2.0.0-20200605151824-2b28d334fa05
	helm.sh/helm/v3 v3.5.0
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/kustomize/api v0.4.1
	sigs.k8s.io/kustomize/kyaml v0.10.5
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// override the go-scm from tekton
	github.com/jenkins-x/go-scm => github.com/jenkins-x/go-scm v1.5.229
	// fix yaml comment parsing issue
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776

	k8s.io/api => k8s.io/api v0.20.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.3
	k8s.io/client-go => k8s.io/client-go v0.20.3

	// fix yaml comment parsing issue
	sigs.k8s.io/kustomize/kyaml => sigs.k8s.io/kustomize/kyaml v0.6.1
	sigs.k8s.io/yaml => sigs.k8s.io/yaml v1.2.0
)

go 1.15
