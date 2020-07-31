FROM gcr.io/jenkinsxio-labs-private/jxl-base:0.0.55

ENTRYPOINT ["jx-gitops"]

COPY ./build/linux/jx-gitops /usr/bin/jx-gitops