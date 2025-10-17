package optimistic

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOnDemandCache_Get(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		assert.NotPanics(t, func() {
			mtx := sync.NewCond(&sync.Mutex{})

			ttl := 50 * time.Millisecond
			callsCounter := new(atomic.Int32)
			getter := func(key int) (string, bool) {
				callsCounter.Add(1)
				time.Sleep(100 * time.Millisecond)
				return strconv.Itoa(key), true
			}
			feature := NewCache[int, string](ttl, getter)
			defer feature.Close()

			const routines = 10
			const key = 10289

			for i := int32(0); i < routines; i++ {
				go func() {
					mtx.L.Lock()
					mtx.Wait()
					mtx.L.Unlock()
					_, _ = feature.Get(key)
				}()
			}

			mtx.L.Lock()
			mtx.Broadcast()
			mtx.L.Unlock()

			{
				v, isExists := feature.Get(key)
				assert.True(t, isExists)
				assert.Equal(t, "10289", v)
				assert.Equal(t, int32(1), callsCounter.Load())
			}
			{
				v, isExists := feature.Get(key)
				assert.True(t, isExists)
				assert.Equal(t, "10289", v)
				assert.Equal(t, int32(1), callsCounter.Load())
			}
			time.Sleep(time.Second)
			{
				v, isExists := feature.Get(key)
				assert.True(t, isExists)
				assert.Equal(t, "10289", v)
				assert.Equal(t, int32(2), callsCounter.Load())
			}
		})
	})

}
