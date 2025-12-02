ARG BUILDER_IMAGE=build-harbor.alauda.cn/devops/nonroot/builder-go:latest@sha256:121028f46dc9c51ad1ade23e72a28ce82f15c1c2a75eeb3be7aaf2772b455203
ARG BASE_IMAGE=build-harbor.alauda.cn/ops/alpine:3.21.5-alauda-202512020944

FROM ${BUILDER_IMAGE} AS builder

USER root

WORKDIR /app

COPY . /app

RUN cd /app &&  go build -o artifact-scanner .

FROM ${BASE_IMAGE}

WORKDIR /app

COPY --from=builder /app/artifact-scanner /app/artifact-scanner

ENTRYPOINT ["artifact-scanner"]
