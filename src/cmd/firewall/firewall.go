package firewall

import (
	"fmt"
	"os"
)

// DefaultAllowedDomains returns the default allowed domains for firewall
func DefaultAllowedDomains() []string {
	return []string{
		"api.anthropic.com",
		"github.com",
		"api.github.com",
		"raw.githubusercontent.com",
		"objects.githubusercontent.com",
		"registry.npmjs.org",
		"pypi.org",
		"files.pythonhosted.org",
		"proxy.golang.org",
		"sum.golang.org",
		"registry-1.docker.io",
		"auth.docker.io",
		"production.cloudflare.docker.com",
		"cdn.jsdelivr.net",
		"unpkg.com",
	}
}

// HandleCommand handles the firewall subcommand
func HandleCommand(args []string) {
	if len(args) == 0 {
		printHelp()
		return
	}

	scope := args[0]

	switch scope {
	case "global":
		handleGlobal(args[1:])
	case "project":
		handleProject(args[1:])
	case "extension":
		handleExtension(args[1:])
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Printf("Unknown firewall scope: %s\n", scope)
		fmt.Println("Use: global, project, or extension")
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`addt firewall - Manage network firewall rules

Usage: addt firewall <scope> <command> [args]

Scopes:
  global                   Manage global firewall rules (~/.addt/config.yaml)
  project                  Manage project firewall rules (.addt.yaml)
  extension <name>         Manage per-extension firewall rules

Commands:
  allow <domain>           Add domain to allowed list
  deny <domain>            Add domain to denied list
  remove <domain>          Remove domain from any list
  list                     List firewall rules
  reset                    Reset to defaults (global) or clear (project/extension)

Examples:
  addt firewall global list
  addt firewall global allow api.example.com
  addt firewall global deny malware.com
  addt firewall global reset

  addt firewall project allow custom-api.com
  addt firewall project list

  addt firewall extension claude allow api.anthropic.com
  addt firewall extension codex allow api.openai.com
  addt firewall extension claude list

Rule Evaluation (layered override, most specific wins):
  Defaults → Extension → Global → Project

  Each layer checks deny first, then allow. First match wins.
  Project rules override global, global overrides extension, etc.

  Example: Defaults allow npm, global denies it, project re-allows it.

Firewall Modes (set via 'addt config'):
  strict      - Block all except allowed (default)
  permissive  - Allow all except denied
  off         - Disabled`)
}
