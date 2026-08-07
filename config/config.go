package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/vincentchyu/sonic-lens/common"
)

var ConfigObj = &Config{}

type Config struct {
	Lastfm        ScrobblerConfig     `yaml:"lastfm"`
	Musixmatch    MusixmatchConfig    `yaml:"musixmatch"`
	Log           LogConfig           `yaml:"log"`
	Database      DatabaseConfig      `yaml:"database"`
	Dashboard     DashboardConfig     `yaml:"dashboard"`
	PlayReplay    PlayReplayConfig    `yaml:"playReplay"`
	HTTP          HTTPConfig          `yaml:"http"`
	Bonjour       BonjourConfig       `yaml:"bonjour"`
	Telemetry     TelemetryConfig     `yaml:"telemetry"`
	Redis         RedisConfig         `yaml:"redis"`
	Cloudflare    CloudflareConfig    `yaml:"cloudflare"`
	ObjectStorage ObjectStorageConfig `yaml:"objectStorage"`
	AI            AIConfig            `yaml:"ai"`
	Scrobblers    []string            `yaml:"scrobblers"`
	IsDev         bool                `yaml:"isDev"`
	Lyrics        LyricsConfig        `yaml:"lyrics"`
}

type ScrobblerConfig struct {
	ApplicationName string `yaml:"applicationName"`
	ApiKey          string `yaml:"apiKey"`
	SharedSecret    string `yaml:"sharedSecret"`
	RegisteredTo    string `yaml:"registeredTo"`
	UserLoginToken  string `yaml:"userLoginToken"`
	UserUsername    string `yaml:"userUsername"`
	UserPassword    string `yaml:"userPassword"`
}

type LogConfig struct {
	Path  string `yaml:"path"`
	Level string `yaml:"level"`
}

type MusixmatchConfig struct {
	ApiKey string `yaml:"apiKey"`
}

type LyricsConfig struct {
	LrcAPI LrcAPIConfig `yaml:"lrcApi"`
}

type LrcAPIConfig struct {
	BaseURL string `yaml:"baseUrl"`
	Token   string `yaml:"token"`
}

type DatabaseConfig struct {
	Type  string      `yaml:"type"` // 当前仅支持 "mysql"
	Path  string      `yaml:"path"`
	Mysql MysqlConfig `yaml:"mysql"`
}

type HTTPConfig struct {
	Port string `yaml:"port"`
}

type BonjourConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Name        string `yaml:"name"`
	ServiceType string `yaml:"serviceType"`
}

type DashboardConfig struct {
	StatRefreshEnabled              bool `yaml:"statRefreshEnabled"`
	StatRefreshIntervalMinutes      int  `yaml:"statRefreshIntervalMinutes"`
	HeavyStatRefreshIntervalMinutes int  `yaml:"heavyStatRefreshIntervalMinutes"`
	HeavyStatOnlyOnNewPlay          bool `yaml:"heavyStatOnlyOnNewPlay"`
	TopN                            int  `yaml:"topN"`
	TrendDays                       int  `yaml:"trendDays"`
	HourlyTrendDays                 int  `yaml:"hourlyTrendDays"`
}

type PlayReplayConfig struct {
	Enabled         bool `yaml:"enabled"`
	IntervalMinutes int  `yaml:"intervalMinutes"`
	BatchSize       int  `yaml:"batchSize"`
	OnlyUnapplied   bool `yaml:"onlyUnapplied"`
	OnlyUnresolved  bool `yaml:"onlyUnresolved"`
	RunOnStartup    bool `yaml:"runOnStartup"`
}

type MysqlConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

func (m MysqlConfig) GetMysqlDSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		m.User, m.Password, m.Host, m.Port, m.Database,
	)
}

type TelemetryConfig struct {
	Name                  string            `yaml:"name"`
	Endpoint              string            `yaml:"endpoint"`
	Sampler               float64           `yaml:"sampler"`
	Insecure              bool              `yaml:"insecure"`
	MetricIntervalSeconds int               `yaml:"metricIntervalSeconds"`
	RuntimeMetricsEnabled bool              `yaml:"runtimeMetricsEnabled"`
	DBStatsMetricsEnabled bool              `yaml:"dbStatsMetricsEnabled"`
	Environment           string            `yaml:"environment"`
	Batcher               string            `yaml:"batcher"` // 兼容旧配置；仅保留 stdout 调试语义
	OtlpHeaders           map[string]string `yaml:"otlpHeaders"`
	OtlpHttpPath          string            `yaml:"otlpHttpPath"`   // 兼容旧配置，当前 gRPC exporter 不使用
	OtlpHttpSecure        bool              `yaml:"otlpHttpSecure"` // 兼容旧配置，当前 gRPC exporter 不使用
	Disabled              bool              `yaml:"disabled"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type CloudflareConfig struct {
	AccountID    string `yaml:"accountId"`
	APIToken     string `yaml:"apiToken"`
	D1DatabaseID string `yaml:"d1DatabaseId"`
	SyncEnabled  bool   `yaml:"syncEnabled"`  // 是否启用 D1 同步
	SyncInterval int    `yaml:"syncInterval"` // 同步间隔(小时)
}

// ObjectStorageConfig 对象存储配置（MinIO/S3/R2/cloudflare 兼容）。
type ObjectStorageConfig struct {
	Enabled         bool   `yaml:"enabled"`
	Provider        string `yaml:"provider"`
	Endpoint        string `yaml:"endpoint"`
	Bucket          string `yaml:"bucket"`
	Region          string `yaml:"region"`
	AccessKeyID     string `yaml:"accessKeyId"`
	SecretAccessKey string `yaml:"secretAccessKey"`
	CDNURL          string `yaml:"cdnUrl"`
	BasePrefix      string `yaml:"basePrefix"`
	OriginalPrefix  string `yaml:"originalPrefix"`
	ThumbnailPrefix string `yaml:"thumbnailPrefix"`
	ForcePathStyle  bool   `yaml:"forcePathStyle"`
	UseSSL          bool   `yaml:"useSSL"`
}

// AIConfig 大模型相关配置
// provider 用于选择具体实现，例如：gemini、ollama、doubao、omlx 等
type AIConfig struct {
	Provider  string       `yaml:"provider"`
	MultiStep bool         `yaml:"multiStep"`
	Gemini    GeminiConfig `yaml:"gemini"`
	Ollama    OllamaConfig `yaml:"ollama"`
	Doubao    DoubaoConfig `yaml:"doubao"`
	OMLX      OMLXConfig   `yaml:"omlx"`
}

// GetAvailableProviders 返回当前配置中所有已配置的 AI 提供商
func (c AIConfig) GetAvailableProviders() []string {
	var providers []string
	if c.Gemini.APIKey != "" {
		providers = append(providers, "gemini")
	}
	if c.Ollama.Host != "" {
		providers = append(providers, "ollama")
	}
	if c.Doubao.APIKey != "" {
		providers = append(providers, "doubao")
	}
	if c.OMLX.APIKey != "" {
		providers = append(providers, "omlx")
	}
	return providers
}

// GetAvailablePlatforms 返回当前配置中可用的 AI 平台。
func (c AIConfig) GetAvailablePlatforms() []common.AIModelPlatform {
	var platforms []common.AIModelPlatform
	for _, provider := range c.GetAvailableProviders() {
		platform := common.ParseAIModelPlatform(provider)
		if platform.IsValid() {
			platforms = append(platforms, platform)
		}
	}
	return platforms
}

// OMLX 配置
type OMLXConfig struct {
	APIKey  string `yaml:"apiKey"`
	BaseURL string `yaml:"baseUrl"`
	Model   string `yaml:"model"`
}

// GeminiConfig Gemini 配置
type GeminiConfig struct {
	APIKey  string `yaml:"apiKey"`
	BaseURL string `yaml:"baseUrl"`
	Model   string `yaml:"model"`
}

// OllamaConfig 本地 Ollama 配置
type OllamaConfig struct {
	Host  string `yaml:"host"`  // 例如 http://127.0.0.1:11434
	Model string `yaml:"model"` // 例如 qwen:latest
}

// DoubaoConfig 豆包/字节系模型配置
type DoubaoConfig struct {
	APIKey              string `yaml:"apiKey"`
	BaseURL             string `yaml:"baseUrl"`
	Model               string `yaml:"model"`
	ManagementAccessKey string `yaml:"managementAccessKey"`
	ManagementSecretKey string `yaml:"managementSecretKey"`
	ManagementRegion    string `yaml:"managementRegion"`
	ProjectName         string `yaml:"projectName"`
}

func InitConfig(filePath string) {
	viper.SetConfigFile(filePath)
	viper.SetConfigType("yaml")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	_ = viper.BindEnv("objectStorage.enabled", "OBJECT_STORAGE_ENABLED")
	_ = viper.BindEnv("objectStorage.provider", "OBJECT_STORAGE_PROVIDER")
	_ = viper.BindEnv("objectStorage.endpoint", "ENDPOINT", "OBJECT_STORAGE_ENDPOINT")
	_ = viper.BindEnv("objectStorage.bucket", "BUCKET", "OBJECT_STORAGE_BUCKET")
	_ = viper.BindEnv("objectStorage.region", "REGION", "OBJECT_STORAGE_REGION")
	_ = viper.BindEnv("objectStorage.accessKeyId", "ACCESS_KEY_ID", "OBJECT_STORAGE_ACCESS_KEY_ID")
	_ = viper.BindEnv("objectStorage.secretAccessKey", "SECRET_ACCESS_KEY", "OBJECT_STORAGE_SECRET_ACCESS_KEY")
	_ = viper.BindEnv("objectStorage.cdnUrl", "CDN_URL", "OBJECT_STORAGE_CDN_URL")
	_ = viper.BindEnv("objectStorage.basePrefix", "BASE_PREFIX", "OBJECT_STORAGE_BASE_PREFIX")
	_ = viper.BindEnv("objectStorage.originalPrefix", "ORIGINAL_PREFIX", "OBJECT_STORAGE_ORIGINAL_PREFIX")
	_ = viper.BindEnv("objectStorage.thumbnailPrefix", "THUMBNAIL_PREFIX", "OBJECT_STORAGE_THUMBNAIL_PREFIX")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
	if err := viper.Unmarshal(ConfigObj); err != nil {
		panic(err)
	}
}
