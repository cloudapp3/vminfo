---
title: Collect Metrics
description: Use vminfo from Go code to collect host metadata and runtime metrics.
---

# Collect Metrics

Use the root package to collect host data from Go.

## Minimal example

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudapp3/vminfo"
)

func main() {
	ctx := context.Background()

	static, _ := vminfo.CollectStatic(ctx)
	stats, _ := vminfo.CollectStats(ctx, vminfo.Options{SampleInterval: time.Second})

	fmt.Println(static.Hostname, stats.CPU)
}
```

## Options

| Field | Description |
| --- | --- |
| `SampleInterval` | Sampling duration used by `CollectStats` and `CollectAll` |

## Notes

- `CollectAll` returns both `StaticInfo` and `RuntimeStats`.
- `CollectStats` uses a cached static snapshot and fresh dynamic samples.
- `SampleInterval` defaults to one second when left unset.

## Related

- [Go Library](/library/)
- [HTTP API](/api)
