package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	var port string
	flag.StringVar(&port, "p", "port", "port to listen on")
	flag.Parse()

	rand.Seed(time.Now().UTC().UnixNano())

	http.HandleFunc("/", randomApproval)
	log.Printf("Starting server on :%v ...\n", port)
	http.ListenAndServe(":"+port, nil)
}

func hex2rand(input string) int {
	// simple way to make this deterministic for same input
	rand.Seed(int64(int(input[len(input)-1])))
	return rand.Intn(100)
}

func randomApproval(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	result := "APPROVED"
	if id != "" && hex2rand(id) >= 70 {
		result = "DISAPPROVED"
	}
	log.Println(id, result)
	fmt.Fprint(w, result)
}
