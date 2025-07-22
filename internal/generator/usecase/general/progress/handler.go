package progress

import (
	"sync"

	"github.com/tarantool/sdvg/internal/generator/usecase"
)

// Handler type is storage for progresses.
type Handler struct {
	progresses map[string]*usecase.Progress
	mutex      *sync.RWMutex
}

func NewHandler() *Handler {
	return &Handler{
		progresses: make(map[string]*usecase.Progress),
		mutex:      &sync.RWMutex{},
	}
}

// Create function creates struct for progress by name.
func (p *Handler) Create(name string, total uint64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if total == 0 {
		return
	}

	p.progresses[name] = &usecase.Progress{
		Done:  0,
		Total: total,
	}
}

// Set function sets progress to selected value by name.
func (p *Handler) Set(name string, done uint64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	progress := p.progresses[name]
	progress.Done = done
}

// Add function add selected value to progress by name.
func (p *Handler) Add(name string, done uint64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	progress := p.progresses[name]
	progress.Done += done
}

// GetAll returns all saved progresses.
func (p *Handler) GetAll() map[string]usecase.Progress {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	results := make(map[string]usecase.Progress, len(p.progresses))
	for name, progress := range p.progresses {
		results[name] = *progress
	}

	return results
}
