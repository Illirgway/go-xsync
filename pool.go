package xsync

import "sync"

// simple generics wrapper around sync.Pool with autonew on Get

type PoolT[V any] struct {
	p sync.Pool
}

/* TODO
type reseter interface {
	Reset()
}
*/

//go:nosplit
func (p *PoolT[V]) Get() (v *V) {

	v, ok := p.p.Get().(*V)

	if !ok || v == nil {
		v = new(V)
	}

	return v
}

//go:nosplit
func (p *PoolT[V]) Put(v *V) {

	if v == nil {
		return
	}

	/* TODO
	if x, ok := v.(reseter); ok {
		x.Reset()
	}
	*/

	p.p.Put(v)
}
