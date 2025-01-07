package main

import (
	"github.com/spf13/cobra"
	"github.com/wabzsy/gonut"
	"log"
)

func main() {
	c := gonut.DefaultConfig()

	cmd := NewCommand(c)

	cmd.RunE = func(*cobra.Command, []string) error {
		o := gonut.New(c)
		if err := o.Create(); err != nil {
			return err
		}
		o.ShowResults()
		return nil
	}

	if err := cmd.Execute(); err != nil {
		log.Println("error:", err)
	}
}

func NewCommand(c *gonut.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gonut",
		Short: "Only the finest artisanal donuts are made of shells.",
		Example: `
  gonut -i c2.dll
  gonut --arch x86 --class TestClass --method RunProcess --args notepad.exe --input loader.dll
  gonut -i loader.dll -c TestClass -m RunProcess -p "calc notepad" -s http://remote_server.com/modules/
  gonut -z2 -k2 -t -i loader.exe -o out.bin
`,
		Version:       "v1.0.0-3",
		SilenceErrors: true,
	}

	// -MODULE OPTIONS-
	cmd.Flags().StringVarP(&c.ModuleName, "modname", "n", "",
		"Module name for HTTP staging. If entropy is enabled, this is generated randomly.")
	cmd.Flags().StringVarP(&c.Server, "server", "s", "",
		"Server that will host the Donut module. Credentials may be provided in the following format: https://username:password@192.168.0.1/")
	cmd.Flags().Uint32VarP((*uint32)(&c.Entropy), "entropy", "e", uint32(gonut.DONUT_ENTROPY_DEFAULT),
		`Entropy:
	1=None
	2=Use random names
	3=Random names + symmetric encryption
`)

	// -PIC/SHELLCODE OPTIONS-
	cmd.Flags().IntVarP((*int)(&c.Arch), "arch", "a", int(gonut.DONUT_ARCH_X96),
		`Target architecture:
	1=x86
	2=amd64
	3=x86+amd64
`)
	cmd.Flags().StringVarP(&c.Output, "output", "o", "",
		"Output file to save loader.")
	cmd.Flags().IntVarP((*int)(&c.Format), "format", "f", int(gonut.DONUT_FORMAT_BINARY),
		`Output format:
	1=Binary
	2=Base64
	3=C
	4=Ruby
	5=Python
	6=Powershell
	7=C#
	8=Hex
	9=UUID
	10=Golang
	11=Rust
`)
	cmd.Flags().Uint32VarP(&c.OEP, "oep", "y", 0,
		"Create thread for loader and continue execution at <addr> supplied. (eg. 0x1234)")
	cmd.Flags().Uint32VarP((*uint32)(&c.ExitOpt), "exit", "x", uint32(gonut.DONUT_OPT_EXIT_THREAD),
		`Exit behaviour:
	1=Exit thread
	2=Exit process
	3=Do not exit or cleanup and block indefinitely
`)

	// -FILE OPTIONS-
	cmd.Flags().StringVarP(&c.Class, "class", "c", "",
		"Optional class name. (required for .NET DLL, format: namespace.class)")
	cmd.Flags().StringVarP(&c.Domain, "domain", "d", "",
		"AppDomain name to create for .NET assembly. If entropy is enabled, this is generated randomly.")
	cmd.Flags().StringVarP(&c.Input, "input", "i", "",
		"Input file to execute in-memory.")
	cmd.Flags().StringVarP(&c.Method, "method", "m", "",
		"Optional method or function for DLL. (a method is required for .NET DLL)")
	cmd.Flags().StringVarP(&c.Args, "args", "p", "",
		"Optional parameters/command line inside quotations for DLL method/function or EXE.")
	cmd.Flags().BoolVarP((*bool)(&c.Unicode), "unicode", "w", false,
		"Command line is passed to unmanaged DLL function in UNICODE format. (default is ANSI)")
	cmd.Flags().StringVarP(&c.Runtime, "runtime", "r", "",
		"CLR runtime version. MetaHeader used by default or v4.0.30319 if none available.")
	cmd.Flags().BoolVarP((*bool)(&c.Thread), "thread", "t", false,
		"Execute the entrypoint of an unmanaged EXE as a thread.")

	// -EXTRA-
	cmd.Flags().Uint32VarP((*uint32)(&c.GonutCompress), "compress", "z", uint32(gonut.GONUT_COMPRESS_NONE),
		`Pack/Compress file:
	1=None
	2=aPLib         [experimental]
	3=LZNT1  (RTL)  [experimental, Windows only]
	4=Xpress (RTL)  [experimental, Windows only]
	5=LZNT1         [experimental]
	6=Xpress        [experimental, recommended]
`)
	cmd.Flags().Uint32VarP((*uint32)(&c.Bypass), "bypass", "b", uint32(gonut.DONUT_BYPASS_CONTINUE),
		`Bypass AMSI/WLDP/ETW:
	1=None
	2=Abort on fail
	3=Continue on fail
`)
	cmd.Flags().Uint32VarP((*uint32)(&c.Headers), "headers", "k", uint32(gonut.DONUT_HEADERS_OVERWRITE),
		`Preserve PE headers:
	1=Overwrite
	2=Keep all
`)
	cmd.Flags().StringVarP(&c.Decoy, "decoy", "j", "",
		"Optional path of decoy module for Module Overloading.")

	// -OTHER-
	cmd.Flags().BoolVarP(&c.Verbose, "verbose", "v", false, "verbose output")

	cmd.Flags().SortFlags = false

	return cmd
}
