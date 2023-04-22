package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

func WriteBatchesInFile(ctx context.Context, name string, in <-chan string, bufferSize int) {
	f, err := os.OpenFile(name+".tsv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("can't proceed with file: %v", err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			log.Fatalf("can't close file: %v", err)
		}
	}()
	w := bufio.NewWriterSize(f, bufferSize)

	for {
		select {
		case <-ctx.Done():
			if err = w.Flush(); err != nil {
				log.Fatalf("can't flush buffer to file: %v", err)
			}
			return
		case data := <-in:
			_, err = w.WriteString(data)
			if err != nil {
				log.Printf("can't write %v to file", data)
			}
		}
	}
}

func UrlAsker(ctx context.Context, wg *sync.WaitGroup, taskPool <-chan string, out chan<- string, nbRetries int, timeout time.Duration) {
	client := http.Client{
		Timeout: timeout,
	}

	if nbRetries <= 0 {
		nbRetries = 1
	}

	for {
		select {
		case <-ctx.Done():
			wg.Done()
			return
		case url := <-taskPool:
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				log.Printf("can't make request to %v: %v", url, err)
				continue
			}

			for i := 0; i < nbRetries; i++ {
				resp, err := client.Do(req)
				if err != nil {
					log.Printf("can't do request to %v: %v", url, err)
					continue
				}

				out <- fmt.Sprintf("%v\t%v\n", url, resp.StatusCode)

				if resp.Body != nil {
					if err = resp.Body.Close(); err != nil {
						log.Printf("failed to close response body: %v", err)
					}
				}
				break
			}
		default:
			wg.Done()
			return
		}
	}
}

func main() {
	absPath, err := filepath.Abs("./config.yml")
	if err != nil {
		log.Fatalf("can't find abs path of file: %v", err)
	}
	cfg, err := NewConfig(absPath)
	if err != nil {
		log.Fatalf("can't create config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	defer cancel()

	c := make(chan string)

	wg := sync.WaitGroup{}

	go WriteBatchesInFile(ctx, cfg.FileName, c, cfg.BufferSize)

	taskPool := make(chan string, len(cfg.Urls))
	for _, url := range cfg.Urls {
		taskPool <- url
	}

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go UrlAsker(ctx, &wg, taskPool, c, cfg.Retries, cfg.Timeout)
	}

	wg.Wait()
}
