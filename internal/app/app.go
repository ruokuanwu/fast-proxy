package app

import (
	"errors"
	"fmt"
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
		Short:         "快速配置本地反向代理",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newAddCommand())
	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newRemoveCommand())
	cmd.AddCommand(newListCommand())

	return cmd
}

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "初始化系统 Caddy 配置",
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

func newAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add <domain> <host:port>",
		Short: "添加本地域名反向代理",
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
		Short:   "按 ID 删除本地域名反向代理",
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
		Short:   "查看本工具管理的代理规则",
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
	fmt.Println("fast-proxy Caddy 配置已初始化")
	return nil
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
			fmt.Printf("规则不存在: %s\n", id)
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
				return "", false, fmt.Errorf("id 前缀不唯一: %s，匹配到: %s", prefix, strings.Join(matches, ", "))
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
		return errors.New("domain 不能为空")
	}
	if domain == "localhost" {
		return errors.New("不允许代理 localhost")
	}
	if strings.ContainsAny(domain, " /:") {
		return fmt.Errorf("domain 格式错误: %s", domain)
	}
	return nil
}

func validateRuleID(id string) error {
	if id == "" {
		return errors.New("id 不能为空")
	}
	if strings.ContainsAny(id, " \t\n\r") {
		return fmt.Errorf("id 格式错误: %s", id)
	}
	return nil
}

func validateTarget(target string) error {
	host, portText, ok := strings.Cut(target, ":")
	if !ok {
		if target == "" || strings.ContainsAny(target, " \t\n\r/") {
			return errors.New("target 格式错误，请使用 host 或 host:port，例如 1.1.1.1 或 localhost:3000")
		}
		return nil
	}
	if host == "" || portText == "" || strings.Contains(portText, ":") {
		return errors.New("target 格式错误，请使用 host:port，例如 localhost:3000")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("target 端口必须是 1-65535 的整数")
	}
	return nil
}
