FROM alpine:3.22

# Install dependencies
RUN apk --no-cache add \
    ca-certificates \
    curl \
    git \
    bash

# Install GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null

# For Alpine, we need to use a different approach for GitHub CLI
RUN apk add --no-cache github-cli

# Install GitLab CLI
RUN curl -s https://gitlab.com/gitlab-org/cli/-/releases/permalink/latest/downloads/glab_linux_amd64.tar.gz | tar -xz -C /tmp \
    && mv /tmp/bin/glab /usr/local/bin/glab \
    && chmod +x /usr/local/bin/glab

# Create non-root user with UID/GID 65532
RUN addgroup -g 65532 nonroot && \
    adduser -D -u 65532 -G nonroot nonroot

# Set working directory
WORKDIR /app

# Change ownership of the working directory
RUN chown -R nonroot:nonroot /app

# Switch to non-root user
USER 65532:65532

# Default command
CMD ["/bin/sh"]
