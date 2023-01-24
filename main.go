package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	r := http.DefaultServeMux
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hostname, _ := os.Hostname()

		resp := fmt.Sprintf("%s\n%s", hostname, os.Getenv("TEXT"))
		w.Write([]byte(resp))
	})

	if err := http.ListenAndServe(":80", r); err != nil {
		log.Fatal(err)
	}
}
