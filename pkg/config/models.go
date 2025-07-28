package config

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
	Host     string `mapstructure:"host" validate:"required,hostname|ip"`
	Port     int    `mapstructure:"port" validate:"required"`
	Database string `mapstructure:"db" validate:"required"`
	User     string `mapstructure:"user" validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
}

type JWTConfig struct {
	WebClientAccess   string `mapstructure:"web_client_access" validate:"required"`
	WebClientRefresh  string `mapstructure:"web_client_refresh" validate:"required"`
	WebClientEncodeID string `mapstructure:"web_client_encode_id" validate:"required"`
}
