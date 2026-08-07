// Package gameserver manages the dedicated-server port pool and spawns the
// headless Godot process for each match.
package gameserver

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"

	"openfire-server/internal/config"
)

// ExitFunc is invoked once when a gameserver process exits. The port it used
// has already been returned to the pool by the time ExitFunc runs.
type ExitFunc func(matchID string, exitCode int)

// Manager owns the port pool and the running game-server processes.
type Manager struct {
	cfg   config.Config
	log   *slog.Logger
	mu    sync.Mutex
	ports map[int]bool // true = free
	procs map[string]*process
}

type process struct {
	cmd  *exec.Cmd
	port int
}

// NewManager builds a Manager and seeds the port pool from config.
func NewManager(cfg config.Config, log *slog.Logger) *Manager {
	m := &Manager{
		cfg:   cfg,
		log:   log,
		ports: make(map[int]bool),
		procs: make(map[string]*process),
	}
	for p := cfg.GameserverPortMin; p <= cfg.GameserverPortMax; p++ {
		m.ports[p] = true
	}
	return m
}

// AllocatePort reserves and returns a free port, or an error if none are free.
func (m *Manager) AllocatePort() (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for p, free := range m.ports {
		if free {
			m.ports[p] = false
			return p, nil
		}
	}
	return 0, fmt.Errorf("no free game-server ports available")
}

// ReleasePort returns a port to the pool. Safe to call even if not tracked.
func (m *Manager) ReleasePort(p int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.ports[p]; ok {
		m.ports[p] = true
	}
}

// Start launches the headless Godot binary for a match. On success a goroutine
// watches the process: when it exits the port is released and onExit is called.
// If the binary does not exist (common in dev) a warning is logged and an error
// is returned without crashing the central server.
func (m *Manager) Start(matchID, mode string, port int, onExit ExitFunc) error {
	binary := m.cfg.GameserverBinary
	if binary == "" {
		return fmt.Errorf("gameserver binary not configured")
	}
	if _, err := os.Stat(binary); err != nil {
		m.log.Warn("gameserver binary not found; cannot start dedicated server (dev mode)",
			"binary", binary, "match_id", matchID, "err", err)
		return fmt.Errorf("gameserver binary not found: %w", err)
	}

	cmd := exec.Command(binary, "--headless")
	cmd.Env = append(os.Environ(),
		"OPENFIRE_DEDICATED=1",
		fmt.Sprintf("OPENFIRE_PORT=%d", port),
		fmt.Sprintf("OPENFIRE_MATCH_ID=%s", matchID),
		fmt.Sprintf("OPENFIRE_CENTRAL_URL=%s", m.cfg.CentralURL),
		fmt.Sprintf("OPENFIRE_INTERNAL_TOKEN=%s", m.cfg.InternalToken),
		fmt.Sprintf("OPENFIRE_MAX_PLAYERS=%d", m.cfg.GameserverMaxPlayers),
		fmt.Sprintf("OPENFIRE_MODE=%s", mode),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start gameserver: %w", err)
	}

	p := &process{cmd: cmd, port: port}
	m.mu.Lock()
	m.procs[matchID] = p
	m.mu.Unlock()

	m.log.Info("gameserver started", "match_id", matchID, "pid", cmd.Process.Pid, "port", port, "mode", mode)

	go m.watch(matchID, p, onExit)
	return nil
}

func (m *Manager) watch(matchID string, p *process, onExit ExitFunc) {
	err := p.cmd.Wait()
	exitCode := 0
	if err != nil {
		exitCode = 1
		m.log.Warn("gameserver process exited with error", "match_id", matchID, "err", err)
	} else {
		m.log.Info("gameserver process exited", "match_id", matchID)
	}
	m.mu.Lock()
	if cur, ok := m.procs[matchID]; ok && cur == p {
		delete(m.procs, matchID)
	}
	m.ports[p.port] = true
	m.mu.Unlock()
	if onExit != nil {
		onExit(matchID, exitCode)
	}
}

// Stop kills the process for a match, if still running. The watch goroutine
// then performs port release and exit notification. Idempotent.
func (m *Manager) Stop(matchID string) {
	m.mu.Lock()
	p := m.procs[matchID]
	m.mu.Unlock()
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return
	}
	_ = p.cmd.Process.Kill()
}
