ARG BUILDER_IMAGE=build-harbor.alauda.cn/devops/nonroot/builder-go:latest@sha256:b29fb8870a2aca59959c7222f2ea2ca46445cd0a3cf1cd783f138500dfb1dc71
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
