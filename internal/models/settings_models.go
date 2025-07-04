package models

type Config struct {
	Release    bool             `mapstructure:"release"`
	WebClient  WebClientConfig  `mapstructure:"web_client"`
	HttpServer HttpServerConfig `mapstructure:"http_server"`
	Host       string           `mapstructure:"host"`
	MySQL      MySQLConfig      `mapstructure:"mysql"`
	JWT        JWTConfig
}

type WebClientConfig struct {
	Domain string `mapstructure:"domain" validate:"required"`
	Port   string `mapstructure:"port" validate:"required"`
}

type HttpServerConfig struct {
	Port string `mapstructure:"port"  validate:"required"`
}

type MySQLConfig struct {
	Host     string `validate:"required,hostname|ip"`
	Port     int    `validate:"required"`
	Database string `validate:"required"`
	User     string `validate:"required"`
	Password string `validate:"required"`
}

type JWTConfig struct {
	WebClientAccess   string `validate:"required"`
	WebClientRefresh  string `validate:"required"`
	WebClientEncodeID string `validate:"required"`
}
