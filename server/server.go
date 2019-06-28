package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/bartke/cadence-withdrawal-approval/common"
	"github.com/bartke/cadence-withdrawal-approval/withdrawal"
	"go.uber.org/cadence/client"
)

/**
 * Supports to list withdrawals, create new withdrawal, update withdrawal state and checking withdrawal state.
 */

var tokenMap = make(map[string][]byte)

var workflowClient client.Client

func main() {
	var h common.SampleHelper
	h.SetupServiceConfig()
	var err error
	workflowClient, err = h.Builder.BuildCadenceClient()
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", listHandler)
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/create", createHandler)
	http.HandleFunc("/action", actionHandler)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/registerCallback", callbackHandler)

	log.Println("Starting server on :8099...")
	http.ListenAndServe(":8099", nil)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "<h1>Withdrawal Approval</h1>"+"<a href=\"/list\">Refresh</a>"+
		"<h3>All withdrawal requests:</h3><table><tr><th>ID</th><th>Sports</th><th>Casino</th><th>Manual</th><th>Payment</th><th>Action</th>")
	keys := []string{}
	for k := range withdrawal.DB {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, id := range keys {
		wd := withdrawal.DB[id]
		actionLink := ""
		if wd.State() == withdrawal.Pending {
			actionLink = fmt.Sprintf("<a href=\"/action?type=approve&domain=manual&id=%s\">"+
				"<button style=\"background-color:#4CAF50;\">APPROVE</button></a>"+
				"&nbsp;&nbsp;<a href=\"/action?type=reject&domain=manual&id=%s\">"+
				"<button style=\"background-color:#f44336;\">REJECT</button></a>", id, id)
		}
		fmt.Fprintf(w, "<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
			id, wd.DomainState(withdrawal.Sports), wd.DomainState(withdrawal.Casino), wd.DomainState(withdrawal.Manual), wd.State(), actionLink)
	}
	fmt.Fprint(w, "</table>")
}

func actionHandler(w http.ResponseWriter, r *http.Request) {
	isAPICall := r.URL.Query().Get("is_api_call") == "true"
	id := r.URL.Query().Get("id")
	wd, ok := withdrawal.DB[id]
	if !ok {
		fmt.Fprint(w, "ERROR:INVALID_ID")
		return
	}
	oldState := wd.State()
	action := withdrawal.ParseAction(r.URL.Query().Get("type"))
	domain := withdrawal.ParseDomain(r.URL.Query().Get("domain"))

	log.Println("received ----> ", action, domain)

	switch action {
	case withdrawal.Approve:
		withdrawal.DB[id].Approve(domain)
	case withdrawal.Reject:
		withdrawal.DB[id].Reject(domain)
	case withdrawal.Payout:
		withdrawal.DB[id].Payout()
	}
	if isAPICall {
		fmt.Fprint(w, "SUCCEED")
	} else {
		listHandler(w, r)
	}

	if oldState == withdrawal.Pending && (withdrawal.DB[id].State() == withdrawal.Approved || withdrawal.DB[id].State() == withdrawal.Rejected) {
		// report state change
		notifyWithdrawalStateChange(id, withdrawal.DB[id].State().String())
	}

	log.Printf("Set state for %s from %s to %s via %v.\n", id, oldState, withdrawal.DB[id].State().String(), domain)

	return
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	isAPICall := r.URL.Query().Get("is_api_call") == "true"
	id := r.URL.Query().Get("id")
	_, ok := withdrawal.DB[id]
	if ok {
		fmt.Fprint(w, "ERROR:ID_ALREADY_EXISTS")
		return
	}

	withdrawal.DB[id] = withdrawal.New(id)
	if isAPICall {
		fmt.Fprint(w, "SUCCEED")
	} else {
		listHandler(w, r)
	}
	log.Printf("pending new withdrawal id:%s.\n", id)
	return
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	wd, ok := withdrawal.DB[id]
	if !ok {
		fmt.Fprint(w, "ERROR:INVALID_ID")
		return
	}

	fmt.Fprint(w, wd.State().String())
	log.Printf("Checking status for %s: %s\n", id, wd.State().String())
	return
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	wd, ok := withdrawal.DB[id]
	if !ok {
		fmt.Fprint(w, "ERROR:INVALID_ID")
		return
	}
	if wd.State() != withdrawal.Pending {
		fmt.Fprint(w, "ERROR:INVALID_STATE")
		return
	}

	err := r.ParseForm()
	if err != nil {
		// Handle error here via logging and then return
		fmt.Fprint(w, "ERROR:INVALID_FORM_DATA")
		return
	}

	taskToken := r.PostFormValue("task_token")
	log.Printf("Registered callback for ID=%s, token=%s\n", id, taskToken)
	tokenMap[id] = []byte(taskToken)
	fmt.Fprint(w, "SUCCEED")
}

func notifyWithdrawalStateChange(id, state string) {
	token, ok := tokenMap[id]
	if !ok {
		log.Printf("Invalid id:%s\n", id)
		return
	}
	err := workflowClient.CompleteActivity(context.Background(), token, state, nil)
	if err != nil {
		log.Printf("Failed to complete activity with error: %+v\n", err)
	} else {
		log.Printf("Successfully complete activity: %s\n", token)
	}
}
