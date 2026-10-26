package app

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Single-instance handoff (WL-2.1, plan §11): the first instance
// owns a Unix socket; later invocations forward validated URIs to it
// and exit (0 when a URI was handed off, else 3 = already-running).
// Stale sockets from a dead owner are reclaimed.

// ClaimInstance returns primary=true for the owner (with a channel of
// forwarded URI strings), or primary=false with the process exit code.
// stop closes the listener; call it on quit (primary only).
func ClaimInstance(uris []string) (primary bool, code int, incoming <-chan string, stop func()) {
	noop := func() {}
	path := sockPath()
	if conn, err := net.Dial("unix", path); err == nil {
		w := bufio.NewWriter(conn)
		for _, u := range uris {
			w.WriteString(u + "\n")
		}
		w.Flush()
		conn.Close()
		if len(uris) > 0 {
			return false, 0, nil, noop
		}
		return false, 3, nil, noop
	}
	os.Remove(path) // stale socket from a dead owner
	ln, err := net.Listen("unix", path)
	if err != nil {
		// Cannot listen (sandboxed runtime?): run unclaimed.
		return true, 0, nil, noop
	}
	ch := make(chan string, 8)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				s := bufio.NewScanner(c)
				for s.Scan() {
					if line := strings.TrimSpace(s.Text()); line != "" {
						ch <- line
					}
				}
			}(c)
		}
	}()
	return true, 0, ch, func() { ln.Close(); os.Remove(path) }
}

func sockPath() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "whatsapp-linux-"+strconv.Itoa(os.Getuid())+".sock")
}
