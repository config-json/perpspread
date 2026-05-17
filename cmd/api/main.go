package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	"github.com/config-json/perpspread/internal/api"
	"github.com/config-json/perpspread/internal/storage"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	s, err := storage.New(ctx)

	if err != nil {
		panic(err)
	}

	defer s.Close()

	server := &http.Server{
		Addr:    ":8000",
		Handler: api.New(s),
	}

	go func() {
		err := server.ListenAndServe()

		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	<-ctx.Done()
	err = server.Shutdown(ctx)

	if err != nil {
		panic(err)
	}
}
