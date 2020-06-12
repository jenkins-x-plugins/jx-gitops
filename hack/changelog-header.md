### Linux

```shell
curl -L https://github.com/jenkins-x/jx-gitops/releases/download/v{{.Version}}/jx-gitops-linux-amd64.tar.gz | tar xzv 
sudo mv jx-gitops /usr/local/bin
```

### macOS

```shell
curl -L  https://github.com/jenkins-x/jx-gitops/releases/download/v{{.Version}}/jx-gitops-darwin-amd64.tar.gz | tar xzv
sudo mv jx-gitops /usr/local/bin
```

