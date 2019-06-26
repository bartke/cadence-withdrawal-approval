package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
)

func main() {
	var port string
	flag.StringVar(&port, "p", "port", "port to listen on")
	flag.Parse()

	http.HandleFunc("/", randomApproval)
	log.Printf("Starting server on :%v ...\n", port)
	http.ListenAndServe(":"+port, nil)
}

func randomApproval(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	result := "APPROVED"
	if rand.Intn(100) >= 70 {
		result = "DISAPPROVED"
	}
	log.Println(id, result)
	fmt.Fprint(w, result)
}
