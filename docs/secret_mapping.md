## Secret Mappings

When using the [jx-gitops extsecret](cmd/jx-gitops_extsecret.md) command to generate [ExternalSecret](https://github.com/godaddy/kubernetes-external-secrets) CRDs you may wish to use a custom mapping of Secret names and data keys to key/properties in Vault.

To do this just create a [.jx/gitops/secret-mappings.yaml](https://github.com/jenkins-x/jx-gitops/blob/master/.jx/gitops/secret-mappings.yaml) file in your directory tree when running the command. 

You can then customise the `key` and/or `property` values that are used in the generated [ExternalSecret](https://github.com/godaddy/kubernetes-external-secrets) CRDs
