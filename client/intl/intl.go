package intl

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"gopkg.in/yaml.v3"
)

//go:embed community/*
var communityFS embed.FS

//go:embed professional/*
var professionalFS embed.FS

//go:embed custom/*
var customFS embed.FS

// MalLevel 表示mal插件的级别
type MalLevel int

const (
	CommunityLevel MalLevel = iota
	ProfessionalLevel
	CustomLevel
)

// EmbeddedMal 表示一个嵌入的mal插件
type EmbeddedMal struct {
	Name     string
	Level    MalLevel
	Manifest *plugin.MalManiFest
	Files    map[string][]byte // 文件路径到内容的映射
}

var (
	embeddedMals = make(map[string]*EmbeddedMal) // name -> EmbeddedMal
	levelOrder   = []MalLevel{CommunityLevel, ProfessionalLevel, CustomLevel}
	levelNames   = map[MalLevel]string{
		CommunityLevel:    "community",
		ProfessionalLevel: "professional",
		CustomLevel:       "custom",
	}
	levelFS = map[MalLevel]embed.FS{
		CommunityLevel:    communityFS,
		ProfessionalLevel: professionalFS,
		CustomLevel:       customFS,
	}
)

// NewEmbedPluginManager 创建intl插件管理器
func NewEmbedPluginManager() *EmbedManager {
	// 加载所有嵌入的mal插件定义
	if err := LoadEmbeddedMals(); err != nil {
		logs.Log.Errorf("Failed to load embedded mals: %v", err)
		return nil
	}

	// 创建插件管理器
	manager := NewEmbedManager()

	// 初始化命令，处理优先级覆盖
	if err := manager.InitializeCommands(); err != nil {
		logs.Log.Errorf("Failed to initialize embedded commands: %v", err)
		return nil
	}

	logs.Log.Infof("Created intl manager with %d embedded mals available", len(GetEmbeddedMals()))
	return manager
}

// LoadEmbeddedMals 加载所有嵌入的mal插件
func LoadEmbeddedMals() error {
	// 按优先级顺序加载每个级别的mal包
	for _, level := range levelOrder {
		if err := loadMalFromLevel(level); err != nil {
			logs.Log.Errorf("Failed to load mal from level %d: %v", level, err)
			continue
		}
	}

	logs.Log.Infof("Loaded %d embedded mals", len(embeddedMals))
	return nil
}

// loadMalFromLevel 从指定级别加载mal包
func loadMalFromLevel(level MalLevel) error {
	embedFS := levelFS[level]
	levelName := levelNames[level]

	// 检查mal.yaml是否存在
	manifestPath := levelName + "/mal.yaml"
	if _, err := embedFS.ReadFile(manifestPath); err != nil {
		// 如果mal.yaml不存在，跳过这个级别
		logs.Log.Debugf("No mal.yaml found for level %s, skipping", levelName)
		return nil
	}

	// 加载mal包
	embeddedMal, err := loadEmbeddedMal(embedFS, levelName, levelName, level)
	if err != nil {
		return fmt.Errorf("failed to load mal from level %s: %w", levelName, err)
	}

	embeddedMals[levelName] = embeddedMal
	logs.Log.Infof("Loaded embedded mal: %s (level: %d)", levelName, level)
	return nil
}

// loadEmbeddedMal 加载单个嵌入的mal插件
func loadEmbeddedMal(embedFS embed.FS, malPath, malName string, level MalLevel) (*EmbeddedMal, error) {
	// 读取manifest文件
	manifestPath := malPath + "/mal.yaml"
	manifestData, err := embedFS.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	// 解析manifest
	manifest := &plugin.MalManiFest{}
	if err := yaml.Unmarshal(manifestData, manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// 加载所有文件
	files := make(map[string][]byte)
	err = fs.WalkDir(embedFS, malPath, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// 跳过.gitkeep文件
		if strings.HasSuffix(filePath, ".gitkeep") {
			return nil
		}

		// 读取文件内容
		content, err := embedFS.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		// 将路径转换为相对于mal根目录的路径
		relPath := strings.TrimPrefix(filePath, malPath+"/")
		files[relPath] = content
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk mal directory: %w", err)
	}

	return &EmbeddedMal{
		Name:     malName,
		Level:    level,
		Manifest: manifest,
		Files:    files,
	}, nil
}

// GetEmbeddedMals 获取所有嵌入的mal插件
func GetEmbeddedMals() map[string]*EmbeddedMal {
	return embeddedMals
}

// GetEmbeddedMal 获取指定名称的嵌入mal插件
func GetEmbeddedMal(name string) (*EmbeddedMal, bool) {
	mal, exists := embeddedMals[name]
	return mal, exists
}

// CreateEmbeddedMalPlugin 为嵌入的mal创建插件实例
func CreateEmbeddedMalPlugin(embeddedMal *EmbeddedMal) (*plugin.DefaultPlugin, error) {
	// 获取入口文件内容
	entryContent, exists := embeddedMal.GetFileContent(embeddedMal.Manifest.EntryFile)
	if !exists {
		return nil, fmt.Errorf("entry file %s not found in embedded mal %s",
			embeddedMal.Manifest.EntryFile, embeddedMal.Name)
	}

	// 创建DefaultPlugin实例
	defaultPlugin := &plugin.DefaultPlugin{
		MalManiFest: embeddedMal.Manifest,
		Enable:      true,
		Content:     entryContent,
		Path:        "", // 嵌入式插件没有实际路径
		CMDs:        make(plugin.Commands),
		Events:      make(map[intermediate.EventCondition]intermediate.OnEventFunc),
	}

	return defaultPlugin, nil
}

// GetFileContent 获取嵌入mal的文件内容
func (e *EmbeddedMal) GetFileContent(filename string) ([]byte, bool) {
	content, exists := e.Files[filename]
	return content, exists
}

// ListFiles 列出嵌入mal的所有文件
func (e *EmbeddedMal) ListFiles() []string {
	var files []string
	for filename := range e.Files {
		files = append(files, filename)
	}
	return files
}

// ListEmbedded 列出所有嵌入的mal插件信息
func ListEmbedded() {
	embeddedMals := GetEmbeddedMals()
	if len(embeddedMals) == 0 {
		logs.Log.Info("No embedded mals found")
		return
	}

	logs.Log.Info("Embedded Mals:")
	for name, mal := range embeddedMals {
		levelName := getLevelName(mal.Level)
		logs.Log.Infof("  - %s (v%s, %s, %s)",
			name,
			mal.Manifest.Version,
			mal.Manifest.Type,
			levelName)
	}
}

// getLevelName 获取级别名称
func getLevelName(level MalLevel) string {
	switch level {
	case CommunityLevel:
		return "Community"
	case ProfessionalLevel:
		return "Professional"
	case CustomLevel:
		return "Custom"
	default:
		return "Unknown"
	}
}
