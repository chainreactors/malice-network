package plugin

import (
	"sync"
	"testing"
)

func TestGetGlobalMalManagerConcurrentInitialization(t *testing.T) {
	oldManager := globalMalManager
	globalMalManager = &MalManager{
		embeddedPlugins:      make(map[string]*EmbedPlugin),
		embeddedLevelPlugins: make(map[MalLevel][]*EmbedPlugin),
		loadedLevelCommands:  make(map[MalLevel]Commands),
		externalPlugins:      make(map[string]Plugin),
		globalPlugins:        make([]*DefaultPlugin, 0),
		loadedCommands:       make(Commands),
	}
	t.Cleanup(func() {
		if globalMalManager.luaVMPool != nil {
			globalMalManager.luaVMPool.Destroy()
		}
		globalMalManager = oldManager
	})

	const goroutines = 16
	start := make(chan struct{})
	results := make(chan *MalManager, goroutines)
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- GetGlobalMalManager()
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	for manager := range results {
		if manager != globalMalManager {
			t.Fatal("GetGlobalMalManager returned different instances")
		}
	}
	if !globalMalManager.initialized {
		t.Fatal("global manager was not initialized")
	}
}
