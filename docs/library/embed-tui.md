---
title: Embed TUI
description: Run the interactive terminal UI from another Go CLI.
---

# Embed TUI

The `github.com/cloudapp3/vminfo/tui` package lets you launch the same interactive UI from your own CLI.

## Example

```go
package main

import (
	"context"
	"log"

	vminfotui "github.com/cloudapp3/vminfo/tui"
)

func main() {
	if err := vminfotui.Run(context.Background(), vminfotui.Options{Lang: "en"}); err != nil {
		log.Fatal(err)
	}
}
```

## Options

| Field | Description |
| --- | --- |
| `Stdout` | Terminal output stream |
| `Stdin` | Keyboard input stream |
| `Lang` | UI language, such as `en` or `zh` |

`tui.Options` also works with custom streams, which is useful for embedded CLIs and tests.
