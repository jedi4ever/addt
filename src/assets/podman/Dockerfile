# Extension image - builds FROM base image for fast caching
ARG BASE_IMAGE=addt-base:latest
FROM ${BASE_IMAGE}

# Build arguments for extensions
ARG ADDT_EXTENSIONS=claude

# Extension version overrides (format: "claude:1.0.5,codex:0.2.0")
# Default versions are defined in each extension's config.yaml
ARG EXTENSION_VERSIONS=""

USER root

# Copy install script and extensions
COPY install.sh /usr/local/share/addt/install.sh
COPY extensions/ /usr/local/share/addt/extensions/
RUN chmod +x /usr/local/share/addt/install.sh && \
    find /usr/local/share/addt/extensions -name "*.sh" -exec chmod +x {} \;

# Install requested extensions (as addt user for go installs)
# EXTENSION_VERSIONS overrides default versions from config.yaml
USER addt
RUN EXTENSION_VERSIONS="${EXTENSION_VERSIONS}" \
    /usr/local/share/addt/install.sh "${ADDT_EXTENSIONS}"

# Add version labels for tracking
LABEL org.opencontainers.image.title="addt"
LABEL org.opencontainers.image.description="Containerized environment for AI coding agents (Podman)"
LABEL org.opencontainers.image.authors="https://github.com/anthropics/claude-code"
