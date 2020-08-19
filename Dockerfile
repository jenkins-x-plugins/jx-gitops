FROM gcr.io/jenkinsxio/jx-cli-base:0.0.10

ENTRYPOINT ["jx-gitops"]

COPY ./build/linux/jx-gitops /usr/bin/jx-gitops