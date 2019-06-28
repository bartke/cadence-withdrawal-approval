package withdrawal

import "strings"

type withdrawal struct {
	id          string
	domainState map[domain]State
	state       State
}

type domain string
type State string
type action string

const (
	Sports        domain = "sports"
	Casino        domain = "casino"
	Manual        domain = "manual"
	UnknownDomain domain = "-"

	Approve       action = "APPROVE"
	Reject        action = "REJECT"
	Payout        action = "PAYOUT"
	UnknownAction action = "-"

	Pending   State = "PENDING"
	Approved  State = "APPROVED"
	Rejected  State = "REJECTED"
	Completed State = "COMPLETED"
)

func (s State) String() string {
	return string(s)
}

func ParseAction(s string) action {
	switch strings.ToLower(s) {
	case "approve":
		return Approve
	case "reject":
		return Reject
	case "payout":
		return Payout
	}
	return UnknownAction
}

func ParseDomain(s string) domain {
	switch strings.ToLower(s) {
	case "sports":
		return Sports
	case "casino":
		return Casino
	case "manual":
		return Manual
	}
	return UnknownDomain
}
