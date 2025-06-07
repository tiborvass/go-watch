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

const (
	DefaultTimeFormat = "Mon Jan _2 15:04:05 2006"
	DefaultInterval   = 2 * time.Second
)

// Watcher allows to set options.
// Zero values are the default.
type Watcher struct {
	// Interval represents the time interval between two loops of command execution.
	// If left zero, it is set to DefaultInterval
	Interval time.Duration
	// If NoTitle is set, no title header is displayed
	NoTitle bool
	// If Exec is set, the commands are executed without an `sh -c` wrapper.
	Exec bool
	// Hostname is the hostname displayed in the header.
	// If nil, it is set to os.Hostname()
	Hostname *string
	// TimeFormat is the format displayed in the header.
	// If empty, it is set to DefaultTimeFormat.
	TimeFormat string
}

// Watch executes the commands passed in cmdArgs in a loop honoring the options defined in Watcher.
func (w Watcher) Watch(cmdArgs ...string) {
	if w.Interval == 0 {
		w.Interval = DefaultInterval
	}
	if w.TimeFormat == "" {
		w.TimeFormat = DefaultTimeFormat
	}
	hostname := ""
	if w.Hostname != nil {
		hostname = *w.Hostname
	} else {
		hostname, _ = os.Hostname()
	}

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

	redraw := func() {
		var buf bytes.Buffer
		// Clear screen & home
		buf.WriteString("\x1b[H\x1b[J")

		width, height, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			width, height = 80, 24
		}

		delta := 0

		if !w.NoTitle {
			delta = 3
			headerLeft := fmt.Sprintf("Every %.1fs: %s",
				w.Interval.Truncate(time.Second/10).Seconds(),
				strings.Join(cmdArgs, " "),
			)
			headerRight := time.Now().Format(w.TimeFormat)
			if hostname != "" {
				headerRight = fmt.Sprintf("%s: %s", hostname, headerRight)
			}
			x := width - len(headerLeft)
			if x < 0 {
				x = 0
			}
			fmt.Fprintf(&buf, "%s%*s\n\n", headerLeft, x, headerRight)
		}

		pr, pw, err := os.Pipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "pipe error: %v\n", err)
			return
		}
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		if !w.Exec {
			cmd = exec.Command("/bin/sh", "-c", strings.Join(cmdArgs, " "))
		}
		cmd.Stdout = pw
		cmd.Stderr = pw

		if err := cmd.Start(); err != nil {
			pw.Close()
			pr.Close()
			fmt.Fprintf(os.Stderr, "start error: %v\n", err)
			return
		}
		pw.Close()

		scanner := bufio.NewScanner(pr)
		for i := 0; i < height-delta && scanner.Scan(); i++ {
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
		pr.Close()
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
