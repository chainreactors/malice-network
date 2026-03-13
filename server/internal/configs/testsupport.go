package configs

import (
	"path/filepath"
	"testing"

	config "github.com/gookit/config/v2"
	yamlDriver "github.com/gookit/config/v2/yaml"
)

func UseTestPaths(t testing.TB, root string) {
	t.Helper()

	oldServerRootPath := ServerRootPath
	oldContextPath := ContextPath
	oldLogPath := LogPath
	oldCertsPath := CertsPath
	oldListenerPath := ListenerPath
	oldTempPath := TempPath
	oldPluginPath := PluginPath
	oldAuditPath := AuditPath
	oldWebsitePath := WebsitePath
	oldProfilePath := ProfilePath
	oldBinPath := BinPath
	oldDatabaseFileName := databaseFileName

	ServerRootPath = root
	ContextPath = filepath.Join(root, "context")
	LogPath = filepath.Join(root, "log")
	CertsPath = filepath.Join(root, "certs")
	ListenerPath = filepath.Join(root, "listener")
	TempPath = filepath.Join(root, "temp")
	PluginPath = filepath.Join(root, "plugins")
	AuditPath = filepath.Join(root, "audit")
	WebsitePath = filepath.Join(root, "web")
	ProfilePath = filepath.Join(root, "profile")
	BinPath = filepath.Join(root, "bin")
	databaseFileName = filepath.Join(root, "malice.db")

	t.Cleanup(func() {
		ServerRootPath = oldServerRootPath
		ContextPath = oldContextPath
		LogPath = oldLogPath
		CertsPath = oldCertsPath
		ListenerPath = oldListenerPath
		TempPath = oldTempPath
		PluginPath = oldPluginPath
		AuditPath = oldAuditPath
		WebsitePath = oldWebsitePath
		ProfilePath = oldProfilePath
		BinPath = oldBinPath
		databaseFileName = oldDatabaseFileName
	})
}

func InitTestConfigRuntime(t testing.TB) {
	t.Helper()

	config.Reset()
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	})
	config.AddDriver(yamlDriver.Driver)
	t.Cleanup(config.Reset)
}
