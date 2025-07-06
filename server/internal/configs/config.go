package configs

import (
	"github.com/chainreactors/files"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/gookit/config/v2"
	"os"
	"path/filepath"
)

func InitConfig() error {
	perm := os.FileMode(0700)
	err := os.MkdirAll(ServerRootPath, perm)
	if err != nil {
		return err
	}
	os.MkdirAll(ContextPath, perm)
	os.MkdirAll(CertsPath, perm)
	os.MkdirAll(TempPath, perm)
	os.MkdirAll(LogPath, perm)
	//os.MkdirAll(PluginPath, perm)
	os.MkdirAll(AuditPath, perm)
	os.MkdirAll(BinPath, perm)
	os.MkdirAll(WebsitePath, perm)
	os.MkdirAll(ListenerPath, perm)
	os.MkdirAll(BuildOutputPath, perm)
	os.MkdirAll(SourceCodePath, perm)
	return nil
}

func FindConfig(filename string) string {
	if filename == "" {
		filename = ServerConfigFileName
	}
	if fileutils.Exist(filename) {
		return filename
	} else if path := filepath.Join(files.GetExcPath(), filename); fileutils.Exist(path) {
		return path
	} else {
		return ""
	}
}

func GetCertDir() string {
	//rootDir := assets.GetRootAppDir()
	// test
	if _, err := os.Stat(CertsPath); os.IsNotExist(err) {
		err := os.MkdirAll(CertsPath, 0700)
		if err != nil {
			logs.Log.Errorf("Failed to create cert dir: %v", err)
		}
	}
	return CertsPath
}

func GetServerConfig() *ServerConfig {
	s := &ServerConfig{}
	err := config.MapStruct("server", s)
	if err != nil {
		logs.Log.Errorf("Failed to map server config %s", err)
		return nil
	}
	return s
}

func GetGithubConfig() *GithubConfig {
	g := &GithubConfig{}
	err := config.MapStruct("server.github", g)
	if err != nil {
		logs.Log.Errorf("Failed to map github config %s", err)
		return nil
	}
	return g
}

func GetNotifyConfig() *NotifyConfig {
	n := &NotifyConfig{}
	err := config.MapStruct("server.notify", n)
	if err != nil {
		logs.Log.Errorf("Failed to map notify config %s", err)
		return nil
	}
	return n
}

func GetSaasConfig() *SaasConfig {
	s := &SaasConfig{}
	err := config.MapStruct("server.saas", s)
	if err != nil {
		logs.Log.Errorf("Failed to map saas config %s", err)
		return nil
	}
	return s

}

func GetListenerConfig() *ListenerConfig {
	l := &ListenerConfig{}
	err := config.MapStruct("listeners", l)
	if err != nil {
		logs.Log.Errorf("Failed to map listener config %s", err)
		return nil
	}
	return l
}

func GetWorkDir() string {
	dir, err := os.Getwd()
	if err != nil {
		logs.Log.Errorf("Failed to get work dir %s", err)
		return ""
	}
	return dir
}

func UpdateGithubConfig(g *GithubConfig) error {
	err := config.Set("server.github", g)
	if err != nil {
		logs.Log.Errorf("Failed to update github config %s", err)
		return err
	}
	return nil
}

func UpdateNotifyConfig(n *NotifyConfig) error {
	err := config.Set("server.notify", n)
	if err != nil {
		logs.Log.Errorf("Failed to update notify config %s", err)
		return err
	}
	return nil
}

func UpdateSaasConfig(n *SaasConfig) error {
	err := config.Set("server.saas", n)
	if err != nil {
		logs.Log.Errorf("Failed to update saas config %s", err)
		return err
	}
	return nil
}
