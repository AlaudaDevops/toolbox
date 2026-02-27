ARG BUILDER_IMAGE=build-harbor.alauda.cn/devops/nonroot/builder-go:latest@sha256:8401fd9dd2d33d623b80ee6408db5bd35d538281f452ec8fd7f07be7177703db
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
