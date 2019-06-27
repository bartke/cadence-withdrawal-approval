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

type Result struct {
	Source string
	Status string
}

// SampleWithdrawalWorkflow workflow decider
func SampleWithdrawalWorkflow(ctx workflow.Context, withdrawalID string) (result string, err error) {
	waitChannel := workflow.NewChannel(ctx)

	// step 1, create new withdrawal report
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
		HeartbeatTimeout:       time.Second * 20,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          time.Second,
			BackoffCoefficient:       2.0,
			MaximumInterval:          time.Minute,
			ExpirationInterval:       time.Minute * 5,
			MaximumAttempts:          10,
			NonRetriableErrorReasons: []string{},
		},
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

	// step 2.1 have one retryable context for the auto approvers
	ao = workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          time.Second,
			BackoffCoefficient:       2.0,
			MaximumInterval:          time.Minute,
			ExpirationInterval:       time.Minute * 5,
			MaximumAttempts:          10,
			NonRetriableErrorReasons: []string{"DISAPPROVED", "disapproved"},
		},
	}
	ctx3 := workflow.WithActivityOptions(ctx, ao)

	// we're trying to reach two auto approvals in parallel

	workflow.Go(ctx3, func(ctx workflow.Context) {
		var status string
		err = workflow.ExecuteActivity(ctx, waitForAutomatedActivity, withdrawalID, ":8091").Get(ctx, &status)
		if err != nil {
			logger.Error("Activity failed", zap.Error(err))
		}
		waitChannel.Send(ctx, Result{"sports", status})
	})

	workflow.Go(ctx3, func(ctx workflow.Context) {
		var status string
		err = workflow.ExecuteActivity(ctx, waitForAutomatedActivity, withdrawalID, ":8092").Get(ctx, &status)
		if err != nil {
			logger.Error("Activity failed", zap.Error(err))
		}
		waitChannel.Send(ctx, Result{"casino", status})
	})

	// add the manual workflow

	workflow.Go(ctx3, func(ctx workflow.Context) {
		var status string
		err = workflow.ExecuteActivity(ctx, waitForManualActivity, withdrawalID).Get(ctx, &status)
		if err != nil {
			logger.Error("Activity failed", zap.Error(err))
		}
		waitChannel.Send(ctx, Result{"manual", status})
	})

	// wait for the coroutinue to check in.

	// NOTE: this state should be kept in an application or behind a poller in
	// the real world, here we keep it exemplary
	var status string
	approvals := map[string]string{
		"sports": "PENDING",
		"casino": "PENDING",
		"manual": "PENDING",
	}
	for {
		if approvals["sports"] == "APPROVED" && approvals["casino"] == "APPROVED" {
			err = workflow.ExecuteActivity(ctx3, autoApprove, withdrawalID).Get(ctx3, nil)
			if err != nil {
				return "", nil
			}
			status = "APPROVED"
			break
		}
		if approvals["manual"] != "PENDING" {
			status = approvals["manual"]
			break
		}

		var v interface{}
		waitChannel.Receive(ctx3, &v)
		switch r := v.(type) {
		case error:
			// ignore
		case Result:
			logger.Info("Result received "+r.Source, zap.String("WithdrawalStatus", status))
			approvals[r.Source] = r.Status
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
