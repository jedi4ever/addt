#!/bin/bash
set -e

# Network firewall initialization for addt (Podman with pasta/slirp4netns support)
# Implements a whitelist-based firewall to restrict outbound network access
# Works with both traditional iptables and nftables (preferred for rootless Podman)

ALLOWED_DOMAINS_FILE="${FIREWALL_CONFIG_FILE:-/home/addt/.addt/firewall/allowed-domains.txt}"

# Check if firewall is disabled
if [ "${ADDT_FIREWALL_MODE}" = "off" ] || [ "${ADDT_FIREWALL_MODE}" = "disabled" ]; then
    echo "Firewall: Disabled by configuration"
    exit 0
fi

echo "Firewall: Initializing network restrictions..."

# Detect available firewall tools
USE_NFTABLES=false
USE_IPTABLES=false

if command -v nft >/dev/null 2>&1; then
    USE_NFTABLES=true
    echo "Firewall: Using nftables"
elif command -v iptables >/dev/null 2>&1; then
    USE_IPTABLES=true
    echo "Firewall: Using iptables"
else
    echo "Firewall: Warning - No firewall tools available (nft/iptables)"
    exit 0
fi

# Create allowed IPs storage
ALLOWED_IPS=""

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

        # Add IPs to list
        for ip in $IPS; do
            if [[ $ip =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
                ALLOWED_IPS="$ALLOWED_IPS $ip"
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

# Container registries
registry-1.docker.io
auth.docker.io
production.cloudflare.docker.com
quay.io
gcr.io
ghcr.io

# Common CDNs
cdn.jsdelivr.net
unpkg.com
EOF

    chown addt:$(id -gn addt) "$ALLOWED_DOMAINS_FILE" 2>/dev/null || true

    echo "Firewall: Default configuration created"
    echo "Firewall: Edit $ALLOWED_DOMAINS_FILE to customize allowed domains"
fi

# Configure firewall rules
if [ "$USE_NFTABLES" = true ]; then
    echo "Firewall: Configuring nftables rules..."

    # Flush existing rules
    nft flush ruleset 2>/dev/null || true

    # Create table and chain
    nft add table inet addt_filter 2>/dev/null || true
    nft add chain inet addt_filter output "{ type filter hook output priority 0; policy drop; }" 2>/dev/null || true

    # Allow loopback
    nft add rule inet addt_filter output oifname "lo" accept

    # Allow established/related connections
    nft add rule inet addt_filter output ct state established,related accept

    # Allow DNS
    nft add rule inet addt_filter output udp dport 53 accept
    nft add rule inet addt_filter output tcp dport 53 accept

    # Allow whitelisted IPs
    for ip in $ALLOWED_IPS; do
        nft add rule inet addt_filter output ip daddr "$ip" accept 2>/dev/null || true
    done

    # Log and handle based on mode
    if [ "${ADDT_FIREWALL_MODE}" = "strict" ] || [ "${ADDT_FIREWALL_MODE}" = "enabled" ]; then
        nft add rule inet addt_filter output log prefix \"ADDT-FIREWALL-BLOCKED: \" level warn
        echo "Firewall: Strict mode enabled - blocking all non-whitelisted traffic"
    elif [ "${ADDT_FIREWALL_MODE}" = "permissive" ]; then
        nft add rule inet addt_filter output log prefix \"ADDT-FIREWALL-WOULD-BLOCK: \" level warn
        nft add rule inet addt_filter output accept
        echo "Firewall: Permissive mode enabled - logging but allowing all traffic"
    else
        # Default to strict
        nft add rule inet addt_filter output log prefix \"ADDT-FIREWALL-BLOCKED: \" level warn
        echo "Firewall: Default strict mode enabled"
    fi

elif [ "$USE_IPTABLES" = true ]; then
    echo "Firewall: Configuring iptables rules..."

    # Create ipset for allowed IPs (if available)
    if command -v ipset >/dev/null 2>&1; then
        ipset create allowed_ips hash:ip hashsize 4096 maxelem 65536 2>/dev/null || true

        # Add IPs to ipset
        for ip in $ALLOWED_IPS; do
            ipset add allowed_ips "$ip" 2>/dev/null || true
        done

        USE_IPSET=true
    else
        USE_IPSET=false
    fi

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
    if [ "$USE_IPSET" = true ]; then
        iptables -A OUTPUT -m set --match-set allowed_ips dst -j ACCEPT
    else
        # Fallback: add individual rules for each IP
        for ip in $ALLOWED_IPS; do
            iptables -A OUTPUT -d "$ip" -j ACCEPT 2>/dev/null || true
        done
    fi

    # Log and drop/accept based on mode
    if [ "${ADDT_FIREWALL_MODE}" = "strict" ] || [ "${ADDT_FIREWALL_MODE}" = "enabled" ]; then
        iptables -A OUTPUT -j LOG --log-prefix "ADDT-FIREWALL-BLOCKED: " --log-level 4
        iptables -A OUTPUT -j DROP
        echo "Firewall: Strict mode enabled - blocking all non-whitelisted traffic"
    elif [ "${ADDT_FIREWALL_MODE}" = "permissive" ]; then
        iptables -A OUTPUT -j LOG --log-prefix "ADDT-FIREWALL-WOULD-BLOCK: " --log-level 4
        iptables -A OUTPUT -j ACCEPT
        echo "Firewall: Permissive mode enabled - logging but allowing all traffic"
    else
        # Default to strict
        iptables -A OUTPUT -j LOG --log-prefix "ADDT-FIREWALL-BLOCKED: " --log-level 4
        iptables -A OUTPUT -j DROP
        echo "Firewall: Default strict mode enabled"
    fi
fi

# Show summary
IP_COUNT=$(echo "$ALLOWED_IPS" | wc -w)
echo "Firewall: Initialized with $IP_COUNT whitelisted IPs"
echo "Firewall: Mode: ${ADDT_FIREWALL_MODE:-strict}"
