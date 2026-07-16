package formats

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ugent-library/sip-creator/sip"
)

// Identificator characterises files with an external format-identification
// tool. It is an optional enricher: fixity is computed natively by the
// store, so a build without an identificator is spec-valid (premis:format
// is a SHOULD).
type Identificator interface {
	// Identify characterises the file at path; (nil, nil) means the tool
	// ran but found no match.
	Identify(path string) (*sip.Format, error)
}

type Factory func(string, []string) (Identificator, error)

var factories = make(map[string]Factory)
var mu sync.RWMutex

func Register(name string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories[name] = factory
}

func New(name, bin, args string) (Identificator, error) {
	mu.RLock()
	factory, ok := factories[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown format identificator '%s'", name)
	}

	// An empty args string means no arguments, not one empty argument.
	var argv []string
	if args != "" {
		argv = strings.Split(args, " ")
	}

	return factory(bin, argv)
}
