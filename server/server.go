package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/bartke/cadence-withdrawal-approval/common"
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
	fmt.Fprint(w, "<h1>Withdrawal Approval</h1>"+"<a href=\"/list\">HOME</a>"+
		"<h3>All withdrawal requests:</h3><table><tr><th>ID</th><th>Status</th><th>Action</th>")
	keys := []string{}
	for k := range withdrawalDB {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, id := range keys {
		withdrawal := withdrawalDB[id]
		actionLink := ""
		if withdrawal.State() == Pending {
			actionLink = fmt.Sprintf("<a href=\"/action?type=approve&id=%s\">"+
				"<button style=\"background-color:#4CAF50;\">APPROVE</button></a>"+
				"&nbsp;&nbsp;<a href=\"/action?type=reject&id=%s\">"+
				"<button style=\"background-color:#f44336;\">REJECT</button></a>", id, id)
		}
		fmt.Fprintf(w, "<tr><td>%s</td><td>%s</td><td>%s</td></tr>", id, withdrawal.State(), actionLink)
	}
	fmt.Fprint(w, "</table>")
}

func actionHandler(w http.ResponseWriter, r *http.Request) {
	isAPICall := r.URL.Query().Get("is_api_call") == "true"
	id := r.URL.Query().Get("id")
	withdrawal, ok := withdrawalDB[id]
	if !ok {
		fmt.Fprint(w, "ERROR:INVALID_ID")
		return
	}
	oldState := withdrawal.State()
	actionType := r.URL.Query().Get("type")
	switch actionType {
	case "approve":
		withdrawalDB[id].Approve()
	case "reject":
		withdrawalDB[id].Reject()
	case "payment":
		withdrawalDB[id].Payout()
	}
	if isAPICall {
		fmt.Fprint(w, "SUCCEED")
	} else {
		listHandler(w, r)
	}

	if oldState == Pending && (withdrawalDB[id].State() == Approved || withdrawalDB[id].State() == Rejected) {
		// report state change
		notifyWithdrawalStateChange(id, withdrawalDB[id].State().String())
	}

	log.Printf("Set state for %s from %s to %s.\n", id, oldState, withdrawalDB[id].State().String())

	return
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	isAPICall := r.URL.Query().Get("is_api_call") == "true"
	id := r.URL.Query().Get("id")
	_, ok := withdrawalDB[id]
	if ok {
		fmt.Fprint(w, "ERROR:ID_ALREADY_EXISTS")
		return
	}

	withdrawalDB[id] = NewWithdrawal(id)
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
	state, ok := withdrawalDB[id]
	if !ok {
		fmt.Fprint(w, "ERROR:INVALID_ID")
		return
	}

	fmt.Fprint(w, state)
	log.Printf("Checking status for %s: %s\n", id, state)
	return
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	withdrawal, ok := withdrawalDB[id]
	if !ok {
		fmt.Fprint(w, "ERROR:INVALID_ID")
		return
	}
	if withdrawal.State() != Pending {
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
