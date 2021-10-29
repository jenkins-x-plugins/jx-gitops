module github.com/jenkins-x-plugins/jx-gitops

require (
	cloud.google.com/go/storage v1.11.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/davecgh/go-spew v1.1.1
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.2.0
	github.com/h2non/gock v1.0.9
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12
	github.com/jenkins-x-plugins/jx-charter v0.0.28
	github.com/jenkins-x/go-scm v1.10.10
	github.com/jenkins-x/jx-api/v4 v4.1.5
	github.com/jenkins-x/jx-helpers/v3 v3.0.127
	github.com/jenkins-x/jx-kube-client/v3 v3.0.2
	github.com/jenkins-x/jx-logging/v3 v3.0.6
	github.com/jenkins-x/lighthouse-client v0.0.233
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/roboll/helmfile v0.139.0
	github.com/rollout/rox-go v0.0.0-20181220111955-29ddae74a8c4
	github.com/spf13/cobra v1.2.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/tektoncd/pipeline v0.26.0
	github.com/vrischmann/envconfig v1.3.0 // indirect
	gopkg.in/validator.v2 v2.0.0-20200605151824-2b28d334fa05
	helm.sh/helm/v3 v3.6.3
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	sigs.k8s.io/kustomize/api v0.8.7
	sigs.k8s.io/kustomize/kyaml v0.10.17
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// helm dependencies
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible

	github.com/go-openapi/spec => github.com/go-openapi/spec v0.20.2

	//knative.dev/pkg => knative.dev/pkg v0.0.0-20210730172132-bb4aaf09c430

	// override the go-scm from tekton
	github.com/jenkins-x/go-scm => github.com/jenkins-x/go-scm v1.10.10

	// for the PipelineRun debug fix see: https://github.com/tektoncd/pipeline/pull/4145
	github.com/tektoncd/pipeline => github.com/jstrachan/pipeline v0.21.1-0.20210811150720-45a86a5488af

	k8s.io/api => k8s.io/api v0.20.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.7
	k8s.io/client-go => k8s.io/client-go v0.20.7

	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20210113233702-8566a335510f
	sigs.k8s.io/kustomize/api => sigs.k8s.io/kustomize/api v0.8.7
	sigs.k8s.io/kustomize/kyaml => sigs.k8s.io/kustomize/kyaml v0.10.17
)

go 1.15
