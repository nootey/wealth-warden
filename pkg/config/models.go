package config

type Config struct {
	Release    bool             `mapstructure:"release"`
	WebClient  WebClientConfig  `mapstructure:"web_client"`
	HttpServer HttpServerConfig `mapstructure:"http_server"`
	Host       string           `mapstructure:"host"`
	Postgres   PostgresConfig   `mapstructure:"postgres"`
	JWT        JWTConfig
	CORS       CorsConfig `mapstructure:"cors"`
	Seed       SeedConfig `mapstructure:"seed"`
}

type WebClientConfig struct {
	Domain string `mapstructure:"domain" validate:"required"`
}

type HttpServerConfig struct {
	Port string `mapstructure:"port"  validate:"required"`
}

type PostgresConfig struct {
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

type CorsConfig struct {
	AllowedOrigins   []string `mapstructure:"allowedOrigins" validate:"required"`
	WildcardSuffixes []string `mapstructure:"wildcardSuffixes"`
	AllowedSchemes   []string `mapstructure:"allowedSchemes" validate:"required"`
}

type SeedConfig struct {
	SuperAdminEmail    string `mapstructure:"super_admin_email"`
	SuperAdminPassword string `mapstructure:"super_admin_password"`
	MemberUserEmail    string `mapstructure:"member_user_email"`
	MemberUserPassword string `mapstructure:"member_user_password"`
}
