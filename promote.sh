#!/bin/bash

echo "promoting the new version ${VERSION} to downstream repositories"

jx step create pr regex --regex '\s+GitOpsVersion = "(?P<version>.*)"' --version ${VERSION} --files pkg/plugins/versions.go --repo https://github.com/jenkins-x/jx-cli.git

jx step create pr regex \
    --regex 'version: (.*)' \
    --version ${VERSION} \
    --files docker/gcr.io/jenkinsxio-labs-private/jx-gitops.yml \
    --repo https://github.com/jenkins-x/jxr-versions.git