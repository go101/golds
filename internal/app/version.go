package app

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"time"

	"go101.org/golds/internal/util"
)

const RoughBuildTime = "2022-08-25"

const Version = "v0.5.3-preview"

func releaseGolds() {
	if _, err := util.RunShell(time.Minute*3, "", nil, "go", "test", "./..."); err != nil {
		log.Println(err)
		return
	}
	if _, err := util.RunShell(time.Minute*3, "", nil, "go", "fmt", "./..."); err != nil {
		log.Println(err)
		return
	}
	if _, err := util.RunShell(time.Minute*3, "", nil, "go", "mod", "tidy"); err != nil {
		log.Println(err)
		return
	}

	const (
		TimeConstPrefix    = `const RoughBuildTime = "`
		VersionConstPrefix = `const Version = "v`
		PreviewSuffix      = "-preview"
	)

	var verisonGoFile = filepath.Join("internal", "app", "version.go")

	oldContent, err := ioutil.ReadFile(verisonGoFile)
	if err != nil {
		log.Printf("failed to load version.go: %s", err)
		return
	}

	i, j := bytes.Index(oldContent, []byte(TimeConstPrefix)), 0
	if i > 0 {
		i += len(TimeConstPrefix)
		j = bytes.IndexByte(oldContent[i:], '"')
		if j >= 0 {
			j += i
		}
	}
	if i <= 0 || j <= 0 {
		log.Printf("RoughBuildTime string not found (%d : %d)", i, j)
		return
	}

	m, n := bytes.Index(oldContent, []byte(VersionConstPrefix)), 0
	if m > 0 {
		m += len(VersionConstPrefix)
		n = bytes.IndexByte(oldContent[m:], '"')
		if n >= 0 {
			n += m
		}
	}
	if m <= 0 || n <= 0 {
		log.Printf("Version string not found (%d : %d)", m, n)
		return
	}
	if m < j || n < i {
		log.Println("Version string should be behind of RoughBuildTime string")
		return
	}

	oldVersion := bytes.TrimSuffix(oldContent[m:n], []byte(PreviewSuffix))
	noPreviewSuffix := len(oldVersion) == n-m
	mmp := bytes.SplitN(oldVersion, []byte{'.'}, -1)
	if len(mmp) != 3 {
		log.Printf("Version string not in MAJOR.MINOR.PATCH format: %s", oldVersion)
		return
	}

	major, err := strconv.Atoi(string(mmp[0]))
	if err != nil {
		log.Printf("parse MAJOR version (%s) error: %s", mmp[0], err)
		return
	}

	minor, err := strconv.Atoi(string(mmp[1]))
	if err != nil {
		log.Printf("parse MINOR version (%s) error: %s", mmp[1], err)
		return
	}

	patch, err := strconv.Atoi(string(mmp[2]))
	if err != nil {
		log.Printf("parse PATCH version (%s) error: %s", mmp[2], err)
		return
	}

	var incVersion = func() {
		patch = (patch + 1) % 10
		if patch == 0 {
			minor = (minor + 1) % 10
			if minor == 0 {
				major++
			}
		}
	}

	newContentLength := len(oldContent) + 1
	if noPreviewSuffix {
		newContentLength += len(PreviewSuffix)
		incVersion()
	}

	var newReleaseTime = time.Now().Format("2006-01-02")
	var nextReleaseTime = time.Now().Add(time.Hour * 24 * 50).Format("2006-01-02")
	var newVersion, newPreviewVersion []byte

	var buf = bytes.NewBuffer(make([]byte, 0, newContentLength))
	{
		buf.Reset()
		fmt.Fprintf(buf, "%d.%d.%d", major, minor, patch)
		newVersion = append(newVersion, buf.Bytes()...)
	}
	{
		incVersion()
		buf.Reset()
		fmt.Fprintf(buf, "%d.%d.%d", major, minor, patch)
		buf.WriteString(PreviewSuffix)
		newPreviewVersion = append(newPreviewVersion, buf.Bytes()...)
	}

	var writeNewContent = func(version []byte, releaseTime string) error {
		buf.Reset()
		buf.Write(oldContent[:i])
		buf.WriteString(releaseTime)
		buf.Write(oldContent[j:m])
		buf.Write(version)
		buf.Write(oldContent[n:])
		return ioutil.WriteFile(verisonGoFile, buf.Bytes(), 0644)
		//log.Printf("%s\n\n", buf.Bytes()[:n+1])
		//return nil
	}

	if err := writeNewContent(newVersion, newReleaseTime); err != nil {
		log.Printf("write release version file error: %s", err)
		return
	}

	var gitTag = fmt.Sprintf("v%s", newVersion)
	if output, err := util.RunShellCommand(time.Second*5, "", nil,
		"git", "commit", "-a", "-m", gitTag); err != nil {
		log.Printf("git commit error: %s\n%s", err, output)
	}
	if output, err := util.RunShellCommand(time.Second*5, "", nil,
		"git", "tag", gitTag); err != nil {
		log.Printf("git commit error: %s\n%s", err, output)
	}

	if err := writeNewContent(newPreviewVersion, nextReleaseTime); err != nil {
		log.Printf("write preview version file error: %s", err)
		return
	}

	log.Printf("new release time: %s", newReleaseTime)
	log.Printf("new version: %s", newVersion)
	log.Printf("new preview version: %s", newPreviewVersion)
}
