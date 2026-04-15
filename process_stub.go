//go:build !linux

package vminfo

import (
	"context"
	"fmt"
	"runtime"
)

func listProcesses(context.Context) ([]ProcessInfo, error) {
	return nil, fmt.Errorf("local process view unsupported on %s", runtime.GOOS)
}

func terminateProcess(context.Context, int32) error {
	return fmt.Errorf("local process terminate unsupported on %s", runtime.GOOS)
}
