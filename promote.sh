#!/bin/bash

echo "promoting the new version ${VERSION} to downstream repositories"

jx step create pr regex --regex '\s+GitOpsVersion = "(?P<version>.*)"' --version ${VERSION} --files pkg/plugins/versions.go --repo https://github.com/jenkins-x/jx-cli.git

