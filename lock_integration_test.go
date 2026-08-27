package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang-redis/handler"
	"golang-redis/repository"
	"golang-redis/service"

	"github.com/gofiber/fiber/v2"
	goredislib "github.com/redis/go-redis/v9"
)

const testRedisAddr = "localhost:6379"

func connectTestRedis(t *testing.T) *goredislib.Client {
	t.Helper()

	client := goredislib.NewClient(&goredislib.Options{Addr: testRedisAddr})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("skipping integration test: redis not reachable: %v", err)
	}

	return client
}

func TestLockService_RunCriticalSection_MutualExclusion(t *testing.T) {
	client := connectTestRedis(t)
	defer client.Close()

	jobID := "mutex-" + t.Name()
	client.Del(context.Background(), "lock:"+jobID)

	lockRepo := repository.NewLockRepositoryRedis(client)
	svc := service.NewLockService(lockRepo, 500*time.Millisecond)

	const workers = 10
	var (
		inCriticalSection atomic.Int32
		maxObserved       atomic.Int32
		wg                sync.WaitGroup
	)

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := svc.RunCriticalSection(context.Background(), service.RunCriticalSectionRequest{
				JobID:      jobID,
				HoldMillis: 50,
			})
			if err != nil {
				return
			}

			n := inCriticalSection.Add(1)
			for {
				max := maxObserved.Load()
				if n <= max || maxObserved.CompareAndSwap(max, n) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			inCriticalSection.Add(-1)
		}()
	}

	wg.Wait()

	if maxObserved.Load() > 1 {
		t.Fatalf("observed %d goroutines inside the critical section concurrently, want at most 1", maxObserved.Load())
	}
}

func TestLockHandler_RunCriticalSection_MutualExclusion(t *testing.T) {
	client := connectTestRedis(t)
	defer client.Close()

	jobID := "http-mutex-" + t.Name()
	client.Del(context.Background(), "lock:"+jobID)

	lockRepo := repository.NewLockRepositoryRedis(client)
	svc := service.NewLockService(lockRepo, 500*time.Millisecond)
	lockHdlr := handler.NewLockHandler(svc)

	app := fiber.New()
	app.Post("/jobs/run", lockHdlr.RunCriticalSection)

	run := func() int {
		var buf bytes.Buffer
		_ = json.NewEncoder(&buf).Encode(service.RunCriticalSectionRequest{JobID: jobID, HoldMillis: 200})

		req := httptest.NewRequest(fiber.MethodPost, "/jobs/run", &buf)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, 3000)
		if err != nil {
			t.Fatalf("app.Test() error = %v", err)
		}
		return resp.StatusCode
	}

	var wg sync.WaitGroup
	statuses := make([]int, 2)
	for i := range 2 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			statuses[i] = run()
		}(i)
	}
	wg.Wait()

	var ok, conflict int
	for _, s := range statuses {
		switch s {
		case fiber.StatusOK:
			ok++
		case fiber.StatusConflict:
			conflict++
		}
	}

	if ok != 1 || conflict != 1 {
		t.Fatalf("statuses = %v, want exactly one 200 and one 409", statuses)
	}
}
