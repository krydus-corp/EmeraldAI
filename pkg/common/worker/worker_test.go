/*
 * File: worker_test.go
 * Project: worker
 * File Created: Friday, 19th May 2023 11:03:34 am
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package worker

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type args struct {
	idx int
	t   *testing.T
}

func (a *args) do() (struct{}, error) {
	a.t.Logf("processing task %d", a.idx)
	return struct{}{}, nil
}

func TestWorker(t *testing.T) {
	sent, received := 0, 0

	pool := New[struct{}](Config{
		Concurrency:      runtime.NumCPU(),
		RetryAttempts:    3,
		RetryWaitSeconds: 1,
		RetryBackoff:     true,
		Name:             "Test Worker",
	}).Start()
	defer pool.Stop()

	// Feed the pool
	t.Log("starting worker feed loop")

	for idx := 0; idx < 3; idx++ {
		args := args{idx: idx, t: t}
		pool.InChan <- args.do
		sent++
	}

	// Retrieve processed tasks
	t.Log("starting worker receive loop")
	started := time.Now()

	go func() {
		for task := range pool.OutChan {
			t.Logf("received task %v", task.Value)
			assert.NoError(t, task.Err)
			received++
		}
	}()

	// Wait for all tasks to complete
	for sent != received {
		time.Sleep(100 * time.Millisecond)
	}
	duration := time.Since(started).Seconds()

	assert.InDelta(t, 0, duration, 0.5)
}

func (a *args) fail() (struct{}, error) {
	return struct{}{}, fmt.Errorf("FAIL!")
}

func TestWorkerRetry(t *testing.T) {
	sent, received := 0, 0

	pool := New[struct{}](Config{
		Concurrency:      runtime.NumCPU(),
		RetryAttempts:    3,
		RetryWaitSeconds: 1,
		RetryBackoff:     true,
		RetryJitter:      false,
		Name:             "Test Worker Fail",
	}).Start()
	defer pool.Stop()

	// Feed the pool
	t.Log("starting worker feed loop")
	started := time.Now()

	args := args{idx: 0, t: t}
	pool.InChan <- args.fail
	sent++

	// Retrieve processed tasks
	t.Log("starting worker receive loop")

	go func() {
		for task := range pool.OutChan {
			t.Logf("received task %v", task.Value)
			assert.Error(t, task.Err)
			received++
		}
	}()

	// Wait for all tasks to complete
	for sent != received {
		time.Sleep(100 * time.Millisecond)
	}
	duration := time.Since(started).Seconds()

	t.Logf("task took %f ms", duration)

	assert.InDelta(t, 3, duration, 0.1)
}
