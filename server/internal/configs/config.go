package configs

import (
	"github.com/chainreactors/logs"
	"github.com/gookit/config/v2"
	"os"
)

func InitConfig() error {
	perm := os.FileMode(0700)
	err := os.MkdirAll(ServerRootPath, perm)
	if err != nil {
		return err
	}
	os.MkdirAll(LogPath, perm)
	os.MkdirAll(CertsPath, perm)
	os.MkdirAll(TempPath, perm)
	//os.MkdirAll(PluginPath, perm)
	os.MkdirAll(AuditPath, perm)
	os.MkdirAll(CachePath, perm)
	os.MkdirAll(WebsitePath, perm)
	os.MkdirAll(ListenerPath, perm)
	os.MkdirAll(BuildPath, perm)
	return nil
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

func GetListenerConfig() *ListenerConfig {
	l := &ListenerConfig{}
	err := config.MapStruct("listeners", l)
	if err != nil {
		logs.Log.Errorf("Failed to map listener config %s", err)
		return nil
	}
	return l
}
