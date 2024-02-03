/*
 * File: worker.go
 * Project: worker
 * File Created: Sunday, 11th September 2022 3:49:56 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package worker

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
)

type Worker[T any] struct {
	InChan  chan func() (T, error)
	OutChan chan Result[T]

	sem      chan struct{}
	stopChan chan struct{}
	wg       sync.WaitGroup

	cfg Config
}

type Config struct {
	Concurrency      int  `yaml:"concurrency,omitempty"`
	RetryAttempts    int  `yaml:"retry_attempts,omitempty"`
	RetryWaitSeconds int  `yaml:"retry_wait_seconds,omitempty"`
	RetryBackoff     bool `yaml:"retry_backoff,omitempty"`
	RetryJitter      bool `yaml:"retry_jitter,omitempty"`
	Name             string
}

func (c *Config) String() string {
	return fmt.Sprintf("concurrency=%d retryAttempts=%d retryWaitSeconds=%d retryBackoff=%v", c.Concurrency, c.RetryAttempts, c.RetryWaitSeconds, c.RetryBackoff)
}

func New[T any](cfg Config) *Worker[T] {
	if cfg.Concurrency == -1 {
		cfg.Concurrency = runtime.NumCPU()
	}
	if cfg.Name == "" {
		cfg.Name = common.ShortUUID(10)
	}

	log.Printf("configured worker pool: %s", cfg.String())

	return &Worker[T]{
		InChan:   make(chan func() (T, error)),
		OutChan:  make(chan Result[T]),
		sem:      make(chan struct{}, cfg.Concurrency),
		stopChan: make(chan struct{}, 1),
		wg:       sync.WaitGroup{},
		cfg:      cfg,
	}
}

func (w *Worker[T]) Start() *Worker[T] {
	go func() {
		log.Printf("starting worker=%s", w.cfg.Name)

		for task := range w.InChan {
			if len(w.stopChan) > 0 {
				break
			}

			w.sem <- struct{}{}

			w.wg.Add(1)
			go func(f func() (T, error)) {
				defer w.wg.Done()

				if w.cfg.RetryAttempts > 0 {
					v, err := Retry(
						w.cfg.RetryAttempts,
						w.cfg.RetryBackoff,
						w.cfg.RetryJitter,
						time.Duration(w.cfg.RetryWaitSeconds*int(time.Second)),
						func() (T, error) {
							return f()
						})
					w.OutChan <- Result[T]{Value: v, Err: err}
					<-w.sem
				} else {
					v, err := f()
					w.OutChan <- Result[T]{Value: v, Err: err}
					<-w.sem
				}
			}(task)
		}
		log.Printf("exiting worker=%s", w.cfg.Name)
	}()

	return w
}

func (w *Worker[T]) Stop() {
	w.stopChan <- struct{}{}

	w.wg.Wait()
	close(w.OutChan)
	close(w.InChan)
}
