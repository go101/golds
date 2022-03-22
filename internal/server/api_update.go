package server

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go101.org/golds/internal/util"
)

const DurationToUpdate = time.Hour * 24 * 64

const (
	UpdateTip_Nothing = iota
	UpdateTip_ToUpdate
	UpdateTip_Updating
	UpdateTip_Updated
)

var UpdateTip2DivID = []string{
	UpdateTip_Nothing:  "",
	UpdateTip_ToUpdate: "to-update",
	UpdateTip_Updating: "updating",
	UpdateTip_Updated:  "updated",
}

// Must be called when locking.
func (ds *docServer) confirmUpdateTip() {
	if ds.updateTip == UpdateTip_Updating {
		return
	}

	d := time.Now().Sub(ds.roughBuildTime())
	needCheckUpdate := d > DurationToUpdate
	if needCheckUpdate {
		ds.updateTip = UpdateTip_ToUpdate
		ds.newerVersionInstalled = false
	} else if ds.newerVersionInstalled {
		ds.updateTip = UpdateTip_Updated
	} else {
		ds.updateTip = UpdateTip_Nothing
	}
}

// update page.
func (ds *docServer) startUpdatingGold() {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	ds.confirmUpdateTip()

	if ds.updateTip == UpdateTip_ToUpdate {
		ds.updateTip = UpdateTip_Updating
		go ds.updateGold()
	}
}

// api:update
// - GET: get current update info.
// - POST: do update
func (ds *docServer) updateAPI(w http.ResponseWriter, r *http.Request) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	ds.confirmUpdateTip()

	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"updateStatus": "%s"}`, UpdateTip2DivID[ds.updateTip])
		return
	}

	if r.Method == http.MethodPost {
		w.WriteHeader(http.StatusAccepted)
		w.Header().Set("Content-Type", "application/json")
		if ds.updateTip == UpdateTip_ToUpdate {
			ds.updateTip = UpdateTip_Updating
			go ds.updateGold()
		}
		fmt.Fprintf(w, `{"updateStatus": "%s"}`, UpdateTip2DivID[ds.updateTip])
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func (ds *docServer) onUpdateDone(succeeded bool) {
	var now = time.Now()

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	ds.roughBuildTime = func() time.Time {
		return now
	}
	ds.newerVersionInstalled = succeeded
	ds.updateTip = UpdateTip_Nothing
}

var versionRegexp = regexp.MustCompile(`go[0-9]+\.?[0-9]*`)

func ParseGoVersion(versionStr []byte) (major, minor int64, err error) {
	version := versionRegexp.Find(versionStr)
	if version == nil {
		err = fmt.Errorf("no version info in %s", versionStr)
		return
	}

	version = version[2:]
	idx := bytes.IndexByte(version, '.')
	if idx < 0 {
		idx = len(version)
	} else {
		minor, _ = strconv.ParseInt(string(version[idx+1:]), 10, 32)
	}
	major, _ = strconv.ParseInt(string(version[:idx]), 10, 32)
	return
}

func GoldsUpdateGoSubCommand(pkgPath string) string {
	output, err := util.RunShellCommand(time.Second*5, "", nil, "go", "version")
	if err != nil {
		return ""
	}

	major, minor, err := ParseGoVersion(output)
	isGo1_16plus := err == nil && major >= 1 && minor >= 16
	if isGo1_16plus {
		return fmt.Sprintf("install %s@latest", pkgPath)
	} else {
		return fmt.Sprintf("get -u %s", pkgPath)
	}
}

func (ds *docServer) updateGold() {
	if err := func() error {
		dir, err := ioutil.TempDir("", "*")
		if err != nil {
			return err
		}

		subCommand := GoldsUpdateGoSubCommand(ds.appPkgPath)
		if subCommand == "" {
			return errors.New("don't how to update Golds")
		}
		ds.updateLogger.Printf("Run: go %s", subCommand)
		output, err := util.RunShellCommand(time.Minute*3, dir, []string{"GO111MODULE=on"}, "go", strings.SplitN(subCommand, " ", -1)...)
		if len(output) > 0 {
			ds.updateLogger.Printf("\n%s\n", output)
		}
		if err != nil {
			return err
		}

		return nil
	}(); err != nil {
		ds.onUpdateDone(false)
		ds.updateLogger.Println("Update Golds error:", err)
	} else {
		ds.onUpdateDone(true)
		ds.updateLogger.Println("Update Golds succeeded.")
	}
}
