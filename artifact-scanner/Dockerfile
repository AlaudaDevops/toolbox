ARG BUILDER_IMAGE=build-harbor.alauda.cn/devops/nonroot/builder-go:latest@sha256:6410143563e944c5859385d175ef3ae338d7f38987b0fbc7f832f2d2cae61d38
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
