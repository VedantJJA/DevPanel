package docker

import "sync"

// BuildLogEntry is a single build/deploy log line stored in the in-process buffer.
// It is distinct from LogEntry (which is Docker's runtime container log format).
type BuildLogEntry struct {
	Timestamp string `json:"timestamp"`
	Stage     string `json:"stage"`   // "init", "detect", "generate", "build", "deploy"
	Service   string `json:"service"`
	Message   string `json:"message"`
	Level     string `json:"level"` // "info", "warn", "error", "success"
}

// LogBuffer is a thread-safe circular buffer for build/deploy logs per service.
type LogBuffer struct {
	mu      sync.RWMutex
	entries []BuildLogEntry
	maxSize int
}

// NewLogBuffer creates a new log buffer with a maximum capacity.
func NewLogBuffer(capacity int) *LogBuffer {
	if capacity <= 0 {
		capacity = 5000
	}
	return &LogBuffer{
		entries: make([]BuildLogEntry, 0, capacity),
		maxSize: capacity,
	}
}

// Append adds a build log entry, dropping the oldest 20 % when full.
func (lb *LogBuffer) Append(entry BuildLogEntry) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if len(lb.entries) >= lb.maxSize {
		drop := lb.maxSize / 5
		lb.entries = lb.entries[drop:]
	}
	lb.entries = append(lb.entries, entry)
}

// AppendLine is a convenience wrapper around Append.
func (lb *LogBuffer) AppendLine(stage, service, message, level string) {
	lb.Append(BuildLogEntry{
		Timestamp: nowRFC3339(),
		Stage:     stage,
		Service:   service,
		Message:   message,
		Level:     level,
	})
}

// GetAll returns a snapshot of all log entries.
func (lb *LogBuffer) GetAll() []BuildLogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	out := make([]BuildLogEntry, len(lb.entries))
	copy(out, lb.entries)
	return out
}

// Len returns the current number of stored entries.
func (lb *LogBuffer) Len() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return len(lb.entries)
}

// Clear resets the buffer. MUST be called before every new build so the
// frontend receives a {"type":"clear"} signal via WebSocket.
func (lb *LogBuffer) Clear() {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.entries = lb.entries[:0]
}

// ---------------------------------------------------------------------------
// LogManager — per-project log buffer registry
// ---------------------------------------------------------------------------

// LogManager manages one LogBuffer per project/service.
type LogManager struct {
	mu      sync.RWMutex
	buffers map[string]*LogBuffer
}

// NewLogManager initialises a LogManager.
func NewLogManager() *LogManager {
	return &LogManager{
		buffers: make(map[string]*LogBuffer),
	}
}

// GetOrCreate returns the existing buffer for a project, or creates one.
func (lm *LogManager) GetOrCreate(projectID string) *LogBuffer {
	lm.mu.RLock()
	buf, ok := lm.buffers[projectID]
	lm.mu.RUnlock()
	if ok {
		return buf
	}
	lm.mu.Lock()
	defer lm.mu.Unlock()
	// Double-check after acquiring write lock.
	if buf, ok = lm.buffers[projectID]; ok {
		return buf
	}
	buf = NewLogBuffer(5000)
	lm.buffers[projectID] = buf
	return buf
}

// ClearProject clears the log buffer for a project (called on new deploy).
func (lm *LogManager) ClearProject(projectID string) {
	lm.mu.RLock()
	buf, ok := lm.buffers[projectID]
	lm.mu.RUnlock()
	if ok {
		buf.Clear()
	}
}

// RemoveProject removes the buffer for a project entirely.
func (lm *LogManager) RemoveProject(projectID string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	delete(lm.buffers, projectID)
}

// globalLogManager is the process-wide singleton used by HTTP handlers.
var globalLogManager = NewLogManager()
