## Cadence Withdrawal Approval

This sample workflow process a withdrawal request workflow with an
asynchronous activity.

### Sample Description

- Create a new withdrawal request.
- Wait for the withdrawal request to be approved. This could take an arbitrary
amount of time. So the activity's Execute method has to return before it is
actually approved. This is done by returning a special error so the framework
knows the activity is not completed yet.
  - When the withdrawal is approved (or rejected), it will notify via a call
  to WorkflowClient.CompleteActivity() to tell cadence service that that
  activity is now completed. In this sample case, the dummy server do this
  job. In real world, you will need to register some listener to the
  withdrawal system or you will need to have a polling agent to check
  for the withdrawal status periodically.
- After the wait activity is completed, it did the payment for the withdrawal.

### Steps

You need a cadence service running. See https://github.com/uber/cadence/blob/master/README.md for more details.

```
dummy-server
```

Start the workflow and activity workers

```
withdrawal -m worker
```

Start the withdrawal workflow by creating a new entry:

```
withdrawal -m trigger
```

Go to [localhost](http://localhost:8080/list) to approve the withdrawal. You
should see the workflow complete after you approve the withdrawal request.
You can also reject it.

