ARG BUILDER_IMAGE=build-harbor.alauda.cn/devops/nonroot/builder-go:latest@sha256:4f740d7c64d43e955f2ab8d02b462ec940e3ba88d9134514f4318c3a82c300e7
ARG BASE_IMAGE=build-harbor.alauda.cn/ops/alpine:3.23.3-alauda-202605061100

FROM ${BUILDER_IMAGE} AS builder

USER root

WORKDIR /app

COPY . /app

RUN cd /app &&  go build -o artifact-scanner .

FROM ${BASE_IMAGE}

WORKDIR /app

COPY --from=builder /app/artifact-scanner /app/artifact-scanner

ENTRYPOINT ["artifact-scanner"]
