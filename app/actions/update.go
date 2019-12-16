package actions

import (
	"net/http"
	"os/exec"
	"strconv"
	"time"

	"github.com/lbryio/lighthouse/meta"

	"github.com/lbryio/lbry.go/v2/extras/api"
	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/travis"

	"github.com/sirupsen/logrus"
)

// AutoUpdateCommand is the path of the shell script to run in the environment chainquery is installed on. It should
// stop the service, download and replace the new binary from https://github.com/lbryio/chainquery/releases, start the
// service.
var AutoUpdateCommand = ""

// AutoUpdateAction takes a travis webhook for a successful deployment and runs an environment script to self update.
func AutoUpdateAction(r *http.Request) api.Response {
	err := travis.ValidateSignature(false, r)
	if err != nil {
		logrus.Info(err)
		return api.Response{Error: err, Status: http.StatusBadRequest}
	}

	webHook, err := travis.NewFromRequest(r)
	if err != nil {
		return api.Response{Error: err}
	}

	if webHook.Commit == meta.GetVersion() {
		logrus.Info("same commit version, skipping automatic update.")
		return api.Response{Data: "same commit version, skipping automatic update."}
	}

	shouldUpdate := webHook.Status == 0 && !webHook.PullRequest && webHook.Tag != "" && webHook.Tag != meta.GetVersion()
	if shouldUpdate { // webHook.ShouldDeploy() doesn't work for lighthouse autoupdate.
		if AutoUpdateCommand == "" {
			err := errors.Base("auto-update triggered, but no auto-update command configured")
			logrus.Error(err)
			return api.Response{Error: err}
		}
		logrus.Info("lighthouse is auto-updating...prepare for shutdown")
		// run auto-update asynchronously
		go func() {
			time.Sleep(1 * time.Second) // leave time for handler to send response
			cmd := exec.Command(AutoUpdateCommand)
			out, err := cmd.Output()
			if err != nil {
				errMsg := "auto-update error: " + errors.FullTrace(err) + "\nStdout: " + string(out)
				if exitErr, ok := err.(*exec.ExitError); ok {
					errMsg = errMsg + "\nStderr: " + string(exitErr.Stderr)
				}
				logrus.Errorln(errMsg)
			}
		}()
		return api.Response{Data: "Successful launch of auto update"}

	}
	message := "Auto-Update should not be deployed for one of the following:" +
		" CI Status-" + webHook.StatusMessage +
		", IsPullRequest-" + strconv.FormatBool(webHook.PullRequest) +
		", TagName-" + webHook.Tag
	return api.Response{Data: message}
}
