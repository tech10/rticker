This package provides a ticker that can be reset, stopped, and closed.

I have come across situations where I would like to use a ticker, but I don't want to signal cancellation or closure with another channel or context. This package solves that problem, wrapping a time.Timer in a custom struct that acts as a ticker. When the custom ticker is stopped, any ticks are paused, but the read only channel remains open. When it is reset, ticks will begin again, or if reset while ticks are occuring, they will begin at the new duration added to the current time. If the ticker is closed, the associated receive channel is also closed, allowing any for range loops over the channel to terminate, helping to prevent leaking goroutines without an external mechanism to do so.

## A simple example

```go
package main

import (
	"fmt"
	"time"

	"github.com/tech10/rticker"
)

func main() {
	t := rticker.New(200 * time.Millisecond)

	count := 0
	for range t.C {
		count++
		fmt.Println("tick", count)
		if count == 3 {
			t.Close() // breaks the loop
		}
	}
	fmt.Println("Ticker closed.")
}
```

# Docs

Read the complete [documentation for this package](https://pkg.go.dev/github.com/tech10/rticker) for full examples and a complete overview on how each function works.