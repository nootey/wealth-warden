package config

type Config struct {
	Release           bool             `mapstructure:"release"`
	FinanceAPIBaseURL string           `mapstructure:"finance_api_base_url"`
	WebClient         WebClientConfig  `mapstructure:"web_client"`
	HttpServer        HttpServerConfig `mapstructure:"http_server"`
	Host              string           `mapstructure:"host"`
	Postgres          PostgresConfig   `mapstructure:"postgres"`
	JWT               JWTConfig        `mapstructure:"jwt"`
	CORS              CorsConfig       `mapstructure:"cors"`
	Seed              SeedConfig       `mapstructure:"seed"`
	Mailer            MailerConfig     `mapstructure:"mailer"`
}

type WebClientConfig struct {
	Domain string `mapstructure:"domain"`
	Port   string `mapstructure:"port"`
}

type HttpServerConfig struct {
	Port       string `mapstructure:"port"`
	ReqTimeout int    `mapstructure:"request_timeout"`
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
	AllowedOrigins   []string `mapstructure:"allowedOrigins"`
	WildcardSuffixes []string `mapstructure:"wildcardSuffixes"`
	AllowedSchemes   []string `mapstructure:"allowedSchemes"`
}

type SeedConfig struct {
	SuperAdminEmail    string `mapstructure:"super_admin_email" validate:"required"`
	SuperAdminPassword string `mapstructure:"super_admin_password" validate:"required"`
	MemberUserEmail    string `mapstructure:"member_user_email"`
	MemberUserPassword string `mapstructure:"member_user_password"`
}

type MailerConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}
