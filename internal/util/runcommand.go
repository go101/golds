package util

import (
	"context"
	"log"
	"os"
	"os/exec"
	"time"
)

func RunShellCommand(timeout time.Duration, wd string, envs []string, cmd string, args ...string) ([]byte, error) {
	if wd == "" {
		var err error
		wd, err = os.Getwd()
		if err != nil {
			log.Println(`Can't get current path. Set it as "."`)
			wd = "."
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, cmd, args...)
	command.Dir = wd
	command.Env = append(os.Environ(), envs...)
	return command.CombinedOutput()
}
