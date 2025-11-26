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
			barrier := make(chan struct{})

			ttl := 50 * time.Millisecond
			callsCounter := new(atomic.Int32)
			getter := func(key int) (string, bool) {
				time.Sleep(5 * time.Millisecond)
				callsCounter.Add(1)
				return strconv.Itoa(key), true
			}
			feature := NewCache[int, string](ttl, getter)
			defer feature.Close()

			const key = 10289

			wg := new(sync.WaitGroup)
			for range 5000 {
				wg.Go(func() {
					<-barrier
					_, _ = feature.Get(key)
				})
			}

			close(barrier)
			wg.Wait()

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
				assert.Equal(t, int32(1), callsCounter.Load())
			}
		})
	})

}
