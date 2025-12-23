package main

import (
	"log"
	"net/http"
)

func main() {
	port := "8080"

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(server.ListenAndServe())
}
