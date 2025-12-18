package constants

import "time"

const (
	AccessCookieTTL       = 10 * time.Minute
	RefreshCookieTTLShort = 24 * time.Hour     // when remember=false
	RefreshCookieTTLLong  = 7 * 24 * time.Hour // when remember=true
)
