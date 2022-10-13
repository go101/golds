package server

import (
	"bytes"
	"errors"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	"go101.org/golds/code"
	"go101.org/golds/internal/util"
)

type BuildSourceLinkFunc func(w writer, commit, extraPath, sourcePath, line, endLine string) error

type CodeHost struct {
	ModulePathPrefix string
	// RepositryCharacteristics and GuessRepositryFromSourceURL should be both blank or both non-blank.
	// If it is not blank, the first one in it must be a prefix starting with "https://".
	// Others are character substrings.
	RepositryCharacteristics []string
	// If len(RepositryCharacteristics) > 0, prefix == RepositryCharacteristics[0].
	// And url is prefixed with prefix.
	GuessRepositryFromSourceURL   func(url, prefix string) (repo string, extra string)
	GuessRepositoryFromModulePath func(modulePath string) (repo string, extra string)

	// Only for the hosts which RepositryCharacteristics[0] is available.
	BuildSourceLink BuildSourceLinkFunc
}

var codeHosts = []CodeHost{
	{
		ModulePathPrefix: "github.com/",
		RepositryCharacteristics: []string{
			"https://github.com/",
			"@github.com:",
		},
		GuessRepositryFromSourceURL: guessRepositryFromSourceURL_1,
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			projecName, extraPath := splitByNthSlash(moduleRelativePath, 2)
			return "https://github.com/" + projecName, extraPath
		},
		BuildSourceLink: buildSourceLinkFunc_github,
	},
	{
		ModulePathPrefix: "gitlab.com/",
		RepositryCharacteristics: []string{
			"https://gitlab.com/",
			"@gitlab.com:",
		},
		GuessRepositryFromSourceURL: guessRepositryFromSourceURL_1,
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			projecName, extraPath := splitByNthSlash(moduleRelativePath, 2)
			// inaccurate for sub-orgonization projects, but not a big problen here?
			// ToDo: Not sure. Need check.
			return "https://gitlab.com/" + projecName, extraPath
		},
		BuildSourceLink: buildSourceLinkFunc_gitlab,
	},
	{
		ModulePathPrefix: "bitbucket.org/",
		RepositryCharacteristics: []string{
			"https://bitbucket.org/",
			"@bitbucket.org:",
		},
		GuessRepositryFromSourceURL: guessRepositryFromSourceURL_1,
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			projecName, extraPath := splitByNthSlash(moduleRelativePath, 2)
			return "https://bitbucket.org/" + projecName, extraPath
		},
		BuildSourceLink: buildSourceLinkFunc_bitbucket,
	},
	{
		ModulePathPrefix: "git.sr.ht/",
		RepositryCharacteristics: []string{
			"https://git.sr.ht/",
			"@git.sr.ht:",
		},
		GuessRepositryFromSourceURL:   guessRepositryFromSourceURL_1,
		GuessRepositoryFromModulePath: nil,
		BuildSourceLink:               buildSourceLinkFunc_sr_ht,
	},

	//===============================================
	{
		ModulePathPrefix: "gopkg.in/",
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			i := strings.LastIndex(moduleRelativePath, ".v")
			if i < 0 {
				return "", ""
			}
			if strings.IndexByte(moduleRelativePath[i:], '/') > 0 {
				return "", ""
			}
			path := moduleRelativePath[:i]
			j := strings.LastIndexByte(path, '/')
			if j < 0 {
				// gopkg.in/pkg.v3 → github.com/go-pkg/pkg
				return "https://github.com/go-" + path + "/" + path, ""
			}
			// gopkg.in/user/pkg.v3 → github.com/user/pkg
			return "https://github.com/" + path, ""
		},
	},
	{
		ModulePathPrefix: "golang.org/x/",
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			projecName, extraPath := splitByNthSlash(moduleRelativePath, 1)
			return "https://github.com/golang/" + projecName, extraPath
		},
	},
	//{
	//	ModulePathPrefix: "go101.org/",
	//	GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
	//		projecName, extraPath := splitByNthSlash(moduleRelativePath, 1)
	//		return "https://github.com/go101/" + projecName, extraPath
	//	},
	//},
	{
		ModulePathPrefix: "gioui.org",
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			projecName, extraPath := splitByNthSlash(moduleRelativePath, 1)
			return "https://git.sr.ht/~eliasnaur/gio" + projecName, extraPath
		},
	},
	{
		ModulePathPrefix: "inet.af/",
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			projecName, extraPath := splitByNthSlash(moduleRelativePath, 1)
			return "https://github.com/inetaf/" + projecName, extraPath
		},
	},
	{
		ModulePathPrefix: "sigs.k8s.io/",
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			projecName, extraPath := splitByNthSlash(moduleRelativePath, 1)
			return "https://github.com/kubernetes-sigs/" + projecName, extraPath
		},
	},
	{
		ModulePathPrefix: "k8s.io/",
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			projecName, extraPath := splitByNthSlash(moduleRelativePath, 1)
			return "https://github.com/kubernetes/" + projecName, extraPath
		},
	},
	{
		ModulePathPrefix: "go.etcd.io/",
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			projecName, extraPath := splitByNthSlash(moduleRelativePath, 1)
			return "https://github.com/etcd-io/" + projecName, extraPath
		},
	},
	{
		ModulePathPrefix: "go.uber.org/",
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			projecName, extraPath := splitByNthSlash(moduleRelativePath, 1)
			return "https://github.com/uber-go/" + projecName, extraPath
		},
	},
	{
		ModulePathPrefix: "cloud.google.com/go",
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			return "https://github.com/googleapis/google-cloud-go", ""
		},
	},
	{
		ModulePathPrefix: "google.golang.org/grpc",
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			return "https://github.com/grpc/grpc-go", ""
		},
	},
	{
		ModulePathPrefix: "google.golang.org/api",
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			return "https://github.com/googleapis/google-api-go-client", ""
		},
	},
	{
		ModulePathPrefix: "google.golang.org/genproto",
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			return "https://github.com/googleapis/go-genproto", ""
		},
	},
	{
		ModulePathPrefix: "google.golang.org/protobuf",
		GuessRepositoryFromModulePath: func(moduleRelativePath string) (string, string) {
			return "https://github.com/protocolbuffers/protobuf-go", ""
		},
	},
}

func buildSourceLinkFunc_github(w writer, commit, extraPath, sourcePath, line, endLine string) error {
	// https://github.com/user/project/blob/commit/extra/path/to/file.go#L11-L22
	items := make([]string, 0, 8)
	items = append(items, "/blob/", commit, extraPath, sourcePath)
	if line != "" {
		items = append(items, "#L", line)
		if endLine != "" {
			items = append(items, "-L", endLine)
		}
	}
	for _, s := range items {
		_, err := w.WriteString(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildSourceLinkFunc_gitlab(w writer, commit, extraPath, sourcePath, line, endLine string) error {
	// https://gitlab.com/user/project/~blob/commit/extra/path/to/file.go#L11-L22
	items := make([]string, 0, 8)
	items = append(items, "/~/blob/", commit, extraPath, sourcePath)
	if line != "" {
		items = append(items, "#L", line)
		if endLine != "" {
			items = append(items, "-", endLine)
		}
	}
	for _, s := range items {
		_, err := w.WriteString(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildSourceLinkFunc_bitbucket(w writer, commit, extraPath, sourcePath, line, endLine string) error {
	// https://bitbucket.org/user/project/src/commit/extra/path/to/file.go#lines-6:10
	items := make([]string, 0, 8)
	items = append(items, "/src/", commit, extraPath, sourcePath)
	if line != "" {
		items = append(items, "#lines-", line)
		if endLine != "" {
			items = append(items, ":", endLine)
		}
	}
	for _, s := range items {
		_, err := w.WriteString(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildSourceLinkFunc_sr_ht(w writer, commit, extraPath, sourcePath, line, endLine string) error {
	// https://git.sr.ht/~user/project/tree/commit/item/extra/path/to/file.go#L7-14
	items := make([]string, 0, 9)
	items = append(items, "/tree/", commit, "/item", extraPath, sourcePath)
	if line != "" {
		items = append(items, "#L", line)
		if endLine != "" {
			items = append(items, "-", endLine)
		}
	}
	for _, s := range items {
		_, err := w.WriteString(s)
		if err != nil {
			return err
		}
	}
	return nil
}

//=====================================
//
//=====================================

func (ds *docServer) tryToCompleteModuleInfo(m *code.Module) {
	//if ds.analysisWorkingDirectory == "" {
	//	ds.analysisWorkingDirectory = util.WorkingDirectory()
	//}

	// ToDo: handle modules feature off case in which module versions will always blank?
	//       Or best not to generate any modules in this case.
	//if m.ActualVersion() == "" && m.Replace.Path == "" { // wd module
	if m == ds.analyzer.WorkingDirectoryModule() {
		//if !strings.HasPrefix(ds.initialWorkingDirectory, m.Dir) {
		//	log.Printf("working directory module dir is not correct:\n\t%s\n\t%s", m.Dir, ds.initialWorkingDirectory)
		//	return
		//}
		// m.Dir might be prefix of ds.initialWorkingDirectory, or vice version:
		// 1. run "golds ./..." in subpackages of a module folder.
		// 2. run "golds foo/..." for the foo module.

		ds.tryRetrievingWorkdingDirectoryModuleInfo(m)
		// ToDo: also need ?go-get=1 query if ...
	} else {
		if strings.HasPrefix(m.Replace.Path, ".") {
			log.Printf("(replace) guess moudle %s repository (to use working directory module)", m.Path)
			return // local replacements will be handled in analyzer.
		}

		foundInVendor := false
		if m.ActualDir() == "" { // this happens for packages in project vendor folder
			func() {
				pkgDir := m.Pkgs[0].Directory
				in, relDir := ds.inVendor(pkgDir)
				if !in {
					return
				}

				path := strings.Replace(m.Path, "/", sep, -1)
				if !strings.HasPrefix(relDir, path) {
					return
				}
				foundInVendor = true

				relDir = relDir[len(m.Path):]
				m.Dir = pkgDir[:len(pkgDir)-len(relDir)]

				if m.RepositoryURL != "" {
					return
				}
				url, extra := guessRepositoryFromModulePath(m.ActualPath())
				if url != "" {

					m.RepositoryURL = url
					m.RepositoryDir = m.Dir[:len(m.Dir)-len(extra)]
					m.ExtraPathInRepository = extra

					if verboseLogs {
						log.Printf("(vendor) guess moudle %s repository: %s", m.Path, m.RepositoryURL)
					}
				}
			}()
		}

		if !foundInVendor && m.RepositoryURL == "" {
			func() {
				const atV = "@v"
				//var i int
				if m.ActualDir() == "" {
					pkgDir := m.Pkgs[0].Directory
					i := strings.LastIndex(pkgDir, atV)
					if i <= 0 {
						return
					}
					k := strings.IndexByte(pkgDir[i+len(atV):], filepath.Separator)
					if k < 0 {
						m.Dir = pkgDir
					} else {
						m.Dir = pkgDir[:i+len(atV)+k]
					}
				}

				url, extra := guessRepositoryFromModulePath(m.ActualPath())
				if url != "" {
					m.RepositoryURL = url
					m.RepositoryDir = m.Dir[:len(m.Dir)-len(extra)]
					m.ExtraPathInRepository = extra

					if verboseLogs {
						log.Printf("(modcache) guess moudle %s repository: %s", m.Path, m.RepositoryURL)
					}
				}
			}()
		}

		if m.RepositoryURL == "" && allowNetworkConnection {
			func() {
				srcRepo, extraPath, err := findSourceRepository(m.ActualPath())
				if err != nil {
					if verboseLogs {
						log.Printf("!!! query source repository for module %s error: %s", m.Path, err)
					}
					return
				}

				if verboseLogs {
					log.Printf(">>> query moudle %s repository: %s", m.Path, srcRepo)
				}

				if strings.HasSuffix(srcRepo, "/") {
					srcRepo = srcRepo[:len(srcRepo)-1]
				}

				m.RepositoryURL = srcRepo
				m.RepositoryDir = m.Dir[:len(m.Dir)-len(extraPath)]
				m.ExtraPathInRepository = extraPath
			}()
		}
	}
}

//===========

func guessRepositoryFromModulePath(modulePath string) (repoURL string, extraPath string) {
	for i := range codeHosts {
		host := &codeHosts[i]
		if strings.HasPrefix(modulePath, host.ModulePathPrefix) {
			if host.GuessRepositoryFromModulePath != nil {
				modulePath = removeVnSuffix(modulePath)
				return host.GuessRepositoryFromModulePath(modulePath[len(host.ModulePathPrefix):])
			}
		}
	}
	return "", ""
}

func removeVnSuffix(moudlePath string) string {
	i := strings.LastIndex(moudlePath, "/v")
	if i <= 0 {
		return moudlePath
	}

	k := i + 2
	for ; k < len(moudlePath); k++ {
		if b := moudlePath[k]; b < '0' || b > '9' {
			return moudlePath
		}
	}
	return moudlePath[:i]
}

//===========

func ensureHttpsRepositoryURL(url string) string {
	for i := range codeHosts {
		host := &codeHosts[i]
		if len(host.RepositryCharacteristics) > 0 {
			httpsPrefix := host.RepositryCharacteristics[0]
			for _, c := range host.RepositryCharacteristics[1:] {
				if k := strings.Index(url, c); k >= 0 {
					return httpsPrefix + url[k+len(c):]
				}
			}
		}
	}
	return url
}

//===========

func guessRepositryFromSourceURL(url string) (repoURL, extraPath string) {
	for i := range codeHosts {
		host := &codeHosts[i]
		if len(host.RepositryCharacteristics) > 0 {
			httpsPrefix := host.RepositryCharacteristics[0]
			if strings.HasPrefix(url, httpsPrefix) {
				if host.GuessRepositryFromSourceURL != nil {
					return host.GuessRepositryFromSourceURL(url, httpsPrefix)
				}
			}
		}
	}
	return "", ""
}

func guessRepositryFromSourceURL_1(url, prefix string) (repoURL, extraPath string) {
	srcPath := url[len(prefix):]
	items := strings.SplitN(srcPath, "/", 3)
	if len(items) < 3 {
		return url, ""
	}
	// ToDo: for gitlab, extraPath might be not blank.
	return url[:len(prefix)+len(items[0])+1+len(items[1])], ""
}

// The second result starts with / if not blank.
func splitByNthSlash(path string, n int) (string, string) {
	if n <= 0 {
		panic("n must be positive integer")
	}
	for p := path; len(p) > 0; {
		i := strings.IndexByte(p, '/')
		if i < 0 {
			break
		}
		p = p[i+1:]
		if n--; n == 0 {
			i = len(path) - len(p) - 1
			return path[:i], path[i:]
		}
	}
	return path, ""
}

const sep = string(filepath.Separator)
const sepVendorSep = sep + "vendor" + sep

func (ds *docServer) inVendor(pkgDir string) (bool, string) {
	dir := pkgDir
	wdModule := ds.analyzer.WorkingDirectoryModule()
	if wdModule == nil {
		panic("should not")
	}
	if !strings.HasPrefix(dir, wdModule.Dir) {
		return false, ""
	}
	dir = dir[len(wdModule.Dir):]
	if !strings.HasPrefix(dir, sepVendorSep) {
		return false, ""
	}
	dir = dir[len(sepVendorSep):]
	return true, dir
}

var dotgit = []byte(".git")
var slash = []byte("/")

// Make sure d.wdModule is conirmed before call this method.
func (ds *docServer) tryRetrievingWorkdingDirectoryModuleInfo(m *code.Module) {

	// ...
	output, err := util.RunShellCommand(time.Second*5, "", nil, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		if verboseLogs {
			log.Println("unable to confirm wording diretory module: not in a CVS (only supports git now) directory")
		}
		return
	}
	projectLocalDir := string(bytes.TrimSpace(output))

	// ...
	output, err = util.RunShellCommand(time.Second*5, "", nil, "git", "rev-parse", "HEAD")
	if err != nil {
		if verboseLogs {
			log.Printf("unable to confirm wording diretory module: git rev-parse HEAD error: %s", err)
		}
		return
	}
	commitHash := bytes.TrimSpace(output)

	// ...
	var remoteName string
	output, err = util.RunShellCommand(time.Second*5, "", nil, "git", "remote")
	if err != nil {
		if verboseLogs {
			log.Printf("unable to confirm wording diretory module: git remote error: %s", err)
		}
		return
	}
	output = bytes.TrimSpace(output)
	if i := bytes.IndexByte(output, '\n'); i < 0 {
		remoteName = string(output)
	} else {
		firstRemote := string(bytes.TrimSpace(output[:i]))

		// output: remote-name/remote-branch
		output, err = util.RunShellCommand(time.Second*5, "", nil, "git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
		if err != nil {
			if verboseLogs {
				log.Printf("unable to confirm wording diretory module: git rev-parse --abbrev-ref --symbolic-full-name @{upstream} error: %s", err)
			}
		} else if i := bytes.IndexByte(output, '/'); i <= 0 {
			if verboseLogs {
				log.Printf("unable to confirm wording diretory module: could not find remote in %s", output)
			}
		} else {
			remoteName = string(bytes.TrimSpace(output[:i]))
		}
		if remoteName == "" {
			remoteName = firstRemote
		}
	}
	output, err = util.RunShellCommand(time.Second*5, "", nil, "git", "remote", "get-url", remoteName)
	if err != nil {
		if verboseLogs {
			log.Printf("unable to confirm wording diretory module: git remote get-url origin %s error: %s", remoteName, output)
		}
		return
	}
	output = bytes.TrimSpace(output)
	if bytes.HasSuffix(output, dotgit) {
		output = output[:len(output)-len(dotgit)]
	}
	if bytes.HasSuffix(output, slash) {
		output = output[:len(output)-1]
	}
	projectRemoteURL := string(output)

	// ...
	var warnings []string
	output, err = util.RunShellCommand(time.Second*15, "", nil, "git", "status", "-s")
	output = bytes.TrimSpace(output)
	if err != nil {
		warnings = append(warnings, "unable to get project CVS commit status.")
		if verboseLogs {
			log.Printf("unable to get wording diretory commit status: git status -s: %s. %s", err, output)
		}
	} else if len(output) != 0 {
		warnings = append(warnings, "something in project haven't been committed yet")
	}
	//output, err = util.RunShellCommand(time.Second*5, "", nil, "git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	//output = bytes.TrimSpace(output)
	//if err != nil {
	//	warnings = append(warnings, "unable to get project CVS push status.")
	//	if verboseLogs {
	//		log.Printf("unable to get wording diretory push status: git rev-parse --abbrev-ref --symbolic-full-name @{upstream}: %s. %s", err, output)
	//	}
	//}
	//originBranch := string(output)
	//output, err = util.RunShellCommand(time.Second*5, "", nil, "git", "diff", originBranch)
	//output = bytes.TrimSpace(output)
	//if err != nil {
	//	warnings = append(warnings, "unable to get project CVS push status.")
	//	if verboseLogs {
	//		log.Printf("unable to get wording diretory push status: (git diff %s: %s. %s", originBranch, err, output)
	//	}
	//} else if len(output) != 0 {
	//	warnings = append(warnings, "something in project haven't been pushed yet to remote CVS")
	//}

	// ...

	if strings.HasPrefix(m.Dir, projectLocalDir) {
		m.ExtraPathInRepository = m.Dir[len(projectLocalDir):]
		m.Version = string(commitHash)
		m.RepositoryDir = projectLocalDir
		m.RepositoryURL = ensureHttpsRepositoryURL(projectRemoteURL)
		ds.wdRepositoryWarnings = warnings
	}

	if verboseLogs {
		log.Printf("(working directory) guess moudle %s repository: %s", m.Path, m.RepositoryURL)
	}
}

// ToDo: not a perfect implementation.
func findSourceRepository(forModule string) (repoURL, extraPath string, err error) {
	gogetURL := "https://" + forModule + "?go-get=1"
	code, _, data, err := util.HttpRequest("GET", gogetURL, nil, 15)
	if err != nil {
		return "", "", fmt.Errorf("GET %s error (code=%d): %s", gogetURL, code, err)
	}
	if code < 200 || code >= 300 {
		return "", "", fmt.Errorf("GET %s (code=%d)", gogetURL, code)
	}

	r := bytes.NewReader(data)
	doc, err := html.Parse(r)
	if err != nil {
		return "", "", fmt.Errorf("Parse HTML error: %s", err)
	}

	var importContnt, sourceContent string
	var f func(*html.Node) bool
	f = func(n *html.Node) bool {
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "meta") {
			name, content := "", ""
			for _, a := range n.Attr {
				if a.Key == "name" {
					name = a.Val
				} else if a.Key == "content" {
					content = a.Val
				}
			}
			if name == "go-source" {
				sourceContent = content
			} else if name == "go-import" {
				importContnt = content
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}

		if sourceContent != "" || importContnt != "" {
			return true
		}

		return false
	}
	f(doc)

	if sourceContent != "" {
		items := strings.Fields(sourceContent)
		if len(items) >= 4 {
			repoURL = items[3]
		} else if len(items) == 3 {
			repoURL = items[2]
		} else if len(items) == 2 {
			repoURL = items[1]
		}
		if repoURL != "" {
			repoURL, extraPath = guessRepositryFromSourceURL(repoURL)
		}
	}
	if repoURL != "" && importContnt != "" {
		items := strings.Fields(sourceContent)
		if len(items) >= 3 {
			repoURL = items[2]
		}
		if repoURL != "" {
			repoURL, extraPath = guessRepositryFromSourceURL(repoURL)
		}
	}
	if repoURL == "" {
		return "", "", errors.New("failed to find out")
	}
	return
}

//======================================

// devel go1.17-326a792517 Tue May 11 02:46:21 2021 +0000
var findGoVersionRegexp = regexp.MustCompile(`devel go[.0-9]+-([0-9a-fA-F]{6,})\s`)

func findGoToolchainVersionFromGoRoot(goroot string) (string, error) {
	versionData, err := ioutil.ReadFile(filepath.Join(goroot, "VERSION"))
	if err == nil {
		return string(bytes.TrimSpace(versionData)), nil
	} else {
		//panic("failed to get Go toolchain version in GOROOT: " + goroot)
	}
	versionData, err = ioutil.ReadFile(filepath.Join(goroot, "VERSION.cache"))
	if err != nil {
		return "", fmt.Errorf("failed to get Go toolchain version in GOROOT (%s): %w", goroot, err)
	}
	matches := findGoVersionRegexp.FindStringSubmatch(string(versionData))
	if len(matches) >= 2 {
		return matches[1], nil
	}
	return "", fmt.Errorf("failed to get Go toolchain version in GOROOT (%s)", goroot)
}

func findToolchainInfo() (toolchain code.ToolchainInfo, err error) {
	if _, err = os.Stat(build.Default.GOROOT); err != nil {
		return
	}
	version, err := findGoToolchainVersionFromGoRoot(build.Default.GOROOT)
	if err != nil {
		return
	}
	cmdPath := filepath.Join(build.Default.GOROOT, "src", "cmd")
	srcPath := filepath.Dir(cmdPath)
	rootPath := filepath.Dir(srcPath)
	toolchain = code.ToolchainInfo{
		Root:    rootPath,
		Src:     srcPath,
		Cmd:     cmdPath,
		Version: version,
	}
	return
}

//======================================

func (ds *docServer) printModulesInfo() {
	ds.analyzer.IterateModule(func(m *code.Module) {
		log.Printf("module: %s@%s (%d pkgs)", m.Path, m.ActualVersion(), len(m.Pkgs))
		log.Printf("            Pkgs[0].Dir: %s", m.Pkgs[0].Directory)
		log.Printf("                    Dir: %s", m.ActualDir())
		log.Printf("          RepositoryDir: %s", m.RepositoryDir)
		log.Printf("          RepositoryURL: %s", m.RepositoryURL)
		log.Printf("              ExtraPath: %s", m.ExtraPathInRepository)
		log.Printf("            Replace.Dir: %s", m.Replace.Dir)
		log.Printf("           Replace.Path: %s", m.Replace.Path)
	})
}

func (ds *docServer) confirmModuleBuildSourceLinkFuncs() {
	maxModuleIndex := -1
	ds.analyzer.IterateModule(func(m *code.Module) {
		if m.Index > maxModuleIndex {
			maxModuleIndex = m.Index
		}

	})
	ds.moduleBuildSourceLinkFuncs = make([]BuildSourceLinkFunc, maxModuleIndex+1)
	ds.analyzer.IterateModule(func(m *code.Module) {
		var f BuildSourceLinkFunc

		for i := range codeHosts {
			host := &codeHosts[i]
			if len(host.RepositryCharacteristics) > 0 {
				prefix := host.RepositryCharacteristics[0]
				if strings.HasPrefix(m.RepositoryURL, prefix) {
					f = host.BuildSourceLink
					break
				}
			}
		}

		ds.moduleBuildSourceLinkFuncs[m.Index] = f
	})

	if sourceReadingStyle == SourceReadingStyle_external {
		writeExternalSourceCodeLink = ds.buildExternelSourceLink
	}
}

// ToDo: pass a *Package instead of pkgPath to optimize.
func (ds *docServer) buildExternelSourceLink(w writer, pkgFile, line, endLine string) (handled bool, err error) {
	srcFile := ds.analyzer.SourceFile(pkgFile)
	if srcFile == nil {
		return false, fmt.Errorf("buildExternelSourceLink: source file %s not found", pkgFile)
	}

	if srcFile.GeneratedFile != "" {
		return
	}

	module := srcFile.Pkg.Module()
	if module == nil {
		return
	}

	buildSourceLinkFunc := ds.moduleBuildSourceLinkFuncs[module.Index]
	if buildSourceLinkFunc == nil {
		return
	}

	if _, err := w.WriteString(module.RepositoryURL); err != nil {
		return true, err
	}

	sourcePath := pkgFile[len(module.Path):]
	if err := buildSourceLinkFunc(w, module.RepositoryCommit, module.ExtraPathInRepository, sourcePath, line, endLine); err != nil {
		return true, err
	}

	return true, nil
}
