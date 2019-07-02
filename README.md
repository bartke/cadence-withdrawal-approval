## Cadence Withdrawal Approval POC

Sample workflow to evaluate cadence for an exemplary withdrawal approval
system based on [https://github.com/samarabbas/cadence-samples](https://github.com/samarabbas/cadence-samples).

![concept](https://github.com/bartke/cadence-withdrawal-approval/blob/master/screenshots/concept.png "Withdrawal Example Concept")

### Description

- Create a new withdrawal request to start the workflow
- try to contact two separate sample auto approval systems
    - if both auto approval systems approve the request, process it
    - if unreachable, keep retrying unless the error is a disapproval
    - if either one of the approval systems rejects or is unrachable wait for user input
- user input can take an arbitrary amount of time
    - register a callback and return `activity.ErrResultPending`
    - when the user approves, notify via `WorkflowClient.CompleteActivity()`
    - note that this could also be accomplished via polling
- the payout is processed if either both approval systems or an end user approves

### Steps to Run

Setup a cadence service running, see [github.com/uber/cadence](https://github.com/uber/cadence/blob/master/README.md).

```
go get -d github.com/uber/cadence
cd $GOPATH/src/github.com/uber/cadence/docker
docker-compose up
```

Start the dummy server:

```
dummy-server
```

Start two sample auto approval systems, both approving randomly.

```
auto-approver -p 8091
auto-approver -p 8092
```

Start the workflow and activity workers

```
withdrawal -m worker
```

Start the withdrawal workflow by creating a new entry:

```
withdrawal -m trigger
```

Go to [localhost](http://localhost:8099/list) to approve the withdrawals if
one of the two auto approvals fail. You should see the workflow complete after
you approve the withdrawal request. You can also reject it.

The system should allow for auto approvers to drop out and in as well as the
dummy server to spawn after we already triggered withdrawals.

