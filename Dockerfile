FROM gcr.io/jenkinsxio/jx-cli-base:latest

ENTRYPOINT ["jx-gitops"]

COPY ./build/linux/jx-gitops /usr/bin/jx-gitops