//go:build windows

package main

import (
	"context"

	"github.com/ZemarLi549/cc-connect-ultra/config"
)

func runRunAsUserStartupChecks(_ context.Context, _ *config.Config) error {
	return nil
}
