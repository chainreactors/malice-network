package build

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/wabzsy/gonut"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func NewMaleficSRDIArtifact(name, typ, src, platform, arch, stage, funcName, dataPath string) (*models.Artifact, []byte, error) {
	builder, err := db.SaveArtifact(name, typ, platform, arch, stage, consts.CommandArtifactUpload)
	if err != nil {
		return nil, nil, err
	}
	bin, err := gonut.DonutShellcodeFromFile(builder.Path, arch, "")
	if err != nil {
		return nil, nil, err
	}
	return builder, bin, nil
}

// for pulse
func OBJCOPYPulse(builder *models.Artifact, platform, arch string) ([]byte, error) {
	absBuildOutputPath, err := filepath.Abs(configs.BuildOutputPath)
	if err != nil {
		return nil, err
	}
	dstPath := filepath.Join(absBuildOutputPath, encoders.UUID())
	cmd := exec.Command("objcopy", "-O", "binary", builder.Path, dstPath)
	cmd.Dir = sourcePath
	output, err := cmd.CombinedOutput()
	logs.Log.Debugf("Objcopy output: %s", output)
	if err != nil {
		return nil, err
	}
	bin, err := os.ReadFile(dstPath)
	if err != nil {
		return nil, err
	}
	return bin, nil
}

func SRDIArtifact(builder *models.Artifact, platform, arch string) ([]byte, error) {
	if !strings.Contains(builder.Target, consts.Windows) {
		return []byte{}, errs.ErrPlartFormNotSupport
	}
	exePath := builder.Path
	if !strings.HasSuffix(exePath, ".exe") {
		exePath = builder.Path + ".exe"
		err := fileutils.CopyFile(builder.Path, exePath)
		if err != nil {
			return nil, fmt.Errorf("copy to .exe failed: %w", err)
		}
	}
	bin, err := gonut.DonutShellcodeFromFile(exePath, arch, "")
	if err != nil {
		return []byte{}, err
	}
	return bin, nil
}
