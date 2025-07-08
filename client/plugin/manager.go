package plugin

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/intl"
	"github.com/chainreactors/mals/m"
)

var (
	globalMalManager *MalManager
	managerOnce      sync.Once
)

// MalManager 统一的mal插件管理器，分别管理嵌入式和外部插件
type MalManager struct {
	mu              sync.RWMutex
	embeddedPlugins map[string]*EmbedPlugin // 嵌入式插件
	externalPlugins map[string]Plugin       // 外部插件
	globalPlugins   []*DefaultPlugin        // 全局库插件（Lib: true的插件）
	loadedCommands  Commands                // 所有已加载的嵌入式命令
	initialized     bool                    // 是否已初始化
}

// GetGlobalMalManager 获取全局mal管理器（单例）
func GetGlobalMalManager() *MalManager {
	managerOnce.Do(func() {
		globalMalManager = &MalManager{
			embeddedPlugins: make(map[string]*EmbedPlugin),
			externalPlugins: make(map[string]Plugin),
			globalPlugins:   make([]*DefaultPlugin, 0),
			loadedCommands:  make(Commands),
		}
		// 初始化时加载所有插件
		globalMalManager.initialize()
	})
	return globalMalManager
}

// initialize 初始化管理器，加载所有插件
func (mm *MalManager) initialize() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.initialized {
		return
	}

	mm.loadEmbeddedMals()
	mm.loadExternalMals()

	mm.initialized = true
	logs.Log.Infof("MalManager initialized with %d embedded and %d external plugins, %d global plugins\n",
		len(mm.embeddedPlugins), len(mm.externalPlugins), len(mm.globalPlugins))
}

// loadEmbeddedMals 直接加载嵌入式mal插件
func (mm *MalManager) loadEmbeddedMals() {
	// 按优先级顺序加载每个级别的mal包
	for _, level := range levelOrder {
		levelName := level.String()

		// 检查mal.yaml是否存在
		manifestPath := levelName + "/mal.yaml"
		if _, err := intl.UnifiedFS.ReadFile(manifestPath); err != nil {
			logs.Log.Debugf("No mal.yaml found for level %s, skipping\n", levelName)
			continue
		}

		// 创建嵌入式插件
		embedPlugin, err := NewEmbedPlugin(levelName, levelName, level)
		if err != nil {
			logs.Log.Errorf("Failed to create embedded plugin %s: %v\n", levelName, err)
			continue
		}

		// 运行插件
		if err := embedPlugin.Run(); err != nil {
			logs.Log.Errorf("Failed to run embedded plugin %s: %v\n", levelName, err)
			continue
		}

		mm.embeddedPlugins[levelName] = embedPlugin
		logs.Log.Infof("Loaded embedded plugin: %s (level: %d)\n", levelName, level)
	}

	for _, levelName := range levelOrder {
		if plugin, exists := mm.embeddedPlugins[levelName.String()]; exists {
			// 添加插件的命令到Commands中，后加载的会覆盖先加载的
			for _, cmd := range plugin.Commands() {
				cmdName := cmd.Command.Name()
				mm.loadedCommands.SetCommand(cmdName, cmd.Command)
				logs.Log.Debugf("Added/Updated embedded command '%s' from %s\n", cmdName, levelName)
			}
		}
	}
}

func (mm *MalManager) loadExternalMals() {
	mm.globalPlugins = LoadGlobalLuaPlugin()

	for _, manifest := range GetPluginManifest() {
		_, err := mm.LoadExternalMal(manifest)
		if err != nil {
			logs.Log.Errorf("Failed to load external mal %s: %v\n", manifest.Name, err)
			continue
		}
	}
}

// LoadExternalMal 加载单个外部mal插件
func (mm *MalManager) LoadExternalMal(manifest *MalManiFest) (Plugin, error) {
	// 检查是否已加载
	if _, exists := mm.externalPlugins[manifest.Name]; exists {
		return nil, fmt.Errorf("external mal %s already loaded\n", manifest.Name)
	}

	var plugin Plugin
	var err error

	switch manifest.Type {
	case LuaScript:
		plugin, err = NewLuaMalPlugin(manifest)
	//case GoPlugin:
	//	plugin, err = NewGoMalPlugin(manifest)
	default:
		return nil, fmt.Errorf("not found valid script type: %s\n", manifest.Type)
	}

	if err != nil {
		return nil, err
	}

	err = plugin.Run()
	if err != nil {
		return nil, err
	}

	mm.externalPlugins[manifest.Name] = plugin
	logs.Log.Infof("Loaded external plugin: %s\n", manifest.Name)

	return plugin, nil
}

// GetEmbedPlugin 获取指定名称的嵌入式插件
func (mm *MalManager) GetEmbedPlugin(name string) (*EmbedPlugin, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	plugin, exists := mm.embeddedPlugins[name]
	return plugin, exists
}

// GetExternalPlugin 获取指定名称的外部插件
func (mm *MalManager) GetExternalPlugin(name string) (Plugin, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	plugin, exists := mm.externalPlugins[name]
	return plugin, exists
}

// GetPlugin 获取指定名称的插件（先查找外部，再查找嵌入式）
func (mm *MalManager) GetPlugin(name string) (Plugin, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// 先查找外部插件
	if plugin, exists := mm.externalPlugins[name]; exists {
		return plugin, true
	}

	// 再查找嵌入式插件
	if plugin, exists := mm.embeddedPlugins[name]; exists {
		return plugin, true
	}

	return nil, false
}

// GetAllEmbeddedPlugins 获取所有嵌入式插件
func (mm *MalManager) GetAllEmbeddedPlugins() map[string]*EmbedPlugin {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	result := make(map[string]*EmbedPlugin, len(mm.embeddedPlugins))
	for name, plugin := range mm.embeddedPlugins {
		result[name] = plugin
	}
	return result
}

// GetAllExternalPlugins 获取所有外部插件
func (mm *MalManager) GetAllExternalPlugins() map[string]Plugin {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	result := make(map[string]Plugin, len(mm.externalPlugins))
	for name, plugin := range mm.externalPlugins {
		result[name] = plugin
	}
	return result
}

// GetPluginManifests 获取所有外部插件清单
func (mm *MalManager) GetPluginManifests() []*MalManiFest {
	return GetPluginManifest()
}

// ReloadExternalMal 重新加载指定的外部mal插件
func (mm *MalManager) ReloadExternalMal(malName string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	plugin, exists := mm.externalPlugins[malName]
	if !exists {
		return fmt.Errorf("external plugin %s not found\n", malName)
	}

	logs.Log.Debugf("Reloading external plugin: %s\n", malName)

	if err := plugin.Destroy(); err != nil {
		logs.Log.Warnf("Failed to destroy plugin %s during reload: %v\n", malName, err)
	}

	delete(mm.externalPlugins, malName)

	manifestPath := filepath.Join(assets.GetMalsDir(), malName, m.ManifestFileName)
	manifest, err := LoadMalManiFest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to reload manifest for %s: %v", malName, err)
	}

	var newPlugin Plugin
	switch manifest.Type {
	case LuaScript:
		newPlugin, err = NewLuaMalPlugin(manifest)
	default:
		return fmt.Errorf("not found valid script type: %s\n", manifest.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to create new plugin %s: %v", malName, err)
	}

	err = newPlugin.Run()
	if err != nil {
		return fmt.Errorf("failed to run new plugin %s: %v", malName, err)
	}

	// 重新添加到映射中
	mm.externalPlugins[malName] = newPlugin
	logs.Log.Infof("Successfully reloaded external plugin: %s\n", malName)

	return nil
}

// GetLoadedMals 获取所有已加载的插件列表
func (mm *MalManager) GetLoadedMals() []string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var plugins []string

	// 添加嵌入式插件
	for name := range mm.embeddedPlugins {
		plugins = append(plugins, name+" (embedded)")
	}

	// 添加外部插件
	for name := range mm.externalPlugins {
		plugins = append(plugins, name+" (external)")
	}

	return plugins
}

// GetGlobalPlugins 获取所有全局插件
func (mm *MalManager) GetGlobalPlugins() []*DefaultPlugin {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// 返回副本以避免并发问题
	result := make([]*DefaultPlugin, len(mm.globalPlugins))
	copy(result, mm.globalPlugins)
	return result
}

// GetGlobalPlugin 获取指定名称的全局插件
func (mm *MalManager) GetGlobalPlugin(name string) (*DefaultPlugin, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	for _, plugin := range mm.globalPlugins {
		if plugin.Name == name {
			return plugin, true
		}
	}
	return nil, false
}

// RemoveExternalMal 移除指定的外部mal插件
func (mm *MalManager) RemoveExternalMal(malName string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	plugin, exists := mm.externalPlugins[malName]
	if !exists {
		return fmt.Errorf("external plugin %s not found\n", malName)
	}

	// 销毁插件
	if err := plugin.Destroy(); err != nil {
		logs.Log.Warnf("Failed to destroy plugin %s: %v\n", malName, err)
	}

	// 从映射中删除
	delete(mm.externalPlugins, malName)
	logs.Log.Infof("Removed external plugin: %s\n", malName)

	return nil
}
