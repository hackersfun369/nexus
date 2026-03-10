package plugin

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorRed    = "\033[31m"
)

// CLI handles plugin subcommands
type CLI struct {
	registry *Registry
	out      io.Writer
}

// NewCLI creates a plugin CLI handler
func NewCLI(registry *Registry) *CLI {
	return &CLI{registry: registry, out: os.Stdout}
}

// Run dispatches plugin subcommands
func (c *CLI) Run(ctx context.Context, args []string) {
	if len(args) == 0 {
		c.printHelp()
		return
	}
	switch args[0] {
	case "list", "ls":
		c.runList(ctx, args[1:])
	case "install", "add":
		c.runInstall(ctx, args[1:])
	case "remove", "rm", "uninstall":
		c.runRemove(ctx, args[1:])
	case "sync", "update":
		c.runSync(ctx, args[1:]...)
	case "search":
		c.runSearch(ctx, args[1:])
	case "info":
		c.runInfo(ctx, args[1:])
	default:
		fmt.Fprintf(c.out, "%sUnknown plugin command: %s%s\n", colorRed, args[0], colorReset)
		c.printHelp()
	}
}

func (c *CLI) runList(ctx context.Context, args []string) {
	filter := Filter{}
	if len(args) > 0 && args[0] == "--installed" {
		filter.Status = StatusInstalled
	}

	plugins, err := c.registry.store.List(ctx, filter)
	if err != nil {
		fmt.Fprintf(c.out, "%sError: %v%s\n", colorRed, err, colorReset)
		return
	}

	if len(plugins) == 0 {
		fmt.Fprintf(c.out, "%sNo plugins found. Run 'nexus plugin sync' to fetch the registry.%s\n",
			colorYellow, colorReset)
		return
	}

	fmt.Fprintf(c.out, "\n%s%s  Plugins (%d)%s\n", colorBold, colorCyan, len(plugins), colorReset)
	fmt.Fprintf(c.out, "%s  ─────────────────────────────────────────────────────────%s\n",
		colorDim, colorReset)

	w := tabwriter.NewWriter(c.out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  %sID\tVERSION\tLANGUAGE\tPLATFORM\tSTATUS%s\n",
		colorBold, colorReset)
	for _, p := range plugins {
		status := statusBadge(p.Status)
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\n",
			p.ID, p.Version, p.Language, p.Platform, status)
	}
	w.Flush()
	fmt.Fprintln(c.out)
}

func (c *CLI) runInstall(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(c.out, "%sUsage: nexus plugin install <id>%s\n", colorYellow, colorReset)
		return
	}
	id := args[0]
	fmt.Fprintf(c.out, "%s  Installing %s...%s\n", colorDim, id, colorReset)

	p, err := c.registry.Install(ctx, id)
	if err != nil {
		fmt.Fprintf(c.out, "%s  ✗ Failed: %v%s\n", colorRed, err, colorReset)
		return
	}
	fmt.Fprintf(c.out, "%s  ✓ Installed %s v%s%s\n", colorGreen, p.Name, p.Version, colorReset)
}

func (c *CLI) runRemove(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(c.out, "%sUsage: nexus plugin remove <id>%s\n", colorYellow, colorReset)
		return
	}
	id := args[0]
	if err := c.registry.Remove(ctx, id); err != nil {
		fmt.Fprintf(c.out, "%s  ✗ Failed: %v%s\n", colorRed, err, colorReset)
		return
	}
	fmt.Fprintf(c.out, "%s  ✓ Removed %s%s\n", colorGreen, id, colorReset)
}

func (c *CLI) runSync(ctx context.Context, args ...string) {
	fmt.Fprintf(c.out, "%s  Syncing plugin registry...%s\n", colorDim, colorReset)
	localFile := ""
	for _, a := range args {
		if a != "--local" {
			localFile = a
		}
	}
	var manifest *RegistryManifest
	var err error
	if localFile != "" {
		manifest, err = c.registry.SyncFromFile(ctx, localFile)
	} else {
		manifest, err = c.registry.Sync(ctx)
	}
	if err != nil {
		fmt.Fprintf(c.out, "%s  ✗ Sync failed: %v%s\n", colorRed, err, colorReset)
		return
	}
	fmt.Fprintf(c.out, "%s  ✓ Synced %d plugins from registry%s\n",
		colorGreen, len(manifest.Plugins), colorReset)
}

func (c *CLI) runSearch(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(c.out, "%sUsage: nexus plugin search <language|platform>%s\n",
			colorYellow, colorReset)
		return
	}
	term := args[0]
	filter := Filter{}

	// Try as platform first
	platforms := map[string]bool{
		"android": true, "web": true, "windows": true,
		"mac": true, "cli": true, "backend": true, "all": true,
	}
	if platforms[term] {
		filter.Platform = term
	} else {
		filter.Language = term
	}

	plugins, err := c.registry.store.List(ctx, filter)
	if err != nil {
		fmt.Fprintf(c.out, "%sError: %v%s\n", colorRed, err, colorReset)
		return
	}
	if len(plugins) == 0 {
		fmt.Fprintf(c.out, "%s  No plugins found for '%s'%s\n", colorYellow, term, colorReset)
		return
	}

	fmt.Fprintf(c.out, "\n%s%s  Results for '%s' (%d)%s\n",
		colorBold, colorCyan, term, len(plugins), colorReset)
	for _, p := range plugins {
		fmt.Fprintf(c.out, "  %s%s%s v%s — %s\n",
			colorGreen, p.ID, colorReset, p.Version, p.Description)
	}
	fmt.Fprintln(c.out)
}

func (c *CLI) runInfo(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(c.out, "%sUsage: nexus plugin info <id>%s\n", colorYellow, colorReset)
		return
	}
	p, err := c.registry.store.Get(ctx, args[0])
	if err != nil {
		fmt.Fprintf(c.out, "%s  Plugin not found: %s%s\n", colorRed, args[0], colorReset)
		return
	}

	fmt.Fprintf(c.out, "\n%s%s  %s%s\n", colorBold, colorCyan, p.Name, colorReset)
	fmt.Fprintf(c.out, "%s  ─────────────────────────────────%s\n", colorDim, colorReset)
	fmt.Fprintf(c.out, "  %sID:%s       %s\n", colorBold, colorReset, p.ID)
	fmt.Fprintf(c.out, "  %sVersion:%s  %s\n", colorBold, colorReset, p.Version)
	fmt.Fprintf(c.out, "  %sLanguage:%s %s\n", colorBold, colorReset, p.Language)
	fmt.Fprintf(c.out, "  %sPlatform:%s %s\n", colorBold, colorReset, p.Platform)
	fmt.Fprintf(c.out, "  %sStatus:%s   %s\n", colorBold, colorReset, statusBadge(p.Status))
	fmt.Fprintf(c.out, "  %sDesc:%s     %s\n", colorBold, colorReset, p.Description)
	if p.InstallPath != "" {
		fmt.Fprintf(c.out, "  %sPath:%s     %s\n", colorBold, colorReset, p.InstallPath)
	}
	fmt.Fprintln(c.out)
}

func (c *CLI) printHelp() {
	fmt.Fprintf(c.out, "\n%s%s  nexus plugin — Plugin Manager%s\n", colorBold, colorCyan, colorReset)
	fmt.Fprintf(c.out, "%s  ─────────────────────────────────────%s\n", colorDim, colorReset)
	cmds := [][2]string{
		{"list [--installed]", "List all plugins"},
		{"install <id>", "Install a plugin"},
		{"remove <id>", "Remove a plugin"},
		{"sync", "Sync registry from plugins.nexus.dev"},
		{"search <term>", "Search plugins by language or platform"},
		{"info <id>", "Show plugin details"},
	}
	for _, cmd := range cmds {
		fmt.Fprintf(c.out, "  %s%-22s%s %s%s%s\n",
			colorGreen, cmd[0], colorReset, colorDim, cmd[1], colorReset)
	}
	fmt.Fprintln(c.out)
}

func statusBadge(s Status) string {
	switch s {
	case StatusInstalled:
		return colorGreen + "installed" + colorReset
	case StatusDisabled:
		return colorYellow + "disabled" + colorReset
	default:
		return colorDim + "available" + colorReset
	}
}
