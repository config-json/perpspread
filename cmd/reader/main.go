package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/config-json/perpspread/internal/reader"
	"github.com/config-json/perpspread/internal/storage"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	r := reader.New(ctx)
	w := storage.NewWriter(ctx)

	r.Start()
	w.Start()

	for {
		select {
		case <-ctx.Done():
			r.Close()
			w.Close()
			return
		case ob := <-r.Output():
			w.Input() <- ob
		case err := <-r.Error():
			log.Println("[Reader] Error:", err)
		case err := <-w.Error():
			log.Println("[Writer] Error:", err)
		}
	}
}
