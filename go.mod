module github.com/jenkins-x/jx-gitops

require (
	cloud.google.com/go v0.76.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/davecgh/go-spew v1.1.1
	github.com/fatih/color v1.10.0 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/google/go-cmp v0.5.4
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.4 // indirect
	github.com/h2non/gock v1.0.9
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.11
	github.com/jenkins-x/go-scm v1.5.216
	github.com/jenkins-x/jx-api/v4 v4.0.24
	github.com/jenkins-x/jx-helpers/v3 v3.0.74
	github.com/jenkins-x/jx-kube-client/v3 v3.0.2
	github.com/jenkins-x/jx-logging/v3 v3.0.3
	github.com/jenkins-x/lighthouse-client v0.0.24
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/roboll/helmfile v0.138.4
	github.com/rollout/rox-go v0.0.0-20181220111955-29ddae74a8c4
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/vrischmann/envconfig v1.3.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/oauth2 v0.0.0-20210201163806-010130855d6c // indirect
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	gopkg.in/validator.v2 v2.0.0-20200605151824-2b28d334fa05
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	helm.sh/helm/v3 v3.5.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/klog/v2 v2.5.0 // indirect
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009 // indirect
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/kustomize/api v0.4.1
	sigs.k8s.io/kustomize/kyaml v0.10.6
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// fix yaml comment parsing issue
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776

	k8s.io/api => k8s.io/api v0.20.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2

	// fix yaml comment parsing issue
	sigs.k8s.io/kustomize/kyaml => sigs.k8s.io/kustomize/kyaml v0.6.1
	sigs.k8s.io/yaml => sigs.k8s.io/yaml v1.2.0

)

go 1.15
