package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

var done = make(chan struct{})

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{\"status\": \"UP\"}"))
	})
	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("server will shutdown within 3s"))
		close(done)
	})

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error {
		log.Println("SERVER START")
		defer log.Println("GOROUTINE QUIT: http server")

		return server.ListenAndServe()
	})

	g.Go(func() error {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGTERM, syscall.SIGINT)

		defer func() {
			close(sigchan)
			log.Println("GOROUTINE QUIT: signal listener")
		}()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case sig := <-sigchan:
			return fmt.Errorf("SIGNAL: %v", sig)
		}
	})

	g.Go(func() error {
		defer log.Println("GOROUTINE QUIT: deamon")

		select {
		case <-ctx.Done():
		case <-done:
		case <-time.After(30 * time.Second):
			log.Println("- This server live for only 30s, cheers! -")
		}

		timedCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return server.Shutdown(timedCtx)
	})

	err := g.Wait()
	if err != nil {
		log.Printf("ERRGROUP QUIT [%v]\n", err)
	}
	log.Println("Bye!")
}
