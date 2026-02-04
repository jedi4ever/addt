#!/bin/bash
# Extension installer for addt
# Usage: install.sh [extension1,extension2,...]
#
# Extensions are directories containing:
#   - config.yaml  - metadata (name, description, entrypoint, dependencies, default_version)
#   - install.sh   - install script
#
# Environment variables:
#   EXTENSION_VERSIONS - Override versions (format: "claude:1.0.5,codex:0.2.0")
#   Default versions come from each extension's config.yaml default_version field

set -e

EXTENSIONS_DIR="${EXTENSIONS_DIR:-/usr/local/share/addt/extensions}"
METADATA_FILE="${METADATA_FILE:-/home/addt/.addt/extensions.json}"
EXTENSIONS="${1:-$ADDT_EXTENSIONS}"

# Parse EXTENSION_VERSIONS into associative array
declare -A VERSION_OVERRIDES
if [ -n "$EXTENSION_VERSIONS" ]; then
    IFS=',' read -ra VERSION_PAIRS <<< "$EXTENSION_VERSIONS"
    for pair in "${VERSION_PAIRS[@]}"; do
        ext_name="${pair%%:*}"
        ext_version="${pair#*:}"
        VERSION_OVERRIDES["$ext_name"]="$ext_version"
    done
fi

# Ensure metadata directory exists
mkdir -p "$(dirname "$METADATA_FILE")"

if [ -z "$EXTENSIONS" ] || [ "$EXTENSIONS" = "none" ]; then
    echo "Extensions: No extensions requested"
    echo '{"extensions":{}}' > "$METADATA_FILE"
    exit 0
fi

# Simple YAML parser - extract value for a key
yaml_get() {
    local file="$1"
    local key="$2"
    grep "^${key}:" "$file" 2>/dev/null | sed "s/^${key}:[[:space:]]*//" | tr -d '"' || echo ""
}

# Parse entrypoint which can be either a string or array
# Returns JSON array format: ["cmd"] or ["cmd", "arg1", "arg2"]
yaml_get_entrypoint_json() {
    local file="$1"
    local line=$(grep "^entrypoint:" "$file" 2>/dev/null)

    if [ -z "$line" ]; then
        echo '[""]'
        return
    fi

    # Check if it's an inline array: entrypoint: ["bash", "-i"]
    if [[ "$line" =~ \[.*\] ]]; then
        # Extract the array part and output as-is (already JSON)
        echo "$line" | sed 's/^entrypoint:[[:space:]]*//'
        return
    fi

    # Check if it's a simple string: entrypoint: bash
    local value=$(echo "$line" | sed 's/^entrypoint:[[:space:]]*//' | tr -d '"')
    if [ -n "$value" ]; then
        printf '["%s"]' "$value"
        return
    fi

    # Check for multi-line array format
    local in_entrypoint=false
    local items=()

    while IFS= read -r l; do
        if [[ "$l" =~ ^entrypoint: ]]; then
            in_entrypoint=true
            continue
        fi
        if $in_entrypoint; then
            # Stop if we hit another top-level key
            if [[ "$l" =~ ^[a-z] ]] && [[ ! "$l" =~ ^[[:space:]] ]]; then
                break
            fi
            # Extract item (- item format)
            if [[ "$l" =~ ^[[:space:]]*-[[:space:]]*(.+) ]]; then
                item="${BASH_REMATCH[1]}"
                item=$(echo "$item" | tr -d '"' | tr -d "'")
                items+=("$item")
            fi
        fi
    done < "$file"

    # Build JSON array
    if [ ${#items[@]} -gt 0 ]; then
        local first=true
        echo -n "["
        for item in "${items[@]}"; do
            if [ "$first" = true ]; then
                first=false
            else
                echo -n ","
            fi
            printf '"%s"' "$item"
        done
        echo -n "]"
    else
        echo '[""]'
    fi
}

# Get version for an extension (override > default from yaml > "latest")
get_extension_version() {
    local ext="$1"
    local config="$2"

    # Check for override first
    if [ -n "${VERSION_OVERRIDES[$ext]}" ]; then
        echo "${VERSION_OVERRIDES[$ext]}"
        return
    fi

    # Read default_version from config.yaml
    local default_ver=$(yaml_get "$config" "default_version")
    if [ -n "$default_ver" ]; then
        echo "$default_ver"
        return
    fi

    # Fallback to latest
    echo "latest"
}

# Parse list items from YAML (handles both [] and list format)
yaml_get_list() {
    local file="$1"
    local key="$2"
    local in_section=false
    local items=""

    while IFS= read -r line; do
        if [[ "$line" =~ ^${key}: ]]; then
            # Check for inline empty array
            if [[ "$line" =~ \[\] ]]; then
                echo ""
                return
            fi
            in_section=true
            continue
        fi
        if $in_section; then
            # Stop if we hit another top-level key
            if [[ "$line" =~ ^[a-z] ]] && [[ ! "$line" =~ ^[[:space:]] ]]; then
                break
            fi
            # Extract item (- item format)
            if [[ "$line" =~ ^[[:space:]]*-[[:space:]]*(.+) ]]; then
                item="${BASH_REMATCH[1]}"
                items="$items $item"
            fi
        fi
    done < "$file"

    echo "$items" | xargs
}

# Parse dependencies from YAML
yaml_get_deps() {
    yaml_get_list "$1" "dependencies"
}

# Parse env_vars from YAML (returns JSON array of strings)
yaml_get_env_vars_json() {
    local file="$1"
    local items=$(yaml_get_list "$file" "env_vars")
    local first=true

    echo -n "["
    for item in $items; do
        if [ "$first" = true ]; then
            first=false
        else
            echo -n ","
        fi
        printf '"%s"' "$item"
    done
    echo -n "]"
}

# Parse mounts from YAML (returns JSON array of {source, target} objects)
yaml_get_mounts_json() {
    local file="$1"
    local in_mounts=false
    local current_source=""
    local current_target=""
    local first=true

    echo -n "["

    # Helper to output current mount
    output_mount() {
        if [ -n "$current_source" ] && [ -n "$current_target" ]; then
            if [ "$first" = true ]; then
                first=false
            else
                echo -n ","
            fi
            printf '{"source":"%s","target":"%s"}' "$current_source" "$current_target"
        fi
        current_source=""
        current_target=""
    }

    while IFS= read -r line || [[ -n "$line" ]]; do
        if [[ "$line" =~ ^mounts: ]]; then
            if [[ "$line" =~ \[\] ]]; then
                echo -n "]"
                return
            fi
            in_mounts=true
            continue
        fi
        if $in_mounts; then
            # Stop if we hit another top-level key
            if [[ "$line" =~ ^[a-z] ]] && [[ ! "$line" =~ ^[[:space:]] ]]; then
                output_mount
                break
            fi
            # New mount entry starts with "- source:"
            if [[ "$line" =~ ^[[:space:]]*-[[:space:]]*source:[[:space:]]*(.+) ]]; then
                output_mount
                current_source="${BASH_REMATCH[1]}"
            # Source without dash (continuation)
            elif [[ "$line" =~ ^[[:space:]]+source:[[:space:]]*(.+) ]]; then
                current_source="${BASH_REMATCH[1]}"
            fi
            # Parse target
            if [[ "$line" =~ ^[[:space:]]*target:[[:space:]]*(.+) ]]; then
                current_target="${BASH_REMATCH[1]}"
            fi
        fi
    done < "$file"

    # Output last mount if any
    output_mount

    echo -n "]"
}

# Parse flags from YAML (returns JSON array of {flag, description} objects)
yaml_get_flags_json() {
    local file="$1"
    local in_flags=false
    local current_flag=""
    local current_desc=""
    local first=true

    echo -n "["

    while IFS= read -r line || [[ -n "$line" ]]; do
        if [[ "$line" =~ ^flags: ]]; then
            if [[ "$line" =~ \[\] ]]; then
                echo -n "]"
                return
            fi
            in_flags=true
            continue
        fi
        if $in_flags; then
            # Stop if we hit another top-level key
            if [[ "$line" =~ ^[a-z] ]] && [[ ! "$line" =~ ^[[:space:]] ]]; then
                break
            fi
            # Parse flag
            if [[ "$line" =~ ^[[:space:]]*-?[[:space:]]*flag:[[:space:]]*[\"\']*([^\"\']+)[\"\']*$ ]]; then
                current_flag="${BASH_REMATCH[1]}"
            fi
            # Parse description
            if [[ "$line" =~ ^[[:space:]]*description:[[:space:]]*[\"\']*([^\"\']+)[\"\']*$ ]]; then
                current_desc="${BASH_REMATCH[1]}"
                # Output the flag entry
                if [ "$first" = true ]; then
                    first=false
                else
                    echo -n ","
                fi
                # Escape quotes in flag and description
                current_flag=$(echo "$current_flag" | sed 's/"/\\"/g')
                current_desc=$(echo "$current_desc" | sed 's/"/\\"/g')
                printf '{"flag":"%s","description":"%s"}' "$current_flag" "$current_desc"
                current_flag=""
                current_desc=""
            fi
        fi
    done < "$file"

    echo -n "]"
}

# Build installation order with dependencies
declare -A installed
declare -a install_order

resolve_extension() {
    local ext="$1"

    # Skip if already processed
    if [ "${installed[$ext]}" = "1" ]; then
        return
    fi

    local ext_dir="$EXTENSIONS_DIR/$ext"
    local config="$ext_dir/config.yaml"
    local script="$ext_dir/install.sh"

    if [ ! -d "$ext_dir" ]; then
        echo "Extensions: Warning - extension '$ext' not found (no $ext/ directory), skipping"
        return
    fi

    if [ ! -f "$config" ]; then
        echo "Extensions: Warning - extension '$ext' has no config.yaml, skipping"
        return
    fi

    # install.sh is optional - extension can be metadata-only

    # Process dependencies first
    local deps=$(yaml_get_deps "$config")
    for dep in $deps; do
        resolve_extension "$dep"
    done

    # Add to install order
    install_order+=("$ext")
    installed[$ext]=1
}

echo "Extensions: Resolving dependencies..."

# Parse comma-separated extension list
IFS=',' read -ra EXT_ARRAY <<< "$EXTENSIONS"
for ext in "${EXT_ARRAY[@]}"; do
    ext=$(echo "$ext" | xargs)  # trim whitespace
    [ -n "$ext" ] && resolve_extension "$ext"
done

if [ ${#install_order[@]} -eq 0 ]; then
    echo "Extensions: No valid extensions to install"
    echo '{"extensions":{}}' > "$METADATA_FILE"
    exit 0
fi

echo "Extensions: Installation order: ${install_order[*]}"

# Install each extension
for ext in "${install_order[@]}"; do
    ext_dir="$EXTENSIONS_DIR/$ext"
    config="$ext_dir/config.yaml"
    script="$ext_dir/install.sh"

    description=$(yaml_get "$config" "description")

    # Get version for this extension (override > yaml default > "latest")
    ext_version=$(get_extension_version "$ext" "$config")

    # Convert extension name to uppercase env var name (e.g., claude -> CLAUDE_VERSION)
    # Handle hyphens by converting to underscores (e.g., claude-flow -> CLAUDE_FLOW_VERSION)
    ext_env_name=$(echo "$ext" | tr '[:lower:]-' '[:upper:]_')
    version_env_var="${ext_env_name}_VERSION"

    echo "=========================================="
    echo "Extensions: Installing '$ext' (version: $ext_version)"
    [ -n "$description" ] && echo "  $description"
    echo "=========================================="

    # Run install.sh if it exists (optional)
    # Export version as <EXT>_VERSION environment variable
    if [ -f "$script" ]; then
        export "$version_env_var=$ext_version"
        bash "$script"
    else
        echo "  (no install.sh - metadata only)"
    fi
done

# Write metadata JSON
echo "Extensions: Writing metadata to $METADATA_FILE"
{
    echo '{"extensions":{'
    first=true
    for ext in "${install_order[@]}"; do
        config="$EXTENSIONS_DIR/$ext/config.yaml"
        name=$(yaml_get "$config" "name")
        description=$(yaml_get "$config" "description")
        entrypoint=$(yaml_get_entrypoint_json "$config")
        auto_mount=$(yaml_get "$config" "auto_mount")
        mounts=$(yaml_get_mounts_json "$config")
        flags=$(yaml_get_flags_json "$config")
        env_vars=$(yaml_get_env_vars_json "$config")

        [ "$first" = true ] && first=false || echo ","
        # Build JSON with optional auto_mount field
        # Note: entrypoint is already JSON array format
        if [ -n "$auto_mount" ]; then
            printf '"%s":{"name":"%s","description":"%s","entrypoint":%s,"auto_mount":%s,"mounts":%s,"flags":%s,"env_vars":%s}' \
                "$ext" "$name" "$description" "$entrypoint" "$auto_mount" "$mounts" "$flags" "$env_vars"
        else
            printf '"%s":{"name":"%s","description":"%s","entrypoint":%s,"mounts":%s,"flags":%s,"env_vars":%s}' \
                "$ext" "$name" "$description" "$entrypoint" "$mounts" "$flags" "$env_vars"
        fi
    done
    echo '}}'
} > "$METADATA_FILE"

echo "=========================================="
echo "Extensions: All extensions installed successfully"
echo "Installed: ${install_order[*]}"
echo "=========================================="
