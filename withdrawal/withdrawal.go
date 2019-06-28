package withdrawal

import (
	"log"
)

// use in-memory db for this example
var DB = make(map[string]*withdrawal)

func New(id string) *withdrawal {
	return &withdrawal{
		id: id,
		domainState: map[domain]State{
			Sports: Pending,
			Casino: Pending,
			Manual: Pending,
		},
		state: Pending,
	}
}

func (w *withdrawal) Approve(key domain) {
	if w.domainState[key] != Pending {
		return
	}
	w.domainState[key] = Approved
	// some trigger
	if w.domainState[Sports] == Approved && w.domainState[Casino] == Approved && w.domainState[Manual] != Rejected || w.domainState[Manual] == Approved {
		w.state = Approved
	}
}

func (w *withdrawal) Reject(key domain) {
	if w.domainState[key] != Pending && key != Manual {
		return
	}
	w.domainState[key] = Rejected
	if key == Manual {
		w.state = Rejected
	}
}

func (w *withdrawal) Payout() {
	if w.state != Approved {
		log.Println("payment blocked for withdrawal", w.id)
		return
	}
	log.Println("payment triggered for withdrawal", w.id)
	// Some logic
	w.state = Completed
}

func (w *withdrawal) DomainState(key domain) State {
	return w.domainState[key]
}

func (w *withdrawal) State() State {
	return w.state
}
