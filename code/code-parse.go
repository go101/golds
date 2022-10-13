package code

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/tools/go/packages"

	"go101.org/golds/internal/util"
)

//func avoidCheckFuncBody(fset *token.FileSet, parseFilename string, _ []byte) (*ast.File, error) {
//	var src interface{}
//	mode := parser.ParseComments // | parser.AllErrors
//	file, err := parser.ParseFile(fset, parseFilename, src, mode)
//	if file == nil {
//		return nil, err
//	}
//	for _, decl := range file.Decls {
//		if fd, ok := decl.(*ast.FuncDecl); ok {
//			fd.Body = nil
//		}
//	}
//	return file, nil
//}

func collectPPackages(ppkgs []*packages.Package) map[string]*packages.Package {
	var allPPkgs = make(map[string]*packages.Package, 1000)
	var regPkgs func(ppkg *packages.Package)
	regPkgs = func(ppkg *packages.Package) {
		if _, present := allPPkgs[ppkg.PkgPath]; present {
			return
		}

		allPPkgs[ppkg.PkgPath] = ppkg
		for _, p := range ppkg.Imports {
			regPkgs(p)
		}
	}

	for _, ppkg := range ppkgs {
		regPkgs(ppkg)
	}

	return allPPkgs
}

func getMatchedPackages(arg string, jsonFormat bool) ([][]byte, error) {
	var output []byte
	var err error
	if jsonFormat {
		output, err = util.RunShell(time.Minute*3, "", nil, "go", "list", "-find", "-json", arg)
	} else {
		output, err = util.RunShell(time.Minute*3, "", nil, "go", "list", "-find", arg)
	}
	if err != nil {
		return nil, fmt.Errorf("go list %s error: %w", arg, err)
	}
	output = bytes.TrimSpace(output)
	//if bytes.HasPrefix(output, []byte("go: ")) {
	//	return nil, fmt.Errorf("go list %s error: %s", arg, output)
	//}
	if bytes.HasPrefix(output, []byte("no required module provides package")) {
		return nil, fmt.Errorf("go list %s error: %s", arg, output)
	}
	if jsonFormat {
		if !bytes.HasPrefix(output, []byte("{")) {
			return nil, fmt.Errorf("go list %s error: %s", arg, output)
		}
	}

	return bytes.Fields(output), nil
}

func hasMatchedPackages(arg string) bool {
	//out, err := getMatchedPackages(arg, true)
	out, err := getMatchedPackages(arg, false)
	return err == nil && len(out) > 0
}

//func collectStdPackages() ([]string, error) {
//	//log.Println("[collect std packages ...]")
//	//defer log.Println("[collect std packages done]")
//
//	var configForCollectStdPkgs = &packages.Config{
//		Tests: false,
//	}
//
//	ppkgs, err := packages.Load(configForCollectStdPkgs, "std")
//	if err != nil {
//		return nil, err
//	}
//
//	pkgs := make([]string, 0, len(ppkgs)+1)
//	pkgs = append(pkgs, "builtin")
//	for _, pp := range ppkgs {
//		pkgs = append(pkgs, pp.PkgPath)
//	}
//
//	return pkgs, nil
//}

func validateArgumentsAndSetOptions(args []string, toolchainPath string) ([]string, bool, error) {
	if len(args) == 0 {
		//panic("should not")
		return []string{"."}, false, nil
	}

	hasToolchain := false
	stdOnly := len(args) == 1 && args[0] == "std"
	if stdOnly {
		// increase the success rate.
		os.Setenv("GO111MODULE", "off")
		os.Setenv("CGO_ENABLED", "0")
	} else {
		hasOthers := false
		oldArgs := args
		args = args[:0]
		for _, p := range oldArgs {
			if p == "std" {
				args = append(args, p)
			} else if strings.HasPrefix(p, "cmd/") || p == "cmd" || p == "toolchain" {
				if p == "toolchain" {
					p = "cmd"
				}

				hasToolchain = true
				if _, err := os.Stat(toolchainPath); errors.Is(err, os.ErrNotExist) {
					log.Printf("the toolchain argument is ignored for the assumed source directory (%s) doesn not exist", toolchainPath)
					continue
				}
				// looks both are ok.
				//args = append(args, toolchainPath + string(filepath.Separator) + "..."
				//args = append(args, "./...")
				args = append(args, p)
			} else if p == "." || strings.HasPrefix(p, "./") {
				args = append(args, p)
			} else if strings.HasPrefix(p, ".\\") {
				args = append(args, strings.Replace(p, "\\", "/", -1))
			} else {
				if !hasMatchedPackages(p) {
					//log.Printf("argument %s does not match any package, so it is discarded", p)
					continue
				}
				hasOthers = true
				//if p == "." || strings.HasPrefix(p, "./") || strings.HasPrefix(p, ".\\") {
				//	dotPath = p
				//}
				args = append(args, p)
			}
		}
		_ = hasOthers
		//if hasToolchain {
		//	if hasOthers {
		//		return nil, hasToolchain, fmt.Errorf("the toolchain pseudo module name can only be used solely or alongside with the std pseudo module name\n")
		//	}
		//	if err := os.Chdir(toolchainPath); err != nil {
		//		return nil, hasToolchain, fmt.Errorf("change dir to toolchain path error: %w", err)
		//	}
		//}
	}

	return args, hasToolchain, nil
}

type LoadError struct {
	Errs []error
}

func (le *LoadError) Error() string { // just to implement error
	return fmt.Sprintf("%d errors", len(le.Errs))
}

// ParsePackages parses input packages.
func (d *CodeAnalyzer) ParsePackages(onSubTaskDone func(int, time.Duration, ...int32), completeModuleInfo func(*Module), toolchain ToolchainInfo, args ...string) error {
	// the length of the input args is not zero for sure.
	oldArgs := args

	args, hasToolchain, err := validateArgumentsAndSetOptions(args, toolchain.Cmd)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		if len(oldArgs) != 1 || strings.HasPrefix(oldArgs[0], ".") {
			return errors.New("no packages matched")
		}

		// construct a temp project, ...

		// go mod init golds.app/tmp
		// go get oldArgs[0]

		tempDir, err := os.MkdirTemp("", "golds-temp-project-*")
		if err != nil {
			return fmt.Errorf("create temp dir error: %w", err)
		}
		defer os.RemoveAll(tempDir)
		// println(tempDir)
		oldDir := util.WorkingDirectory()
		err = os.Chdir(tempDir)
		if err != nil {
			return fmt.Errorf("enter temp dir error: %w", err)
		}
		defer os.Chdir(oldDir)

		_, err = util.RunShell(time.Minute*3, "", nil, "go", "mod", "init", "golds.app/tmp")
		if err != nil {
			return fmt.Errorf("go mod init error: %w", err)
		}
		_, err = util.RunShell(time.Minute*3, "", nil, "go", "get", "-d", oldArgs[0])
		if err != nil {
			return fmt.Errorf("go get %s error: %w", oldArgs[0], err)
		}

		atIndex := strings.IndexByte(oldArgs[0], '@')
		if atIndex > 0 {
			oldArgs[0] = oldArgs[0][0:atIndex]
		}

		args = oldArgs

		//log.Println("tempDir:", tempDir)
		//log.Println("args:", args)
	}

	var stopWatch = util.NewStopWatch()
	if onSubTaskDone == nil {
		onSubTaskDone = func(int, time.Duration, ...int32) {}
	}
	var logProgress = func(resetWatch bool, task int, args ...int32) {
		onSubTaskDone(task, stopWatch.Duration(resetWatch), args...)
	}

	// ...
	args = args[:0:len(args)]
	for _, arg := range args[:cap(args)] {
		if arg == "builtin" {
			//goto Start
		} else {
			args = append(args, arg)
		}
	}

	//

	var numParsedPackages int32

	var configForParsing = &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps |
			packages.NeedTypes | packages.NeedExportsFile | packages.NeedFiles |
			packages.NeedCompiledGoFiles | packages.NeedTypesSizes |
			packages.NeedSyntax | packages.NeedTypesInfo,
		Tests: false, // ToDo: parse tests
		// It looks, if Tests is set to true, "golds std" panics with error:
		// * panic: TypeName for reflect.EmbedWithUnexpMeth not found
		// * or panic: TypeName for runtime.LFNode not found

		//Logf: func(format string, args ...interface{}) {
		//	log.Println("================================================\n", args)
		//},

		ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			if num := atomic.AddInt32(&numParsedPackages, 1); num&(num-1) == 0 {
				if num == 1 {
					logProgress(true, SubTask_PreparationDone)
				} else {
					logProgress(false, SubTask_NFilesParsed, num)
				}
			}

			//defer log.Println("parsed", filename)
			const mode = parser.AllErrors | parser.ParseComments
			return parser.ParseFile(fset, filename, src, mode)
		},

		// Reasons to disable this:
		// 1. to suppress "imported but not used" errors
		// 2. to implemente "voew code" and "jump to definition" features.
		// It looks the memory comsumed will be doubled.
		//ParseFile: avoidCheckFuncBody,

		//       NeedTypes: NeedTypes adds Types, Fset, and IllTyped.
		//       Why can't only Fset be got?
		// ToDo: modify go/packages code to not use go/types.
		//       use ast only and build type info tailored for docs and code reading.
		//       But it looks NeedTypes doesn't consume much more memory, so ...
		//       And, go/types can be used to verify the correctness of the custom implementation.
	}

	ppkgs, err := packages.Load(configForParsing, args...)
	if err != nil {
		return fmt.Errorf("packages.Load (parse packages): %w", err)
	}

	var hasRuntime bool
	var loadErrs = make([]error, 0, len(ppkgs))
	for _, ppkg := range ppkgs {
		switch ppkg.PkgPath {
		case "runtime":
			hasRuntime = true
		case "builtin":
			// skip "illegal cycle in declaration of int" alike errors.
		//case "unsafe":
		default:
			// ToDo: how to judge "imported but not used" errors to ignore them?

			for _, e := range ppkg.Errors {
				loadErrs = append(loadErrs, e)
			}
		}
	}

	if len(loadErrs) > 0 {
		//return errors.New("code parsing errors")
		return &LoadError{Errs: loadErrs}
	}

	// For "golds main.go" cases.
	if !hasRuntime {
		runtimePPkgs, err := packages.Load(configForParsing, "runtime")
		if err != nil {
			return fmt.Errorf("packages.Load (parse runtime package): %w", err)
		}
		if len(runtimePPkgs) < 1 {
			return errors.New("packages.Load: load runtime page error (unknown).")
		}
		numParsedPackages += int32(len(runtimePPkgs))
		ppkgs = append(ppkgs, runtimePPkgs...)
	}

	//if num := numParsedPackages; num&(num-1) != 0 {
	//	logProgress(true, SubTask_ParsePackagesDone, num)
	//}

	builtinPPkgs, err := packages.Load(configForParsing, "builtin")
	if err != nil {
		return fmt.Errorf("packages.Load (parse builtin package): %w", err)
	}
	if len(builtinPPkgs) != 1 {
		return errors.New("packages.Load: load builtin page error (unknown).")
	}
	numParsedPackages++
	logProgress(true, SubTask_ParsePackagesDone, numParsedPackages)

	//...

	//stdPkgs, err := collectStdPackages()
	stdPkgs, err := getMatchedPackages("std", false)
	if err != nil {
		return fmt.Errorf("failed to collect std packages: %w", err)
	}

	var allPPkgs = collectPPackages(ppkgs)
	var builtinPPkg = builtinPPkgs[0]
	allPPkgs[builtinPPkg.PkgPath] = builtinPPkg

	d.packageList = make([]*Package, 0, len(allPPkgs))
	d.packageTable = make(map[string]*Package, len(allPPkgs))

	//if len(d.packageList) != len(allPPkgs) {
	//	//panic("package counts not match! " + strconv.Itoa(len(d.packageList)) + " != " + strconv.Itoa(len(allPPkgs)))
	//}

	//if len(d.packageTable) != len(allPPkgs) {
	//	//panic("package counts not match! " + strconv.Itoa(len(d.packageTable)) + " != " + strconv.Itoa(len(allPPkgs)))
	//}

	// It looks the AST info of the parsed "unsafe" package is blank.
	// So we fill the info manually to simplify some implementations later.
	if unsafePPkg, builtinPPkg := allPPkgs["unsafe"], allPPkgs["builtin"]; unsafePPkg != nil && builtinPPkg != nil {
		//log.Println("====== 111", unsafePPkg.Fset.Base(), builtinPPkg.Fset.Base(), allPPkgs["bytes"].Fset.Base())
		err := fillUnsafePackage(unsafePPkg, builtinPPkg)
		if err != nil {
			return err
		}
	}

	//var packageListChanged = false
	for path, ppkg := range allPPkgs {
		if path != ppkg.PkgPath {
			panic("path != ppkg.PkgPath")
		}

		pkg := d.packageTable[path]
		if pkg == nil {
			//packageListChanged = true

			pkg := &Package{PPkg: ppkg, Path: path}
			d.packageTable[path] = pkg
			d.packageList = append(d.packageList, pkg)

			//log.Println("     [parsed]", path)
		} else {
			pkg.PPkg = ppkg
			pkg.Path = path

			log.Println("     [parsed]", path, "(duplicated?)")
		}
		//if len(ppkg.Errors) > 0 {
		//	for _, err := range ppkg.Errors {
		//		log.Printf("          error: %#v", err)
		//	}
		//}
	}
	d.builtinPkg = d.packageTable["builtin"]

	d.stats.Packages = int32(len(d.packageList))

	var pkgNumDepedBys = make(map[*Package]uint32, len(allPPkgs))
	for _, pkg := range d.packageList {
		pkg.Deps = make([]*Package, 0, len(pkg.PPkg.Imports))
		//for path := range pkg.PPkg.Imports // the path never starts with "vendor/"
		for _, ppkg := range pkg.PPkg.Imports {
			path := ppkg.PkgPath // may start with "vendor/"
			depPkg := d.packageTable[path]
			if depPkg == nil {
				panic("ParsePackages: dependency package " + path + " not found")
			}
			pkg.Deps = append(pkg.Deps, depPkg)
			pkgNumDepedBys[depPkg]++
		}
	}
	for _, pkg := range d.packageList {
		pkg.DepedBys = make([]*Package, 0, pkgNumDepedBys[pkg])
	}
	for _, pkg := range d.packageList {
		for _, dep := range pkg.Deps {
			dep.DepedBys = append(dep.DepedBys, pkg)
		}
	}

	logProgress(true, SubTask_CollectPackages, int32(len(d.packageList)))

	//log.Println("[parse packages done]")

	// To be filled later
	d.stdModule = &Module{
		Path:    "", //"std",
		Version: "",
		Dir:     "",
		Pkgs:    make([]*Package, 0, len(stdPkgs)+1), // might be a little wasting
	}

	for _, path := range stdPkgs {
		pkg := d.packageTable[string(path)]
		if pkg != nil {
			pkg.module = d.stdModule
			d.stdModule.Pkgs = append(d.stdModule.Pkgs, pkg)
		}
	}
	d.builtinPkg.module = d.stdModule
	d.stdModule.Pkgs = append(d.stdModule.Pkgs, d.builtinPkg)

	// ToDo: this is some slow. Try to parse go.mod files manually?
	err = d.confirmPackageModules(args, hasToolchain, toolchain, completeModuleInfo)
	if err != nil {
		return err
	}

	for _, m := range d.modulesByPath {
		m.buildPackageHierarchy()
	}

	logProgress(true, SubTask_CollectModules, int32(len(d.modulesByPath)))

	// ...

	return nil
}

//var newlineBrace = []byte{'\n', '{'}
//var newline = []byte{'\n'}
//var space = []byte{' '}

func (d *CodeAnalyzer) confirmPackageModules(args []string, hasToolchain bool, toolchain ToolchainInfo, completeModuleInfo func(*Module)) error {
	// go list -deps -json [args]

	// There is a bug https://github.com/golang/go/issues/45649
	// which makes the command return some incorrect modules for some packages.

	// In the output, packages under GOROOT have not .module info.
	cmdAndArgs := append([]string{"go", "list", "-deps", "-json"}, args...)
	output, err := util.RunShell(time.Minute*3, "", nil, cmdAndArgs...)
	if err != nil {
		// log.Printf("%s", output) // debug(ToDo: need a debug verbose flag)
		return fmt.Errorf("unable to list packages and modules info: %s: %w", strings.Join(cmdAndArgs, " "), err)
	}
	// Sometimes, "go list ./..." output "go: warning: "./..." matched no packages" without error code.
	// So the ./... argument might be not filter off by hasMatchedPackages in validateArgumentsAndSetOptions.
	// (ToDo: the above two line comment are invalid now, ...)
	output = bytes.TrimSpace(output)
	if i := bytes.IndexByte(output, '{'); i > 0 {
		output = output[i:]
	}

	type pkg struct {
		ImportPath string
		Dir        string
		Module     Module
		Doc        string // package documentation string
		// Root   string // Go root or Go path dir containing this package

		Goroot   bool // is this package in the Go root? When it is true, Standard is also.
		Standard bool // is this package part of the standard Go library?
	}

	//numToolchainPkgs, modulesNumPkgs := 0, make(map[string]int, 256)
	//count := bytes.Count(output, newlineBrace) + 1
	//pkgs := make([]pkg, count)
	//
	//for i := 0; i < count; i++ {
	//	end := bytes.Index(output, newlineBrace)
	//	if end < 0 {
	//		end = len(output)
	//	}
	//	p := &pkgs[i]
	//	err = json.Unmarshal(output[:end], p)
	//	if err != nil {
	//		log.Printf("Unmarshal package#%d: %s for %s", i, err, output[:end])
	//		return
	//	}
	//	if p.module.Path != "" { // must be not std or toolchain mobule
	//		modulesNumPkgs[p.module.Path]++
	//	} else if strings.HasPrefix(p.Dir, toolchain.Cmd) {
	//		numToolchainPkgs++
	//	}
	//	//log.Println("===", p.ImportPath, p.module.Path)
	//	if end == len(output) {
	//		break
	//	}
	//	output = output[end+1:]
	//}

	numToolchainPkgs, modulesNumPkgs := 0, make(map[string]int, 256)
	count := bytes.Count(output, []byte("ImportPath"))
	if count == 0 {
		return fmt.Errorf("go list nothing: %s", output)
	}

	pkgs := make([]pkg, count)

	var i = 0
	for dec := json.NewDecoder(bytes.NewBuffer(output)); dec.More(); {
		p := &pkgs[i]
		err = dec.Decode(p)
		if err != nil {
			return fmt.Errorf("decode package#%d json error: %w", i, err)
		}
		if p.Module.Path != "" { // must be not std or toolchain mobule
			modulesNumPkgs[p.Module.Path]++
		} else if strings.HasPrefix(p.Dir, toolchain.Cmd) {
			numToolchainPkgs++
		}
		i++
	}
	if i != count {
		return fmt.Errorf("decoded package json count (%d) != result of bytes.Count (%d)", i, count)
	}

	if numToolchainPkgs == 0 {
		if hasToolchain {
			log.Println("!!! hasToolchain==true but numToolchainPkgs==0, weird")
			hasToolchain = false
		}
	} else {
		if !hasToolchain {
			//log.Println("!!! hasToolchain==false but numToolchainPkgs>0, weird")
			// Not weird. run "golds ./..." in GOROOT/src/cmd directory.
			hasToolchain = true
		}
		d.wdModule = &Module{
			Path:    "cmd", // "toolchain"
			Version: "",
			Dir:     toolchain.Cmd,
			Pkgs:    make([]*Package, 0, numToolchainPkgs),
		}
	}

	d.nonToolchainModules = make([]Module, 0, len(modulesNumPkgs))
	numAllModules := len(modulesNumPkgs) + 1 // including the std module
	if hasToolchain {
		numAllModules++ // the cmd toolchain module
	}
	d.modulesByPath = make(map[string]*Module, numAllModules)
	d.stdModule.Index = len(d.modulesByPath)
	d.modulesByPath["std"] = d.stdModule
	if hasToolchain {
		d.wdModule.Index = len(d.modulesByPath)
		d.modulesByPath[d.wdModule.Path] = d.wdModule
	}
	numPackagesWithoutModule := 0
	for i := range pkgs {
		p := &pkgs[i]
		pkg := d.packageTable[p.ImportPath]
		if pkg == nil {
			fmt.Printf("!!! package %s is not found, weird\n", p.ImportPath)
			continue
		}

		pkg.OneLineDoc = p.Doc
		pkg.Directory = p.Dir

		if p.Module.Path != "" {
			if hasToolchain {
				// log.Printf("!!! hasToolchain==true but package %s is not in toolchain directory, weird", p.ImportPath)
				// Not weird. Toolchain depends on some golang.org/x/... packages.
			}

			var mPath = p.Module.Path
			if mPath == "std" { // || mPath == "cmd" {
				mPath += " (fake)"
				// This case happens when pasred a non-current std package.
			}

			m := d.modulesByPath[mPath]
			if m == nil {
				d.nonToolchainModules = append(d.nonToolchainModules, p.Module)
				m = &d.nonToolchainModules[len(d.nonToolchainModules)-1]
				m.Index = len(d.modulesByPath)
				d.modulesByPath[mPath] = m
				m.Pkgs = make([]*Package, 0, modulesNumPkgs[m.Path])
			}
			pkg.module = m // ToDo: use substrings of Dir and Path of pkg to save some memory.
			m.Pkgs = append(m.Pkgs, pkg)
		} else if strings.HasPrefix(p.Dir, toolchain.Cmd) {
			if pkg.module != nil {
				if !d.IsStandardPackage(pkg) {
					log.Printf("!!! the module of toolchain package %s is already found, weird", p.ImportPath)
				}
			} else {
				pkg.module = d.wdModule
				d.wdModule.Pkgs = append(d.wdModule.Pkgs, pkg)
			}
		} else if p.Standard {
			if pkg.module != d.stdModule {
				log.Printf("!!! the module of standard package %s is still not confirmed, weird", p.ImportPath)
			}
		} else {
			numPackagesWithoutModule++
			//log.Printf("!!! package %s is not a toolchain package, some weird: %v", p.ImportPath, d.wdModule)
			// The reason why entring this branch might be the modules feature is off,
		}
	}

	if len(d.nonToolchainModules) != len(modulesNumPkgs) {
		panic(fmt.Sprintf("non-std moduels count wrong (%d : %d)", len(d.nonToolchainModules), len(modulesNumPkgs)))
	}
	if numPackagesWithoutModule > 0 && len(d.nonToolchainModules) > 0 {
		log.Println("!!! in modules mode, but some packages are not in any module, weird")
	}

	// confirm std and cmd module version, ...
	d.stdModule.Version = toolchain.Version
	d.stdModule.Dir = filepath.Dir(toolchain.Src) // filepath.Join(build.Default.GOROOT, "src")
	d.stdModule.RepositoryCommit = toolchain.Version
	d.stdModule.RepositoryDir = build.Default.GOROOT
	d.stdModule.RepositoryURL = "https://github.com/golang/go"
	d.stdModule.ExtraPathInRepository = "/src/"
	if hasToolchain {
		d.wdModule.Version = toolchain.Version
		d.wdModule.Dir = toolchain.Cmd
		d.wdModule.RepositoryCommit = toolchain.Version
		d.wdModule.RepositoryDir = build.Default.GOROOT
		d.wdModule.RepositoryURL = "https://github.com/golang/go"
		d.wdModule.ExtraPathInRepository = "/src/cmd"

		//if n := len(d.nonToolchainModules); n != 0 {
		//	log.Printf("!!! toolchain==true, but %d other modules are found, weird", n)
		//}
		// Not weird. Toolchain depends on some golang.org/x/... packages.
	}

	for i := range d.nonToolchainModules {
		m := &d.nonToolchainModules[i]
		if m.ActualVersion() == "" && m.Replace.Path == "" {
			d.wdModule = m
		}
	}
	// Confirm wdModule firstly so that the vendor directory could be determined,
	if completeModuleInfo != nil {
		var wg sync.WaitGroup
		for i := range d.nonToolchainModules {
			m := &d.nonToolchainModules[i]
			wg.Add(1)
			go func() {
				defer wg.Done()
				completeModuleInfo(m)
			}()
		}
		wg.Wait()
	}
	for i := range d.nonToolchainModules {
		m := &d.nonToolchainModules[i]
		if m.ActualVersion() == "" {
			//panic("should not")
			continue // don't confirm repo for modules which versions are blank.
		}
		confirmModuleReposotoryCommit(m)
	}
	if completeModuleInfo != nil && d.wdModule != nil {
		for i := range d.nonToolchainModules {
			m := &d.nonToolchainModules[i]
			if strings.HasPrefix(m.Replace.Path, ".") {
				moduleDir := m.ActualDir()
				if !strings.HasPrefix(moduleDir, d.wdModule.Dir) {
					panic("should not\n" + moduleDir + "\n" + d.wdModule.Dir + "\n" + m.Replace.Dir)
				}
				path := moduleDir[len(d.wdModule.Dir):]
				m.ExtraPathInRepository = d.wdModule.ExtraPathInRepository + path
				m.RepositoryCommit = d.wdModule.RepositoryCommit
				m.RepositoryDir = d.wdModule.RepositoryDir
				m.RepositoryURL = d.wdModule.RepositoryURL
			}
		}
	}

	for i := range d.nonToolchainModules {
		m := &d.nonToolchainModules[i]
		if m != d.wdModule && m.ActualVersion() == "" && strings.HasPrefix(m.Replace.Dir, ".") {
			log.Printf("!!! the version of module %s is not confirmed, weird", m.Path)
		}
	}
	for _, pkg := range d.packageList {
		if pkg.module == nil || pkg.module == d.stdModule {
			//log.Printf("!!! the module of package %s is not confirmed, weird (or not)", pkg.Path)
			continue
		}
		if !strings.HasPrefix(pkg.Path, pkg.module.Path) {
			if pkg.Path != "command-line-arguments" { // sourced from golang.org/x/tools/go/packages
				log.Println("!!! wrong prefix:", pkg.Path, pkg.module.Path)
				pkg.wrongModule = true
				pkg.module.hasWrongPkgs = true
			}
		}
	}

	return nil
}

// v0.0.0-20180917221912-90fa682c2a6e
// v0.4.2-0.20210302225053-d515b24adc21
var findCommentRegexp = regexp.MustCompile(`v[0-9]\S*[0-9]{8,}-([0-9a-f]{6,})`)

const incompatibleSuffix = "+incompatible"

func confirmModuleReposotoryCommit(m *Module) {
	version := m.ActualVersion()
	if i := strings.Index(version, incompatibleSuffix); i > 0 {
		version = version[:i]
	}
	matches := findCommentRegexp.FindStringSubmatch(version)
	if len(matches) >= 2 {
		m.RepositoryCommit = matches[1]
		return
	}

	// ToDo: valid for all code hosting websites?
	if extra := m.ExtraPathInRepository; extra != "" {
		if strings.HasPrefix(extra, "/") {
			extra = extra[1:]
		}
		if strings.HasSuffix(extra, "/") {
			m.RepositoryCommit = extra + version
		} else {
			m.RepositoryCommit = extra + "/" + version
		}
		return
	}

	m.RepositoryCommit = version
}

func fillUnsafePackage(unsafePPkg *packages.Package, builtinPPkg *packages.Package) error {
	intType := builtinPPkg.Types.Scope().Lookup("int").Type()

	//log.Println("====== 000", unsafePPkg.PkgPath)
	//log.Println("====== 222", unsafePPkg.Fset.PositionFor(token.Pos(0), false))

	buildPkg, err := build.Import("unsafe", "", build.FindOnly)
	if err != nil {
		//log.Fatal(fmt.Errorf("build.Import: %w", err))
		return fmt.Errorf("build.Import: %w", err)
	}

	filter := func(fi os.FileInfo) bool {
		return strings.HasSuffix(fi.Name(), ".go") && !strings.HasSuffix(fi.Name(), "_test.go")
	}

	//log.Println("====== 333", buildPkg.Dir)
	fset := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fset, buildPkg.Dir, filter, parser.ParseComments)
	if err != nil {
		//log.Fatal(fmt.Errorf("parser.ParseDir: %w", err))
		return fmt.Errorf("parser.ParseDir: %w", err)
	}

	astPkg := astPkgs["unsafe"]
	if astPkg == nil {
		//log.Fatal("ast package for unsafe is not found")
		return errors.New("ast package for unsafe is not found")
	}

	//fset := token.NewFileSet()
	//f, err := parser.ParseFile(fset, "unsafe.go", unsafe_go, parser.ParseComments)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}

	// It is strange that unsafePPkg.Fset is not blank
	// (it looks all parsed packages (by go/Packages.Load) share the same FileSet)
	// even if unsafePPkg.GoFiles and unsafePPkg.Syntax (and more) are both blank.
	// This is why the current function tries to fill them.
	unsafePPkg.TypesInfo.Defs = make(map[*ast.Ident]types.Object)
	unsafePPkg.TypesInfo.Types = make(map[ast.Expr]types.TypeAndValue)
	unsafePPkg.Fset = fset

	var artitraryExpr, intExpr1, intExpr2 ast.Expr
	var artitraryType types.Type

	var artitraryName, integerName *ast.Ident

	//for filename, astFile := range map[string]*ast.File{"unsafe.go": f} {
	for filename, astFile := range astPkg.Files {
		unsafePPkg.GoFiles = append(unsafePPkg.GoFiles, filename)
		unsafePPkg.CompiledGoFiles = append(unsafePPkg.CompiledGoFiles, filename)
		//log.Println("unsafe filename:", filename)
		unsafePPkg.Syntax = append(unsafePPkg.Syntax, astFile)
		//unsafePPkg.Fset.AddFile(filename, unsafePPkg.Fset.Base(), int(astFile.End()-astFile.Pos()))
		//log.Println("====== 444", filename, unsafePPkg.Fset.Base(), int(astFile.End()-astFile.Pos()))

		for _, decl := range astFile.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok {
				//if fd.Name.IsExported() {
				//	log.Printf("     func declaration: %s", fd.Name.Name)
				//}

				obj := types.Unsafe.Scope().Lookup(fd.Name.Name)
				if obj == nil {
					if fd.Name.Name == "Add" || fd.Name.Name == "Slice" {
						return fmt.Errorf(`unsafe.%s is introduced in Go 1.17.
Please re-install Golds with Go toolchain v1.17+ with:
	go install go101.org/golds@latest`,
							fd.Name.Name)
					}
					panic(fd.Name.Name + " is not found in unsafe scope")
				}

				unsafePPkg.TypesInfo.Defs[fd.Name] = obj
				continue
			}

			gn, ok := decl.(*ast.GenDecl)
			if !ok || gn.Tok != token.TYPE {
				continue
			}

			for _, spec := range gn.Specs {
				typeSpec := spec.(*ast.TypeSpec)
				//if typeSpec.Name.IsExported() {
				//	log.Printf("     type declaration: %s", typeSpec.Name.Name)
				//}

				obj := types.Unsafe.Scope().Lookup(typeSpec.Name.Name)
				typeObj, _ := obj.(*types.TypeName)
				switch typeSpec.Name.Name {
				default:
					panic("Unexpected type name in unsafe: " + typeSpec.Name.Name)
				case "Pointer":
					if typeObj == nil {
						panic("a non-nil type object for unsafe.Pointer is expected")
					}

					// ToDo: is this a bug?
					//       should get the base type expr instead?
					//       type Pointer *ArbitraryType
					//artitraryExpr = typeSpec.Type

					//starExpr, ok := typeSpec.Type.(*ast.StarExpr)
					//if !ok {
					//	panic("not a *ast.StarExpr")
					//}
					//artitraryExpr = starExpr.X

					// Not a bug, the old way is right. (Really?)
					artitraryExpr = typeSpec.Type
				case "ArbitraryType":
					intExpr1 = typeSpec.Type

					// ToDo: need invesgate: in testing running, typeObj != nil
					//       but in normal running, it is nil!!!
					//log.Println("ArbitraryType obj:", typeObj)
					//if typeObj != nil {
					//	panic("a nil type object for ArbitraryType is expected")
					//}
					//log.Println("    ", typeSpec.Name.Name, "is not found in unsafe scope. Create one manually.")

					//log.Printf("%T %T\n", intType, intType.Underlying())

					// ToDo:
					// The last argument is nil is because the creations of
					// the following two objects depend on each other.
					// ;(, Maybe I have not found the right solution.
					typeObj = types.NewTypeName(typeSpec.Pos(), types.Unsafe, typeSpec.Name.Name, nil)
					unsafePPkg.Types.Scope().Insert(typeObj)
					artitraryType = types.NewNamed(typeObj, intType.Underlying(), nil)

					artitraryName = typeSpec.Name
				case "IntegerType": // introduced in Go 1.17
					intExpr2 = typeSpec.Type

					// ToDo: see the ToDo above for ArbitraryType
					typeObj = types.NewTypeName(typeSpec.Pos(), types.Unsafe, typeSpec.Name.Name, nil)
					unsafePPkg.Types.Scope().Insert(typeObj)

					integerName = typeSpec.Name
				}

				// new declared type (source type will be set below)
				unsafePPkg.TypesInfo.Defs[typeSpec.Name] = typeObj
			}
		}

		// ToDo: how init and _ functions are parsed.
	}

	if artitraryExpr == nil {
		panic("artitraryExpr is nil")
	}

	if intExpr1 == nil {
		panic("intExpr is nil")
	}

	if artitraryType == nil {
		panic("artitraryType is nil")
	}

	// source types
	unsafePPkg.TypesInfo.Types[intExpr1] = types.TypeAndValue{Type: intType}
	if intExpr2 != nil {
		unsafePPkg.TypesInfo.Types[intExpr2] = types.TypeAndValue{Type: intType}
	}
	unsafePPkg.TypesInfo.Types[artitraryExpr] = types.TypeAndValue{Type: artitraryType}

	//tn1 := unsafePPkg.TypesInfo.Defs[artitraryName].(*types.TypeName)
	//tn2 := unsafePPkg.TypesInfo.Defs[integerName].(*types.TypeName)
	//fmt.Printf("111111: %[1]T, %[1]v ===  %[2]T, %[2]v \n", unsafePPkg.TypesInfo.Defs[artitraryName], tn1.Type())
	//fmt.Printf("222222: %[1]T, %[1]v ===  %[2]T, %[2]v \n", unsafePPkg.TypesInfo.Defs[integerName], tn2.Type())
	_ = artitraryName

	// ToDo: golang.org/x/tools might change implementation so that this handling is not essential now.
	if tn, _ := unsafePPkg.TypesInfo.Defs[integerName].(*types.TypeName); tn != nil {
		if tn.Type() == nil {
			// :(, The type of types.TypeName is not modifiable now.
			// tn.SetType(types.NewNamed(...))

			// ToDo: now, in analyzing, when anObj.Type() == nil, treat as types.Invalid
		}
	}

	return nil
}
