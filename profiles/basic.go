package profiles

import (
	"github.com/ugent-library/sip-creator/sip"
	"github.com/ugent-library/sip-creator/store"
)

// Basic builds a meemoo SIP 2.0 basic-profile package in two phases:
// assemble the complete package graph (no disk writes), then emit it in
// the canonical order. Assembly failures happen before the store exists,
// so a bad input leaves no partial package dir behind.
func (p *Profile) Basic() (*sip.Package, error) {
	p.Logger.Info("starting...")

	pkg, err := p.assemble()
	if err != nil {
		return nil, err
	}

	st := store.New(pkg.Location)
	if err := p.write(st, pkg); err != nil {
		return nil, err
	}

	p.Logger.Info("finished.")
	return pkg, nil
}
