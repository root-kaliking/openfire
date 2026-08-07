// Package gs manages Godot headless dedicated-server subprocesses: a
// bounded port pool and process lifecycle. When a match is created the
// central server launches a Godot process with the match's port/id
// passed via environment variables; when the process exits, the match
// is marked abandoned unless the GS already reported a finished result.
package gs

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// Manager owns the GS port pool and running processes.
type Manager struct {
	godotBin      string
	projectPath   string
	centralURL    string
	internalToken string
	portMin       int
	portMax       int
	logsDir       string

	mu        sync.Mutex
	usedPorts map[int]bool
	procs     map[string]*os.Process // matchID -> process
	db        *sql.DB
}

// New constructs a Manager. db is used to mark matches abandoned when a
// process exits without reporting a result.
func New(db *sql.DB, godotBin, projectPath, centralURL, internalToken string, portMin, portMax int, logsDir string) *Manager {
	return &Manager{
		godotBin:      godotBin,
		projectPath:   projectPath,
		centralURL:    centralURL,
		internalToken: internalToken,
		portMin:       portMin,
		portMax:       portMax,
		logsDir:       logsDir,
		usedPorts:     make(map[int]bool),
		procs:         make(map[string]*os.Process),
		db:            db,
	}
}

// StartMatch allocates a free port and launches a Godot headless
// dedicated-server process for the given match id. It returns the
// allocated port.
func (m *Manager) StartMatch(matchID string) (int, error) {
	port, err := m.allocatePort()
	if err != nil {
		return 0, err
	}
	if err := m.launch(matchID, port); err != nil {
		m.releasePort(port)
		return 0, err
	}
	return port, nil
}

// Stop kills the process for a match (best-effort), if still running.
func (m *Manager) Stop(matchID string) {
	m.mu.Lock()
	p := m.procs[matchID]
	m.mu.Unlock()
	if p != nil {
		_ = p.Kill()
	}
}

func (m *Manager) allocatePort() (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for p := m.portMin; p <= m.portMax; p++ {
		if !m.usedPorts[p] {
			m.usedPorts[p] = true
			return p, nil
		}
	}
	return 0, fmt.Errorf("no free GS port in range %d-%d", m.portMin, m.portMax)
}

func (m *Manager) releasePort(p int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.usedPorts, p)
}

func (m *Manager) launch(matchID string, port int) error {
	if err := os.MkdirAll(m.logsDir, 0o755); err != nil {
		return fmt.Errorf("create logs dir: %w", err)
	}
	logPath := filepath.Join(m.logsDir, fmt.Sprintf("gs-%s.log", matchID))
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("create gs log file: %w", err)
	}

	cmd := exec.Command(m.godotBin,
		"--headless",
		"--path", m.projectPath,
		"res://scenes/dedicated_server.tscn",
	)
	cmd.Dir = m.projectPath
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("OPENFIRE_GS_PORT=%d", port),
		fmt.Sprintf("OPENFIRE_GS_MATCH_ID=%s", matchID),
		fmt.Sprintf("OPENFIRE_INTERNAL_TOKEN=%s", m.internalToken),
		fmt.Sprintf("OPENFIRE_CENTRAL_URL=%s", m.centralURL),
	)
	// GS stdout/stderr go to its log file; stderr is also tee'd to the
	// central server's stderr for live debugging.
	cmd.Stdout = logFile
	cmd.Stderr = io.MultiWriter(logFile, os.Stderr)

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("start godot: %w", err)
	}

	m.mu.Lock()
	m.procs[matchID] = cmd.Process
	m.mu.Unlock()

	go func() {
		_ = cmd.Wait()
		logFile.Close()
		m.mu.Lock()
		delete(m.procs, matchID)
		m.mu.Unlock()
		m.releasePort(port)
		m.markAbandonedIfNotFinished(matchID)
	}()
	return nil
}

// markAbandonedIfNotFinished flips a still-running match to abandoned
// when its GS process exits without the GS having reported a result.
func (m *Manager) markAbandonedIfNotFinished(matchID string) {
	if m.db == nil {
		return
	}
	_, err := m.db.Exec(
		`UPDATE matches SET status='abandoned', finished_at=NOW()
		 WHERE id=$1 AND status NOT IN ('finished','abandoned')`,
		matchID)
	if err != nil {
		log.Printf("gs: failed to mark match %s abandoned: %v", matchID, err)
	}
}
