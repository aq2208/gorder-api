package configs

import (
	"fmt"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	App struct {
		Name     string `koanf:"name"`
		HTTPAddr string `koanf:"http_addr"`
		LogLevel string `koanf:"log_level"`
	} `koanf:"app"`

	HTTP struct {
		ReadTimeout  time.Duration `koanf:"read_timeout"`
		WriteTimeout time.Duration `koanf:"write_timeout"`
		IdleTimeout  time.Duration `koanf:"idle_timeout"`
	} `koanf:"http"`

	MySQL struct {
		DSN             string        `koanf:"dsn"`
		MaxOpenConns    int           `koanf:"max_open_conns"`
		MaxIdleConns    int           `koanf:"max_idle_conns"`
		ConnMaxLifetime time.Duration `koanf:"conn_max_lifetime"`
	} `koanf:"mysql"`

	Redis struct {
		Addr     string `koanf:"addr"`
		Password string `koanf:"password"`
	} `koanf:"redis"`

	Idempotency struct {
		TTL time.Duration `koanf:"ttl"`
	} `koanf:"idempotency"`

	Rabbit struct {
		URL        string `koanf:"url"`
		Exchange   string `koanf:"exchange"`
		RoutingKey string `koanf:"routing_key"`
	} `koanf:"rabbitmq"`

	Kafka struct {
		Brokers     []string `koanf:"brokers"`
		TopicEvents string   `koanf:"topic_events"`
	} `koanf:"kafka"`

	Security struct {
		JWTSecret string        `koanf:"jwt_secret"`
		Issuer    string        `koanf:"issuer"`
		Audience  string        `koanf:"audience"`
		TTL       time.Duration `koanf:"ttl"`
	} `koanf:"security"`

	CryptoConfig struct {
		KeyID     string `koanf:"key_id"`
		AES256B64 string `koanf:"aes256_b64url"`
		RSAPubPEM string `koanf:"rsa_pub_pem"`
		RSAPriPEM string `koanf:"rsa_pri_pem"`
	}
}

func Load(pathDir, envName string) (Config, error) {
	k := koanf.New(".")
	// 1) base
	if err := k.Load(file.Provider(fmt.Sprintf("%s/base.yaml", pathDir)), yaml.Parser()); err != nil {
		return Config{}, fmt.Errorf("load base: %w", err)
	}

	// 2) env override (dev/staging/prod). Optional: allow missing for local runs.
	_ = k.Load(file.Provider(fmt.Sprintf("%s/%s.yaml", pathDir, envName)), yaml.Parser())

	// 3) environment variables override (prefix ORDERAPI_, nested with __)
	// e.g. ORDERAPI_MYSQL__DSN, ORDERAPI_REDIS__PASSWORD
	if err := k.Load(env.Provider("ORDERAPI_", ".", func(s string) string {
		s = strings.TrimPrefix(s, "ORDERAPI_")
		s = strings.ReplaceAll(s, "__", ".")
		return strings.ToLower(s)
	}), nil); err != nil {
		return Config{}, fmt.Errorf("env overlay: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.App.HTTPAddr == "" {
		return fmt.Errorf("app.http_addr required")
	}
	if c.MySQL.DSN == "" {
		return fmt.Errorf("mysql.dsn required")
	}
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("kafka.brokers required (can be dummy for now)")
	}
	return nil
}
