ARG BUILDER_IMAGE=build-harbor.alauda.cn/devops/nonroot/builder-go:latest@sha256:116009fc6f39cfa317c8cbd4b3c88e0800fe18f4f789a9c774011fb1c4c69a9a
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
