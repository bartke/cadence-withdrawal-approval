package main

import (
	"time"

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

// This is registration process where you register all your workflow handlers.
func init() {
	workflow.Register(SampleWithdrawalWorkflow)
}

// SampleWithdrawalWorkflow workflow decider
func SampleWithdrawalWorkflow(ctx workflow.Context, withdrawalID string) (result string, err error) {
	waitChannel := workflow.NewChannel(ctx)

	// step 1, create new withdrawal report
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
		HeartbeatTimeout:       time.Second * 20,
	}
	ctx1 := workflow.WithActivityOptions(ctx, ao)
	logger := workflow.GetLogger(ctx)

	err = workflow.ExecuteActivity(ctx1, createWithdrawalActivity, withdrawalID).Get(ctx1, nil)
	if err != nil {
		logger.Error("Failed to create withdrawal report", zap.Error(err))
		return "", err
	}

	// step 2, wait for the withdrawal report to be approved (or rejected)
	ao = workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
	}
	ctx2 := workflow.WithActivityOptions(ctx, ao)

	// have one retryable context
	ao = workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          time.Second,
			BackoffCoefficient:       2.0,
			MaximumInterval:          time.Minute,
			ExpirationInterval:       time.Minute * 5,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{"DISAPPROVED"},
		},
	}
	ctx3 := workflow.WithActivityOptions(ctx, ao)

	// Notice that we set the timeout to be 10 minutes for this sample demo.
	// If the expected time for the activity to complete (waiting for human to
	// approve the request) is longer, you should set the timeout accordingly
	// so the cadence system will wait accordingly. Otherwise, cadence system
	// could mark the activity as failure by timeout.

	workflow.Go(ctx3, func(ctx workflow.Context) {
		var status string
		err = workflow.ExecuteActivity(ctx, waitForAutomatedActivity, withdrawalID, ":8091").Get(ctx, &status)
		if err != nil {
			logger.Error("Activity failed", zap.Error(err))
		}
		waitChannel.Send(ctx, status)
	})

	workflow.Go(ctx3, func(ctx workflow.Context) {
		var status string
		err = workflow.ExecuteActivity(ctx, waitForAutomatedActivity, withdrawalID, ":8092").Get(ctx, &status)
		if err != nil {
			logger.Error("Activity failed", zap.Error(err))
		}
		waitChannel.Send(ctx, status)
	})

	// wait for both of the coroutinue to complete.
	var status, statusA, statusB string
	waitChannel.Receive(ctx3, &statusA)
	waitChannel.Receive(ctx3, &statusB)

	if statusA == "APPROVED" && statusB == "APPROVED" {
		status = "APPROVED"
	} else {
		// step 2.1, optionally taking the manual branch if automated approval
		// fails
		logger.Info("Workflow taking manual branch.", zap.String("WithdrawalStatus", status))
		err = workflow.ExecuteActivity(ctx2, waitForManualActivity, withdrawalID).Get(ctx2, &status)
		if err != nil {
			return "", err
		}
	}

	if status != "APPROVED" {
		logger.Info("Workflow completed.", zap.String("WithdrawalStatus", status))
		return "", nil
	}

	// step 3, trigger payment to the withdrawal
	err = workflow.ExecuteActivity(ctx2, paymentActivity, withdrawalID).Get(ctx2, nil)
	if err != nil {
		logger.Info("Workflow completed with payment failed.", zap.Error(err))
		return "", err
	}

	logger.Info("Workflow completed with withdrawal payment completed.")
	return "COMPLETED", nil
}
