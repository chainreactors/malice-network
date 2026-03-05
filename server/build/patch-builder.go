package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	selfType "github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/internal/mutant"
)

type PatchBuilder struct {
	config   *clientpb.BuildConfig
	artifact *models.Artifact
	output   string
}

func NewPatchBuilder(req *clientpb.BuildConfig) *PatchBuilder {
	os.MkdirAll(configs.BuildOutputPath, 0700)
	InitTemplatePath()
	return &PatchBuilder{
		config: req,
	}
}

func (p *PatchBuilder) Generate() (*clientpb.Artifact, error) {
	if p.config.BuildName == "" {
		p.config.BuildName = codenames.GetCodename()
	}

	if p.config.ProfileName != "" && p.config.MaleficConfig == nil {
		implant, _, _, err := db.GetProfileFullConfig(p.config.ProfileName)
		if err != nil {
			return nil, fmt.Errorf("failed to get profile config: %s", err)
		}
		p.config.MaleficConfig = implant
	}

	if p.config.MaleficConfig == nil {
		return nil, fmt.Errorf("implant config (MaleficConfig) is required for patch build")
	}

	_, err := selfType.LoadProfile(p.config.MaleficConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse implant config: %s", err)
	}

	artifact, err := db.SaveArtifactFromConfig(p.config)
	if err != nil {
		return nil, fmt.Errorf("failed to save artifact: %s", err)
	}
	p.artifact = artifact
	db.UpdateBuilderStatus(p.artifact.ID, consts.BuildStatusRunning)

	return artifact.ToProtobuf([]byte{}), nil
}

func (p *PatchBuilder) Execute() error {
	transport := DetectTransport(p.config.MaleficConfig)
	logs.Log.Infof("[patch-builder] Detected transport: %s", transport)

	templatePath, err := FindTemplate(transport, p.config.Target)
	if err != nil {
		db.UpdateBuilderStatus(p.artifact.ID, consts.BuildStatusFailure)
		return fmt.Errorf("failed to find template: %w", err)
	}
	logs.Log.Infof("[patch-builder] Using template: %s", templatePath)

	templateBin, err := os.ReadFile(templatePath)
	if err != nil {
		db.UpdateBuilderStatus(p.artifact.ID, consts.BuildStatusFailure)
		return fmt.Errorf("failed to read template: %w", err)
	}

	patched, err := mutant.PatchConfig(&mutant.PatchConfigRequest{
		TemplateBin: templateBin,
		ImplantYaml: p.config.MaleficConfig,
	})
	if err != nil {
		db.UpdateBuilderStatus(p.artifact.ID, consts.BuildStatusFailure)
		return fmt.Errorf("failed to patch config: %w", err)
	}

	target, _ := consts.GetBuildTarget(p.config.Target)
	ext := ""
	if target.OS == consts.Windows {
		ext = ".exe"
	}
	outputName := fmt.Sprintf("%s%s", p.config.BuildName, ext)
	p.output = filepath.Join(configs.BuildOutputPath, outputName)

	if err := os.WriteFile(p.output, patched, 0700); err != nil {
		db.UpdateBuilderStatus(p.artifact.ID, consts.BuildStatusFailure)
		return fmt.Errorf("failed to write output: %w", err)
	}

	db.UpdateBuilderStatus(p.artifact.ID, consts.BuildStatusCompleted)
	logs.Log.Infof("[patch-builder] Patched binary written to %s (%d bytes)", p.output, len(patched))
	return nil
}

func (p *PatchBuilder) Collect() (string, string, error) {
	absPath, err := filepath.Abs(p.output)
	if err != nil {
		return "", consts.BuildStatusFailure, fmt.Errorf("failed to resolve output path: %w", err)
	}

	p.artifact.Path = absPath
	if err := db.UpdateBuilderPath(p.artifact); err != nil {
		return "", consts.BuildStatusFailure, fmt.Errorf("failed to update artifact path: %w", err)
	}

	return absPath, consts.BuildStatusCompleted, nil
}
