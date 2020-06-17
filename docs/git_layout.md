## Git Layout

A given cluster/environment should have a directory in git where all the kubernetes resources reside in a format as close to their natural Kubernetes YAML format. We default to the directory `config-root` but any folder will do. 
                                                                                                                           
Many tools can be used to fetch the YAML files from repositories and modify them such as any permutation of:

* git 
* [helm](https://helm.sh/) 
* [kpt](https://googlecontainertools.github.io/kpt/)
* [kustomize](https://kustomize.io/)

However we see those as tools that should be used at build time to generate the YAML and then check it into Git. The most important thing is to standise on what gets checked into git + that gets released: namely the standard kubernetes resources and custom resources.

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

e.g. use [jx-gitops split](https://github.com/jenkins-x/jx-gitops/blob/master/docs/cmd/jx-gitops_split.md) in your `Makefile`.


Tool agnostic - close to real 
PR include changes
Ext secrets 
