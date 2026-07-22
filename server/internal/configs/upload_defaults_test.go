package configs

import "testing"

func TestUploadConfigUsesDefaultsWhenSectionMissing(t *testing.T) {
	configPath := writeTestFile(t, t.TempDir(), "config.yaml", "server:\n  enable: true\n")
	loadTestConfig(t, configPath)

	upload := GetUploadConfig()
	if upload.MaxChunkBytes != DefaultUploadMaxChunkBytes ||
		upload.MaxFileBytes != 20*1024*1024*1024 ||
		upload.MaxStagingBytes != 20*1024*1024*1024 ||
		upload.MinFreeDiskBytes != DefaultUploadMinFreeDiskBytes ||
		upload.MaxActivePerSession != DefaultUploadMaxActivePerSession ||
		upload.StagingTTLSeconds != 6*60*60 {
		t.Fatalf("unexpected default upload config: %#v", upload)
	}
}
