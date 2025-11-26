package handlers_test

import (
	"wealth-warden/internal/tests"
)

func init() {
	if err := tests.Setup(); err != nil {
		panic(err)
	}
}
