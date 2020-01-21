package xz

import (
	"sync"
	"testing"
)

func TestMapStoreLoadDelete(t *testing.T) {
	m := NewMap()
	m.Store("key", "value")

	v, ok := m.Load("key")
	if !ok {
		t.Fatal("value not loaded")
	}

	s, ok := v.(string)
	if !ok {
		t.Fatalf("value invalid type: %s", v)
	}

	if s != "value" {
		t.Fatalf(`invalid value: %s, expecting "value"`, v)
	}

	m.Delete("key")
	v, ok = m.Load("key")
	if ok {
		t.Fatalf("found deleted value %s", v)
	}

}

func TestMapRange(t *testing.T) {
	m := NewMap()
	m.Store("key1", "value1")
	m.Store("key2", "value2")
	m.Store("key3", "value3")
	m.Store("key4", "value4")

	c := 0
	m.Range(func(key interface{}, value interface{}) bool {
		c++
		return true
	})

	if c != 4 {
		t.Fatalf("map range failed iterations got: %d, expected: %d", c, 4)
	}
}

func TestMapLoadOrStore(t *testing.T) {
	m := NewMap()
	m.Store("key1", "value1")

	v, loaded := m.LoadOrStore("key2", "value2")
	if loaded {
		t.Fatalf("failed to store for key2")
	}

	if v != "value2" {
		t.Fatalf("unexpected value for key2: %v", v)
	}

	v, loaded = m.LoadOrStore("key1", "value2")
	if !loaded {
		t.Fatalf("failed to load for key1")
	}

	if v != "value1" {
		t.Fatalf("unexpected value for key2: %v", v)
	}

}

func TestMapConcurrency(t *testing.T) {
	m := NewMap()
	wg := sync.WaitGroup{}
	start := make(chan struct{})

	for x := 0; x < 4; x++ {
		wg.Add(1)
		go func(c int) {
			<-start
			m.Store(c*1, "value1")
			m.Store(c*2, "value2")
			m.Store(c*3, "value3")
			m.Store(c*4, "value4")

			m.Range(func(key interface{}, value interface{}) bool {
				k := key.(int)
				if k%2 == 0 {
					m.Delete(key)
				}
				return true
			})
			wg.Done()
		}(x)
	}
	close(start)
	wg.Wait()

	if _, ok := m.Load(1); !ok {
		t.Fatalf("could not find key %d", 1)
	}

	if _, ok := m.Load(3); !ok {
		t.Fatalf("could not find key %d", 3)
	}

	if _, ok := m.Load(9); !ok {
		t.Fatalf("could not find key %d", 9)
	}
}

func BenchmarkMapStoreRangeDelete(b *testing.B) {
	m := NewMap()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m.Store("key1", "value1")
		m.Store("key2", "value2")
		m.Store("key3", "value3")
		m.Store("key4", "value4")

		c := 0
		m.Range(func(key interface{}, value interface{}) bool {
			m.Delete(key)
			c++
			return true
		})

		if c != 4 {
			b.Fatalf("map range failed iterations got: %d, expected: %d", c, 4)
		}
	}
}
