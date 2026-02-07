#!/bin/bash
set -e

# Network firewall initialization for addt
# Implements a whitelist-based firewall to restrict outbound network access

ALLOWED_DOMAINS_FILE="${FIREWALL_CONFIG_FILE:-/home/addt/.addt/firewall/allowed-domains.txt}"

# Check if firewall is disabled
if [ "${ADDT_FIREWALL_MODE}" = "off" ] || [ "${ADDT_FIREWALL_MODE}" = "disabled" ]; then
    echo "Firewall: Disabled by configuration"
    exit 0
fi

echo "Firewall: Initializing network restrictions..."

# Create ipset for allowed IPs
ipset create allowed_ips hash:ip hashsize 4096 maxelem 65536 2>/dev/null || true

# Read domains from config file
if [ -f "$ALLOWED_DOMAINS_FILE" ]; then
    echo "Firewall: Loading allowed domains from $ALLOWED_DOMAINS_FILE"

    # Read domains, filter comments and empty lines
    while IFS= read -r domain || [ -n "$domain" ]; do
        # Skip comments and empty lines
        [[ "$domain" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$domain" ]] && continue

        # Trim whitespace
        domain=$(echo "$domain" | xargs)

        # Resolve domain to IPs
        echo "  Resolving: $domain"

        # Try dig first (preferred)
        if command -v dig >/dev/null 2>&1; then
            IPS=$(dig +short "$domain" A | grep -E '^[0-9]+\.' || true)
        # Fallback to host
        elif command -v host >/dev/null 2>&1; then
            IPS=$(host "$domain" | grep "has address" | awk '{print $4}' || true)
        else
            echo "  Warning: No DNS tools available (dig/host)"
            continue
        fi

        # Add IPs to ipset
        for ip in $IPS; do
            if [[ $ip =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
                ipset add allowed_ips "$ip" 2>/dev/null || true
                echo "    Added: $ip"
            fi
        done
    done < "$ALLOWED_DOMAINS_FILE"
else
    echo "Firewall: Warning - No allowed domains file found at $ALLOWED_DOMAINS_FILE"
    echo "Firewall: Creating default configuration..."

    # Create directory if needed
    mkdir -p "$(dirname "$ALLOWED_DOMAINS_FILE")"

    # Create default allowed domains
    cat > "$ALLOWED_DOMAINS_FILE" << 'EOF'
# Default allowed domains for addt
# Lines starting with # are comments

# Anthropic API
api.anthropic.com

# GitHub
github.com
api.github.com
raw.githubusercontent.com
objects.githubusercontent.com

# npm registry
registry.npmjs.org

# PyPI
pypi.org
files.pythonhosted.org

# Go modules
proxy.golang.org
sum.golang.org

# Docker Hub (if needed)
registry-1.docker.io
auth.docker.io
production.cloudflare.docker.com

# Common CDNs
cdn.jsdelivr.net
unpkg.com
EOF

    chown "$(id -u addt):$(id -g addt)" "$ALLOWED_DOMAINS_FILE" 2>/dev/null || true

    echo "Firewall: Default configuration created"
    echo "Firewall: Edit $ALLOWED_DOMAINS_FILE to customize allowed domains"

    # Re-read the freshly created file to resolve domains
    echo "Firewall: Resolving default domains..."
    while IFS= read -r domain || [ -n "$domain" ]; do
        [[ "$domain" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$domain" ]] && continue
        domain=$(echo "$domain" | xargs)
        echo "  Resolving: $domain"
        if command -v dig >/dev/null 2>&1; then
            IPS=$(dig +short "$domain" A | grep -E '^[0-9]+\.' || true)
        elif command -v host >/dev/null 2>&1; then
            IPS=$(host "$domain" | grep "has address" | awk '{print $4}' || true)
        else
            continue
        fi
        for ip in $IPS; do
            if [[ $ip =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
                ipset add allowed_ips "$ip" 2>/dev/null || true
                echo "    Added: $ip"
            fi
        done
    done < "$ALLOWED_DOMAINS_FILE"
fi

# Set up iptables rules
echo "Firewall: Configuring iptables rules..."

# Flush existing rules
iptables -F OUTPUT 2>/dev/null || true

# Allow loopback
iptables -A OUTPUT -o lo -j ACCEPT

# Allow established/related connections
iptables -A OUTPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

# Allow DNS (needed for resolution)
iptables -A OUTPUT -p udp --dport 53 -j ACCEPT
iptables -A OUTPUT -p tcp --dport 53 -j ACCEPT

# Allow traffic to whitelisted IPs
iptables -A OUTPUT -m set --match-set allowed_ips dst -j ACCEPT

# Log and drop everything else
if [ "${ADDT_FIREWALL_MODE}" = "strict" ] || [ "${ADDT_FIREWALL_MODE}" = "enabled" ]; then
    iptables -A OUTPUT -j LOG --log-prefix "DCLAUDE-FIREWALL-BLOCKED: " --log-level 4
    iptables -A OUTPUT -j DROP
    echo "Firewall: Strict mode enabled - blocking all non-whitelisted traffic"
elif [ "${ADDT_FIREWALL_MODE}" = "permissive" ]; then
    iptables -A OUTPUT -j LOG --log-prefix "DCLAUDE-FIREWALL-WOULD-BLOCK: " --log-level 4
    iptables -A OUTPUT -j ACCEPT
    echo "Firewall: Permissive mode enabled - logging but allowing all traffic"
else
    # Default to strict
    iptables -A OUTPUT -j LOG --log-prefix "DCLAUDE-FIREWALL-BLOCKED: " --log-level 4
    iptables -A OUTPUT -j DROP
    echo "Firewall: Default strict mode enabled"
fi

# Show summary
IP_COUNT=$(ipset list allowed_ips | grep -c "^[0-9]" || echo "0")
echo "Firewall: Initialized with $IP_COUNT whitelisted IPs"
echo "Firewall: Mode: ${ADDT_FIREWALL_MODE:-strict}"
