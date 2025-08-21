package util

import (
	"bytes"
	"context"
	"fmt"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	if cmd == "go" {
		envs = addGOTOOLCHAIN(envs)
		if build.Default.GOROOT != "" {
			cmd = filepath.Join(build.Default.GOROOT, "bin", "go")
		}
	}

	//println("======== cmd:", cmd);
	command.Env = removeGODEBUG(append(os.Environ(), envs...))
	var erroutput bytes.Buffer
	command.Stderr = &erroutput
	output, err := command.Output()
	if err != nil {
		if erroutput.Len() > 0 {
			err = fmt.Errorf("%w\n\n%s", err, erroutput.Bytes())
		}
	}

	//log.Println(">>>", cmd, args)
	//log.Printf("=== wd: %s", wd)
	//log.Printf("=== envs: %s", envs)
	//log.Printf("=== error: %s", err)
	//log.Printf("=== output: %s", output)

	return output, err
}

func RunShell(timeout time.Duration, wd string, envs []string, cmdAndArgs ...string) ([]byte, error) {
	if len(cmdAndArgs) == 0 {
		panic("command is not specified")
	}

	return RunShellCommand(timeout, wd, envs, cmdAndArgs[0], cmdAndArgs[1:]...)
}

func addGOTOOLCHAIN(envs []string) []string {
	for _, e := range envs {
		if strings.HasPrefix(e, "GOTOOLCHAIN=") {
			return envs
		}
	}
	return append(envs, "GOTOOLCHAIN=local")
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
