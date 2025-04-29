package config

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func init() {
	if err := loadConfig(); err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}
}

var conf BizConf

type OpenAIConf struct {
	BaseURL      string `yaml:"base_url"`
	APIKey       string `yaml:"api_key"`
	Model        string `yaml:"model"`
	Voice        string `yaml:"voice"`
	SystemPrompt string `yaml:"system_prompt"`
}

type XiaozhiConf struct {
	Format        string `yaml:"format"`
	Transport     string `yaml:"transport"`
	SampleRate    int    `yaml:"sample_rate"`
	Channels      int    `yaml:"channels"`
	FrameDuration int    `yaml:"frame_duration"`
}

type ProviderConf struct {
	Name    string          `yaml:"name"`
	Xiaozhi XiaozhiProvider `yaml:"xiaozhi"`
}

type XiaozhiProvider struct {
	BaseURL string `yaml:"base_url"`
}

type BizConf struct {
	Provider ProviderConf `yaml:"provider"`
	OpenAI   OpenAIConf   `yaml:"openai"`
	Xiaozhi  XiaozhiConf  `yaml:"xiaozhi"`
	Audio    struct {
		InputFormat  string `yaml:"input_format"`
		OutputFormat string `yaml:"output_format"`
		SampleRate   int    `yaml:"sample_rate"`
		Channels     int    `yaml:"channels"`
		MaxDuration  int    `yaml:"max_duration"`
	} `yaml:"audio"`
	DefaultParams struct {
		ChatCompletions struct {
			FrequencyPenalty *float32 `yaml:"frequency_penalty"`
			Temperature      *float32 `yaml:"temperature"`
			TopP             *float32 `yaml:"top_p"`
		} `yaml:"chat_completions"`
	} `yaml:"default_params"`
}

func Get() *BizConf {
	return &conf
}

func OpenAIConfig() *OpenAIConf {
	return &conf.OpenAI
}

func Xiaozhi() *XiaozhiConf {
	return &conf.Xiaozhi
}

func Provider() *ProviderConf {
	return &conf.Provider
}

func loadConfig() error {
	v := viper.New()
	v.SetConfigName("biz")
	v.SetConfigType("yaml")
	v.AddConfigPath("conf")
	v.AddConfigPath(".")

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           &conf,
		TagName:          "yaml",
	}); err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	} else if err = decoder.Decode(v.AllSettings()); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	conf.LoadEnv(v)
	if err := conf.Validate(); err != nil {
		return err
	}

	v.WatchConfig()
	return nil
}

func GetConfigFilePath() string {
	return viper.ConfigFileUsed()
}

func ReloadConfig() error {
	return loadConfig()
}

func (c *BizConf) LoadEnv(v *viper.Viper) {
	c.OpenAI.APIKey = strings.TrimSpace(v.GetString("XDIM_STEP_API_KEY"))
	if c.OpenAI.APIKey == "" {
		panic("api_key is required")
	}
}

func (c *BizConf) Validate() error {
	if c.OpenAI.APIKey == "" {
		return fmt.Errorf("openai.api_key is required")
	}
	if c.OpenAI.BaseURL == "" {
		return fmt.Errorf("openai.base_url is required")
	}
	if c.Xiaozhi.Format == "" {
		return fmt.Errorf("xiaozhi.format is required")
	}
	if c.Xiaozhi.Transport == "" {
		return fmt.Errorf("xiaozhi.transport is required")
	}
	return nil
}
