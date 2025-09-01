// *********************************************/
//                     _ooOoo_
//                    o8888888o
//                    88" . "88
//                    (| -_- |)
//                    O\  =  /O
//                 ____/`---'\____
//               .'  \\|     |//  `.
//              /  \\|||  :  |||//  \
//             /  _||||| -:- |||||-  \
//             |   | \\\  -  /// |   |
//             | \_|  ''\---/''  |   |
//             \  .-\__  `-`  ___/-. /
//           ___`. .'  /--.--\  `. . __
//        ."" '<  `.___\_<|>_/___.'  >'"".
//       | | :  `- \`.;`\ _ /`;.`/ - ` : | |
//       \  \ `-.   \_ __\ /__ _/   .-` /  /
//  ======`-.____`-.___\_____/___.-`____.-'======
//                     `=---='
// ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
//             佛祖保佑       永无BUG
//             心外无法       法外无心
//             三宝弟子       飞猪宏愿
// *********************************************/

package action

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	rice "github.com/GeertJohan/go.rice"
	"github.com/pkg/errors"
	"snz1.cn/snz1dp/snz1dpctl/down"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

func resolveNodejsPath(s Action) (mkfpath string, err error) {
	setting := s.GlobalSetting()
	var (
		gzfpath string
	)

	nodejsDir := getNodejsDir(setting)
	mkfpath, err = nodejsFileExisted(setting)
	if err != nil {
		if gzfpath, err = downloadNodejs(s, setting.OutOrStdout(), false); err != nil {
			return
		}
		os.RemoveAll(nodejsDir)
		if err = unarchiveNodejs(gzfpath, setting.GetBinDir()); err != nil {
			return
		}
	}
	if runtime.GOOS == "windows" {
		err = utils.FixBinSearchPath(nodejsDir)
	} else {
		err = utils.FixBinSearchPath(nodejsDir + "/bin")
	}
	if err != nil {
		return
	}
	return
}

func resolveNrmPath(s Action) (mkfpath string, err error) {
	setting := s.GlobalSetting()
	var (
		gzfpath string
	)

	mkfpath, err = nrmFileExisted(setting)
	if err != nil {
		err = nil
		startNvmCmd := NewStartNvm(setting)
		nvmNodeVersion, _ := startNvmCmd.GetNodeVersion()

		if nvmNodeVersion == BaseConfig.Node.Version {
			if gzfpath, err = downloadNodejs(s, setting.OutOrStdout(), false); err != nil {
				return
			}
			os.RemoveAll(getNodejsDir(setting))
			if err = unarchiveNodejs(gzfpath, setting.GetBinDir()); err != nil {
				return
			}
		}
		npmPaths, err := resolveNpmPaths(s)
		if err != nil {
			s.ErrorExit("%s", err)
		}
		npmCmd := npmPaths[0]
		npmArgs := npmPaths[1:]
		npmArgs = append(npmArgs,
			"install",
			"-g",
			"nrm",
		)
		var stdOutBuffer bytes.Buffer
		npmInstallNrm := exec.CommandContext(
			context.Background(),
			npmCmd, npmArgs...,
		)
		npmInstallNrm.Stdout = &stdOutBuffer
		if err = npmInstallNrm.Run(); err != nil {
			s.ErrorExit("node-v%s install nrm error: %s", nvmNodeVersion, stdOutBuffer.String())
		}
	}
	return
}

func resolveNrmPaths(s Action) (mkfpaths []string, err error) {
	isWin := false
	switch runtime.GOOS {
	case "windows":
		isWin = true
	}
	var (
		nodejspath, mkfpath string
	)
	if !isWin {
		if nodejspath, err = resolveNodejsPath(s); err != nil {
			return
		}
		mkfpaths = append(mkfpaths, nodejspath)
	}
	if mkfpath, err = resolveNrmPath(s); err != nil {
		return
	}
	mkfpaths = append(mkfpaths, mkfpath)
	return
}

func resolveNpmPath(s Action) (mkfpath string, err error) {
	setting := s.GlobalSetting()
	var (
		gzfpath string
	)

	mkfpath, err = npmFileExisted(setting)
	if err != nil {
		startNvmCmd := NewStartNvm(setting)
		nvmNodeVersion, _ := startNvmCmd.GetNodeVersion()

		if nvmNodeVersion == BaseConfig.Node.Version {
			if gzfpath, err = downloadNodejs(s, setting.OutOrStdout(), false); err != nil {
				return
			}
			os.RemoveAll(getNodejsDir(setting))
			if err = unarchiveNodejs(gzfpath, setting.GetBinDir()); err != nil {
				return
			}
		} else {
			err = errors.Errorf("node found npm in node-v%s", nvmNodeVersion)
		}
	}
	return
}

func resolveNpmPaths(s Action) (mkfpaths []string, err error) {
	isWin := false
	switch runtime.GOOS {
	case "windows":
		isWin = true
	}
	var (
		nodejspath, mkfpath string
	)
	if nodejspath, err = resolveNodejsPath(s); err != nil {
		return
	}
	if !isWin {
		mkfpaths = append(mkfpaths, nodejspath)
	}
	if mkfpath, err = resolveNpmPath(s); err != nil {
		return
	}
	mkfpaths = append(mkfpaths, mkfpath)
	return
}

func getNodejsDir(setting *GlobalSetting) string {
	nodejsPath := getNodejsFile(setting)
	switch runtime.GOOS {
	case "windows":
		return path.Dir(nodejsPath)
	default:
		return path.Dir(path.Dir(nodejsPath))
	}
}

var _nodejsFile string

func getNodejsFile(setting *GlobalSetting) string {
	if _nodejsFile != "" {
		return _nodejsFile
	}
	bindir := setting.GetBinDir()

	startNvmCmd := NewStartNvm(setting)
	nvmNodeVersion, _ := startNvmCmd.GetNodeVersion()

	if nvmNodeVersion == BaseConfig.Node.Version {
		fcname := fmt.Sprintf("%s-v%s-%s-%s/bin/%s", BaseConfig.Node.Name, BaseConfig.Node.Version, runtime.GOOS, runtime.GOARCH, BaseConfig.Node.Name)
		switch runtime.GOOS {
		case "windows":
			fcname = fmt.Sprintf("%s-v%s-%s-%s/%s", BaseConfig.Node.Name, BaseConfig.Node.Version, runtime.GOOS, runtime.GOARCH, BaseConfig.Node.Name)
			fcname = fcname + ".exe"
		}
		_nodejsFile = path.Join(bindir, fcname)
		return _nodejsFile
	} else {
		switch runtime.GOOS {
		case "windows":
			return path.Join(bindir, "nodejs", "node.exe")
		default:
			fcname, _ := utils.ExpandUserDir(fmt.Sprintf("~/.nvm/versions/node/v%s/bin/node", nvmNodeVersion))
			_nodejsFile = fcname
			return _nodejsFile
		}
	}
}

func getNrmFile(setting *GlobalSetting) string {
	nodejsDir := getNodejsDir(setting)
	fcname := fmt.Sprintf("%s/bin/nrm", nodejsDir)
	switch runtime.GOOS {
	case "windows":
		fcname = fmt.Sprintf("%s/nrm.cmd", nodejsDir)
	}
	return fcname
}

func getNpmFile(setting *GlobalSetting) string {
	nodejsDir := getNodejsDir(setting)
	fcname := fmt.Sprintf("%s/bin/npm", nodejsDir)
	switch runtime.GOOS {
	case "windows":
		fcname = fmt.Sprintf("%s/npm.cmd", nodejsDir)
	}
	return fcname
}

func nodejsFileExisted(setting *GlobalSetting) (kf string, err error) {
	kf = getNodejsFile(setting)
	var lst os.FileInfo
	if lst, err = os.Stat(kf); err == nil && !lst.IsDir() {
		return
	}
	if lst != nil && lst.IsDir() {
		os.RemoveAll(kf)
	}
	err = errors.Errorf("not existed")
	return
}

func nrmFileExisted(setting *GlobalSetting) (kf string, err error) {
	kf = getNrmFile(setting)
	var lst os.FileInfo
	if lst, err = os.Stat(kf); err == nil && !lst.IsDir() {
		return
	}
	if lst != nil && lst.IsDir() {
		os.RemoveAll(kf)
	}
	err = errors.Errorf("not existed")
	return
}

func npmFileExisted(setting *GlobalSetting) (kf string, err error) {
	kf = getNpmFile(setting)
	var lst os.FileInfo
	if lst, err = os.Stat(kf); err == nil && !lst.IsDir() {
		return
	}
	if lst != nil && lst.IsDir() {
		os.RemoveAll(kf)
	}
	err = errors.Errorf("not existed")
	return
}

func unarchiveNodejs(gzfpath string, binbasedir string) error {
	var (
		err error
	)
	if err = UnarchiveBundle(gzfpath, binbasedir); err != nil {
		return err
	}
	return nil
}

// StartNode -
type StartNodejs struct {
	BaseAction
	Args []string
}

// StartNodejs -
func NewStartNodejs(setting *GlobalSetting) *StartNodejs {
	return &StartNodejs{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *StartNodejs) Run() error {
	setting := s.GlobalSetting()
	var (
		mkfpath string
		err     error
	)

	setting.InitLogger(BaseConfig.Node.Name)

	mkfpath, err = resolveNodejsPath(s)
	if err != nil {
		s.ErrorExit("%s", err.Error())
	}

	nodejsArgs := s.Args
	nodejs := exec.CommandContext(context.Background(), mkfpath, nodejsArgs...)
	nodejs.Stdin = setting.InOrStdin()
	nodejs.Stdout = setting.OutOrStdout()
	nodejs.Stderr = setting.ErrOrStderr()
	if err = nodejs.Run(); err != nil {
		s.ErrorExit("%v", err)
	}
	return nil
}

// StartNode -
type StartNrm struct {
	BaseAction
	Args []string
}

// NewStartNrm -
func NewStartNrm(setting *GlobalSetting) *StartNrm {
	return &StartNrm{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *StartNrm) Run() error {
	setting := s.GlobalSetting()
	var (
		mkfpath []string
		err     error
	)

	setting.InitLogger(BaseConfig.Node.Name)

	mkfpath, err = resolveNrmPaths(s)
	if err != nil {
		s.ErrorExit("%s", err.Error())
	}

	nrmCmd := mkfpath[0]
	nrmArgs := mkfpath[1:]
	nrmArgs = append(nrmArgs, s.Args...)
	nodejs := exec.CommandContext(context.Background(), nrmCmd, nrmArgs...)
	nodejs.Stdin = setting.InOrStdin()
	nodejs.Stdout = setting.OutOrStdout()
	nodejs.Stderr = setting.ErrOrStderr()
	if err = nodejs.Run(); err != nil {
		s.ErrorExit("%v", err)
	}
	return nil
}

// StartNode -
type StartNpm struct {
	BaseAction
	Args []string
}

// NewStartNpm -
func NewStartNpm(setting *GlobalSetting) *StartNpm {
	return &StartNpm{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *StartNpm) Run() error {
	setting := s.GlobalSetting()
	var (
		mkfpath []string
		err     error
	)

	setting.InitLogger(BaseConfig.Node.Name)

	mkfpath, err = resolveNpmPaths(s)
	if err != nil {
		s.ErrorExit("%s", err.Error())
	}

	npmCmd := mkfpath[0]
	npmArgs := mkfpath[1:]
	npmArgs = append(npmArgs, s.Args...)
	nodejs := exec.CommandContext(context.Background(), npmCmd, npmArgs...)
	nodejs.Stdin = setting.InOrStdin()
	nodejs.Stdout = setting.OutOrStdout()
	nodejs.Stderr = setting.ErrOrStderr()
	if err = nodejs.Run(); err != nil {
		s.ErrorExit("%v", err)
	}
	return nil
}

// CreateVueProject -
type CreateVueProject struct {
	BaseAction
	Name    string
	Path    string
	Api     string
	Version string
	Force   bool
}

// NewCreateVueProject -
func NewCreateVueProject(setting *GlobalSetting) *CreateVueProject {
	return &CreateVueProject{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run
func (c *CreateVueProject) Run() (err error) {
	var (
		setting        *GlobalSetting = c.GlobalSetting()
		bundleDir      string         = setting.GetBundleDir()
		spinner        *utils.WaitSpinner
		vueProjectURL  *url.URL
		backendApiURL  *url.URL
		zipFileName    string
		zipFilepath    string
		force, dodown  bool = c.Force, false
		fst            os.FileInfo
		failedRemove   bool = true
		vueProjectInfo VueProjectInfo
		ic             *InstallConfiguration
		vueprojectURL  string = BaseConfig.Vueproject.URL
	)

	// 下载组件
	if _, err := os.Stat(bundleDir); err != nil {
		os.MkdirAll(bundleDir, os.ModePerm)
	}

	if c.Name == "" {
		c.ErrorExit("vue project name is empty!")
		return
	}

	if c.Path == "/" {
		c.ErrorExit("web path error!")
		return
	}

	vueProjectInfo.Name = c.Name
	if backendApiURL, err = url.Parse(c.Api); err != nil {
		c.ErrorExit("backend api url %s error: %s", c.Api, err)
	}

	if backendApiURL.Path == "" || backendApiURL.Path == "/" {
		c.ErrorExit("backend api url %s error: %s", c.Api, "web path error")
	}

	if _, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		c.ErrorExit("%s", err)
		return
	}

	apiSchema := "http:"
	if backendApiURL.Scheme != "" {
		apiSchema = backendApiURL.Scheme
	}

	apiHost := "localhost"
	if backendApiURL.Host != "" {
		apiHost = backendApiURL.Host
	}

	vueProjectInfo.ApiPath = backendApiURL.Path
	vueProjectInfo.ApiURL = fmt.Sprintf("%s://%s%s", apiSchema, apiHost, backendApiURL.Path)
	vueProjectInfo.Version = c.Version
	vueProjectInfo.ApiBase = fmt.Sprintf("%s://%s", apiSchema, apiHost)

	if c.Path == "" {
		vueProjectInfo.BasePath = "/" + c.Name
	} else {
		vueProjectInfo.BasePath = c.Path
		if vueProjectInfo.BasePath[len(vueProjectInfo.BasePath)-1] == '/' {
			vueProjectInfo.BasePath = vueProjectInfo.BasePath[0 : len(vueProjectInfo.BasePath)-2]
		}
	}

	if *ic.Snz1dp.NpmRepo.Private {
		vueProjectInfo.DockerHub = ic.Snz1dp.Registry.URL
		if !strings.HasSuffix(vueProjectInfo.DockerHub, "/") {
			vueProjectInfo.DockerHub += "/"
		}
	} else {
		vueProjectInfo.DockerHub = "snz1dp/"
	}

	vueProjectInfo.ServerURL = ic.Snz1dp.Server.URL
	if vueProjectInfo.ServerURL == BaseConfig.Snz1dp.Server.URL {
		vueProjectInfo.ServerURL = ""
	}

	vueProjectInfo.DownloadURL = ic.Snz1dp.Server.DownloadPrefix

	vueprojectURL = setting.ResolveDownloadURL(BaseConfig.Vueproject.URL)
	if vueProjectURL, err = url.Parse(vueprojectURL); err != nil {
		c.ErrorExit("vue project url %s error: %s", vueprojectURL, err)
		return
	}

	zipFileName = filepath.Base(vueProjectURL.Path)
	zipFilepath = path.Join(bundleDir, zipFileName)

	if force {
		dodown = true
	} else {
		if fst, err = os.Stat(zipFilepath); err == nil && !fst.IsDir() {
			sumData, err := utils.FileChecksum(zipFilepath, sha256.New())
			fcksum := hex.EncodeToString(sumData)
			dodown = err != nil || down.VerifyBundle(zipFilepath+".sha256", fcksum) != nil
		} else {
			if fst != nil {
				os.RemoveAll(zipFilepath)
			}
			dodown = true
		}
	}

	if dodown {
		spinner = utils.NewSpinner(fmt.Sprintf("download %s...", vueProjectURL.String()), setting.OutOrStdout())
		zipFilepath, err = down.NewBundleDownloader(setting.OutOrStdout(), vueProjectURL.String(), down.VerifyAlways).Download(
			setting.GetBundleDir(), zipFileName)

		spinner.Close()
		if err != nil {
			c.ErrorExit("failed: %s", err.Error())
			return
		}
		c.Println("ok!")
	}

	var (
		curdir string
		prjdir string
	)

	curdir, err = os.Getwd()
	if err != nil {
		c.ErrorExit("%s", err)
		return
	}

	// 项目目录
	prjdir = path.Join(curdir, c.Name)
	if _, err = os.Stat(prjdir); err == nil {
		if !utils.Confirm("project directory "+prjdir+" existed, Proceed? (y/N)", setting.InOrStdin(), setting.OutOrStdout()) {
			c.Println("Cancelled.")
			return
		}
		failedRemove = false
	}

	// 创建目录
	if err = os.MkdirAll(prjdir, os.ModePerm); err != nil {
		c.ErrorExit("create directory %s error: %s", prjdir, err)
		return
	}

	// 切换到目录
	if err = os.Chdir(prjdir); err != nil {
		c.ErrorExit("cd %s error: %s", prjdir, err)
		return
	}

	// 解压
	if err = UnarchiveBundle(zipFilepath, prjdir); err != nil {
		os.Chdir(curdir)
		if failedRemove {
			os.RemoveAll(prjdir)
		}
		c.ErrorExit("unzip %s error: %s", zipFilepath, err)
		return
	}

	var (
		tpl    *template.Template
		bDatas []byte
	)

	confBox, err := rice.FindBox("../asset/vue")
	if err != nil {
		panic(err)
	}

	for _, templateFile := range []string{
		"BUILD.yaml", "config/index.js", "config/prod.env.js", "package.json", "VERSION", "Makefile", "Dockerfile",
		"Jenkinsfile", ".workflow/build.sh", ".workflow/publish.sh", ".workflow/deploy.sh",
	} {
		bDatas, _ = confBox.Bytes(templateFile)
		if tpl, err = template.New("render").Parse(string(bDatas)); err != nil {
			os.Chdir(curdir)
			if failedRemove {
				os.RemoveAll(prjdir)
			}
			c.ErrorExit("load %s template error: %s", templateFile, err)
			return
		}

		var tbuf bytes.Buffer
		if err = tpl.Execute(&tbuf, &vueProjectInfo); err != nil {
			os.Chdir(curdir)
			if failedRemove {
				os.RemoveAll(prjdir)
			}
			c.ErrorExit("render %s template error: %s", templateFile, err)
			return
		}

		os.WriteFile(path.Join(prjdir, templateFile), tbuf.Bytes(), os.ModePerm)
	}

	os.Remove(path.Join(prjdir, "package-lock.json"))

	c.Println("success create %s.", c.Name)

	return
}

type VueProjectInfo struct {
	Name        string
	Version     string
	BasePath    string
	ApiPath     string
	ApiBase     string
	ApiURL      string
	DockerHub   string
	ServerURL   string
	DownloadURL string
}
