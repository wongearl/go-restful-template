package config

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	authoptions "github.com/wongearl/go-restful-template/pkg/aiserver/authentication/options"
	"github.com/wongearl/go-restful-template/pkg/client/cache"
	"github.com/wongearl/go-restful-template/pkg/client/k8s"
	"k8s.io/klog"
)

var (
	_config = defaultConfig()
)

const (
	// DefaultConfigurationName is the default name of configuration
	defaultConfigurationName = "ai-config"

	// DefaultConfigurationPath the default location of the configuration file
	defaultConfigurationPath = "/etc/ai"
)

type config struct {
	cfg         *Config
	cfgChangeCh chan Config
	watchOnce   sync.Once
	loadOnce    sync.Once
}

func (c *config) watchConfig() <-chan Config {
	c.watchOnce.Do(func() {
		viper.WatchConfig()
		viper.OnConfigChange(func(in fsnotify.Event) {
			cfg := New()
			if err := viper.Unmarshal(cfg); err != nil {
				klog.Warning("config reload error", err)
			} else {
				c.cfgChangeCh <- *cfg
			}
		})
	})
	return c.cfgChangeCh
}

func (c *config) loadFromDisk() (*Config, error) {
	var err error
	c.loadOnce.Do(func() {
		if err = viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				err = fmt.Errorf("error parsing configuration file %s", err)
			}
		}
		err = viper.Unmarshal(c.cfg)
	})
	return c.cfg, err
}

func defaultConfig() *config {
	viper.SetConfigName(defaultConfigurationName)
	viper.AddConfigPath(defaultConfigurationPath)

	viper.AddConfigPath(".")

	return &config{
		cfg:         New(),
		cfgChangeCh: make(chan Config),
		watchOnce:   sync.Once{},
		loadOnce:    sync.Once{},
	}
}

// Config defines everything needed for apiserver to deal with external services
type Config struct {
	KubernetesOptions *k8s.KubernetesOptions `json:"kubernetes,omitempty" yaml:"kubernetes,omitempty" mapstructure:"kubernetes"`
	AiOptions         *AiOptions             `json:"ai" yaml:"ai" mapstructure:"ai"`
	// 缓存配置，缓存token
	CacheOptions *cache.Options `json:"cache,omitempty" yaml:"cache,omitempty" mapstructure:"cache"`
	// 认证服务配置，开启关闭等
	AuthenticationOptions *authoptions.AuthenticationOptions `json:"authentication,omitempty" yaml:"authentication,omitempty" mapstructure:"authentication"`
}

func New() *Config {
	return &Config{
		KubernetesOptions:     k8s.NewKubernetesOptions(),
		AuthenticationOptions: authoptions.NewAuthenticateOptions(),
		AiOptions:             &AiOptions{},
		CacheOptions:          cache.NewCacheOptions(),
	}
}

// AiOptions defines all the needs from ai
type AiOptions struct {
	Namespace  string `json:"namespace" yaml:"namespace"`
	NamePrefix string `json:"namePrefix" yaml:"namePrefix"`
}

func TryLoadFromDisk() (*Config, error) {
	return _config.loadFromDisk()
}

func WatchConfigChange() <-chan Config {
	return _config.watchConfig()
}
