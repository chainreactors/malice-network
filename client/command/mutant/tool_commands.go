package mutant

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func parseToolMapping(raw, defaultRemote string) (toolInput, error) {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) == 1 {
		return toolInput{Remote: defaultRemote, Local: raw}, nil
	}
	if parts[0] == "" || parts[1] == "" {
		return toolInput{}, fmt.Errorf("invalid mapping %q, expected remote:local", raw)
	}
	return toolInput{Remote: parts[0], Local: parts[1]}, nil
}

func parseOutputMapping(raw, defaultRemote string) (toolOutput, error) {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) == 1 {
		return toolOutput{Remote: defaultRemote, Local: raw}, nil
	}
	if parts[0] == "" || parts[1] == "" {
		return toolOutput{}, fmt.Errorf("invalid mapping %q, expected remote:local", raw)
	}
	return toolOutput{Remote: parts[0], Local: parts[1]}, nil
}

func MutantRunCmd(cmd *cobra.Command, con *core.Console) error {
	inputMappings, _ := cmd.Flags().GetStringArray("input-file")
	outputMappings, _ := cmd.Flags().GetStringArray("output-file")
	if len(cmd.Flags().Args()) == 0 {
		return fmt.Errorf("tool args are required after --")
	}

	inputs := make([]toolInput, 0, len(inputMappings))
	for i, raw := range inputMappings {
		input, err := parseToolMapping(raw, fmt.Sprintf("input-%d", i))
		if err != nil {
			return err
		}
		inputs = append(inputs, input)
	}

	outputs := make([]toolOutput, 0, len(outputMappings))
	for i, raw := range outputMappings {
		output, err := parseOutputMapping(raw, fmt.Sprintf("output-%d", i))
		if err != nil {
			return err
		}
		outputs = append(outputs, output)
	}

	resp, err := callMutantTool(con, cmd.Flags().Args(), inputs, outputs, getTimeout(cmd))
	if err != nil {
		return err
	}
	printToolStdout(con, resp)
	return writeToolOutputs(resp, outputs)
}

func PatchCmd(cmd *cobra.Command, con *core.Console) error {
	input, _ := cmd.Flags().GetString("input")
	config, _ := cmd.Flags().GetString("config")
	fromImplant, _ := cmd.Flags().GetString("from-implant")
	output, _ := cmd.Flags().GetString("output")
	obfSeed, _ := cmd.Flags().GetUint64("obf-seed")
	sets, _ := cmd.Flags().GetStringArray("set")
	if input == "" {
		return fmt.Errorf("input file is required")
	}
	if config == "" && fromImplant == "" {
		return fmt.Errorf("config or from-implant is required")
	}
	if output == "" {
		output = defaultOutputPath(input, ".patched")
	}

	remoteInput := remoteFileName(input, "input.bin")
	remoteOutput := remoteFileName(output, "patched.bin")
	args := []string{"patch", "-i", remoteInput, "-o", remoteOutput}
	inputs := []toolInput{{Remote: remoteInput, Local: input}}
	if fromImplant != "" {
		args = append(args, "--from-implant", "implant.yaml")
		inputs = append(inputs, toolInput{Remote: "implant.yaml", Local: fromImplant})
	}
	if config != "" {
		args = append(args, "-c", "runtime-config")
		inputs = append(inputs, toolInput{Remote: "runtime-config", Local: config})
	}
	if obfSeed != 0 {
		args = append(args, "--obf-seed", strconv.FormatUint(obfSeed, 10))
	}
	for _, set := range sets {
		args = append(args, "--set", set)
	}

	resp, err := callMutantTool(con, args, inputs, []toolOutput{{Remote: remoteOutput, Local: output}}, getTimeout(cmd))
	if err != nil {
		return err
	}
	printToolStdout(con, resp)
	return writeToolOutputs(resp, []toolOutput{{Remote: remoteOutput, Local: output}}, remoteOutput)
}

func ObjcopyCmd(cmd *cobra.Command, con *core.Console) error {
	input, _ := cmd.Flags().GetString("input")
	output, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")
	if input == "" || output == "" {
		return fmt.Errorf("input and output are required")
	}
	remoteInput := remoteFileName(input, "input.bin")
	remoteOutput := remoteFileName(output, "output.bin")
	args := []string{"objcopy", "-f", format, "-i", remoteInput, "-o", remoteOutput}
	resp, err := callMutantTool(con, args, []toolInput{{Remote: remoteInput, Local: input}}, []toolOutput{{Remote: remoteOutput, Local: output}}, getTimeout(cmd))
	if err != nil {
		return err
	}
	printToolStdout(con, resp)
	return writeToolOutputs(resp, []toolOutput{{Remote: remoteOutput, Local: output}}, remoteOutput)
}

func EncodeCmd(cmd *cobra.Command, con *core.Console) error {
	list, _ := cmd.Flags().GetBool("list")
	if list {
		resp, err := callMutantTool(con, []string{"encode", "--list"}, nil, nil, getTimeout(cmd))
		if err != nil {
			return err
		}
		printToolStdout(con, resp)
		return nil
	}
	input, _ := cmd.Flags().GetString("input")
	output, _ := cmd.Flags().GetString("output")
	encoding, _ := cmd.Flags().GetString("encoding")
	format, _ := cmd.Flags().GetString("format")
	if input == "" {
		return fmt.Errorf("input file is required")
	}
	if output == "" {
		output = defaultOutputPath(input, ".encoded")
	}

	remoteInput := remoteFileName(input, "payload.bin")
	remoteOutput := remoteFileName(output, "encoded")
	args := []string{"encode", "-i", remoteInput, "-e", encoding, "-f", format, "-o", remoteOutput}
	outputs := []toolOutput{{Remote: remoteOutput, Local: output}}
	if format == "all" {
		stem := strings.TrimSuffix(remoteOutput, filepath.Ext(remoteOutput))
		localStem := strings.TrimSuffix(output, filepath.Ext(output))
		outputs = []toolOutput{
			{Remote: stem + ".bin", Local: localStem + ".bin"},
			{Remote: stem + ".key", Local: localStem + ".key"},
			{Remote: stem + ".extra", Local: localStem + ".extra"},
			{Remote: stem + ".c", Local: localStem + ".c"},
			{Remote: stem + ".rs", Local: localStem + ".rs"},
		}
	}
	resp, err := callMutantTool(con, args, []toolInput{{Remote: remoteInput, Local: input}}, outputs, getTimeout(cmd))
	if err != nil {
		return err
	}
	printToolStdout(con, resp)
	return writeToolOutputs(resp, outputs)
}

func EntropyCmd(cmd *cobra.Command, con *core.Console) error {
	input, _ := cmd.Flags().GetString("input")
	output, _ := cmd.Flags().GetString("output")
	threshold, _ := cmd.Flags().GetFloat64("threshold")
	strategy, _ := cmd.Flags().GetString("strategy")
	maxGrowth, _ := cmd.Flags().GetFloat64("max-growth")
	measureOnly, _ := cmd.Flags().GetBool("measure-only")
	if input == "" {
		return fmt.Errorf("input file is required")
	}
	remoteInput := remoteFileName(input, "input.bin")
	args := []string{
		"entropy", "-i", remoteInput,
		"-t", strconv.FormatFloat(threshold, 'f', -1, 64),
		"-s", strategy,
		"--max-growth", strconv.FormatFloat(maxGrowth, 'f', -1, 64),
	}
	var outputs []toolOutput
	var required []string
	if measureOnly {
		args = append(args, "--measure-only")
	} else {
		if output == "" {
			output = defaultOutputPath(input, ".entropy")
		}
		remoteOutput := remoteFileName(output, "entropy-output.bin")
		args = append(args, "-o", remoteOutput)
		outputs = []toolOutput{{Remote: remoteOutput, Local: output}}
		required = []string{remoteOutput}
	}
	resp, err := callMutantTool(con, args, []toolInput{{Remote: remoteInput, Local: input}}, outputs, getTimeout(cmd))
	if err != nil {
		return err
	}
	printToolStdout(con, resp)
	return writeToolOutputs(resp, outputs, required...)
}

func BinderCmd(cmd *cobra.Command, con *core.Console) error {
	switch cmd.CalledAs() {
	case "bind":
		primary, _ := cmd.Flags().GetString("primary")
		secondary, _ := cmd.Flags().GetString("secondary")
		output, _ := cmd.Flags().GetString("output")
		if primary == "" || secondary == "" || output == "" {
			return fmt.Errorf("primary, secondary, and output are required")
		}
		remoteOutput := remoteFileName(output, "bound.exe")
		args := []string{"binder", "bind", "-p", "primary.exe", "-s", "secondary.exe", "-o", remoteOutput}
		inputs := []toolInput{{Remote: "primary.exe", Local: primary}, {Remote: "secondary.exe", Local: secondary}}
		outputs := []toolOutput{{Remote: remoteOutput, Local: output}}
		resp, err := callMutantTool(con, args, inputs, outputs, getTimeout(cmd))
		if err != nil {
			return err
		}
		printToolStdout(con, resp)
		return writeToolOutputs(resp, outputs, remoteOutput)
	case "extract":
		input, _ := cmd.Flags().GetString("input")
		output, _ := cmd.Flags().GetString("output")
		if input == "" || output == "" {
			return fmt.Errorf("input and output are required")
		}
		remoteOutput := remoteFileName(output, "extracted.exe")
		args := []string{"binder", "extract", "-i", "bound.exe", "-o", remoteOutput}
		outputs := []toolOutput{{Remote: remoteOutput, Local: output}}
		resp, err := callMutantTool(con, args, []toolInput{{Remote: "bound.exe", Local: input}}, outputs, getTimeout(cmd))
		if err != nil {
			return err
		}
		printToolStdout(con, resp)
		return writeToolOutputs(resp, outputs, remoteOutput)
	case "check":
		input, _ := cmd.Flags().GetString("input")
		if input == "" {
			return fmt.Errorf("input file is required")
		}
		resp, err := callMutantTool(con, []string{"binder", "check", "-i", "input.exe"}, []toolInput{{Remote: "input.exe", Local: input}}, nil, getTimeout(cmd))
		if err != nil {
			return err
		}
		printToolStdout(con, resp)
		return nil
	default:
		return cmd.Help()
	}
}

func IconCmd(cmd *cobra.Command, con *core.Console) error {
	switch cmd.CalledAs() {
	case "replace":
		input, _ := cmd.Flags().GetString("input")
		ico, _ := cmd.Flags().GetString("ico")
		output, _ := cmd.Flags().GetString("output")
		if input == "" || ico == "" || output == "" {
			return fmt.Errorf("input, ico, and output are required")
		}
		remoteOutput := remoteFileName(output, "icon-output.exe")
		args := []string{"icon", "replace", "-i", "input.exe", "--ico", "icon.ico", "-o", remoteOutput}
		inputs := []toolInput{{Remote: "input.exe", Local: input}, {Remote: "icon.ico", Local: ico}}
		outputs := []toolOutput{{Remote: remoteOutput, Local: output}}
		resp, err := callMutantTool(con, args, inputs, outputs, getTimeout(cmd))
		if err != nil {
			return err
		}
		printToolStdout(con, resp)
		return writeToolOutputs(resp, outputs, remoteOutput)
	case "extract":
		input, _ := cmd.Flags().GetString("input")
		output, _ := cmd.Flags().GetString("output")
		if input == "" || output == "" {
			return fmt.Errorf("input and output are required")
		}
		remoteOutput := remoteFileName(output, "icon.ico")
		args := []string{"icon", "extract", "-i", "input.exe", "-o", remoteOutput}
		outputs := []toolOutput{{Remote: remoteOutput, Local: output}}
		resp, err := callMutantTool(con, args, []toolInput{{Remote: "input.exe", Local: input}}, outputs, getTimeout(cmd))
		if err != nil {
			return err
		}
		printToolStdout(con, resp)
		return writeToolOutputs(resp, outputs, remoteOutput)
	default:
		return cmd.Help()
	}
}

func MutateCmd(cmd *cobra.Command, con *core.Console) error {
	list, _ := cmd.Flags().GetBool("list")
	if list {
		resp, err := callMutantTool(con, []string{"mutate", "--list"}, nil, nil, getTimeout(cmd))
		if err != nil {
			return err
		}
		printToolStdout(con, resp)
		return nil
	}
	input, _ := cmd.Flags().GetString("input")
	config, _ := cmd.Flags().GetString("config")
	output, _ := cmd.Flags().GetString("output")
	outDir, _ := cmd.Flags().GetString("out-dir")
	count, _ := cmd.Flags().GetUint("count")
	encoding, _ := cmd.Flags().GetString("encoding")
	format, _ := cmd.Flags().GetString("format")
	carrier, _ := cmd.Flags().GetString("carrier")
	technique, _ := cmd.Flags().GetString("technique")
	noRelink, _ := cmd.Flags().GetBool("no-relink")
	stubPoly, _ := cmd.Flags().GetBool("stub-poly")
	stubEncrypt, _ := cmd.Flags().GetBool("stub-encrypt")
	addSection, _ := cmd.Flags().GetBool("add-section")
	block, _ := cmd.Flags().GetBool("block")
	blockMethod, _ := cmd.Flags().GetString("block-method")
	srdi, _ := cmd.Flags().GetString("srdi")
	srdiArch, _ := cmd.Flags().GetString("srdi-arch")
	srdiFunction, _ := cmd.Flags().GetString("srdi-function")
	srdiUserdata, _ := cmd.Flags().GetString("srdi-userdata")
	lnkMethod, _ := cmd.Flags().GetString("lnk-method")
	lnkTarget, _ := cmd.Flags().GetString("lnk-target")
	lnkIcon, _ := cmd.Flags().GetString("lnk-icon")
	if input == "" {
		return fmt.Errorf("input file is required")
	}
	if output != "" && outDir != "" {
		return fmt.Errorf("use either output or out-dir, not both")
	}

	args := []string{"mutate", "-i", "input.bin"}
	inputs := []toolInput{{Remote: "input.bin", Local: input}}
	args = appendFlag(args, "-e", encoding)
	args = appendFlag(args, "-f", format)
	args = appendFlag(args, "--technique", technique)
	args = appendFlag(args, "--block-method", blockMethod)
	args = appendFlag(args, "--srdi", srdi)
	args = appendFlag(args, "--srdi-arch", srdiArch)
	args = appendFlag(args, "--srdi-function", srdiFunction)
	args = appendFlag(args, "--lnk-method", lnkMethod)
	args = appendFlag(args, "--lnk-target", lnkTarget)
	args = appendBoolFlag(args, "--no-relink", noRelink)
	args = appendBoolFlag(args, "--stub-poly", stubPoly)
	args = appendBoolFlag(args, "--stub-encrypt", stubEncrypt)
	args = appendBoolFlag(args, "--add-section", addSection)
	args = appendBoolFlag(args, "--block", block)
	if count > 0 {
		args = append(args, "-n", strconv.FormatUint(uint64(count), 10))
	}
	if config != "" {
		args = append(args, "-c", "implant.yaml")
		inputs = append(inputs, toolInput{Remote: "implant.yaml", Local: config})
	}
	if carrier != "" {
		args = append(args, "--carrier", "carrier.exe")
		inputs = append(inputs, toolInput{Remote: "carrier.exe", Local: carrier})
	}
	if srdiUserdata != "" {
		args = append(args, "--srdi-userdata", "srdi-userdata.bin")
		inputs = append(inputs, toolInput{Remote: "srdi-userdata.bin", Local: srdiUserdata})
	}
	if lnkIcon != "" {
		args = append(args, "--lnk-icon", "lnk-icon.ico")
		inputs = append(inputs, toolInput{Remote: "lnk-icon.ico", Local: lnkIcon})
	}

	var outputs []toolOutput
	var required []string
	if output != "" {
		remoteOutput := remoteFileName(output, "mutated.bin")
		args = append(args, "-o", remoteOutput)
		outputs = []toolOutput{{Remote: remoteOutput, Local: output}}
		required = []string{remoteOutput}
	} else {
		if outDir == "" {
			outDir = filepath.Join(assets.GetTempDir(), "mutate")
		}
		args = append(args, "--out-dir", "out")
		outputs = []toolOutput{{Remote: "out", Local: outDir}}
	}

	resp, err := callMutantTool(con, args, inputs, outputs, getTimeout(cmd))
	if err != nil {
		return err
	}
	printToolStdout(con, resp)
	return writeToolOutputs(resp, outputs, required...)
}

func BdfCmd(cmd *cobra.Command, con *core.Console) error {
	input, _ := cmd.Flags().GetString("input")
	payload, _ := cmd.Flags().GetString("payload")
	output, _ := cmd.Flags().GetString("output")
	if input == "" {
		return fmt.Errorf("input file is required")
	}
	args := []string{"bdf", "-i", "input.exe"}
	inputs := []toolInput{{Remote: "input.exe", Local: input}}
	args = appendBoolFlag(args, "--find-caves", mustBool(cmd, "find-caves"))
	args = appendBoolFlag(args, "--add-section", mustBool(cmd, "add-section"))
	args = appendBoolFlag(args, "--marker", mustBool(cmd, "marker"))
	args = appendBoolFlag(args, "--aslr", mustBool(cmd, "aslr"))
	args = appendBoolFlag(args, "--no-zero-cert", mustBool(cmd, "no-zero-cert"))
	args = appendBoolFlag(args, "--stub-poly", mustBool(cmd, "stub-poly"))
	args = appendBoolFlag(args, "--stub-encrypt", mustBool(cmd, "stub-encrypt"))
	args = appendBoolFlag(args, "--block", mustBool(cmd, "block"))
	args = appendFlag(args, "--section-name", mustString(cmd, "section-name"))
	args = appendFlag(args, "--min-cave", mustString(cmd, "min-cave"))
	args = appendFlag(args, "--wait", mustString(cmd, "wait"))
	args = appendFlag(args, "-t", mustString(cmd, "technique"))
	args = appendFlag(args, "--stub-hash", mustString(cmd, "stub-hash"))
	args = appendFlag(args, "--evasion", mustString(cmd, "evasion"))
	args = appendFlag(args, "--seed", mustString(cmd, "seed"))
	args = appendFlag(args, "--block-method", mustString(cmd, "block-method"))
	if payload != "" {
		args = append(args, "-p", "payload.bin")
		inputs = append(inputs, toolInput{Remote: "payload.bin", Local: payload})
	}
	var outputs []toolOutput
	var required []string
	if output != "" {
		args = append(args, "-o", "bdf-output.exe")
		outputs = []toolOutput{{Remote: "bdf-output.exe", Local: output}}
		required = []string{"bdf-output.exe"}
	}
	resp, err := callMutantTool(con, args, inputs, outputs, getTimeout(cmd))
	if err != nil {
		return err
	}
	printToolStdout(con, resp)
	return writeToolOutputs(resp, outputs, required...)
}

func WatermarkCmd(cmd *cobra.Command, con *core.Console) error {
	switch cmd.CalledAs() {
	case "write":
		input, _ := cmd.Flags().GetString("input")
		output, _ := cmd.Flags().GetString("output")
		method, _ := cmd.Flags().GetString("method")
		watermark, _ := cmd.Flags().GetString("watermark")
		if input == "" || output == "" || method == "" || watermark == "" {
			return fmt.Errorf("input, output, method, and watermark are required")
		}
		args := []string{"watermark", "write", "-i", "input.exe", "-o", "marked.exe", "-m", method, "-w", watermark}
		outputs := []toolOutput{{Remote: "marked.exe", Local: output}}
		resp, err := callMutantTool(con, args, []toolInput{{Remote: "input.exe", Local: input}}, outputs, getTimeout(cmd))
		if err != nil {
			return err
		}
		printToolStdout(con, resp)
		return writeToolOutputs(resp, outputs, "marked.exe")
	case "read":
		input, _ := cmd.Flags().GetString("input")
		method, _ := cmd.Flags().GetString("method")
		size, _ := cmd.Flags().GetString("size")
		if input == "" || method == "" {
			return fmt.Errorf("input and method are required")
		}
		args := []string{"watermark", "read", "-i", "input.exe", "-m", method}
		args = appendFlag(args, "-s", size)
		resp, err := callMutantTool(con, args, []toolInput{{Remote: "input.exe", Local: input}}, nil, getTimeout(cmd))
		if err != nil {
			return err
		}
		printToolStdout(con, resp)
		return nil
	default:
		return cmd.Help()
	}
}

func LnkCmd(cmd *cobra.Command, con *core.Console) error {
	switch cmd.CalledAs() {
	case "exec":
		output, _ := cmd.Flags().GetString("output")
		command, _ := cmd.Flags().GetString("command")
		if output == "" {
			return fmt.Errorf("output is required")
		}
		if command == "" {
			return fmt.Errorf("command is required")
		}
		inputs := []toolInput{}
		args := []string{"lnk", "exec", "-c", command, "-o", "output.lnk"}
		args = appendFlag(args, "-t", mustString(cmd, "target"))
		if icon := mustString(cmd, "icon"); icon != "" {
			args = append(args, "--icon", "icon.ico")
			inputs = append(inputs, toolInput{Remote: "icon.ico", Local: icon})
		}
		args = appendFlag(args, "--icon-index", mustString(cmd, "icon-index"))
		args = appendFlag(args, "--name", mustString(cmd, "name"))
		args = appendFlag(args, "--padding", mustString(cmd, "padding"))
		outputs := []toolOutput{{Remote: "output.lnk", Local: output}}
		resp, err := callMutantTool(con, args, inputs, outputs, getTimeout(cmd))
		if err != nil {
			return err
		}
		printToolStdout(con, resp)
		return writeToolOutputs(resp, outputs, "output.lnk")
	case "embed":
		input, _ := cmd.Flags().GetString("input")
		output, _ := cmd.Flags().GetString("output")
		if input == "" || output == "" {
			return fmt.Errorf("input and output are required")
		}
		args := []string{"lnk", "embed", "-i", "payload.bin", "-o", "output.lnk"}
		args = appendFlag(args, "-t", mustString(cmd, "target"))
		args = appendFlag(args, "-c", mustString(cmd, "command"))
		args = appendFlag(args, "--method", mustString(cmd, "method"))
		args = appendFlag(args, "--xor-key", mustString(cmd, "xor-key"))
		args = appendFlag(args, "--padding", mustString(cmd, "padding"))
		inputs := []toolInput{{Remote: "payload.bin", Local: input}}
		if loader := mustString(cmd, "loader"); loader != "" {
			args = append(args, "--loader", "loader.exe")
			inputs = append(inputs, toolInput{Remote: "loader.exe", Local: loader})
		}
		if icon := mustString(cmd, "icon"); icon != "" {
			args = append(args, "--icon", "icon.ico")
			inputs = append(inputs, toolInput{Remote: "icon.ico", Local: icon})
		}
		outputs := []toolOutput{{Remote: "output.lnk", Local: output}}
		resp, err := callMutantTool(con, args, inputs, outputs, getTimeout(cmd))
		if err != nil {
			return err
		}
		printToolStdout(con, resp)
		return writeToolOutputs(resp, outputs, "output.lnk")
	default:
		return cmd.Help()
	}
}

func RelinkCmd(cmd *cobra.Command, con *core.Console) error {
	input, _ := cmd.Flags().GetString("input")
	output, _ := cmd.Flags().GetString("output")
	if input == "" || output == "" {
		return fmt.Errorf("input and output are required")
	}
	args := []string{"relink", "-i", "input.exe", "-o", "output.exe"}
	args = appendFlag(args, "--seed", mustString(cmd, "seed"))
	args = appendFlag(args, "--skip", mustString(cmd, "skip"))
	args = appendFlag(args, "--only", mustString(cmd, "only"))
	args = appendFlag(args, "--padding-size", mustString(cmd, "padding-size"))
	args = appendBoolFlag(args, "--dry-run", mustBool(cmd, "dry-run"))
	outputs := []toolOutput{{Remote: "output.exe", Local: output}}
	resp, err := callMutantTool(con, args, []toolInput{{Remote: "input.exe", Local: input}}, outputs, getTimeout(cmd))
	if err != nil {
		return err
	}
	printToolStdout(con, resp)
	if mustBool(cmd, "dry-run") {
		return nil
	}
	return writeToolOutputs(resp, outputs, "output.exe")
}

func HijackCmd(cmd *cobra.Command, con *core.Console) error {
	input, _ := cmd.Flags().GetString("input")
	payload, _ := cmd.Flags().GetString("payload")
	output, _ := cmd.Flags().GetString("output")
	if input == "" {
		return fmt.Errorf("input file is required")
	}
	args := []string{"hijack", "-i", "input.dll"}
	inputs := []toolInput{{Remote: "input.dll", Local: input}}
	args = appendBoolFlag(args, "--analyze", mustBool(cmd, "analyze"))
	args = appendBoolFlag(args, "--verify", mustBool(cmd, "verify"))
	args = appendFlag(args, "--traces", mustString(cmd, "traces"))
	args = appendFlag(args, "--min-cave", mustString(cmd, "min-cave"))
	args = appendFlag(args, "--call-index", mustString(cmd, "call-index"))
	args = appendFlag(args, "--cave-index", mustString(cmd, "cave-index"))
	if payload != "" {
		args = append(args, "-p", "payload.bin")
		inputs = append(inputs, toolInput{Remote: "payload.bin", Local: payload})
	}
	var outputs []toolOutput
	var required []string
	if output != "" {
		args = append(args, "-o", "output.dll")
		outputs = []toolOutput{{Remote: "output.dll", Local: output}}
		required = []string{"output.dll"}
	}
	resp, err := callMutantTool(con, args, inputs, outputs, getTimeout(cmd))
	if err != nil {
		return err
	}
	printToolStdout(con, resp)
	return writeToolOutputs(resp, outputs, required...)
}

func mustString(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)
	return value
}

func mustBool(cmd *cobra.Command, name string) bool {
	value, _ := cmd.Flags().GetBool(name)
	return value
}
