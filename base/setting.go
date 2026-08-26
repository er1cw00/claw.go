package base

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProviderConfig holds the common LLM provider configuration.
type ProviderConfig struct {
	APIKey  string `yaml:"api_key"`
	APIBase string `yaml:"api_base"`
	Proxy   string `yaml:"proxy"`
}

type AgentConfig struct {
	Name              string  `yaml:"name"`
	Workspace         string  `yaml:"workspace"`
	Provider          string  `yaml:"provider"`
	Model             string  `yaml:"model"`
	MaxTokens         int     `yaml:"max_tokens"`
	ContentWindow     int     `yaml:"content_window"`
	Temperature       float64 `yaml:"temperature"`
	MaxToolIterations int     `yaml:"max_tool_iterations"`
	Timezone          string  `yaml:"timezone"`
}

type Settings struct {
	Web struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"web"`

	Log struct {
		Path  string `yaml:"path"`
		Level string `yaml:"level"`
	} `yaml:"log"`

	Channels struct {
		Telegram struct {
			Enable bool   `yaml:"enable"`
			Token  string `yaml:"token"`
			Proxy  string `yaml:"proxy"`
		} `yaml:"telegram"`
		WeChat struct {
			Enable bool `yaml:"enable"`
		} `yaml:"wechat"`
	} `yaml:"channels"`

	Providers struct {
		Custom     ProviderConfig `yaml:"custom"`
		OpenRouter ProviderConfig `yaml:"openrouter"`
		DeepSeek   ProviderConfig `yaml:"deepseek"`
	} `yaml:"providers"`

	Agent AgentConfig `yaml:"agent"`

	WorkPath  string `yaml:"work_path"`
	MediaPath string `yaml:"media_path"`
}

var settings = new(Settings)

func GetSettings() *Settings {
	return settings
}

// GetProviderConfig returns the configuration for the named provider.
// Supported names: "custom", "openrouter", "deepseek" (case-insensitive).
func GetProviderConfig(name string) ProviderConfig {
	switch strings.ToLower(name) {
	case "custom":
		return settings.Providers.Custom
	case "openrouter":
		return settings.Providers.OpenRouter
	case "deepseek":
		return settings.Providers.DeepSeek
	default:
		return ProviderConfig{}
	}
}

func (*Settings) String() string {
	b, err := yaml.Marshal(settings)
	if err != nil {
		return fmt.Sprintf("yaml Marshal Fail, err: %v", err)
	}
	return string(b)
}

func parseSettings(path string) error {
	settings = new(Settings)

	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(yamlFile, settings); err != nil {
		return err
	}

	if settings.Log.Level == "" {
		settings.Log.Level = "WARNING"
	}
	if settings.Log.Path != "" {
		settings.Log.Path, err = filepath.Abs(settings.Log.Path)
		if err != nil {
			return err
		}
	}

	if settings.WorkPath == "" {
		settings.WorkPath = settings.Agent.Workspace
	}
	if settings.WorkPath, err = filepath.Abs(settings.WorkPath); err != nil {
		fmt.Printf("workspace path unknown")
		return err
	}
	if err = Mkdir(settings.WorkPath); err != nil {
		fmt.Printf("mkdir work path fail, err: %v", err)
		return err
	}
	if settings.MediaPath == "" {
		settings.MediaPath = filepath.Join(settings.WorkPath, "media")
	}
	if err = Mkdir(settings.MediaPath); err != nil {
		fmt.Printf("mkdir media path fail, err: %v", err)
		return err
	}
	return nil
}
