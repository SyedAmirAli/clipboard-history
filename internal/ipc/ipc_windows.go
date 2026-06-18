//go:build windows

package ipc

import (
	"fmt"
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

// Listener owns a Windows named-pipe server for IPC. The Go standard library's
// net package has no "pipe" network, so we use go-winio (already pulled in via
// Wails' dependency tree) to create a real named pipe.
type Listener struct {
	listener net.Listener
	done     chan struct{}
}

// Listen creates a named-pipe listener (e.g. \\.\pipe\clipd) and dispatches each
// received command string to handler.
func Listen(pipePath string, handler func(string)) (*Listener, error) {
	listener, err := winio.ListenPipe(pipePath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create named pipe listener: %w", err)
	}
	ln := &Listener{listener: listener, done: make(chan struct{})}
	go ln.acceptLoop(handler)
	return ln, nil
}

func (ln *Listener) acceptLoop(handler func(string)) {
	defer close(ln.done)
	for {
		conn, err := ln.listener.Accept()
		if err != nil {
			return // listener closed
		}
		go func(c net.Conn) {
			defer c.Close()
			_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 256)
			n, err := c.Read(buf)
			if err != nil {
				return
			}
			handler(string(buf[:n]))
		}(conn)
	}
}

// Close stops the listener and waits for the accept loop to finish.
func (ln *Listener) Close() error {
	if ln.listener != nil {
		ln.listener.Close()
		<-ln.done
	}
	return nil
}

// Send connects to the running instance's pipe and writes a single command.
func Send(pipePath string, cmd string) error {
	timeout := 2 * time.Second
	conn, err := winio.DialPipe(pipePath, &timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to named pipe: %w", err)
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}
	return nil
}

// IsRunning reports whether a clipd instance is listening on the pipe.
func IsRunning(pipePath string) bool {
	timeout := 500 * time.Millisecond
	conn, err := winio.DialPipe(pipePath, &timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
