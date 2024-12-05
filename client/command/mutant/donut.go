package mutant

import (
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
	"github.com/wabzsy/gonut"
)

func DonutCmd(cmd *cobra.Command, con *repl.Console) error {
	donutConfig := gonut.DefaultConfig()

	// 获取标志值并进行强制类型转换
	donutConfig.ModuleName, _ = cmd.Flags().GetString("modname")
	donutConfig.Server, _ = cmd.Flags().GetString("server")

	// 强制类型转换
	entropy, _ := cmd.Flags().GetUint32("entropy")
	donutConfig.Entropy = gonut.EntropyType(entropy)

	arch, _ := cmd.Flags().GetInt("arch")
	donutConfig.Arch = gonut.ArchType(arch)

	donutConfig.Output, _ = cmd.Flags().GetString("output")

	format, _ := cmd.Flags().GetInt("format")
	donutConfig.Format = gonut.FormatType(format)

	oep, _ := cmd.Flags().GetUint32("oep")
	donutConfig.OEP = oep

	exitOpt, _ := cmd.Flags().GetUint32("exit")
	donutConfig.ExitOpt = gonut.ExitType(exitOpt)

	donutConfig.Class, _ = cmd.Flags().GetString("class")
	donutConfig.Domain, _ = cmd.Flags().GetString("domain")
	donutConfig.Input, _ = cmd.Flags().GetString("input")
	donutConfig.Method, _ = cmd.Flags().GetString("method")
	donutConfig.Args, _ = cmd.Flags().GetString("args")

	// 强制类型转换
	unicode, _ := cmd.Flags().GetBool("unicode")
	donutConfig.Unicode = gonut.BoolType(unicode)

	// 强制类型转换
	thread, _ := cmd.Flags().GetBool("thread")
	donutConfig.Thread = gonut.BoolType(thread)

	// 强制类型转换
	compress, _ := cmd.Flags().GetUint32("compress")
	donutConfig.GonutCompress = gonut.CompressionType(compress)

	// 强制类型转换
	bypass, _ := cmd.Flags().GetUint32("bypass")
	donutConfig.Bypass = gonut.BypassType(bypass)

	// 强制类型转换
	headers, _ := cmd.Flags().GetUint32("headers")
	donutConfig.Headers = gonut.HeadersType(headers)

	donutConfig.Decoy, _ = cmd.Flags().GetString("decoy")

	// 强制类型转换
	verbose, _ := cmd.Flags().GetBool("verbose")
	donutConfig.Verbose = verbose
	o := gonut.New(donutConfig)
	if err := o.Create(); err != nil {
		return err
	}
	o.ShowResults()
	return nil
}
