#!/bin/bash
set -e

# Debug log file for entrypoint execution
DEBUG_LOG_FILE="/tmp/addt-entrypoint-debug.log"

# Debug logging function
# Write to both stderr (for docker logs) and debug log file (for docker cp)
debug_log() {
    if [ "${ADDT_LOG_LEVEL:-INFO}" = "DEBUG" ]; then
        timestamp=$(date '+%Y-%m-%d %H:%M:%S')
        log_msg="[${timestamp}] [DEBUG] $*"
        echo "$log_msg" >&2
        # Also write to debug log file
        echo "$log_msg" >> "$DEBUG_LOG_FILE" 2>/dev/null || true
    fi
}

# Initialize debug log file
if [ "${ADDT_LOG_LEVEL:-INFO}" = "DEBUG" ]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Entrypoint script started" > "$DEBUG_LOG_FILE"
fi

debug_log "Entrypoint script started (uid=$(id -u))"
debug_log "ADDT_LOG_LEVEL=${ADDT_LOG_LEVEL:-INFO}"
debug_log "ADDT_COMMAND=${ADDT_COMMAND:-not set}"
debug_log "Arguments: $*"

# --- Root phase: do privileged ops then re-exec as addt ---
if [ "$(id -u)" = "0" ]; then
    debug_log "Running as root, performing privileged operations"

    # Initialize firewall if enabled
    if [ "${ADDT_FIREWALL_ENABLED}" = "true" ] && [ -f /usr/local/bin/init-firewall.sh ]; then
        debug_log "Firewall enabled, initializing (as root)"
        /usr/local/bin/init-firewall.sh
    fi

    # Start Docker daemon if in DinD mode
    if [ "$ADDT_DOCKER_DIND_ENABLE" = "true" ]; then
        debug_log "DinD mode enabled, starting Docker daemon (as root)"
        echo "Starting Docker daemon in isolated mode..."

        dockerd --host=unix:///var/run/docker.sock >/tmp/docker.log 2>&1 &

        echo "Waiting for Docker daemon..."
        for i in $(seq 1 30); do
            if [ -S /var/run/docker.sock ]; then
                chmod 666 /var/run/docker.sock
                if docker info >/dev/null 2>&1; then
                    echo "Docker daemon ready (isolated environment)"
                    break
                fi
            fi
            if [ "$i" -eq 30 ]; then
                echo "Error: Docker daemon failed to start"
                cat /tmp/docker.log 2>/dev/null || echo "No log file available"
                exit 1
            fi
            sleep 1
        done
    fi

    # Fix secrets ownership so addt user can read/delete them
    # Use numeric IDs to avoid group name resolution issues (on macOS, host GID
    # may conflict with an existing Debian group, so 'addt' group may not exist)
    if [ -f /run/secrets/.secrets ]; then
        chown "$(id -u addt):$(id -g addt)" /run/secrets/.secrets
        debug_log "Fixed secrets file ownership for addt user"
    fi

    # Re-exec this script as addt user
    debug_log "Dropping privileges: exec gosu addt $0 $*"
    exec gosu addt "$0" "$@"
fi

# --- Normal phase: running as addt user ---
debug_log "Running as addt user"

# Copy host .gitconfig to writable location (bind-mounted single files can't be
# atomically replaced by git, causing "Device or resource busy" errors)
if [ -f "$HOME/.gitconfig.host" ]; then
    cp "$HOME/.gitconfig.host" "$HOME/.gitconfig"
    debug_log "Copied .gitconfig.host to .gitconfig"
fi

# Set up SSH agent proxy via TCP (macOS + podman: Unix sockets can't be mounted)
# The host runs an SSH proxy on TCP; socat bridges it to a local Unix socket.
if [ -n "$ADDT_SSH_PROXY_HOST" ] && [ -n "$ADDT_SSH_PROXY_PORT" ]; then
    debug_log "Setting up SSH agent TCP bridge to $ADDT_SSH_PROXY_HOST:$ADDT_SSH_PROXY_PORT"
    SSH_SOCK_DIR=$(mktemp -d /tmp/ssh-sock-XXXXXX)
    SSH_SOCK_LOCAL="$SSH_SOCK_DIR/agent.sock"
    if command -v socat >/dev/null 2>&1; then
        setsid socat UNIX-LISTEN:"$SSH_SOCK_LOCAL",fork,mode=600 \
              TCP:"$ADDT_SSH_PROXY_HOST":"$ADDT_SSH_PROXY_PORT" &
        export SSH_AUTH_SOCK="$SSH_SOCK_LOCAL"
        debug_log "SSH agent bridge started at $SSH_AUTH_SOCK"
    else
        echo "Warning: socat not found, SSH agent forwarding unavailable"
    fi
fi

# Set up GPG agent proxy via TCP (macOS + podman: Unix sockets can't be mounted)
if [ -n "$ADDT_GPG_PROXY_HOST" ] && [ -n "$ADDT_GPG_PROXY_PORT" ]; then
    debug_log "Setting up GPG agent TCP bridge to $ADDT_GPG_PROXY_HOST:$ADDT_GPG_PROXY_PORT"
    GPG_SOCK="$HOME/.gnupg/S.gpg-agent"
    GPG_SOCK_DIR=$(mktemp -d /tmp/gpg-sock-XXXXXX)
    GPG_SOCK_LOCAL="$GPG_SOCK_DIR/S.gpg-agent"
    if command -v socat >/dev/null 2>&1; then
        # Create socket on local filesystem to avoid virtiofs chmod issues,
        # then symlink from the expected location
        rm -f "$GPG_SOCK"
        setsid socat UNIX-LISTEN:"$GPG_SOCK_LOCAL",fork,mode=600 \
              TCP:"$ADDT_GPG_PROXY_HOST":"$ADDT_GPG_PROXY_PORT" &
        ln -sf "$GPG_SOCK_LOCAL" "$GPG_SOCK"
        debug_log "GPG agent bridge started at $GPG_SOCK -> $GPG_SOCK_LOCAL"
    else
        echo "Warning: socat not found, GPG agent forwarding unavailable"
    fi
fi

# Set up tmux proxy via TCP (macOS + podman: Unix sockets can't be mounted)
if [ -n "$ADDT_TMUX_PROXY_HOST" ] && [ -n "$ADDT_TMUX_PROXY_PORT" ]; then
    debug_log "Setting up tmux TCP bridge to $ADDT_TMUX_PROXY_HOST:$ADDT_TMUX_PROXY_PORT"
    TMUX_SOCK_DIR=$(mktemp -d /tmp/tmux-sock-XXXXXX)
    TMUX_SOCK="$TMUX_SOCK_DIR/default"
    if command -v socat >/dev/null 2>&1; then
        setsid socat UNIX-LISTEN:"$TMUX_SOCK",fork,mode=600 \
              TCP:"$ADDT_TMUX_PROXY_HOST":"$ADDT_TMUX_PROXY_PORT" &
        # Reconstruct TMUX env: socket_path,pid,window
        if [ -n "$ADDT_TMUX_PARTS" ]; then
            export TMUX="$TMUX_SOCK,$ADDT_TMUX_PARTS"
        else
            export TMUX="$TMUX_SOCK"
        fi
        debug_log "Tmux bridge started: TMUX=$TMUX"
    else
        echo "Warning: socat not found, tmux forwarding unavailable"
    fi
fi

# Load secrets from file if present (copied via docker cp to tmpfs)
# Secrets are written to tmpfs at /run/secrets/.secrets by the host
# This approach keeps secrets out of environment variables entirely
if [ -f /run/secrets/.secrets ]; then
    debug_log "Loading secrets from /run/secrets/.secrets"
    # Parse JSON and export directly to environment
    eval "$(node -e '
        const fs = require("fs");
        const data = fs.readFileSync("/run/secrets/.secrets", "utf8");
        const secrets = JSON.parse(data);
        for (const [key, value] of Object.entries(secrets)) {
            // Escape single quotes in value for shell safety
            const escaped = value.replace(/'"'"'/g, "'"'"'\\'"'"''"'"'");
            console.log(`export ${key}='"'"'${escaped}'"'"'`);
        }
    ')"

    # Overwrite secrets file with random data before deleting to prevent recovery
    if [ -f /run/secrets/.secrets ]; then
        filesize=$(stat -c %s /run/secrets/.secrets 2>/dev/null || stat -f %z /run/secrets/.secrets 2>/dev/null || echo 256)
        dd if=/dev/urandom of=/run/secrets/.secrets bs="$filesize" count=1 conv=notrunc 2>/dev/null
        sync
    fi
    rm -f /run/secrets/.secrets
    debug_log "Secrets loaded, scrubbed, and file removed"
fi

# Note: DinD and firewall initialization are handled in the root phase above.
# When the container starts as root, those ops run before dropping to addt.

# Run extension setup scripts (if not already run in this session)
EXTENSIONS_DIR="/usr/local/share/addt/extensions"
EXTENSIONS_JSON="$HOME/.addt/extensions.json"
SETUP_MARKER="$HOME/.addt/.setup-done"

if [ -f "$EXTENSIONS_JSON" ] && [ ! -f "$SETUP_MARKER" ]; then
    debug_log "Running extension setup scripts"
    # Extract extension names from the top-level "extensions" object in JSON
    extensions=$(node -e "const d=JSON.parse(require('fs').readFileSync('$EXTENSIONS_JSON','utf8'));Object.keys(d.extensions||{}).forEach(e=>console.log(e))" 2>/dev/null)

    for ext in $extensions; do
        # Convert extension name to uppercase env var prefix (e.g., claude -> CLAUDE)
        ext_upper=$(echo "$ext" | tr '[:lower:]-' '[:upper:]_')

        # Check for config override env vars first (set by host), fall back to extensions.json
        override_trust_var="ADDT_${ext_upper}_WORKDIR_AUTOTRUST"
        override_login_var="ADDT_${ext_upper}_AUTH_AUTOLOGIN"
        override_method_var="ADDT_${ext_upper}_AUTH_METHOD"

        # workdir.autotrust: per-extension override > global > extension default
        if [ -n "${!override_trust_var}" ]; then
            autotrust="${!override_trust_var}"
        elif [ -n "$ADDT_WORKDIR_AUTOTRUST" ]; then
            autotrust="$ADDT_WORKDIR_AUTOTRUST"
        else
            autotrust="false"
        fi

        # auth.autologin: per-extension override > global > extension default
        if [ -n "${!override_login_var}" ]; then
            autologin="${!override_login_var}"
        elif [ -n "$ADDT_AUTH_AUTOLOGIN" ]; then
            autologin="$ADDT_AUTH_AUTOLOGIN"
        else
            autologin=$(node -e "const d=JSON.parse(require('fs').readFileSync('$EXTENSIONS_JSON','utf8'));console.log(d.extensions['$ext']?.auth?.autologin||false)" 2>/dev/null || echo "false")
        fi

        # auth.method: per-extension override > global > extension default
        if [ -n "${!override_method_var}" ]; then
            auth_method="${!override_method_var}"
        elif [ -n "$ADDT_AUTH_METHOD" ]; then
            auth_method="$ADDT_AUTH_METHOD"
        else
            auth_method=$(node -e "const d=JSON.parse(require('fs').readFileSync('$EXTENSIONS_JSON','utf8'));console.log(d.extensions['$ext']?.auth?.method||'auto')" 2>/dev/null || echo "auto")
        fi

        export ADDT_EXT_WORKDIR_AUTOTRUST="$autotrust"
        export ADDT_EXT_AUTH_AUTOLOGIN="$autologin"
        export ADDT_EXT_AUTH_METHOD="$auth_method"
        debug_log "Extension $ext: autotrust=$autotrust, autologin=$autologin, auth_method=$auth_method"

        setup_script="$EXTENSIONS_DIR/$ext/setup.sh"
        if [ -f "$setup_script" ]; then
            debug_log "Running setup for extension: $ext"
            echo "Running setup for extension: $ext"
            bash "$setup_script" || echo "Warning: setup.sh for $ext failed"
        fi
    done

    # Mark setup as done for this session
    touch "$SETUP_MARKER"
    debug_log "Extension setup complete"
fi

# Clear credential env vars after setup so they don't leak into shell sessions
# Overwrite with random data before unsetting to prevent recovery
# from /proc/*/environ or memory dumps
# Inspired by: https://github.com/IngmarKrusch/claude-docker
if [ -n "$ADDT_CREDENTIAL_VARS" ]; then
    IFS=',' read -ra CRED_VARS <<< "$ADDT_CREDENTIAL_VARS"
    for var in "${CRED_VARS[@]}"; do
        if [ -n "${!var+x}" ]; then
            eval "val_len=\${#$var}"
            if [ "$val_len" -gt 0 ]; then
                random_data=$(head -c "$val_len" /dev/urandom | base64 | head -c "$val_len")
                export "$var=$random_data"
            fi
        fi
        unset "$var" 2>/dev/null || true
    done
    export ADDT_CREDENTIAL_VARS="$(head -c ${#ADDT_CREDENTIAL_VARS} /dev/urandom | base64 | head -c ${#ADDT_CREDENTIAL_VARS})"
    unset ADDT_CREDENTIAL_VARS
fi

# Scope GH_TOKEN to allowed repos via git credential-cache
# Inspired by: https://github.com/IngmarKrusch/claude-docker
if [ "$ADDT_GITHUB_SCOPE_TOKEN" = "true" ] && [ -n "$GH_TOKEN" ]; then
    debug_log "Scoping GH_TOKEN to allowed repos via git credential-cache"

    # Configure git credential-cache with useHttpPath (scopes by repo path)
    git config --global credential.helper 'cache --timeout=86400'
    git config --global credential.useHttpPath true

    # Helper: cache a credential for a given owner/repo on github.com
    cache_repo_credential() {
        local repo_path="$1"
        printf 'protocol=https\nhost=github.com\npath=%s\nusername=x-access-token\npassword=%s\n\n' \
            "$repo_path" "$GH_TOKEN" | git credential-cache store
        debug_log "Cached credential for github.com/$repo_path"
    }

    # 1. Auto-detect and cache workspace repo (default scope)
    if [ -d /workspace/.git ] || git -C /workspace rev-parse --git-dir >/dev/null 2>&1; then
        REPO_URL=$(git -C /workspace remote get-url origin 2>/dev/null || echo "")
        if [ -n "$REPO_URL" ]; then
            REPO_PATH=""
            case "$REPO_URL" in
                https://github.com/*)
                    REPO_PATH=$(echo "$REPO_URL" | sed 's|https://github.com/||' | sed 's/\.git$//')
                    ;;
                git@github.com:*)
                    REPO_PATH=$(echo "$REPO_URL" | sed 's|git@github.com:||' | sed 's/\.git$//')
                    ;;
            esac
            if [ -n "$REPO_PATH" ]; then
                cache_repo_credential "$REPO_PATH"
            fi
        fi
    fi

    # 2. Cache additional repos from ADDT_GITHUB_SCOPE_REPOS (comma-separated owner/repo)
    if [ -n "$ADDT_GITHUB_SCOPE_REPOS" ]; then
        IFS=',' read -ra EXTRA_REPOS <<< "$ADDT_GITHUB_SCOPE_REPOS"
        for repo in "${EXTRA_REPOS[@]}"; do
            repo=$(echo "$repo" | xargs)  # trim whitespace
            if [ -n "$repo" ]; then
                cache_repo_credential "$repo"
            fi
        done
    fi

    # 3. Store token in gh CLI config (so gh pr/issue/api still work)
    echo "$GH_TOKEN" | gh auth login --with-token 2>/dev/null || true

    # 4. Scrub GH_TOKEN from environment (overwrite with random data then unset)
    token_len=${#GH_TOKEN}
    if [ "$token_len" -gt 0 ]; then
        random_data=$(head -c "$token_len" /dev/urandom | base64 | head -c "$token_len")
        export GH_TOKEN="$random_data"
    fi
    unset GH_TOKEN 2>/dev/null || true

    # Scrub control vars
    unset ADDT_GITHUB_SCOPE_TOKEN 2>/dev/null || true
    unset ADDT_GITHUB_SCOPE_REPOS 2>/dev/null || true
    debug_log "GH_TOKEN scoped and scrubbed"
fi

# Build system prompt for port mappings (exported for args.sh to use)
export ADDT_SYSTEM_PROMPT=""

if [ -n "$ADDT_PORT_MAP" ]; then
    # Parse port mappings (format: "3000:30000,8080:30001")
    ADDT_SYSTEM_PROMPT="# Port Mapping Information

When you start a service inside this container on certain ports, tell the user the correct HOST port to access it from their browser.

Port mappings (container→host):
"
    IFS=',' read -ra MAPPINGS <<< "$ADDT_PORT_MAP"
    for mapping in "${MAPPINGS[@]}"; do
        IFS=':' read -ra PORTS <<< "$mapping"
        CONTAINER_PORT="${PORTS[0]}"
        HOST_PORT="${PORTS[1]}"
        ADDT_SYSTEM_PROMPT+="- Container port $CONTAINER_PORT → Host port $HOST_PORT (user accesses: http://localhost:$HOST_PORT)
"
    done

    ADDT_SYSTEM_PROMPT+="
IMPORTANT:
- When testing/starting services inside the container, use the container ports (e.g., http://localhost:3000)
- When telling the USER where to access services in their browser, use the HOST ports (e.g., http://localhost:30000)
- Always remind the user to use the host port in their browser"
fi

# Set npm global prefix to user-owned directory (so addt user can install/uninstall without sudo)
export NPM_CONFIG_PREFIX="$HOME/.npm-global"
mkdir -p "$NPM_CONFIG_PREFIX"
# Set npm cache inside npm-global so it works with readonly rootfs (tmpfs on /home/addt)
export NPM_CONFIG_CACHE="$NPM_CONFIG_PREFIX/.cache"

# Ensure ~/.local/bin, npm-global/bin and ~/go/bin are in PATH
export PATH="$HOME/.local/bin:$NPM_CONFIG_PREFIX/bin:$HOME/go/bin:$PATH"

# Neutralize git hooks if enabled (prevents malicious .git/hooks/* execution)
# Creates a wrapper that forces core.hooksPath=/dev/null via GIT_CONFIG_COUNT
# Inspired by: https://github.com/IngmarKrusch/claude-docker
if [ "$ADDT_GIT_DISABLE_HOOKS" = "true" ]; then
    debug_log "Git hooks neutralization enabled, creating wrapper"
    REAL_GIT=$(command -v git 2>/dev/null || echo "/usr/bin/git")
    mkdir -p "$HOME/.local/bin"
    cat > "$HOME/.local/bin/git" <<WRAPPER
#!/bin/sh
export GIT_CONFIG_COUNT=\${GIT_CONFIG_COUNT:-0}
n=\$GIT_CONFIG_COUNT
export GIT_CONFIG_KEY_\$n=core.hooksPath
export GIT_CONFIG_VALUE_\$n=/dev/null
export GIT_CONFIG_COUNT=\$((n + 1))
exec "$REAL_GIT" "\$@"
WRAPPER
    chmod +x "$HOME/.local/bin/git"
    debug_log "Git wrapper created at $HOME/.local/bin/git (real git: $REAL_GIT)"
    unset ADDT_GIT_DISABLE_HOOKS
fi

# Determine which command to run (entrypoint can be array: ["bash", "-i"])
ADDT_CMD=""
ADDT_CMD_ARGS=()

if [ -n "$ADDT_COMMAND" ]; then
    ADDT_CMD="$ADDT_COMMAND"
    debug_log "Using ADDT_COMMAND: $ADDT_CMD"
elif [ -f "$EXTENSIONS_JSON" ]; then
    debug_log "Auto-detecting command from extensions.json"
    # Auto-detect from first installed extension
    # Entrypoint is now a JSON array, e.g., ["bash","-i"] or ["claude"]
    entrypoint_json=$(grep -oE '"entrypoint":[[:space:]]*\[[^]]*\]' "$EXTENSIONS_JSON" | head -1 | sed 's/.*"entrypoint":[[:space:]]*//')

    if [ -n "$entrypoint_json" ]; then
        # Parse JSON array: ["cmd", "arg1", "arg2"] -> cmd and args
        # Remove brackets and quotes, split by comma
        entrypoint_clean=$(echo "$entrypoint_json" | tr -d '[]"' | sed 's/,/ /g')
        read -ra entrypoint_parts <<< "$entrypoint_clean"

        if [ ${#entrypoint_parts[@]} -gt 0 ]; then
            ADDT_CMD="${entrypoint_parts[0]}"
            ADDT_CMD_ARGS=("${entrypoint_parts[@]:1}")
        fi
    fi
fi

# Fallback to claude if still not set
ADDT_CMD="${ADDT_CMD:-claude}"
debug_log "Final command: $ADDT_CMD"
debug_log "Command args: ${ADDT_CMD_ARGS[*]}"

# Find the extension directory for args.sh
EXTENSIONS_DIR="/usr/local/share/addt/extensions"
ARGS_SCRIPT=""

# Look for args.sh in the extension matching the command
for ext_dir in "$EXTENSIONS_DIR"/*/; do
    if [ -f "$ext_dir/config.yaml" ]; then
        # Get entrypoint command (first element if array)
        ep_line=$(grep "^entrypoint:" "$ext_dir/config.yaml" 2>/dev/null)
        if [[ "$ep_line" =~ \[ ]]; then
            # Array format - extract first element
            entrypoint=$(echo "$ep_line" | sed 's/^entrypoint:[[:space:]]*//' | tr -d '[]"' | cut -d',' -f1 | xargs)
        else
            # String format
            entrypoint=$(echo "$ep_line" | sed 's/^entrypoint:[[:space:]]*//' | tr -d '"')
        fi

        if [ "$entrypoint" = "$ADDT_CMD" ] && [ -f "$ext_dir/args.sh" ]; then
            ARGS_SCRIPT="$ext_dir/args.sh"
            break
        fi
    fi
done

# Transform args through extension's args.sh if it exists
if [ -n "$ARGS_SCRIPT" ] && [ -f "$ARGS_SCRIPT" ]; then
    debug_log "Using args.sh: $ARGS_SCRIPT"
    debug_log "Original args: $*"
    # Run args.sh and read transformed args (null-delimited to handle multi-line values)
    # Use timeout to prevent hangs (5 seconds should be plenty for arg transformation)
    if command -v timeout >/dev/null 2>&1; then
        mapfile -t -d '' TRANSFORMED_ARGS < <(timeout 5 bash "$ARGS_SCRIPT" "$@")
    else
        mapfile -t -d '' TRANSFORMED_ARGS < <(bash "$ARGS_SCRIPT" "$@")
    fi
    FINAL_ARGS=("${ADDT_CMD_ARGS[@]}" "${TRANSFORMED_ARGS[@]}")
    debug_log "Transformed args: ${FINAL_ARGS[*]}"
else
    # No args.sh - pass args directly
    debug_log "No args.sh found, passing args directly"
    FINAL_ARGS=("${ADDT_CMD_ARGS[@]}" "$@")
fi

# Execute with optional time limit
debug_log "Executing: $ADDT_CMD ${FINAL_ARGS[*]}"
if [ -n "$ADDT_TIME_LIMIT_SECONDS" ] && [ "$ADDT_TIME_LIMIT_SECONDS" -gt 0 ]; then
    echo "Time limit: $((ADDT_TIME_LIMIT_SECONDS / 60)) minutes"
    exec timeout --signal=TERM "$ADDT_TIME_LIMIT_SECONDS" "$ADDT_CMD" "${FINAL_ARGS[@]}"
else
    exec "$ADDT_CMD" "${FINAL_ARGS[@]}"
fi
