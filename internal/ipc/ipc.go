// Package ipc provides a tiny Unix-domain-socket control channel so that a
// second `clipd` invocation can drive the already-running instance — e.g.
// `clipd toggle` to pop the window open. This is the command-line alternative
// to the global Super+V hotkey, useful where X11 hotkey grabs don't work
// (Wayland, WSLg, remote sessions, or when the user just prefers a command).
package ipc

import (
	"bufio"
	"errors"
	"net"
	"os"
	"strings"
	"time"
)

// Commands understood by the listener.
const (
	CmdToggle = "toggle"
	CmdShow   = "show"
	CmdHide   = "hide"
	CmdQuit   = "quit"
)

// ValidCommand reports whether cmd is a recognised control command.
func ValidCommand(cmd string) bool {
	switch cmd {
	case CmdToggle, CmdShow, CmdHide, CmdQuit:
		return true
	default:
		return false
	}
}

// IsRunning reports whether a clipd instance is currently listening on the
// socket at path.
func IsRunning(path string) bool {
	conn, err := net.DialTimeout("unix", path, 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// Send dials the control socket at path and delivers cmd to a running
// instance. It returns an error if no instance is listening (i.e. the socket
// is absent or stale), which the caller can use to decide whether to launch.
func Send(path, cmd string) error {
	conn, err := net.DialTimeout("unix", path, 500*time.Millisecond)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
	if _, err := conn.Write([]byte(cmd + "\n")); err != nil {
		return err
	}
	return nil
}

// Listener wraps the accepting socket so it can be cleanly closed on shutdown.
type Listener struct {
	ln   net.Listener
	path string
}

// Listen starts accepting control connections on the socket at path and
// invokes handler with each received command. A pre-existing (stale) socket
// file is removed first. The accept loop runs in its own goroutine; callers
// should defer Close.
func Listen(path string, handler func(cmd string)) (*Listener, error) {
	// Clear a stale socket left behind by a previous crash. If a live
	// instance owned it, the caller's earlier Send would have succeeded and
	// we would never reach this point.
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		// Not fatal on its own — net.Listen will surface a real conflict.
	}

	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	l := &Listener{ln: ln, path: path}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				line, err := bufio.NewReader(c).ReadString('\n')
				if err != nil && line == "" {
					return
				}
				cmd := strings.TrimSpace(line)
				if ValidCommand(cmd) {
					handler(cmd)
				}
			}(conn)
		}
	}()

	return l, nil
}

// Close stops accepting connections and removes the socket file.
func (l *Listener) Close() {
	if l == nil {
		return
	}
	if l.ln != nil {
		_ = l.ln.Close()
	}
	_ = os.Remove(l.path)
}
