package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"
)

// This is registration process where you register all your activity handlers.
func init() {
	activity.Register(createWithdrawalActivity)
	activity.Register(waitForManualActivity)
	activity.Register(waitForAutomatedActivity)
	activity.Register(autoAction)
	activity.Register(paymentActivity)
	activity.Register(getStatus)
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

// waitForManualActivity waits for the withdrawal decision. This activity will complete asynchronously. When this method
// returns error activity.ErrResultPending, the cadence client recognize this error, and won't mark this activity
// as failed or completed. The cadence server will wait until Client.CompleteActivity() is called or timeout happened
// whichever happen first. In this sample case, the CompleteActivity() method is called by our dummy withdrawal server when
// the withdrawal is approved.
func waitForManualActivity(ctx context.Context, withdrawalID string) (string, error) {
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
		logger.Info("waitForManualActivity failed to register callback.", zap.Error(err))
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

func address(domain string) string {
	if domain == "sports" {
		return autoApprovalSystemSports
	}
	return autoApprovalSystemCasino
}

func waitForAutomatedActivity(ctx context.Context, withdrawalID, domain string) (string, error) {
	if len(withdrawalID) == 0 {
		return "", errors.New("withdrawal id is empty")
	}

	resp, err := http.Get(address(domain) + "/?id=" + withdrawalID)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}

	if string(body) != "APPROVE" && string(body) != "REJECT" {
		activity.GetLogger(ctx).Info("paymentActivity auto action:"+string(body), zap.String("WithdrawalID", withdrawalID))
		// non retryable path
		return "", cadence.NewCustomError(string(body))
	}

	return string(body), nil
}

func autoAction(ctx context.Context, withdrawalID, domain, action string) error {
	activity.GetLogger(ctx).Info("paymentActivity try to auto approved", zap.String("WithdrawalID", withdrawalID))

	// approve in the system
	approveURL := withdrawalServerHostPort + "/action?is_api_call=true&domain=" + domain + "&type=" + strings.ToLower(action) + "&id=" + withdrawalID
	resp, err := http.Get(approveURL)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	if string(body) != "SUCCEED" {
		activity.GetLogger(ctx).Info("paymentActivity auto action failed", zap.String("WithdrawalID", withdrawalID))
		return errors.New(string(body))
	}

	// feedback
	activity.GetLogger(ctx).Info("paymentActivity auto action succeeded", zap.String("WithdrawalID", withdrawalID))
	return nil
}

func getStatus(ctx context.Context, withdrawalID string) (string, error) {
	if len(withdrawalID) == 0 {
		return "", errors.New("withdrawal id is empty")
	}

	resp, err := http.Get(withdrawalServerHostPort + "/status?id=" + withdrawalID)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func paymentActivity(ctx context.Context, withdrawalID string) error {
	if len(withdrawalID) == 0 {
		return errors.New("withdrawal id is empty")
	}

	resp, err := http.Get(withdrawalServerHostPort + "/action?is_api_call=true&type=payout&id=" + withdrawalID)
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
