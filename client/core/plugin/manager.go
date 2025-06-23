package plugin

import (
	"fmt"
	"sync"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/intl"
	"github.com/spf13/cobra"
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

var (
	globalMalManager *MalManager
	managerOnce      sync.Once
)

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
	mm.globalPlugins = LoadGlobalLuaPlugin()
	mm.processEmbeddedCommands()

	mm.initialized = true
	logs.Log.Infof("MalManager initialized with %d embedded and %d external plugins, %d global plugins\n",
		len(mm.embeddedPlugins), len(mm.externalPlugins), len(mm.globalPlugins))
}

// loadEmbeddedMals 直接加载嵌入式mal插件
func (mm *MalManager) loadEmbeddedMals() {
	// 按优先级顺序加载每个级别的mal包
	levelOrder := []MalLevel{CommunityLevel, ProfessionalLevel, CustomLevel}
	levelNames := map[MalLevel]string{
		CommunityLevel:    "community",
		ProfessionalLevel: "professional",
		CustomLevel:       "custom",
	}

	for _, level := range levelOrder {
		levelName := levelNames[level]

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
}

// processEmbeddedCommands 处理嵌入式命令优先级和覆盖
func (mm *MalManager) processEmbeddedCommands() {
	// 按优先级顺序加载嵌入式命令：community -> professional -> custom
	levelOrder := []string{"community", "professional", "custom"}

	for _, levelName := range levelOrder {
		if plugin, exists := mm.embeddedPlugins[levelName]; exists {
			// 添加插件的命令到Commands中，后加载的会覆盖先加载的
			for _, cmd := range plugin.Commands() {
				cmdName := cmd.Command.Name()
				mm.loadedCommands.SetCommand(cmdName, cmd.Command)
				logs.Log.Debugf("Added/Updated embedded command '%s' from %s\n", cmdName, levelName)
			}

		}
	}
}

// LoadExternalMal 加载单个外部mal插件
func (mm *MalManager) LoadExternalMal(manifest *MalManiFest) (Plugin, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

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

// RegisterEmbeddedCommandsTo 注册嵌入式命令到指定的cobra命令
func (mm *MalManager) RegisterEmbeddedCommandsTo(rootCmd *cobra.Command) int {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	if !mm.initialized {
		logs.Log.Warn("MalManager not initialized\n")
		return 0
	}

	count := 0
	for cmdName, cmd := range mm.loadedCommands {
		rootCmd.AddCommand(cmd.Command)
		logs.Log.Debugf("Registered embedded command: %s\n", cmdName)
		count++
	}

	return count
}

// RegisterExternalCommandsTo 注册外部命令到指定的cobra命令
func (mm *MalManager) RegisterExternalCommandsTo(rootCmd *cobra.Command) int {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	count := 0
	for pluginName, plugin := range mm.externalPlugins {
		for _, cmd := range plugin.Commands() {
			rootCmd.AddCommand(cmd.Command)
			logs.Log.Debugf("Registered external command '%s' from %s\n", cmd.Command.Name(), pluginName)
			count++
		}
	}

	return count
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

	// TODO: 实现外部插件的重新加载逻辑
	// 1. 卸载当前插件
	// 2. 重新读取manifest
	// 3. 重新加载插件

	_ = plugin // 避免未使用变量警告
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

// GetCommandCount 获取已加载的命令数量
func (mm *MalManager) GetCommandCount() int {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	embeddedCount := len(mm.loadedCommands)
	externalCount := 0
	for _, plugin := range mm.externalPlugins {
		externalCount += len(plugin.Commands())
	}

	return embeddedCount + externalCount
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
