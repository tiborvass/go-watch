package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/tiborvass/go-watch"
)

func main() {
	n := flag.Float64("n", 2, "seconds between updates")
	flag.Parse()
	interval := time.Duration(math.Round(*n * float64(time.Second)))
	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [-n seconds] command [args...]\n", os.Args[0])
		os.Exit(2)
	}
	cmdArgs := flag.Args()

	w := watch.Watcher{Interval: interval}
	w.Watch(cmdArgs...)
}
