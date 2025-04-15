FROM docker-mirrors.alauda.cn/library/golang:1.24-alpine AS builder

WORKDIR /app

COPY . /app

RUN cd /app &&  go build -o artifact-scanner .

FROM registry.alauda.cn:60070/ops/alpine:3.20

WORKDIR /app

COPY --from=builder /app/artifact-scanner /app/artifact-scanner

ENTRYPOINT ["artifact-scanner"]
