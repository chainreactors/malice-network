package gonut

type Config struct {
	Len  uint32 // original length of input file
	ZLen uint32 // compressed length of input file

	// general / misc options for loader

	Arch     ArchType             // target architecture
	Bypass   BypassType           // bypass option for AMSI/WLDP
	Headers  HeadersType          // preserve PE headers option
	Compress DonutCompressionType // engine to use when compressing file via RtlCompressBuffer
	Entropy  EntropyType          // entropy/encryption level
	Format   FormatType           // output format for loader
	ExitOpt  ExitType             // return to caller, invoke RtlExitUserProcess to terminate the host process, or block indefinitely
	Thread   BoolType             // run entrypoint of unmanaged EXE as a thread. attempts to intercept calls to exit-related API
	OEP      uint32               // original entrypoint of target host file

	// files in/out
	Input    string // name of input file to read and load in-memory
	InputBin []byte
	Output   string // name of output file to save loader

	// .NET stuff
	Runtime string // runtime version to use for CLR
	Domain  string // name of domain to create for .NET DLL/EXE
	Class   string // name of class with optional namespace for .NET DLL
	Method  string // name of method or DLL function to invoke for .NET DLL and unmanaged DLL

	// command line for DLL/EXE
	Args    string   // command line to use for unmanaged DLL/EXE and .NET DLL/EXE
	Unicode BoolType // param is passed to DLL function without converting to unicode

	// module overloading stuff
	Decoy string // path of decoy module

	// HTTP/DNS staging information
	Server     string // points to root path of where module will be stored on remote HTTP server or DNS server
	Auth       string // username and password for web server
	ModuleName string // name of module written to disk for http stager

	// DONUT_MODULE
	ModuleType ModuleType

	// DONUT_INSTANCE
	InstanceType InstanceType

	Verbose bool //  verbose output

	// Gonut only
	GonutCompress CompressionType // Gonut compression engine
}

func DefaultConfig() *Config {
	return &Config{
		Arch:          DONUT_ARCH_X96,
		Bypass:        DONUT_BYPASS_CONTINUE,
		Headers:       DONUT_HEADERS_OVERWRITE,
		Format:        DONUT_FORMAT_BINARY,
		GonutCompress: GONUT_COMPRESS_NONE,
		Entropy:       DONUT_ENTROPY_DEFAULT,
		ExitOpt:       DONUT_OPT_EXIT_PROCESS,
	}
}
