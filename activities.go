package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"go.uber.org/cadence/activity"
	"go.uber.org/zap"
)

// This is registration process where you register all your activity handlers.
func init() {
	activity.Register(createWithdrawalActivity)
	activity.Register(waitForDecisionActivity)
	activity.Register(paymentActivity)
}

func createWithdrawalActivity(ctx context.Context, withdrawalID string) error {
	if len(withdrawalID) == 0 {
		return errors.New("withdrawal id is empty")
	}

	resp, err := http.Get(withdrawalServerHostPort + "/create?is_api_call=true&id=" + withdrawalID)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	if string(body) == "SUCCEED" {
		activity.GetLogger(ctx).Info("Withdrawal created.", zap.String("WithdrawalID", withdrawalID))
		return nil
	}

	return errors.New(string(body))
}

// waitForDecisionActivity waits for the withdrawal decision. This activity will complete asynchronously. When this method
// returns error activity.ErrResultPending, the cadence client recognize this error, and won't mark this activity
// as failed or completed. The cadence server will wait until Client.CompleteActivity() is called or timeout happened
// whichever happen first. In this sample case, the CompleteActivity() method is called by our dummy withdrawal server when
// the withdrawal is approved.
func waitForDecisionActivity(ctx context.Context, withdrawalID string) (string, error) {
	if len(withdrawalID) == 0 {
		return "", errors.New("withdrawal id is empty")
	}

	logger := activity.GetLogger(ctx)

	// save current activity info so it can be completed asynchronously when withdrawal is approved/rejected
	activityInfo := activity.GetInfo(ctx)
	formData := url.Values{}
	formData.Add("task_token", string(activityInfo.TaskToken))

	registerCallbackURL := withdrawalServerHostPort + "/registerCallback?id=" + withdrawalID
	resp, err := http.PostForm(registerCallbackURL, formData)
	if err != nil {
		logger.Info("waitForDecisionActivity failed to register callback.", zap.Error(err))
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}

	status := string(body)
	if status == "SUCCEED" {
		// register callback succeed
		logger.Info("Successfully registered callback.", zap.String("WithdrawalID", withdrawalID))

		// ErrActivityResultPending is returned from activity's execution to indicate the activity is not completed when it returns.
		// activity will be completed asynchronously when Client.CompleteActivity() is called.
		return "", activity.ErrResultPending
	}

	logger.Warn("Register callback failed.", zap.String("WithdrawalStatus", status))
	return "", fmt.Errorf("register callback failed status:%s", status)
}

func paymentActivity(ctx context.Context, withdrawalID string) error {
	if len(withdrawalID) == 0 {
		return errors.New("withdrawal id is empty")
	}

	resp, err := http.Get(withdrawalServerHostPort + "/action?is_api_call=true&type=payment&id=" + withdrawalID)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	if string(body) == "SUCCEED" {
		activity.GetLogger(ctx).Info("paymentActivity succeed", zap.String("WithdrawalID", withdrawalID))
		return nil
	}

	return errors.New(string(body))
}
