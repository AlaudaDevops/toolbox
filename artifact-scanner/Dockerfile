ARG BUILDER_IMAGE=build-harbor.alauda.cn/devops/nonroot/builder-go:latest@sha256:13bbea24d40934f8f22f9573b04dc296ce3349645669693f7b3ac655587c3429
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
