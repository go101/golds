package util

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func RunShellCommand(timeout time.Duration, wd string, envs []string, cmd string, args ...string) ([]byte, error) {
	if wd == "" {
		var err error
		wd, err = os.Getwd()
		if err != nil {
			log.Println(`Can't get current path. Set it as "."`)
			//wd = "."
			wd = ""
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, cmd, args...)
	command.Dir = wd
	command.Env = removeGODEBUG(append(os.Environ(), envs...))
	return command.CombinedOutput() // ToDo: maybe it is better not to combine.
}

func RunShell(timeout time.Duration, wd string, envs []string, cmdAndArgs ...string) ([]byte, error) {
	if len(cmdAndArgs) == 0 {
		panic("command is not specified")
	}

	return RunShellCommand(timeout, wd, envs, cmdAndArgs[0], cmdAndArgs[1:]...)
}

func removeGODEBUG(envs []string) []string {
	r := envs[:0]
	for _, e := range envs {
		if !strings.HasPrefix(e, "GODEBUG=") {
			r = append(r, e)
		}
	}
	return r
}
