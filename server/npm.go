package server

import (
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
)

type npmCmd struct {
	*exec.Cmd
}

func npmExec() *npmCmd {
	// Create the command for the frontend build server in development mode.
	// We consider "running on port 3000" to be development mode.
	var cmd *exec.Cmd
	if isDevelopment {
		cmd = exec.Command("npm", "start")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			logrus.Errorf("Error starting npm command: %+v", err)
		}
	}
	return &npmCmd{cmd}
}

func (cmd *npmCmd) stop() {
	if err := cmd.Process.Kill(); err != nil {
		logrus.Errorf("Could not kill the npm command: %+v", err)
	}
	logrus.Infof("Stopped npm command with PID %d", cmd.Process.Pid)
}
