## Git Layout

A given cluster/environment should have a directory in git where all the kubernetes resources reside in a format as close to their natural Kubernetes YAML format. We default to the directory `config-root` but any folder will do. 
                                                                                                                           
We follow the [naming conventions](https://cloud.google.com/anthos-config-management/docs/concepts/repo) defined by [Anthos Config Management](https://cloud.google.com/anthos/config-management) such that within the cluster/environments directory in git:

* `cluster/`  contains all the sources which are cluster wide (like Custom Resource Definitions or these kubernetes resources: `Namespace`, `ClusterRole`, `ClusterRoleBinding`)
* `namespaces/` contains all the namespaced resources. We also use the first level directory to denote the directory name. So for resources in namespace `foo` may be in `namespaces/foo/myapp/deployment.yaml` for example.

Also note that any resource in a specific namespace will need the `metadata.namespace` property set to the namespace. Otherwise resources default to using the `default` namespace.

You can use the [jx-gitops namespace](https://github.com/jenkins-x-plugins/jx-gitops/blob/master/docs/cmd/jx-gitops_namespace.md) command to set the namespaces on your kubernetes resources. Or to ensure namespaces are set based on the child directory name within `namespaces/` use [jx-gitops namespace --dir-mode --dir config-root/namespaces](https://github.com/jenkins-x-plugins/jx-gitops/blob/master/docs/cmd/jx-gitops_namespace.md)  

  
Many tools can be used to fetch the YAML files from repositories and modify them such as any permutation of:

* git 
* [helm](https://helm.sh/) 
* [kpt](https://googlecontainertools.github.io/kpt/)
* [kustomize](https://kustomize.io/)

However we see those as tools that should be used at build time to generate the YAML and then check it into Git. The most important thing is to standise on what gets checked into git + that gets released: namely the standard kubernetes resources and custom resources.

### Secrets

We obviously don't want to commit raw kubernetes `Secret` YAML into git!

You can use sealed secrets. 

We highly recommend using [Kubernetes External Secrets](https://github.com/godaddy/kubernetes-external-secrets) which means you can check in the `ExternalSecret` resources into git which are a reference to the actual secret values from some provider:

* Alibaba Cloud KMS Secret Manager
* Amazon Secret Manager
* Amazon Parameter Store
* Azure Key Vault 
* Google Secret Manager
* HashiCorp Vault

If you use [jx-gitops extsecret](https://github.com/jenkins-x-plugins/jx-gitops/blob/master/docs/cmd/jx-gitops_extsecret.md) [jx-gitops helm template](https://github.com/jenkins-x-plugins/jx-gitops/blob/master/docs/cmd/jx-gitops_helm_template.md) commands all of your kubernetes `Secret`resources will get automatically converted to `ExternalSecret` resources so you can safely check them into your git repository. You may also find the [document on Secret Mapping](secret_mapping.md) useful


### Deployment Tools

We should strive to support as many deployment tools as possible. The above layouts should work fine with tools such as:

* `kubectl apply`
* Tekton Pipelines using `kubectl apply`
* [Flux](https://fluxcd.io/)
* [Anthos Config Management](https://cloud.google.com/anthos/config-management)
 
### Pull Requests

Proposing pull requests on your git repositories are an excellent way to get reviews from your team and to run any automated tooling to verify the resources.

We recommend your Pull Request pipeline should re-run the generation scripts and check in the output to your Pull Requests to ensure that the Pull Request is complete and folks can review it to see exactly what is going to be changed.

e.g. by running:

```bash 
make build commit
```

If a Pull Request upgrades a version of a [helm](https://helm.sh/) chart or [kpt](https://googlecontainertools.github.io/kpt/) package with a simple one liner, seeing a second commit on the Pull Request with the actual changes to the kubernetes `Deployment` resource in terms of changes to images, volumes, environment variables and so forth is extremely useful.


### Makefile

This is not a requirement but using a `Makefile` to trigger the various tools like [helm](https://helm.sh/), [kpt](https://googlecontainertools.github.io/kpt/) or [kustomize](https://kustomize.io/) can help provide the same CLI UX and use the same pipeline irrespective of which permutation of tools are used.

e.g. to fetch/build/enrich the YAML:

```bash
make all 
```

To apply into the current kubernetes cluster:

```bash
make apply 
```

### Linting

We recommend you lint your YAML files to ensure a consistent layout. If using tools like `helm` we recommend splitting YAML files into a file per resource to simplify understanding and to make things easier to process with tools. 

e.g. use [jx-gitops split](https://github.com/jenkins-x-plugins/jx-gitops/blob/master/docs/cmd/jx-gitops_split.md) in your `Makefile`.