package discovery

import "sync"

// PendingResumes tracks sessions being resumed. When gmuxr registers a new
// session, we check if it matches a pending resume (by cwd+kind) and merge
// it into the existing store entry instead of creating a new one.
type PendingResumes struct {
	mu sync.Mutex
	// key: "cwd|kind" → store session ID (the file-xxx entry)
	pending map[string]string
}

func NewPendingResumes() *PendingResumes {
	return &PendingResumes{pending: make(map[string]string)}
}

// Add records that a resumable session is being resumed.
func (pr *PendingResumes) Add(cwd, kind, sessionID string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.pending[cwd+"|"+kind] = sessionID
}

// Take checks if there's a pending resume for the given cwd+kind.
// If found, removes it and returns the existing session ID.
func (pr *PendingResumes) Take(cwd, kind string) (string, bool) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	key := cwd + "|" + kind
	id, ok := pr.pending[key]
	if ok {
		delete(pr.pending, key)
	}
	return id, ok
}
