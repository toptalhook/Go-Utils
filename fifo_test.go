package utils

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func ExampleFIFO() {
	f := NewFIFO()
	f.Put(1)
	v := f.Get()
	fmt.Println(v)
	// Output: 1
}

func Test_UnsafePtr(t *testing.T) {
	var a int

	addr := unsafe.Pointer(&a)

	b := *(*int)(atomic.LoadPointer(&addr))
	require.Equal(t, a, b)
}

func TestNewFIFO(t *testing.T) {
	f := NewFIFO()
	var pool errgroup.Group
	start := make(chan struct{})

	var mu sync.Mutex
	var cnt int32
	var got []interface{}

	for i := 0; i < 100; i++ {
		pool.Go(func() error {
			<-start
			for i := 0; i < 100; i++ {
				switch rand.Intn(2) {
				case 0:
					f.Put(i)
					atomic.AddInt32(&cnt, 1)
				case 1:
					v := f.Get()
					if v != nil {
						mu.Lock()
						got = append(got, v)
						mu.Unlock()
					}
				}
			}

			return nil
		})
	}

	time.Sleep(time.Second)
	close(start)
	err := pool.Wait()
	require.NoError(t, err)
	f.Len()

	for {
		v := f.Get()
		if v == nil {
			break
		}

		got = append(got, v)
	}

	require.Equal(t, 0, f.Len())
	require.Len(t, got, int(cnt))
}

// BenchmarkFIFO
//
//   cpu: Intel(R) Core(TM) i7-4790 CPU @ 3.60GHz
//   BenchmarkFIFO-8   	  368847	      3330 ns/op	      15 B/op	       0 allocs/op
func BenchmarkFIFO(b *testing.B) {
	f := NewFIFO()
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			switch rand.Intn(2) {
			case 0:
				f.Put(2)
			case 1:
				_ = f.Get()
			}
		}
	})
}
