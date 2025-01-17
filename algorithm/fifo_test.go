package algorithm

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
	if v == nil {
		panic("empty")
	}

	fmt.Println(v.(int))
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
	var got []any

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

	require.Equal(t, 0, f.Len(), "empty")
	require.Len(t, got, int(cnt), "total len")
}

// BenchmarkFIFO
//
// cpu: AMD Ryzen 7 5700G with Radeon Graphics
// BenchmarkFIFO/fifo-16         	  752448	      1654 ns/op	      12 B/op	       0 allocs/op
func BenchmarkFIFO(b *testing.B) {
	b.Run("fifo", func(b *testing.B) {
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
	})
}

// cpu: AMD Ryzen 7 5700G with Radeon Graphics
// BenchmarkFIFOAndChan/fifo-16         	  733112	      1608 ns/op	      12 B/op	       0 allocs/op
// BenchmarkFIFOAndChan/channel_struct-16         	 1809888	       653.2 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFIFOAndChan/channel_int-16            	 1862325	       647.9 ns/op	       0 B/op	       0 allocs/op
func BenchmarkFIFOAndChan(b *testing.B) {

	b.Run("fifo", func(b *testing.B) {
		f := NewFIFO()
		b.RunParallel(func(p *testing.PB) {
			for p.Next() {
				for p.Next() {
					switch rand.Intn(2) {
					case 0:
						f.Put(2)
					case 1:
						_ = f.Get()
					}
				}
			}
		})
	})

	b.Run("channel struct", func(b *testing.B) {
		ch := make(chan struct{}, 10)
		b.RunParallel(func(p *testing.PB) {
			for p.Next() {
				for p.Next() {
					switch rand.Intn(2) {
					case 0:
						select {
						case ch <- struct{}{}:
						default:
						}
					case 1:
						select {
						case <-ch:
						default:
						}
					}
				}
			}
		})
	})

	b.Run("channel int", func(b *testing.B) {
		ch := make(chan int, 10)
		b.RunParallel(func(p *testing.PB) {
			for p.Next() {
				switch rand.Intn(2) {
				case 0:
					select {
					case ch <- 2:
					default:
					}
				case 1:
					select {
					case <-ch:
					default:
					}
				}
			}
		})
	})
}
