# jx-gitops

[![Documentation](https://godoc.org/github.com/jenkins-x/jx-gitops?status.svg)](https://pkg.go.dev/mod/github.com/jenkins-x/jx-gitops)
[![Go Report Card](https://goreportcard.com/badge/github.com/jenkins-x/jx-gitops)](https://goreportcard.com/report/github.com/jenkins-x/jx-gitops)
[![Releases](https://img.shields.io/github/release-pre/jenkins-x/jx-gitops.svg)](https://github.com/jenkins-x/jx-gitops/releases)
[![LICENSE](https://img.shields.io/github/license/jenkins-x/jx-gitops.svg)](https://github.com/jenkins-x/jx-gitops/blob/master/LICENSE)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://slack.k8s.io/)

jx-gitops is a small command line tool working with Kubernetes resources and CRD files in a GitOps repository.

## Commands

See the [jx-gitops command reference](docs/cmd/jx-gitops.md#see-also)

## Documentation

* [Git Repository Layout](docs/git_layout.md) on how to structure the source code of a GitOps repository
* [Secret Mapping](docs/secret_mapping.md) for mapping Secrets to External Secrets and underlying storage
    * [Secret Mapping Reference](docs/config.md) reference guide for configuring secret mappings
* [Kind Filters](docs/kind_filters.md) for how to filter resources by `kind` with some of the [commands](docs/cmd/jx-gitops.md)   
