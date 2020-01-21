package xz

import "sync"

// Map follows the same interface of sync.Map and is safe for concurrent use,
// it is however targeted to the RW scenario and allocates much less.
type Map struct {
	mtx  sync.RWMutex
	data map[interface{}]interface{}
}

// NewMap creates a new map with the specified shards count.
// If the shards argument is < 2, the DefaultShardCount will be used.
func NewMap() (m *Map) {
	m = &Map{}
	m.data = make(map[interface{}]interface{})

	return m
}

// Count returns the number of entries in the map
func (m *Map) Count() (n int) {
	m.mtx.RLock()
	n = len(m.data)
	m.mtx.RUnlock()

	return n
}

// Delete deletes the value for a key.
func (m *Map) Delete(key interface{}) {
	m.mtx.Lock()
	delete(m.data, key)
	m.mtx.Unlock()

	return
}

// Load returns the value stored in the map for a key, or nil if no value is present.
// The ok result indicates whether value was found in the map.
func (m *Map) Load(key interface{}) (value interface{}, ok bool) {
	m.mtx.RLock()
	value, ok = m.data[key]
	m.mtx.RUnlock()

	return value, ok
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value. The loaded result is true if the value was loaded, false if stored.
func (m *Map) LoadOrStore(key, value interface{}) (actual interface{}, loaded bool) {
	m.mtx.Lock()
	v, loaded := m.data[key]

	switch loaded {
	case true:
		actual = v
	case false:
		actual = value
		m.data[key] = value
	}

	m.mtx.Unlock()
	return actual, loaded
}

// Range calls f sequentially for each key and value present in the map. If f returns false, range stops the iteration.
//
// Range does not necessarily correspond to any consistent snapshot of the Map's contents:
// no key will be visited more than once, but if the value for any key is stored or deleted concurrently,
// Range may reflect any mapping for that key from any point during the Range call.
func (m *Map) Range(f func(key, value interface{}) bool) {
	m.mtx.RLock()
	kv := make([]interface{}, 0, len(m.data)*2)
	for key, value := range m.data {
		kv = append(kv, key, value)
	}
	m.mtx.RUnlock()

	for x := 0; x < len(kv)-1; x++ {
		if !f(kv[x], kv[x+1]) {
			return
		}
		x++
	}

	return
}

// Store sets the value for a key.
func (m *Map) Store(key, value interface{}) {
	m.mtx.Lock()
	m.data[key] = value
	m.mtx.Unlock()

	return
}

// Clear the map contents but retains the allocated shards.
func (m *Map) Clear() {
	m.mtx.Lock()
	for k := range m.data {
		delete(m.data, k)
	}
	m.mtx.Unlock()

	return
}
