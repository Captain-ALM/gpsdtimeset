package utils

import (
	"errors"
	"os/exec"
	"time"
)

func SetTime(time time.Time) error {
	ccmd := exec.Command("date", time.Format("010215042006.05"))
	err := ccmd.Run()
	if err != nil {
		return err
	}
	if ccmd.ProcessState.Success() {
		return nil
	}
	return errors.New("process exit code failed")
}
