package main

import "log"

// use memory store for this example
var withdrawalDB = make(map[string]*withdrawal)

type state string

const (
	Pending   state = "PENDING"
	Approved  state = "APPROVED"
	Rejected  state = "REJECTED"
	Completed state = "COMPLETED"
)

func (w *withdrawal) Approve() {
	if w.state != Pending {
		return
	}
	w.state = Approved
}

func (w *withdrawal) Reject() {
	if w.state != Pending {
		return
	}
	w.state = Rejected
}

func (w *withdrawal) Payout() {
	if w.state != Approved {
		log.Println("payment blocked for withdrawal", w.id)
		return
	}
	log.Println("payment triggered for withdrawal", w.id)
	w.state = Completed
}

func (w *withdrawal) State() state {
	return w.state
}

func (s state) String() string {
	return string(s)
}

type withdrawal struct {
	id    string
	state state
}

func NewWithdrawal(id string) *withdrawal {
	return &withdrawal{
		id:    id,
		state: Pending,
	}
}
