package constants

import "time"

const (
	AccessCookieTTL       = 15 * time.Minute
	RefreshCookieTTLShort = 24 * time.Hour      // when remember=false
	RefreshCookieTTLLong  = 14 * 24 * time.Hour // when remember=true
)
