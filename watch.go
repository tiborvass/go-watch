package watch

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/term/termios"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

const defaultTimeFormat = "Mon Jan _2 15:04:05 2006"

type Watcher struct {
	Interval   time.Duration
	Hostname   *string
	TimeFormat string
}

func (w Watcher) Watch(cmdArgs ...string) {
	// cbreak+noecho, keep ISIG (so Ctrl-C still works) ===
	fd := uintptr(os.Stdin.Fd())
	var oldState unix.Termios
	if err := termios.Tcgetattr(fd, &oldState); err != nil {
		fmt.Fprintf(os.Stderr, "Tcgetattr error: %v\n", err)
		os.Exit(1)
	}
	rawState := oldState
	rawState.Lflag &^= unix.ICANON | unix.ECHO // disable canonical mode & echo
	rawState.Lflag |= unix.ISIG                // keep signal-generation
	if err := termios.Tcsetattr(fd, termios.TCSANOW, &rawState); err != nil {
		fmt.Fprintf(os.Stderr, "Tcsetattr error: %v\n", err)
		os.Exit(1)
	}
	defer termios.Tcsetattr(fd, termios.TCSANOW, &oldState)

	// switch to alternate screen buffer + hide cursor
	fmt.Print("\x1b[?1049h\x1b[?25l")
	// ensure we restore both on exit
	defer fmt.Print("\x1b[?25h\x1b[?1049l")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH)

	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	timeFormat := w.TimeFormat
	if timeFormat == "" {
		timeFormat = defaultTimeFormat
	}

	hostname := ""
	if w.Hostname != nil {
		hostname = *w.Hostname
	} else {
		hostname, _ = os.Hostname()
	}

	redraw := func() {
		var buf bytes.Buffer
		// Clear screen & home
		buf.WriteString("\x1b[H\x1b[J")

		width, height, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			width, height = 80, 24
		}

		headerLeft := fmt.Sprintf("Every %.1fs: %s",
			w.Interval.Truncate(time.Second/10).Seconds(),
			strings.Join(cmdArgs, " "),
		)
		headerRight := time.Now().Format(timeFormat)
		if hostname != "" {
			headerRight = fmt.Sprintf("%s: %s", hostname, headerRight)
		}
		x := width - len(headerLeft)
		if x < 0 {
			x = 0
		}
		fmt.Fprintf(&buf, "%s%*s\n\n", headerLeft, x, headerRight)

		r, w, err := os.Pipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "pipe error: %v\n", err)
			return
		}
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = w
		cmd.Stderr = w

		if err := cmd.Start(); err != nil {
			w.Close()
			r.Close()
			fmt.Fprintf(os.Stderr, "start error: %v\n", err)
			return
		}
		w.Close()

		scanner := bufio.NewScanner(r)
		for i := 0; i < height-3 && scanner.Scan(); i++ {
			line := scanner.Bytes()
			if len(line) > 0 {
				if len(line) > width {
					buf.Write(line[:width])
				} else {
					buf.Write(line)
				}
				buf.WriteByte('\n')
			}
			if err != nil {
				if err != io.EOF {
					fmt.Fprintf(os.Stderr, "read error: %v\n", err)
				}
				break
			}
		}
		r.Close()
		cmd.Wait()

		os.Stdout.Write(buf.Bytes())
	}

	redraw()

	for {
		select {
		case sig := <-sigs:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				return
			case syscall.SIGWINCH:
				redraw()
			}
		case <-ticker.C:
			redraw()
		}
	}
}
