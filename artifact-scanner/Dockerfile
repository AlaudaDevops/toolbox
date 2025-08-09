ARG BUILDER_IMAGE=build-harbor.alauda.cn/devops/nonroot/builder-go:latest@sha256:9e65584ae990290fd3ec70b4a8c28c949ec8bb7d909089401aac53db292c7edf
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
