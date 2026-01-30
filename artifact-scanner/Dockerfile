ARG BUILDER_IMAGE=build-harbor.alauda.cn/devops/nonroot/builder-go:latest@sha256:157fa53c5c70bd6fcf7e2f29e539e494f65af4d719570c667c993b64e33b4c1b
ARG BASE_IMAGE=build-harbor.alauda.cn/ops/alpine:3.23.3-alauda-202601301100

FROM ${BUILDER_IMAGE} AS builder

USER root

WORKDIR /app

COPY . /app

RUN cd /app &&  go build -o artifact-scanner .

FROM ${BASE_IMAGE}

WORKDIR /app

COPY --from=builder /app/artifact-scanner /app/artifact-scanner

ENTRYPOINT ["artifact-scanner"]
