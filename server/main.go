package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/estnafinema0/Github-Trends-Aggregator/server/api"
	"github.com/estnafinema0/Github-Trends-Aggregator/server/scheduler"
	"github.com/estnafinema0/Github-Trends-Aggregator/server/store"
	"github.com/estnafinema0/Github-Trends-Aggregator/server/ws"
	"github.com/estnafinema0/Github-Trends-Aggregator/server/config"
	"github.com/estnafinema0/Github-Trends-Aggregator/server/email"
	"github.com/gorilla/mux"
	
)
func main() {
	// Initialize logger
	l := log.New(os.Stdout, "server-api: ", log.LstdFlags)

	if !config.LoadSecrets() {
		l.Printf("Failed to load secrets")
		return
	}
	// Initialize In-memory store
	store := store.NewStore()

	// Initialize WebSocket hub and run it in a goroutine
	hub := ws.NewHub()
	go hub.Run(l)

	go email.StartEmail(store, l)

	go scheduler.StartScheduler(store, hub, l)

	sm := mux.NewRouter()

	sm.HandleFunc("/trends", api.GetTrendsHandler(store)).Methods("GET")
	sm.HandleFunc("/trends/{id:[0-9]+}", api.GetRepoHandler(store)).Methods("GET")
	sm.HandleFunc("/stats", api.GetStatsHandler(store)).Methods("GET")
	sm.HandleFunc("/ws", ws.ServeWsHandler(hub, l))
	sm.HandleFunc("/", api.GetIndexHandler(store)).Methods("GET")
	sm.HandleFunc("/subscribe", api.GetSubscribeHandler(store)).Methods("GET")
	sm.HandleFunc("/subscribed", api.GetSubscribedHandler(store)).Methods("GET", "POST")

	sm.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Handler:      sm,
		Addr:         ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	l.Printf("Starting server on port %s\n", srv.Addr)
	log.Fatal(srv.ListenAndServe())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	l.Println("Received terminate, graceful shutdown", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
