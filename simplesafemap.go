package xsync

import (
	"sync"
)

// simple generic thread-safe map like sync.Map

type SimpleSafeMapT[K comparable, V any] struct {
	lock sync.RWMutex
	m    map[K]V
}

// thread-unsafe
//
//go:nosplit
func (m *SimpleSafeMapT[K, V]) init() {
	m.m = make(map[K]V)
}

//go:nosplit
func (m *SimpleSafeMapT[K, V]) Load(key K) (v V, ok bool) {

	m.lock.RLock()
	v, ok = m.m[key]
	m.lock.RUnlock()

	return v, ok
}

//go:nosplit
func (m *SimpleSafeMapT[K, V]) Store(key K, value V) {

	m.lock.Lock()
	defer m.lock.Unlock()

	if m.m == nil {
		m.init()
	}

	m.m[key] = value
}

func (m *SimpleSafeMapT[K, V]) tryLoadOrStore(key K, value V) (actual V, loaded bool) {

	m.lock.Lock()
	defer m.lock.Unlock()

	// should ignore absent map
	if loaded = m.m != nil; loaded {
		actual, loaded = m.m[key]
	} else { // !loaded === `m.m == nil`
		m.init()
	}

	if !loaded {
		m.m[key] = value
		actual, loaded = value, false
	}

	return actual, loaded
}

//go:nosplit
func (m *SimpleSafeMapT[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {

	// fast-path
	actual, loaded = m.Load(key)

	if !loaded {
		// slow-path
		actual, loaded = m.tryLoadOrStore(key, value)
	}

	return actual, loaded
}

//go:nosplit
func (m *SimpleSafeMapT[K, V]) LoadAndDelete(key K) (value V, loaded bool) {

	m.lock.Lock()
	defer m.lock.Unlock()

	// fast-path
	if len(m.m) == 0 /* implies `m.m == nil` */ {
		return value, false
	}

	value, loaded = m.m[key]

	// optimization: delete only if exists
	if loaded {
		delete(m.m, key)
	}

	return value, loaded
}

//go:nosplit
func (m *SimpleSafeMapT[K, V]) Delete(key K) {

	m.lock.Lock()
	defer m.lock.Unlock()

	if len(m.m) > 0 /* implies `m.m != nil` */ {
		delete(m.m, key)
	}
}

func (m *SimpleSafeMapT[K, V]) Swap(key K, value V) (previous V, loaded bool) {

	m.lock.Lock()
	defer m.lock.Unlock()

	// fetch prev
	// should ignore absent map
	if loaded = m.m != nil; loaded {
		previous, loaded = m.m[key]
	} else { // !loaded === `m.m == nil`
		m.init()
	}

	// store new
	m.m[key] = value

	return previous, loaded
}

/* TODO
func (m *SimpleSafeMapT[K, V]) CompareAndSwap(key K, old, new V) (swapped bool) {

	m.lock.Lock()
	defer m.lock.Unlock()

	// fast-check
	if len(m.m) == 0 { // implies `m.m == nil`
		return false
	}

	v, exists := m.m[key]

	if !exists { // if key not found
		return false
	}

	if swapped = v == old; swapped {
		m.m[key] = new
	}

	return swapped
}

func (m *SimpleSafeMapT[K, V]) CompareAndDelete(key K, old V) (deleted bool) {

	m.lock.Lock()
	defer m.lock.Unlock()

	// fast-check
	if len(m.m) == 0 { // implies `m.m == nil`
		return false
	}

	v, exists := m.m[key]

	if !exists {
		return false
	}

	if deleted = v == old; deleted {
		delete(m.m, key)
	}

	return deleted
}
*/

func (m *SimpleSafeMapT[K, V]) Range(f func(key K, value V) (next bool)) {

	m.lock.RLock()
	defer m.lock.RUnlock() // defer due to f

	// fast-check
	if len(m.m) == 0 /* implies `m.m == nil` */ {
		return
	}

	for k, v := range m.m {
		if !f(k, v) {
			break
		}
	}
}

func (m *SimpleSafeMapT[K, V]) RangeDelete(f func(key K, value V) (remove bool, next bool)) {

	m.lock.Lock()
	defer m.lock.Unlock() // defer due to f and delete

	// fast-check
	if len(m.m) == 0 /* implies `m.m == nil` */ {
		return
	}

	for k, v := range m.m {

		remove, next := f(k, v)

		if remove {
			delete(m.m, k)
		}

		if !next {
			break
		}
	}
}

//go:nosplit
func (m *SimpleSafeMapT[K, V]) Keys() (keys []K) {

	m.lock.RLock()
	defer m.lock.RUnlock() // defer due to memallocs

	// fast-check
	if len(m.m) == 0 /* implies `m.m == nil` */ {
		return nil
	}

	keys = make([]K, len(m.m))

	i := uint(0)

	for k := range m.m {
		keys[i] = k
		i++
	}

	return keys
}

//go:nosplit
func (m *SimpleSafeMapT[K, V]) Values() (values []V) {

	m.lock.RLock()
	defer m.lock.RUnlock() // defer due to memallocs

	// fast-check
	if len(m.m) == 0 /* implies `m.m == nil` */ {
		return nil
	}

	values = make([]V, len(m.m))

	i := uint(0)

	for _, v := range m.m {
		values[i] = v
		i++
	}

	return values
}

//go:nosplit
func (m *SimpleSafeMapT[K, V]) Clear() {

	m.lock.Lock()
	defer m.lock.Unlock()

	// fast-path
	if len(m.m) == 0 /* implies `m.m == nil` */ {
		return
	}

	// COMPILER OPTIMIZED FORM CL 110055 mapclr
	// https://go-review.googlesource.com/c/go/+/110055/
	for k := range m.m {
		delete(m.m, k)
	}
}

//go:nosplit
func NewSimpleSafeMapT[K comparable, V any]() *SimpleSafeMapT[K, V] {
	return &SimpleSafeMapT[K, V]{
		m: make(map[K]V),
	}
}
