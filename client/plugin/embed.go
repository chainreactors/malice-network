package plugin

import (
	"embed"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/intl"
)

// MalLevel 表示mal插件的级别
type MalLevel int

func (l MalLevel) String() string {
	switch l {
	case CommunityLevel:
		return "community"
	case ProfessionalLevel:
		return "professional"
	default:
		return "unknown"
	}
}

const (
	CommunityLevel MalLevel = iota
	ProfessionalLevel
	CustomLevel
)

var (
	levelOrder = []MalLevel{CommunityLevel, ProfessionalLevel, CustomLevel}
)

// EmbedPlugin 嵌入式Lua插件，直接实现Plugin接口
type EmbedPlugin struct {
	*LuaPlugin
	// 嵌入式插件特有的信息
	Level    MalLevel
	FS       embed.FS
	RootPath string // 在embed.FS中的根路径，如"community"、"professional"等
}

// NewEmbedPlugin 创建嵌入式插件
func NewEmbedPlugin(malPath, malName string, level MalLevel) (*EmbedPlugin, error) {
	// 读取manifest文件
	manifestPath := malPath + "/mal.yaml"
	manifestData, err := intl.UnifiedFS.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	// 解析manifest
	manifest := &MalManiFest{}
	if err := yaml.Unmarshal(manifestData, manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// 设置manifest类型为embed
	manifest.Type = EmbedType

	// 读取main.lua内容从embed.FS
	var content []byte
	if manifest.EntryFile != "" {
		entryPath := malPath + "/" + manifest.EntryFile
		content, err = intl.UnifiedFS.ReadFile(entryPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read entry file %s: %w", manifest.EntryFile, err)
		}
	}

	// 为嵌入式插件创建DefaultPlugin
	defaultPlugin := &DefaultPlugin{
		MalManiFest: manifest,
		Enable:      true,
		Content:     content,
		Path:        malPath, // 使用嵌入式路径
		CMDs:        make(Commands),
		Events:      make(map[intermediate.EventCondition]intermediate.OnEventFunc),
	}

	// 创建LuaPlugin
	luaPlugin := &LuaPlugin{
		DefaultPlugin: defaultPlugin,
		vmFns:         make(map[string]lua.LGFunction),
	}

	// 创建嵌入式插件
	embedPlugin := &EmbedPlugin{
		LuaPlugin: luaPlugin,
		Level:     level,
		FS:        intl.UnifiedFS,
		RootPath:  malPath,
	}

	return embedPlugin, nil
}

func (plug *EmbedPlugin) Run() error {
	var err error
	plug.vmPool, err = NewLuaVMPool(10, string(plug.Content), plug.Name)
	if err != nil {
		return err
	}
	plug.registerLuaFunction()
	plug.setContext = func(vm *lua.LState) error {
		return plug.addEmbedLoader(vm)
	}
	plug.registerEmbedResourceFunctions()
	err = plug.registerLuaOnHooks()
	if err != nil {
		return err
	}
	return nil
}

// GetFileContent 获取文件内容 - 直接从embed.FS读取
func (plug *EmbedPlugin) GetFileContent(filename string) ([]byte, bool) {
	fullPath := plug.RootPath + "/" + filename
	content, err := plug.FS.ReadFile(fullPath)
	if err != nil {
		return nil, false
	}
	return content, true
}

// FileExists 检查文件是否存在
func (plug *EmbedPlugin) FileExists(filename string) bool {
	fullPath := plug.RootPath + "/" + filename
	_, err := plug.FS.Open(fullPath)
	return err == nil
}

// ReadDir 读取目录内容
func (plug *EmbedPlugin) ReadDir(dirname string) ([]fs.DirEntry, error) {
	fullPath := plug.RootPath + "/" + dirname
	return plug.FS.ReadDir(fullPath)
}

// GetLevel 获取插件级别
func (plug *EmbedPlugin) GetLevel() MalLevel {
	return plug.Level
}

// registerEmbedResourceFunctions 注册嵌入式资源相关的Lua函数
func (plug *EmbedPlugin) registerEmbedResourceFunctions() {
	// 重写script_resource函数 - 返回资源文件路径
	plug.registerFunction("script_resource", func(filename string) (string, error) {
		resourcePath := "resources/" + filename
		if _, exists := plug.GetFileContent(resourcePath); exists {
			return fmt.Sprintf("embed://%s/%s", plug.Name, resourcePath), nil
		}

		// 回退到文件系统
		resourceFile := filepath.Join(assets.GetMalsDir(), plug.Name, "resources", filename)
		return resourceFile, nil
	}, nil)

	// 重写global_resource函数 - 返回全局资源文件路径
	plug.registerFunction("global_resource", func(filename string) (string, error) {
		// 从全局管理器查找
		if globalManager := GetGlobalMalManager(); globalManager != nil {
			reverseLevelOrder := []string{"custom", "professional", "community"}

			for _, levelName := range reverseLevelOrder {
				if plugin, exists := globalManager.GetEmbedPlugin(levelName); exists {
					resourcePath := "resources/" + filename
					if _, fileExists := plugin.GetFileContent(resourcePath); fileExists {
						return fmt.Sprintf("embed://%s/%s", levelName, resourcePath), nil
					}
				}
			}
		}

		resourceFile := filepath.Join(assets.GetResourceDir(), filename)
		return resourceFile, nil
	}, nil)

	// 重写find_resource函数 - 查找架构特定的资源文件
	plug.registerFunction("find_resource", func(sess *core.Session, base string, ext string) (string, error) {
		// 这里简化处理，直接使用默认架构
		filename := fmt.Sprintf("%s.%s.%s", base, sess.Os.Arch, ext)

		resourcePath := "resources/" + filename
		if _, exists := plug.GetFileContent(resourcePath); exists {
			return fmt.Sprintf("embed://%s/%s", plug.Name, resourcePath), nil
		}

		resourceFile := filepath.Join(assets.GetMalsDir(), plug.Name, "resources", filename)
		return resourceFile, nil
	}, nil)

	// 重写find_global_resource函数 - 查找全局架构特定的资源文件
	plug.registerFunction("find_global_resource", func(sess *core.Session, base string, ext string) (string, error) {
		filename := fmt.Sprintf("%s.%s.%s", base, sess.Os.Arch, ext)

		if globalManager := GetGlobalMalManager(); globalManager != nil {
			reverseLevelOrder := []string{"custom", "professional", "community"}

			for _, levelName := range reverseLevelOrder {
				if plugin, exists := globalManager.GetEmbedPlugin(levelName); exists {
					resourcePath := "resources/" + filename
					if _, fileExists := plugin.GetFileContent(resourcePath); fileExists {
						return fmt.Sprintf("embed://%s/%s", levelName, resourcePath), nil
					}
				}
			}
		}

		// 回退到文件系统
		resourceFile := filepath.Join(assets.GetResourceDir(), filename)
		return resourceFile, nil
	}, nil)

	// 重写read_resource函数 - 读取当前插件的资源文件内容
	plug.registerFunction("read_resource", func(filename string) (string, error) {
		// 先尝试从嵌入式资源读取
		resourcePath := "resources/" + filename
		if content, exists := plug.GetFileContent(resourcePath); exists {
			return string(content), nil
		}

		// 回退到文件系统
		fsPath := filepath.Join(assets.GetMalsDir(), plug.Name, "resources", filename)
		content, err := os.ReadFile(fsPath)
		if err != nil {
			return "", fmt.Errorf("resource file not found: %s", filename)
		}
		return string(content), nil
	}, nil)

	// 重写read_global_resource函数 - 读取全局资源文件内容（按优先级查找）
	plug.registerFunction("read_global_resource", func(filename string) (string, error) {
		// 从plugin包获取全局嵌入式mal管理器
		if globalManager := GetGlobalMalManager(); globalManager != nil {
			// 按优先级顺序查找：custom -> professional -> community
			reverseLevelOrder := []string{"custom", "professional", "community"}

			for _, levelName := range reverseLevelOrder {
				if plugin, exists := globalManager.GetEmbedPlugin(levelName); exists {
					resourcePath := "resources/" + filename
					if content, fileExists := plugin.GetFileContent(resourcePath); fileExists {
						return string(content), nil
					}
				}
			}
		}

		// 回退到文件系统
		fsPath := filepath.Join(assets.GetResourceDir(), filename)
		content, err := os.ReadFile(fsPath)
		if err != nil {
			return "", fmt.Errorf("global resource file not found: %s", filename)
		}
		return string(content), nil
	}, nil)

	// 新增read_embed_resource函数 - 专门用于读取嵌入式资源，支持embed://路径
	plug.registerFunction("read_embed_resource", func(resourcePath string) (string, error) {
		if strings.HasPrefix(resourcePath, "embed://") {
			// 解析嵌入式资源路径: embed://pluginName/resourcePath
			parts := strings.TrimPrefix(resourcePath, "embed://")
			pathParts := strings.SplitN(parts, "/", 2)
			if len(pathParts) != 2 {
				return "", fmt.Errorf("invalid embedded resource path: %s", resourcePath)
			}

			pluginName := pathParts[0]
			filename := strings.TrimPrefix(pathParts[1], "resources/")

			// 如果是当前插件的资源
			if pluginName == plug.Name {
				resourceFilePath := "resources/" + filename
				if content, exists := plug.GetFileContent(resourceFilePath); exists {
					return string(content), nil
				}
			}

			// 从全局管理器查找其他插件的资源
			if globalManager := GetGlobalMalManager(); globalManager != nil {
				if plugin, exists := globalManager.GetEmbedPlugin(pluginName); exists {
					resourceFilePath := "resources/" + filename
					if content, fileExists := plugin.GetFileContent(resourceFilePath); fileExists {
						return string(content), nil
					}
				}
			}

			return "", fmt.Errorf("embedded resource not found: %s", resourcePath)
		}

		// 如果不是embed://路径，直接从文件系统读取
		content, err := os.ReadFile(resourcePath)
		if err != nil {
			return "", fmt.Errorf("file not found: %s", resourcePath)
		}
		return string(content), nil
	}, nil)
}

// addEmbedLoader 添加embed fs的require loader
func (plug *EmbedPlugin) addEmbedLoader(vm *lua.LState) error {
	// 获取package.loaders表
	loaders, ok := vm.GetField(vm.Get(lua.RegistryIndex), "_LOADERS").(*lua.LTable)
	if !ok {
		return fmt.Errorf("package.loaders must be a table")
	}

	// 创建embed loader函数
	embedLoader := vm.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)

		// 将模块名转换为路径 (将点替换为斜杠)
		luaPath := strings.Replace(name, ".", "/", -1) + ".lua"

		// 先尝试从当前插件的embed.FS中查找
		if content, exists := plug.GetFileContent(luaPath); exists {
			// 编译lua脚本
			fn, err := L.LoadString(string(content))
			if err != nil {
				L.Push(lua.LString(fmt.Sprintf("error loading embedded module '%s': %s", name, err.Error())))
				return 1
			}
			L.Push(fn)
			return 1
		}

		//// 尝试从全局管理器的其他embed插件中查找
		//if globalManager := GetGlobalMalManager(); globalManager != nil {
		//	// 按优先级顺序查找：custom -> professional -> community
		//	levelOrder := []string{"custom", "professional", "community"}
		//
		//	for _, levelName := range levelOrder {
		//		if embedPlugin, exists := globalManager.GetEmbedPlugin(levelName); exists {
		//			if content, fileExists := embedPlugin.GetFileContent(luaPath); fileExists {
		//				// 编译lua脚本
		//				fn, err := L.LoadString(string(content))
		//				if err != nil {
		//					L.Push(lua.LString(fmt.Sprintf("error loading embedded module '%s' from %s: %s", name, levelName, err.Error())))
		//					return 1
		//				}
		//				L.Push(fn)
		//				return 1
		//			}
		//		}
		//	}
		//}

		// 没有找到模块
		L.Push(lua.LString(fmt.Sprintf("no embedded module '%s'", name)))
		return 1
	})

	loadersLen := loaders.Len()
	loaders.RawSetInt(loadersLen+1, embedLoader)

	return nil
}
