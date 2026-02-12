package flags

import (
	"strings"

	"github.com/reeflective/console"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	DefaultTimeout = 60
)

// Bind is a convenience function to bind flags to a command, through newly created
// Bind 是一个方便的函数，通过新创建的方法将标志绑定到命令
// pflag.Flagset type. This function can be called any number of times for any command.
// pflag.Flagset type. This 函数可以对任何 command. 调用任意多次
// desc        - An optional name for the flag set (can be empty, but might end up useful).
// desc - An 标志集的可选名称（可以为空，但最终可能有用）。
// persistent  - If true, the flags bound will apply to all subcommands of this command.
// 持久 - If true，绑定的标志将应用于此 command. 的所有子命令
// cmd         - The pointer to the command the flags should be bound to.
// cmd - The 指向命令的指针，标志应绑定 to.
// flags       - A function using this flag set as parameter, for you to register flags.
// flags - A 函数使用此标志集作为参数，用于注册 flags.
func Bind(desc string, persistent bool, cmd *cobra.Command, flags func(f *pflag.FlagSet)) {
	flagSet := pflag.NewFlagSet(desc, pflag.ContinueOnError)
	flags(flagSet)

	if persistent {
		cmd.PersistentFlags().AddFlagSet(flagSet)
	} else {
		cmd.Flags().AddFlagSet(flagSet)
	}
}

// RestrictTargets generates a cobra annotation map with a single console.CommandHiddenFilter key
// RestrictTargets 使用单个 console.CommandHiddenFilter 键生成眼镜蛇注释图
// to a comma-separated list of filters to use in order to expose/hide commands based on requirements.
// 到要使用的 comma__PH0__ 过滤器列表，以便基于 requirements. 执行 expose/hide 命令
// Ex: cmd.Annotations = RestrictTargets("windows") will only show the command if the target is Windows.
// Ex: cmd.Annotations = RestrictTargets(__PH0__) 仅当目标是 Windows. 时才会显示该命令
// Ex: cmd.Annotations = RestrictTargets("windows", "beacon") show the command if target is a beacon on Windows.
// Ex: cmd.Annotations = RestrictTargets(__PH0__, __PH1__) 如果目标是 Windows. 上的 beacon，则显示该命令
func RestrictTargets(filters ...string) map[string]string {
	if len(filters) == 0 {
		return nil
	}

	if len(filters) == 1 {
		return map[string]string{
			console.CommandFilterKey: filters[0],
		}
	}

	filts := strings.Join(filters, ",")

	return map[string]string{
		console.CommandFilterKey: filts,
	}
}

// NewCompletions registers the command to the application completion engine and returns
// NewCompletions 将命令注册到应用程序完成引擎并返回
// you a type through which you can register all sorts of completions for this command,
// 您可以通过它来注册此命令的各种完成的类型，
// from flag arguments, positional ones, per index or remaining, etc.
// 来自标志参数、位置参数、每个索引或剩余的 etc.
//
//	See https://rsteube.github.io/carapace/ for a complete documentation of carapace completions.
//	See __PH0__ 甲壳的完整文档 completions.
func NewCompletions(cmd *cobra.Command) *carapace.Carapace {
	return carapace.Gen(cmd)
}

// BindFlagCompletions is a convenience function for binding completers to flags requiring arguments.
// BindFlagCompletions 是一个方便的函数，用于将完成者绑定到需要 arguments. 的标志
// (It wraps a few steps to be used through the *carapace.Carapace type so you don't have to bother).
// （It 包装了通过 *carapace.Carapace 类型使用的几个步骤，因此您不必费心）。
// cmd   - The target command/subcommand which flags to be completed.
// cmd - The 目标 command/subcommand 标记为 completed.
// bind  - A function using a map "flag-name":carapace.Action for you to bind completions to the flag.
// 绑定 - A 函数使用映射 __PH0__:carapace.Action 来将完成绑定到 flag.
//
//	See https://rsteube.github.io/carapace/ for a complete documentation of carapace completions.
//	See __PH0__ 甲壳的完整文档 completions.
func BindFlagCompletions(cmd *cobra.Command, bind func(comp *carapace.ActionMap)) {
	comps := make(carapace.ActionMap)
	bind(&comps)

	carapace.Gen(cmd).FlagCompletion(comps)
}
