ARG NODE_VERSION=20
FROM node:${NODE_VERSION}-slim

# Build arguments for user ID and group ID
ARG USER_ID=1000
ARG GROUP_ID=1000
ARG USERNAME=claude
ARG CLAUDE_VERSION=latest
ARG GO_VERSION=1.23

# Install dependencies, GitHub CLI, Docker CLI, and Docker daemon (for DinD)
RUN apt-get update && apt-get install -y \
    curl \
    gnupg \
    git \
    sudo \
    ripgrep \
    ca-certificates \
    iptables \
    supervisor \
    && curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian bookworm stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update \
    && apt-get install -y gh docker-ce-cli docker-ce containerd.io \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Go
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "amd64" ]; then GO_ARCH="amd64"; \
    elif [ "$ARCH" = "arm64" ]; then GO_ARCH="arm64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -o /tmp/go.tar.gz && \
    tar -C /usr/local -xzf /tmp/go.tar.gz && \
    rm /tmp/go.tar.gz

# Add Go to PATH
ENV PATH="/usr/local/go/bin:${PATH}"

# Install Claude Code globally (specific version or latest)
RUN if [ "$CLAUDE_VERSION" = "latest" ]; then \
        npm install -g @anthropic-ai/claude-code; \
    else \
        npm install -g @anthropic-ai/claude-code@$CLAUDE_VERSION; \
    fi

# Create user with matching UID/GID from host
# Note: GID 20 may already exist (staff/dialout group), so we just add user to that group
RUN (groupadd -g ${GROUP_ID} ${USERNAME} 2>/dev/null || true) \
    && useradd -m -u ${USER_ID} -g ${GROUP_ID} -s /bin/bash ${USERNAME} 2>/dev/null || usermod -u ${USER_ID} ${USERNAME} \
    && echo "${USERNAME} ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Set working directory
WORKDIR /workspace

# Change ownership of workspace (use numeric IDs to avoid group name issues)
RUN chown -R ${USER_ID}:${GROUP_ID} /workspace

# Copy entrypoint wrapper for DinD support
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Switch to non-root user
USER ${USERNAME}

# Add version labels for tracking
LABEL org.opencontainers.image.title="dclaude"
LABEL org.opencontainers.image.description="Claude Code with Git, GitHub CLI, and Ripgrep"
LABEL org.opencontainers.image.authors="https://github.com/anthropics/claude-code"

# Tool version labels (populated at build time)
LABEL tools.claude="installed"
LABEL tools.git="installed"
LABEL tools.gh="installed"
LABEL tools.ripgrep="installed"
LABEL tools.node="installed"
LABEL tools.go="installed"

# Entry point will be our wrapper script (handles DinD mode)
# Empty CMD means interactive session by default
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD []
