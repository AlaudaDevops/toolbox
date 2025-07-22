ARG BUILDER_IMAGE=golang:1.24-alpine
ARG BASE_IMAGE=alpine:3.22

FROM ${BUILDER_IMAGE} AS builder

WORKDIR /app

COPY . /app

RUN cd /app &&  go build -o artifact-scanner .

FROM ${BASE_IMAGE}

WORKDIR /app

COPY --from=builder /app/artifact-scanner /app/artifact-scanner

ENTRYPOINT ["artifact-scanner"]
