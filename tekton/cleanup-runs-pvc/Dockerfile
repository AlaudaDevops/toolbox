# use curl to download cli
ARG BUILDER_IMAGE=build-harbor.alauda.cn/ops/photon:5-alauda-202511251100
ARG RUNTIME_IMAGE=build-harbor.alauda.cn/ops/alpine:3.21.5-alauda-202512020944
FROM ${BUILDER_IMAGE} AS download

# renovate: datasource=github-releases depName=kubectl packageName=AlaudaDevops/kubernetes
ARG KUBECTL_VERSION=v1.33.5-alauda-6

RUN set -eux; \
    if [ "$(uname -m)" = "arm64" ] || [ "$(uname -m)" = "aarch64" ]; then \
      export ARCH="arm64"; \
    else \
      export ARCH="x86_64"; \
    fi; \
    curl --retry 6 --create-dirs -fLo /tmp/download.tar.gz https://github.com/AlaudaDevops/kubernetes/releases/download/${KUBECTL_VERSION}/kubectl_Linux_${ARCH}.tar.gz && \
    tar -xzf /tmp/download.tar.gz -C /tmp && \
    mv /tmp/alauda-kubectl /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl

# check binary executable
RUN kubectl version --client

# 使用轻量级的Alpine Linux作为基础镜像
FROM ${RUNTIME_IMAGE}

# 安装必要的工具
RUN apk add --no-cache \
    bash \
    curl \
    jq \
	coreutils \
    ca-certificates \
    && rm -rf /var/cache/apk/*

COPY --from=download /usr/local/bin/kubectl /usr/local/bin/kubectl

# 创建应用目录
WORKDIR /app

# 复制脚本文件
COPY --chown=65532:65532 ./ /app/

RUN chmod a+x /app/cleanup-tekton-pvcs.sh

# 创建审计目录
RUN mkdir -p /app/audit

RUN deluser --remove-home nonroot ; \
    adduser -u 65532 -h /home/nonroot -D nonroot && \
    chmod a+rx /docker-entrypoint.sh

# 设置脚本可执行权限

USER 65532

# 默认命令
ENTRYPOINT ["/bin/bash", "/app/cleanup-tekton-pvcs.sh"]
CMD ["--help"]
