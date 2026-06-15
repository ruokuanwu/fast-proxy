package app

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"fast-proxy/internal/caddy"
	"fast-proxy/internal/config"
	"fast-proxy/internal/hosts"

	"github.com/spf13/cobra"
)

func Run(args []string) error {
	cmd := newRootCommand()
	cmd.SetArgs(args)
	return cmd.Execute()
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "fast-proxy",
		Short:         "Quickly configure local reverse proxies",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newAddCommand())
	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newDoctorCommand())
	cmd.AddCommand(newSyncCommand())
	cmd.AddCommand(newRemoveCommand())
	cmd.AddCommand(newListCommand())

	return cmd
}

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize fast-proxy Caddy configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, caddyManager, err := newRuntime()
			if err != nil {
				return err
			}
			return initCaddy(caddyManager)
		},
	}
}

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check the fast-proxy runtime environment",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, _, caddyManager, err := newRuntime()
			if err != nil {
				return err
			}
			return doctor(store, caddyManager)
		},
	}
}

func newSyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Resync hosts and Caddy configuration from the state file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, hostFile, caddyManager, err := newRuntime()
			if err != nil {
				return err
			}
			return syncState(store, hostFile, caddyManager)
		},
	}
}

func newAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add <domain> <host:port>",
		Short: "Add a local domain reverse proxy",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, hostFile, caddyManager, err := newRuntime()
			if err != nil {
				return err
			}
			return add(store, hostFile, caddyManager, args[0], args[1])
		},
	}
}

func newRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <id> [id...]",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove local domain reverse proxies by ID",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, hostFile, caddyManager, err := newRuntime()
			if err != nil {
				return err
			}
			return remove(store, hostFile, caddyManager, args)
		},
	}
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List proxy rules managed by this tool",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, _, _, err := newRuntime()
			if err != nil {
				return err
			}
			return list(store)
		},
	}
}

func newRuntime() (*config.Store, *hosts.File, *caddy.Manager, error) {
	paths, err := config.ResolvePaths()
	if err != nil {
		return nil, nil, nil, err
	}

	store := config.NewStore(paths.ConfigFile)
	hostFile := hosts.NewFile("/etc/hosts")
	caddyManager := caddy.NewManager(paths.Caddyfile, paths.SitesDir)
	return store, hostFile, caddyManager, nil
}

func add(store *config.Store, hostFile *hosts.File, caddyManager *caddy.Manager, domain, target string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}
	if err := validateTarget(target); err != nil {
		return err
	}

	state, err := store.Load()
	if err != nil {
		return err
	}
	wasLocalTarget := false
	for _, rule := range state.Rules {
		if rule.Domain == domain && isLocalTarget(rule.Target) {
			wasLocalTarget = true
			break
		}
	}

	if err := state.Upsert(config.Rule{Domain: domain, Target: target}); err != nil {
		return err
	}
	if err := store.Save(state); err != nil {
		return err
	}
	if err := hostFile.Sync(state.Rules); err != nil {
		return err
	}
	if !isLocalTarget(target) {
		if wasLocalTarget {
			if err := caddyManager.Sync(localTargetRules(state.Rules)); err != nil {
				return err
			}
			if err := caddyManager.Reload(); err != nil {
				return err
			}
		}
		printRulesTable([]config.Rule{findRuleByDomain(state.Rules, domain)})
		return nil
	}
	if err := caddyManager.Sync(localTargetRules(state.Rules)); err != nil {
		return err
	}
	if err := caddyManager.Reload(); err != nil {
		return err
	}

	printRulesTable([]config.Rule{findRuleByDomain(state.Rules, domain)})
	return nil
}

func initCaddy(caddyManager *caddy.Manager) error {
	if err := caddyManager.Init(); err != nil {
		return err
	}
	if err := caddyManager.Reload(); err != nil {
		return err
	}
	fmt.Println("fast-proxy initialization completed.")
	fmt.Println()
	fmt.Println("Completed:")
	fmt.Println("  ✓ Checked Caddy")
	fmt.Printf("  ✓ Created site directory: %s\n", caddyManager.SitesDir())
	fmt.Printf("  ✓ Updated Caddyfile: %s\n", caddyManager.Caddyfile())
	fmt.Println("  ✓ Validated Caddy configuration")
	fmt.Println("  ✓ Reloaded Caddy")
	fmt.Println()
	fmt.Println("You can now add a proxy:")
	fmt.Println("  sudo fp add app.test localhost:3000")
	return nil
}

func doctor(store *config.Store, caddyManager *caddy.Manager) error {
	fmt.Println("fast-proxy doctor")
	fmt.Println()

	if !caddy.IsInstalled() {
		printCheck(false, "Caddy is not installed")
		fmt.Println()
		fmt.Println(caddy.InstallInstructions())
		return nil
	}
	version, err := caddy.Version()
	if err != nil {
		printCheck(false, err.Error())
	} else {
		printCheck(true, "Caddy is installed: "+version)
	}

	if _, err := os.Stat(caddyManager.Caddyfile()); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			printCheck(false, "Caddyfile does not exist: "+caddyManager.Caddyfile())
		} else {
			printCheck(false, "Unable to read Caddyfile: "+err.Error())
		}
	} else {
		printCheck(true, "Caddyfile exists: "+caddyManager.Caddyfile())
	}

	if stat, err := os.Stat(caddyManager.SitesDir()); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			printCheck(false, "Site directory does not exist: "+caddyManager.SitesDir())
		} else {
			printCheck(false, "Unable to read site directory: "+err.Error())
		}
	} else if !stat.IsDir() {
		printCheck(false, "Site path is not a directory: "+caddyManager.SitesDir())
	} else {
		printCheck(true, "Site directory exists: "+caddyManager.SitesDir())
	}

	if ok, err := caddyManager.HasImport(); err != nil {
		printCheck(false, "Unable to check fast-proxy import: "+err.Error())
	} else if !ok {
		printCheck(false, "fast-proxy import is not configured; run: sudo fp init")
	} else {
		printCheck(true, "fast-proxy import is configured")
	}

	if _, err := store.Load(); err != nil {
		printCheck(false, "State file error: "+err.Error())
	} else {
		printCheck(true, "State file is valid")
	}

	if err := caddyManager.Validate(); err != nil {
		printCheck(false, err.Error())
	} else {
		printCheck(true, "Caddy configuration is valid")
	}

	if status, err := caddy.ServiceStatus(); err != nil {
		printCheck(false, "Caddy service status: "+status)
		fmt.Println()
		fmt.Println("Suggested fix:")
		fmt.Println("  sudo systemctl enable --now caddy")
	} else {
		printCheck(true, "Caddy service status: "+status)
	}

	return nil
}

func syncState(store *config.Store, hostFile *hosts.File, caddyManager *caddy.Manager) error {
	state, err := store.Load()
	if err != nil {
		return err
	}
	if err := hostFile.Sync(state.Rules); err != nil {
		return err
	}
	if err := caddyManager.Sync(localTargetRules(state.Rules)); err != nil {
		return err
	}
	if err := caddyManager.Reload(); err != nil {
		return err
	}

	fmt.Println("Synced:")
	fmt.Println("  ✓ /etc/hosts")
	fmt.Printf("  ✓ %s/*.caddy\n", caddyManager.SitesDir())
	fmt.Println("  ✓ Caddy reload")
	fmt.Println()
	printRulesTable(state.Rules)
	return nil
}

func printCheck(ok bool, message string) {
	if ok {
		fmt.Println("✓ " + message)
		return
	}
	fmt.Println("✗ " + message)
}

func remove(store *config.Store, hostFile *hosts.File, caddyManager *caddy.Manager, ids []string) error {
	for _, id := range ids {
		if err := validateRuleID(id); err != nil {
			return err
		}
	}

	state, err := store.Load()
	if err != nil {
		return err
	}
	resolvedIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		resolvedID, ok, err := resolveRuleIDPrefix(state.Rules, id)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		resolvedIDs = append(resolvedIDs, resolvedID)
	}
	if len(resolvedIDs) == 0 {
		printRulesTable(nil)
		return nil
	}
	localBefore := len(localTargetRules(state.Rules))

	removed := make([]config.Rule, 0, len(resolvedIDs))
	for _, id := range resolvedIDs {
		rule, ok := state.RemoveByID(id)
		if !ok {
			fmt.Printf("Rule does not exist: %s\n", id)
			continue
		}
		removed = append(removed, rule)
	}
	if len(removed) == 0 {
		printRulesTable(removed)
		return nil
	}

	if err := store.Save(state); err != nil {
		return err
	}
	if err := hostFile.Sync(state.Rules); err != nil {
		return err
	}
	localAfterRules := localTargetRules(state.Rules)
	if localBefore != len(localAfterRules) || containsLocalTarget(removed) {
		if err := caddyManager.Sync(localAfterRules); err != nil {
			return err
		}
		if err := caddyManager.Reload(); err != nil {
			return err
		}
	}

	printRulesTable(removed)
	return nil
}

func resolveRuleIDPrefix(rules []config.Rule, prefix string) (string, bool, error) {
	for _, rule := range rules {
		if rule.ID == prefix {
			return rule.ID, true, nil
		}
	}

	matches := make([]string, 0, 2)
	for _, rule := range rules {
		if strings.HasPrefix(rule.ID, prefix) {
			matches = append(matches, rule.ID)
			if len(matches) > 1 {
				return "", false, fmt.Errorf("id prefix is ambiguous: %s; matches: %s", prefix, strings.Join(matches, ", "))
			}
		}
	}
	if len(matches) == 0 {
		return "", false, nil
	}
	return matches[0], true, nil
}

func list(store *config.Store) error {
	state, err := store.Load()
	if err != nil {
		return err
	}
	printRulesTable(state.Rules)
	return nil
}

func printRulesTable(rules []config.Rule) {
	widths := []int{12, len("DOMAIN"), len("TARGET")}
	for _, rule := range rules {
		widths[0] = max(widths[0], len(rule.ID))
		widths[1] = max(widths[1], len(rule.Domain))
		widths[2] = max(widths[2], len(rule.Target))
	}

	printTableBorder(widths)
	printTableRow(widths, []string{"ID", "DOMAIN", "TARGET"})
	printTableBorder(widths)
	for _, rule := range rules {
		printTableRow(widths, []string{rule.ID, rule.Domain, rule.Target})
	}
	printTableBorder(widths)
}

func printTableBorder(widths []int) {
	fmt.Print("+")
	for _, width := range widths {
		fmt.Print(strings.Repeat("-", width+2), "+")
	}
	fmt.Println()
}

func printTableRow(widths []int, values []string) {
	fmt.Print("|")
	for i, value := range values {
		fmt.Printf(" %-*s |", widths[i], value)
	}
	fmt.Println()
}

func findRuleByDomain(rules []config.Rule, domain string) config.Rule {
	for _, rule := range rules {
		if rule.Domain == domain {
			return rule
		}
	}
	return config.Rule{Domain: domain}
}

func localTargetRules(rules []config.Rule) []config.Rule {
	local := make([]config.Rule, 0, len(rules))
	for _, rule := range rules {
		if isLocalTarget(rule.Target) {
			local = append(local, rule)
		}
	}
	return local
}

func containsLocalTarget(rules []config.Rule) bool {
	for _, rule := range rules {
		if isLocalTarget(rule.Target) {
			return true
		}
	}
	return false
}

func isLocalTarget(target string) bool {
	host, _, ok := strings.Cut(target, ":")
	return ok && (host == "localhost" || host == "127.0.0.1")
}

func targetHost(target string) string {
	host, _, ok := strings.Cut(target, ":")
	if ok {
		return host
	}
	return target
}

func validateDomain(domain string) error {
	if domain == "" {
		return errors.New("domain cannot be empty")
	}
	if domain == "localhost" {
		return errors.New("proxying localhost is not allowed")
	}
	if strings.ContainsAny(domain, " /:") {
		return fmt.Errorf("invalid domain format: %s", domain)
	}
	return nil
}

func validateRuleID(id string) error {
	if id == "" {
		return errors.New("id cannot be empty")
	}
	if strings.ContainsAny(id, " \t\n\r") {
		return fmt.Errorf("invalid id format: %s", id)
	}
	return nil
}

func validateTarget(target string) error {
	host, portText, ok := strings.Cut(target, ":")
	if !ok {
		if target == "" || strings.ContainsAny(target, " \t\n\r/") {
			return errors.New("invalid target format; use host or host:port, for example 1.1.1.1 or localhost:3000")
		}
		return nil
	}
	if host == "" || portText == "" || strings.Contains(portText, ":") {
		return errors.New("invalid target format; use host:port, for example localhost:3000")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("target port must be an integer from 1 to 65535")
	}
	return nil
}
