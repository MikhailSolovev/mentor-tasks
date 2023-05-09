package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// TODO: add tests and avoid a lot of anti-patterns
// Honestly you can use it just for main ideas of other algorithms, real system is distributed,
// therefore need to use patterns like gateway or make ratelimiter stateless (Redis, Kafka etc.)

var LimitExceed = errors.New("429")
var count = 100
var ch = make(chan int, count)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	var wg sync.WaitGroup

	RPCCallWithRateLimiter := WithRateLimiter(RPCCall, RateLimiterOptions{
		rateLimiter: FixedWindowRateLimiter,
		limit:       10,
		unit:        time.Second,
	})

	go func() {
		for value := range ch {
			log.Println(value)
		}
	}()

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			value, err := RPCCallWithRateLimiter()
			if err != nil {
				if errors.Is(err, LimitExceed) {
					log.Printf("rate limit exceed: %v", err)
				}
			} else {
				ch <- value
			}
		}()
	}

	wg.Wait()
	close(ch)
}

func RPCCall() (int, error) {
	return rand.Int(), nil
}

type RateLimiterOptions struct {
	rateLimiter RateLimiter
	limit       int64
	unit        time.Duration
}

func WithRateLimiter(f func() (int, error), rateLimiterOptions RateLimiterOptions) func() (int, error) {
	return func() (int, error) {
		if err := rateLimiterOptions.rateLimiter(rateLimiterOptions.limit, rateLimiterOptions.unit); err != nil {
			return 0, err
		}
		return f()
	}
}

type RateLimiter func(limit int64, unit time.Duration) error

var bucket int64
var once sync.Once

func TokenBucketRateLimiter(limit int64, unit time.Duration) error {
	once.Do(func() {
		go func() {
			refreshRate := unit / time.Duration(limit)

			atomic.StoreInt64(&bucket, limit)

			time.Sleep(unit)
			for {
				select {
				case <-time.After(refreshRate):
					atomic.AddInt64(&bucket, 1)
				}
			}
		}()
	})

	// --- It all must be atomic operation!

	if atomic.LoadInt64(&bucket) == 0 {
		return LimitExceed
	}

	atomic.AddInt64(&bucket, -1)

	// ---

	return nil
}

var queue = make(chan struct{}, 10)

func LeakyBucketRateLimiter(limit int64, unit time.Duration) error {
	once.Do(func() {
		go func() {
			leakRate := unit / time.Duration(limit)

			<-queue
			value, _ := RPCCall()
			ch <- value

			for {
				select {
				case <-time.After(leakRate):
					<-queue
					value, _ = RPCCall()
					ch <- value
				}
			}
		}()
	})

	select {
	case queue <- struct{}{}:
		return fmt.Errorf("placed in queue")
	default:
		return LimitExceed
	}
}

var windows = sync.Map{}

func FixedWindowRateLimiter(limit int64, unit time.Duration) error {
	once.Do(func() {
		go func() {
			currentWindow := time.Now().Round(unit)
			windows.Store(currentWindow, 0)
			nextWindow := time.Now().Add(unit).Round(unit)
			windows.Store(nextWindow, 0)

			for {
				select {
				case <-time.After(unit):
					prevWindow := time.Now().Add(unit * time.Duration(-1)).Round(unit)
					windows.Delete(prevWindow)
					nextWindow = time.Now().Add(unit).Round(unit)
					windows.Store(nextWindow, 0)
				}
			}
		}()
	})

	now := time.Now().Round(unit)

	// --- It all must be atomic operation!

	value, _ := windows.Load(now)
	v, _ := value.(int64)

	if v == limit {
		return LimitExceed
	}

	v++
	windows.Store(now, v)

	// ---

	return nil
}

// TODO: implement it
func SlidingWindowRateLimiter(limit int64, unit time.Duration) error {
	return nil
}
