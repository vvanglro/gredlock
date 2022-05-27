## gredlock

RedLock in Golang

## Installation

```shell
go get -u github.com/vvanglro/gredlock@v0.1.0
```

## Quickstart
```go
package main

import (
	"context"
	"fmt"
	red "github.com/go-redis/redis/v8"
	"github.com/vvanglro/gredlock/redlock"
)

func main() {
	ctx := context.Background()
	options := []*red.Options{
		{
			Network: "tcp",
			Addr:    "localhost:56379",
		},
		{
			Network: "tcp",
			Addr:    "localhost:56378",
		},
	}

	locker := redlock.NewRedisLock(ctx, options...)
	lock, err := locker.SetLock(ctx, "my-key", "123", 50)
	fmt.Println(lock)
	fmt.Println(err)
	ttl, err := locker.GetLockTtl(ctx, "my-key", "123")
	fmt.Println(ttl)
	fmt.Println(err)
	isLock := locker.IsLocked(ctx, "my-key")
	fmt.Println(isLock)
	unlock, err := locker.UnSetLock(ctx, "my-key", "123")
	fmt.Println(unlock)
	fmt.Println(err)
}
```