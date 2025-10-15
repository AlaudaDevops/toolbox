ARG BUILDER_IMAGE=build-harbor.alauda.cn/devops/nonroot/builder-go:latest@sha256:925a02b543a4315a083c5de502cee1f47ca65f677ff4385da13fdfe5de79818e
ARG BASE_IMAGE=build-harbor.alauda.cn/ops/alpine:3.21.3-alauda-202510091100

FROM ${BUILDER_IMAGE} AS builder

USER root

WORKDIR /app

COPY . /app

RUN cd /app &&  go build -o artifact-scanner .

FROM ${BASE_IMAGE}

WORKDIR /app

COPY --from=builder /app/artifact-scanner /app/artifact-scanner

ENTRYPOINT ["artifact-scanner"]
