package main

import (
	cfg "documents/internal/config"
	st "documents/internal/storage"

	mw "github.com/go-chi/chi/middleware"

	"log"
	"net/http"

	"github.com/go-chi/chi"
)

func main() {
	config, err := cfg.Load(".")

	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	storage, err := st.New(config.Storage.Dsn, config.Storage.Name)

	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}

	defer storage.Close()

	if err := storage.Ping(); err != nil {
		log.Fatalf("failed to ping storage: %v", err)
	}

	if err := storage.CheckCollections(); err != nil {
		log.Fatalf("failed to create collections in storage: %v", err)
	}

	cache := cache.New(config.Cache.LiveTime)

	handler := dh.New(storage, cache)

	router := chi.NewRouter()

	router.Use(mw.RequestID)
	router.Use(mw.Logger)
	router.Use(mw.Recoverer)

	router.Get("/documents", handler.GetDocuments)
	router.Get("/documents/{id}", handler.GetDocument)
	router.Post("/documents", handler.CreateDocument)
	router.Put("/documents/{id}", handler.UpdateDocument)
	router.Delete("/documents/{id}", handler.DeleteDocument)

	server := http.Server{
		Addr:        config.Server.Port,
		Handler:     router,
		IdleTimeout: config.Server.IdleTimeout,
	}

	server.ListenAndServe()
}
