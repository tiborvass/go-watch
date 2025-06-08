# go-watch

<a href="https://opensource.org/licenses/Apache-2.0">
    <img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg">
</a>
<a href="https://pkg.go.dev/github.com/tiborvass/go-watch#section-documentation" rel="nofollow"><img src="https://pkg.go.dev/badge/github.com/tiborvass/go-watch" alt="Documentation"></a>

Simple Go implementation of the `watch` UNIX command, both as a binary and a library.

## binary

```
% go install github.com/tiborvass/go-watch/cmd/watch@latest
% $(go env GOPATH)/bin/watch -n 1 date -Ins
```

## library

[Package Documentation](https://pkg.go.dev/github.com/tiborvass/go-watch#section-documentation)

### Examples

#### Direct exec

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
w := watch.Watcher{Interval: 500 * time.Millisecond}
w.Watch(ctx, "date", "-Ins")
```

#### Wrapped in sh -c

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
w := watch.Watcher{Interval: 500 * time.Millisecond}
w.WatchShell(ctx, "date -Ins")
```
