ARG BUILDER_IMAGE=build-harbor.alauda.cn/devops/nonroot/builder-go:latest@sha256:4938e20f09631bdde309facc5cae264766d3974311bc5b76ab589f8df7ed6b80
ARG BASE_IMAGE=build-harbor.alauda.cn/ops/alpine:3.21.3-alauda-202502271510

FROM ${BUILDER_IMAGE} AS builder

USER root

WORKDIR /app

COPY . /app

RUN cd /app &&  go build -o artifact-scanner .

FROM ${BASE_IMAGE}

WORKDIR /app

COPY --from=builder /app/artifact-scanner /app/artifact-scanner

ENTRYPOINT ["artifact-scanner"]
