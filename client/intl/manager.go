package intl

import (
	"fmt"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/spf13/cobra"
)

// EmbedManager 嵌入式插件管理器
type EmbedManager struct {
	loadedMals map[string]plugin.Plugin
	Commands   plugin.Commands // 维护经过优先级处理的最终命令集合
}

// NewEmbedManager 创建新的嵌入式插件管理器
func NewEmbedManager() *EmbedManager {
	return &EmbedManager{
		loadedMals: make(map[string]plugin.Plugin),
		Commands:   make(plugin.Commands),
	}
}

// InitializeCommands 初始化并处理命令优先级覆盖
func (em *EmbedManager) InitializeCommands() error {
	embeddedMals := GetEmbeddedMals()

	// 按优先级顺序加载：community -> professional -> custom
	levelOrder := []string{"community", "professional", "custom"}

	for _, levelName := range levelOrder {
		if embeddedMal, exists := embeddedMals[levelName]; exists {
			malPlugin, err := em.loadMalPlugin(levelName)
			if err != nil {
				logs.Log.Errorf("Failed to load embedded mal %s: %v", levelName, err)
				continue
			}

			// 将插件的命令添加到Commands中，自动处理覆盖
			for _, cmd := range malPlugin.Commands() {
				cmdName := cmd.Command.Name()

				// 使用Commands.SetCommand处理覆盖逻辑
				em.Commands.SetCommand(cmdName, cmd.Command)

				logs.Log.Debugf("Added/Updated command '%s' from %s mal", cmdName, levelName)
			}

			logs.Log.Infof("Processed embedded mal: %s (level: %s)", embeddedMal.Name, levelName)
		}
	}

	totalCommands := len(em.Commands)
	logs.Log.Infof("Initialized embedded mal commands: %d commands ready", totalCommands)
	return nil
}

// loadMalPlugin 加载单个mal插件
func (em *EmbedManager) loadMalPlugin(malName string) (plugin.Plugin, error) {
	embeddedMal, exists := GetEmbeddedMal(malName)
	if !exists {
		return nil, fmt.Errorf("embedded mal %s not found", malName)
	}

	// 创建DefaultPlugin实例
	defaultPlugin, err := CreateEmbeddedMalPlugin(embeddedMal)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedded mal plugin: %w", err)
	}

	// 创建Lua插件
	luaPlugin := &plugin.LuaPlugin{
		DefaultPlugin: defaultPlugin,
	}

	// 运行插件初始化
	if err := luaPlugin.Run(); err != nil {
		return nil, fmt.Errorf("failed to run embedded lua plugin: %w", err)
	}

	em.loadedMals[embeddedMal.Name] = luaPlugin
	return luaPlugin, nil
}

// GetCommands 获取处理后的命令集合
func (em *EmbedManager) GetCommands() plugin.Commands {
	return em.Commands
}

// RegisterCommandsTo 将所有命令注册到指定的根命令
func (em *EmbedManager) RegisterCommandsTo(rootCmd *cobra.Command) int {
	count := 0
	for _, cmd := range em.Commands {
		if cmd.Command != nil {
			rootCmd.AddCommand(cmd.Command)
			count++
		}
	}
	return count
}

// GetLoadedMals 获取已加载的嵌入式mal插件
func (em *EmbedManager) GetLoadedMals() map[string]plugin.Plugin {
	return em.loadedMals
}
