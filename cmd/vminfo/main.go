package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/cloudapp3/vminfo/internal/app"
)

func main() {
	if err := app.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "vminfo: %v\n", err)
		if errors.Is(err, app.ErrUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
