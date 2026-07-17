package config

type Config struct {
	Release           bool             `mapstructure:"release"`
	FinanceAPIBaseURL string           `mapstructure:"finance_api_base_url"`
	WebClient         WebClientConfig  `mapstructure:"web_client"`
	HttpServer        HttpServerConfig `mapstructure:"http_server"`
	Host              string           `mapstructure:"host"`
	Postgres          PostgresConfig   `mapstructure:"postgres"`
	Redis             RedisConfig      `mapstructure:"redis"`
	Session           SessionConfig    `mapstructure:"session"`
	CORS              CorsConfig       `mapstructure:"cors"`
	Seed              SeedConfig       `mapstructure:"seed"`
	Mailer            MailerConfig     `mapstructure:"mailer"`
	Scheduler         SchedulerConfig  `mapstructure:"scheduler"`
	Otel              OtelConfig       `mapstructure:"otel"`
	Queue             QueueConfig      `mapstructure:"queue"`
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

type RedisConfig struct {
	Host     string `mapstructure:"host" validate:"required,hostname|ip"`
	Port     int    `mapstructure:"port" validate:"required"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type SessionConfig struct {
	TTLHours           int `mapstructure:"ttl_hours" validate:"required,min=1"`
	RememberMeTTLHours int `mapstructure:"remember_me_ttl_hours" validate:"required,min=1"`
	MaxLifetimeHours   int `mapstructure:"max_lifetime_hours" validate:"required,min=1"`
}

type CorsConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	WildcardSuffixes []string `mapstructure:"wildcard_suffixes"`
	AllowedSchemes   []string `mapstructure:"allowed_schemes"`
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

type SchedulerConfig struct {
	ImmediateJobs     []string `mapstructure:"immediate_jobs"`
	ConcurrentWorkers int      `mapstructure:"concurrent_workers"`
}

type OtelConfig struct {
	OTLPEndpoint string `mapstructure:"otlp_endpoint"`
	ServiceName  string `mapstructure:"service_name"`
}

type QueueConfig struct {
	Workers                   int `mapstructure:"workers"`
	MaxAttempts               int `mapstructure:"max_attempts"`
	PollIntervalMs            int `mapstructure:"poll_interval_ms"`
	RetryInitialBackoffSec    int `mapstructure:"retry_initial_backoff_sec"`
	RetrySubsequentBackoffSec int `mapstructure:"retry_subsequent_backoff_sec"`
	VisibilityTimeoutSec      int `mapstructure:"visibility_timeout_sec"` // reclaims jobs stuck in 'processing' after a crash.
}
