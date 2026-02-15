ARG BUILDER_IMAGE=build-harbor.alauda.cn/devops/nonroot/builder-go:latest@sha256:8b98b5818c1c99a19351ac147a487ae54d5a299c27bfab5f6fccc3b1a4dbcd44
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
