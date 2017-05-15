package cycle

import (
	"reflect"
	"strings"
	"testing"
)

type Pool struct {
	*R
	vs [][]uint8
}

func (p *Pool) get(i int) []uint8 {
	return p.vs[p.Map(i)]
}

func (p *Pool) set(i int, v []uint8) {
	p.vs[p.Map(i)] = v
}

func NewPool(z, i, n, nset int) (*Pool, error) {
	r, err := New(z, i, n, nset)
	if err != nil {
		return nil, err
	}
	return &Pool{R: r, vs: make([][]uint8, n)}, nil
}

// N is the length of parent set.
const N = 15

// load represents some finite set to cycle that can't be held in memory.
func load(i int) []uint8 {
	if i < 0 || i >= N {
		panic("index out of bounds")
	}
	return []uint8{uint8(i) + 1}
}

func TestCycle(t *testing.T) {
	read := func(pool *Pool) (p [][]uint8) {
		pool.Do(pool.Left(), 1, func(i int) { p = append(p, pool.get(i)) })
		return p
	}

	pool, err := NewPool(-N/4, 0, N/2, N)
	if err != nil {
		t.Fatalf("NewUint8Slice: %v", err)
	}

	pool.Do(pool.Index(), 1, func(i int) {
		pool.set(i, load(i))
	})

	if want, have := [][]uint8{{1}, {2}, {3}, {4}, nil, nil, nil}, pool.vs; !reflect.DeepEqual(want, have) {
		t.Fatalf("write i=%v s=%v failed\nwant: %+v\nhave: %+v\n", pool.Index(), 1, want, have)
	}

	pool.Do(pool.Index()-1, -1, func(i int) {
		pool.set(i, load(i))
	})

	if want, have := [][]uint8{{1}, {2}, {3}, {4}, {13}, {14}, {15}}, pool.vs; !reflect.DeepEqual(want, have) {
		t.Fatalf("write i=%v s=%v failed\nwant: %+v\nhave: %+v\n", pool.Index()-1, -1, want, have)
	}

	if want, have := [][]uint8{{13}, {14}, {15}, {1}, {2}, {3}, {4}}, read(pool); !reflect.DeepEqual(want, have) {
		t.Fatalf("read failed\nwant: %+v\nhave: %+v\n", want, have)
	}

	if want, have := []uint8{1}, pool.get(0); !reflect.DeepEqual(want, have) {
		t.Fatalf("read zero index failed\nwant: %+v\nhave: %+v\n", want, have)
	}

	wi, ws, err := pool.Cycle(1, 0)
	if err != nil {
		t.Fatalf("shift: %v", err)
	}
	pool.Do(wi, ws, func(i int) {
		pool.set(i, load(i))
	})

	if want, have := [][]uint8{{1}, {2}, {3}, {4}, {5}, {14}, {15}}, pool.vs; !reflect.DeepEqual(want, have) {
		t.Fatalf("shift write failed\nwant: %+v\nhave: %+v\n", want, have)
	}

	if want, have := []uint8{1}, pool.get(0); !reflect.DeepEqual(want, have) {
		t.Fatalf("repeat read of zero index failed\nwant: %+v\nhave: %+v\n", want, have)
	}
}

func TestVerify(t *testing.T) {
	var pros []pro

	// l < r errors
	pros = []pro{
		pro{l: 10, i: 5, r: 20},
		pro{l: 10, i: 25, r: 20},
	}
	for _, o := range pros {
		err := o.verify()
		if err == nil {
			t.Errorf("err == nil for %+v", o)
		} else if !strings.HasPrefix(err.Error(), "l < r") {
			t.Errorf("wrong err for %+v: %s", o, err)
		}
	}

	// r < l errors
	pros = []pro{
		pro{l: 20, i: 15, r: 10},
	}
	for _, o := range pros {
		err := o.verify()
		if err == nil {
			t.Errorf("err == nil for %+v", o)
		} else if !strings.HasPrefix(err.Error(), "r < l") {
			t.Errorf("wrong err for %+v: %s", o, err)
		}
	}

	// no errors
	pros = []pro{
		// l < r
		pro{l: 0, i: 1, r: 2},
		pro{l: 10, i: 15, r: 20},
		// r < l
		pro{l: 20, i: 25, r: 10},
		pro{l: 20, i: 5, r: 10},
	}
	for _, o := range pros {
		if err := o.verify(); err != nil {
			t.Errorf("expected nil err for %+v: %s", o, err)
		}
	}
}
