package main

import (
	// сделал импорты через goimports
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"project-sem/internal/controllers"
	"project-sem/internal/DB"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	if err := DB.InitDB(); err != nil {
		return err
	}
	defer DB.CloseDB()

	r := mux.NewRouter()
	r.HandleFunc("/api/v0/prices", handlers.HandlerPostPrices()).Methods("POST")
	r.HandleFunc("/api/v0/prices", handlers.HandlerGetPrices()).Methods("GET")

	err := http.ListenAndServe(`:8080`, r)
	if err != nil {
		return err
	}
	return nil
}