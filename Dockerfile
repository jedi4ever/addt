FROM node:20-slim

# Build arguments for user ID and group ID
ARG USER_ID=1000
ARG GROUP_ID=1000
ARG USERNAME=claude

# Install dependencies and GitHub CLI
RUN apt-get update && apt-get install -y \
    curl \
    gnupg \
    git \
    sudo \
    ripgrep \
    && curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update \
    && apt-get install -y gh \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Claude Code globally
RUN npm install -g @anthropic-ai/claude-code

# Create user with matching UID/GID from host
# Note: GID 20 may already exist (staff/dialout group), so we just add user to that group
RUN (groupadd -g ${GROUP_ID} ${USERNAME} 2>/dev/null || true) \
    && useradd -m -u ${USER_ID} -g ${GROUP_ID} -s /bin/bash ${USERNAME} 2>/dev/null || usermod -u ${USER_ID} ${USERNAME} \
    && echo "${USERNAME} ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Set working directory
WORKDIR /workspace

# Change ownership of workspace (use numeric IDs to avoid group name issues)
RUN chown -R ${USER_ID}:${GROUP_ID} /workspace

# Switch to non-root user
USER ${USERNAME}

# Entry point will be claude command
# Empty CMD means interactive session by default
ENTRYPOINT ["claude"]
CMD []
