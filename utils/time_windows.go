package utils

import (
	"errors"
	"os/exec"
	"time"
)

func SetTime(time time.Time) error {
	ccmd := exec.Command("date", time.Format("02-01-2006"))
	err := ccmd.Run()
	if err != nil {
		return err
	}
	if !ccmd.ProcessState.Success() {
		return errors.New("process exit code failed")
	}
	ccmd = exec.Command("time", time.Format("15:04:05"))
	err = ccmd.Run()
	if err != nil {
		return err
	}
	if ccmd.ProcessState.Success() {
		return nil
	}
	return errors.New("process exit code failed")
}
