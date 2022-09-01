## Chart Repository

[Helm](https://helm.sh) must be installed to use the charts.
Please refer to Helm's [documentation](https://helm.sh/docs/) to get started.

### Searching for charts

Once Helm is set up properly, add the repo as follows:

    helm repo add myrepo http://bucketrepo.jx.svc.cluster.local/bucketrepo/pages

you can search the charts via:

    helm search repo myfilter

## View the YAML

You can have a look at the underlying charts YAML at: [index.yaml](index.yaml)
