## Kind Filters

Its useful to query YAML files to modify (e.g. `jx-gitops label`) via the resource kind.

You can filter by the `kind` property without the `apiVersion` property like this:

```bash 
jx-gitops label --kind Deployment mylabel=somevalue 
```

If you want to filter by an `apiVersion` too you can add it as a prefix using `/` as a separator:

```bash 
jx-gitops label --kind app/v1/Deployment mylabel=somevalue 
```

You can use a prefix of the apiVersion too if you want:

```bash 
jx-gitops label --kind app/Deployment mylabel=somevalue 
```

You can omit the kind to match all resources of an `apiVersion` (or `apiVersion` prefix)

```bash 
jx-gitops label --kind app/v1/ mylabel=somevalue 
```

