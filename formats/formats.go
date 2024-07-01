package formats

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ugent-library/sip-creator/sip"
)

type Identificator interface {
	Process(string) *sip.File
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
		return nil, fmt.Errorf("uknown format identificator '%s'", name)
	}

	return factory(bin, strings.Split(args, " "))
}
