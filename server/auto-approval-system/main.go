package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"github.com/bartke/cadence-withdrawal-approval/withdrawal"
)

var port string

func main() {
	flag.StringVar(&port, "p", "port", "port to listen on")
	flag.Parse()

	http.HandleFunc("/", randomApproval)
	log.Printf("Starting server on :%v ...\n", port)
	http.ListenAndServe(":"+port, nil)
}

func hex2rand(input string) int {
	// simple way to make this deterministic for same input
	rand.Seed(int64(int(port[len(port)-1])) + int64(int(input[len(input)-1])))
	return rand.Intn(100)
}

func randomApproval(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	result := withdrawal.Approve
	if id != "" && hex2rand(id) >= 80 {
		result = withdrawal.Reject
	}
	log.Println(id, result)
	fmt.Fprint(w, result)
}
