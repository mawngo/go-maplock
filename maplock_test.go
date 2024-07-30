package maplock

import (
	"sync"
	"testing"
	"time"
)

func Test1(_ *testing.T) {
	m := New[string]()
	m.Lock("foo")
	m.Unlock("foo")
}

func Test2(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(3)

	m := New[string]()
	n := 0
	m.Lock("foo")
	n++
	m.Unlock("foo")
	if n != 1 {
		t.Fatal("not 1")
	}
	wg.Done()

	go func() {
		m.Lock("foo")
		n++
		m.Unlock("foo")
		if n != 2 {
			panic("not 2")
		}
		wg.Done()
	}()
	go func() {
		time.Sleep(1 * time.Millisecond)
		m.Lock("foo")
		n++
		m.Unlock("foo")
		if n != 3 {
			panic("not 3")
		}
		wg.Done()
	}()
	wg.Wait()
}
