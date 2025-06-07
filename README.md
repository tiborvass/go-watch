# watch binary

```
% go install github.com/tiborvass/go-watch/cmd/watch@latest
% $(go env GOPATH)/bin/watch -n 1 date -Ins
```

# go-watch library

## Godoc

[Godoc](https://pkg.go.dev/github.com/tiborvass/go-watch)

## Direct exec

```go
w := watch.Watcher{Interval: 500 * time.Millisecond}
w.Watch(ctx, "date", "-Ins")
```

## Wrapped in sh -c

```go
w := watch.Watcher{Interval: 500 * time.Millisecond}
w.WatchShell(ctx, "date -Ins")
```
