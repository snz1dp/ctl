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
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

// ProjectPublish -
type ProjectPublish struct {
	BaseAction
	BuildArgs       []string
	Labels          []string
	SkipBuildDocker bool
	SkipBuildSource bool
	SkipPackage     bool
	HelmRepo        string
	HelmUserName    string
	HelmUserPwd     string
}

// NewProjectPublish -
func NewProjectPublish(setting *GlobalSetting) *ProjectPublish {
	return &ProjectPublish{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (d *ProjectPublish) Run() (err error) {
	var (
		curdir string
		pbc    *ProjectBuildConfig
	)
	curdir, err = os.Getwd()
	if err != nil {
		d.ErrorExit("%s", err)
	}

	pbc, err = LoadProjectBuildFromDirectory(curdir)
	if err != nil {
		d.ErrorExit("%s", err)
	}
	pbc.action = d
	pbc.SkipBuildDocker = d.SkipBuildDocker
	pbc.SkipBuildSource = d.SkipBuildSource
	pbc.SkipPackage = d.SkipPackage

	if pbc.Docker == nil {
		pbc.Docker = &DockerBuildConfig{}
	}

	pbc.Docker.Args = make(map[string]*string)
	pbc.Docker.Labels = make(map[string]string)

	if pbc.HelmPushConfig == nil {
		pbc.HelmPushConfig = &HelmPushConfig{}
	}

	if d.HelmRepo != "" {
		pbc.HelmPushConfig.RepoName = d.HelmRepo
	}

	if d.HelmUserName != "" {
		pbc.HelmPushConfig.RepoUser = d.HelmUserName
	}

	if d.HelmUserPwd != "" {
		pbc.HelmPushConfig.RepoPwd = d.HelmUserPwd
	}

	for _, v := range d.BuildArgs {
		eqidx := strings.Index(v, "=")
		var val = ""
		if eqidx > 0 {
			val = v[eqidx+1:]
			pbc.Docker.Args[v[0:eqidx]] = &val
		} else {
			pbc.Docker.Args[v] = &val
		}
	}

	for _, v := range d.Labels {
		eqidx := strings.Index(v, "=")
		if eqidx > 0 {
			var val = v[eqidx+1:]
			pbc.Docker.Labels[v[0:eqidx]] = val
		} else {
			pbc.Docker.Labels[v] = ""
		}
	}

	return pbc.Publish()
}

// ProjectBuild -
type ProjectBuild struct {
	BaseAction
}

// NewProjectBuild -
func NewProjectBuild(setting *GlobalSetting) *ProjectBuild {
	return &ProjectBuild{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// ProjectInit -
type ProjectInit struct {
	BaseAction
}

// NewProjectBuild -
func NewProjectInit(setting *GlobalSetting) *ProjectInit {
	return &ProjectInit{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (d *ProjectInit) Run() (err error) {
	var (
		curdir string
		pbc    *ProjectBuildConfig
	)
	curdir, err = os.Getwd()
	if err != nil {
		d.ErrorExit("%s", err)
	}

	pbc, err = LoadProjectBuildFromDirectory(curdir)
	if err != nil {
		d.ErrorExit("%s", err)
	}
	pbc.action = d
	return pbc.Init()
}

// Run -
func (d *ProjectBuild) Run() (err error) {
	var (
		curdir string
		pbc    *ProjectBuildConfig
	)
	curdir, err = os.Getwd()
	if err != nil {
		d.ErrorExit("%s", err)
	}

	pbc, err = LoadProjectBuildFromDirectory(curdir)
	if err != nil {
		d.ErrorExit("%s", err)
	}
	pbc.action = d
	return pbc.Build()
}

// ProjectClean -
type ProjectClean struct {
	BaseAction
}

// NewProjectClean -
func NewProjectClean(setting *GlobalSetting) *ProjectClean {
	return &ProjectClean{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (d *ProjectClean) Run() (err error) {
	var (
		curdir string
		pbc    *ProjectBuildConfig
	)
	curdir, err = os.Getwd()
	if err != nil {
		d.ErrorExit("%s", err)
		return
	}

	pbc, err = LoadProjectBuildFromDirectory(curdir)
	if err != nil {
		d.ErrorExit("%s", err)
		return
	}
	pbc.action = d
	return pbc.Clean()
}

// ProjectRun -
type ProjectRun struct {
	BaseAction
}

// NewProjectRun -
func NewProjectRun(setting *GlobalSetting) *ProjectRun {
	return &ProjectRun{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (d *ProjectRun) Run() (err error) {
	var (
		curdir string
		pbc    *ProjectBuildConfig
	)
	curdir, err = os.Getwd()
	if err != nil {
		d.ErrorExit("%s", err)
	}

	pbc, err = LoadProjectBuildFromDirectory(curdir)
	if err != nil {
		d.ErrorExit("%v", err)
	}
	pbc.action = d
	return pbc.Run()
}

// ProjectTest -
type ProjectTest struct {
	BaseAction
}

// NewProjectTest -
func NewProjectTest(setting *GlobalSetting) *ProjectTest {
	return &ProjectTest{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (d *ProjectTest) Run() (err error) {
	var (
		curdir string
		pbc    *ProjectBuildConfig
	)
	curdir, err = os.Getwd()
	if err != nil {
		d.ErrorExit("%s", err)
	}

	pbc, err = LoadProjectBuildFromDirectory(curdir)
	if err != nil {
		d.ErrorExit("%v", err)
	}
	pbc.action = d
	return pbc.Test()
}

// ProjectPackage -
type ProjectPackage struct {
	BaseAction

	SkipBuildDocker     bool
	SkipBuildSource     bool
	PushDockerImage     bool
	SaveDockerImage     bool
	DockerImagePlatform string
}

// NewProjectPackage -
func NewProjectPackage(setting *GlobalSetting) *ProjectPackage {
	return &ProjectPackage{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (d *ProjectPackage) Run() (err error) {
	var (
		curdir string
		pbc    *ProjectBuildConfig
	)
	curdir, err = os.Getwd()
	if err != nil {
		d.ErrorExit("%s", err)
	}

	pbc, err = LoadProjectBuildFromDirectory(curdir)
	if err != nil {
		d.ErrorExit("%v", err)
	}

	pbc.action = d
	pbc.SkipBuildDocker = d.SkipBuildDocker
	pbc.SkipBuildSource = d.SkipBuildSource
	pbc.SaveDockerImage = d.SaveDockerImage
	pbc.Docker.currentPlatform = d.DockerImagePlatform

	if err = pbc.Docker.ValidatePlatform(d.DockerImagePlatform); err != nil {
		d.ErrorExit("%v", err)
	}
	return pbc.Package(d.PushDockerImage)
}

// ProjectDocker -
type ProjectDocker struct {
	BaseAction

	Dockerfile string

	BuildArgs map[string]*string
	Labels    map[string]string
	ImageTag  string
	SSHKey    []string

	SkipBuildSource bool
	Push            bool
	Platform        string
}

// NewProjectDocker -
func NewProjectDocker(setting *GlobalSetting) *ProjectDocker {
	return &ProjectDocker{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (d *ProjectDocker) Run() (err error) {
	var (
		curdir string
		pbc    *ProjectBuildConfig
	)
	curdir, err = os.Getwd()
	if err != nil {
		d.ErrorExit("%s", err)
	}

	pbc, err = LoadProjectBuildFromDirectory(curdir)
	if err != nil {
		d.ErrorExit("%v", err)
	}
	pbc.action = d

	if pbc.Docker == nil {
		pbc.Docker = &DockerBuildConfig{}
	}

	pbc.Docker.Args = make(map[string]*string)
	pbc.Docker.Labels = make(map[string]string)

	if d.Dockerfile != "" {
		pbc.Docker.Dockerfile = d.Dockerfile
	}

	if d.ImageTag != "" {
		pbc.Docker.Tag = d.ImageTag
	}

	for k, v := range d.BuildArgs {
		pbc.Docker.Args[k] = v
	}

	for k, v := range d.Labels {
		pbc.Docker.Labels[k] = v
	}

	pbc.Docker.SSHKeys = d.SSHKey

	pbc.SkipBuildSource = d.SkipBuildSource

	if !d.Push {
		if err = pbc.Docker.ValidatePlatform(d.Platform); err != nil {
			d.ErrorExit("%v", err)
			return
		}
		pbc.Docker.currentPlatform = d.Platform
	}

	return pbc.BuildDocker(d.Push)
}

// ProjectStandalone -
type ProjectStandalone struct {
	ProjectDocker
	Command string

	Follow     bool
	Since      string
	Tail       string
	Timestamps bool
	Details    bool
	Until      string

	Really bool

	ForcePullImage  bool
	LoadImageLocal  bool
	SkipBuildDocker bool
}

// NewProjectStandalone -
func NewProjectStandalone(setting *GlobalSetting) *ProjectStandalone {
	return &ProjectStandalone{
		ProjectDocker: ProjectDocker{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// Run -
func (d *ProjectStandalone) Run() (err error) {
	var (
		curdir string
		pbc    *ProjectBuildConfig
	)
	curdir, err = os.Getwd()
	if err != nil {
		d.ErrorExit("%s", err)
	}

	pbc, err = LoadProjectBuildFromDirectory(curdir)
	if err != nil {
		d.ErrorExit("%v", err)
	}
	pbc.action = d

	if pbc.Docker == nil {
		pbc.Docker = &DockerBuildConfig{}
	}

	pbc.Docker.Args = make(map[string]*string)
	pbc.Docker.Labels = make(map[string]string)

	for k, v := range d.BuildArgs {
		pbc.Docker.Args[k] = v
	}

	for k, v := range d.Labels {
		pbc.Docker.Labels[k] = v
	}

	pbc.Follow = d.Follow
	pbc.Since = d.Since
	pbc.Tail = d.Tail
	pbc.Timestamps = d.Timestamps
	pbc.Details = d.Details
	pbc.Until = d.Until
	pbc.Really = d.Really

	pbc.ForcePullImage = d.ForcePullImage
	pbc.LoadImageLocal = d.LoadImageLocal
	pbc.SkipBuildDocker = d.SkipBuildDocker

	sd := strings.ToLower(d.Command)
	switch sd {
	case "start":
		return pbc.StartStandalone()
	case "stop":
		return pbc.StopStandalone(false)
	case "stop-all":
		pbc.StopStandalone(false)
		return pbc.StopStandalone(true)
	case "clean":
		return pbc.CleanStandalone(false)
	case "clean-all":
		return pbc.CleanStandalone(true)
	case "logs":
		return pbc.LogStandalone()
	case "restart":
		pbc.StopStandalone(false)
		return pbc.StartStandalone()
	case "restart-all":
		pbc.CleanStandalone(true)
		return pbc.StartStandalone()
	case "develop", "profile":
		return pbc.StartDevelopProfile()
	case "apply":
		return pbc.ApplyDeveleopConfig()
	default:
		d.ErrorExit("error standalone command: %s", sd)
	}
	return
}

// MakeInstall -
type MakeInstall struct {
	BaseAction
	// 安装的名字空间
	Namespace string
	Overlay   bool
}

// NewMakeInstall -
func NewMakeInstall(setting *GlobalSetting) *MakeInstall {
	return &MakeInstall{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (m *MakeInstall) Run() (err error) {
	var (
		curdir string
		pbc    *ProjectBuildConfig
	)
	curdir, err = os.Getwd()
	if err != nil {
		m.ErrorExit("%s", err)
	}

	pbc, err = LoadProjectBuildFromDirectory(curdir)
	if err != nil {
		m.ErrorExit("%s", err)
	}
	pbc.action = m
	if err = pbc.Package(false); err != nil {
		m.ErrorExit("%s", err)
		return
	}

	install := NewInstallBundle(m.GlobalSetting())
	install.From = filepath.Join(pbc.basedir, "out", fmt.Sprintf("%s-%s.tgz", pbc.Name, pbc.GetVersion()))
	install.Namespace = m.Namespace
	install.Overlay = m.Overlay
	return install.Run()
}

// MakeSonar -
type MakeSonar struct {
	BaseAction
	Token    string
	SonarURL string
}

// NewMakeSonar -
func NewMakeSonar(setting *GlobalSetting) *MakeSonar {
	return &MakeSonar{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

func getSonarScannerFile(bindir string) string {
	fcname := "sonar-scanner"
	switch runtime.GOOS {
	case "windows":
		fcname = fcname + ".bat"
	}
	return path.Join(bindir, fmt.Sprintf("%s-%s-%s", BaseConfig.Sonar.Name, BaseConfig.Sonar.Version, runtime.GOOS), "bin", fcname)
}

func sonarScannerFileExisted(bindir string) (kf string, err error) {
	kf = getSonarScannerFile(bindir)
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

func unarchiveSonarScanner(gzfpath string, binbasedir string) error {
	var (
		err error
	)
	if err = UnarchiveBundle(gzfpath, binbasedir); err != nil {
		return err
	}
	return nil
}

// Run -
func (m *MakeSonar) Run() (err error) {
	var (
		gzfpath string
		setting *GlobalSetting = m.setting
	)

	_, err = sonarScannerFileExisted(setting.GetBinDir())
	if err != nil {
		if gzfpath, err = downloadSonarScanner(m, setting.OutOrStdout(), false); err != nil {
			m.ErrorExit("%s", err.Error())
			return err
		}
		if err = unarchiveSonarScanner(gzfpath, setting.GetBinDir()); err != nil {
			m.ErrorExit("%s", err.Error())
			return err
		}
	}

	var (
		curdir string
		pbc    *ProjectBuildConfig
	)
	curdir, err = os.Getwd()
	if err != nil {
		m.ErrorExit("%s", err)
	}

	pbc, err = LoadProjectBuildFromDirectory(curdir)
	if err != nil {
		m.ErrorExit("%s", err)
	}
	pbc.action = m

	if pbc.Sonar == nil {
		pbc.Sonar = new(ProjectSonarConfig)
	}

	if m.Token != "" {
		pbc.Sonar.Token = m.Token
	}

	if m.SonarURL != "" {
		pbc.Sonar.URL = m.SonarURL
	} else {
		pbc.Sonar.URL = BaseConfig.Snz1dp.Sonar.URL
	}

	if err = pbc.SonarScanner(); err != nil {
		m.ErrorExit("%s", err)
		return
	}
	return
}

type GetBundleInfo struct {
	BaseAction
	ShowName    bool
	ShowVersion bool
}

// NewGetBundleInfo -
func NewGetBundleInfo(setting *GlobalSetting) *GetBundleInfo {
	return &GetBundleInfo{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (d *GetBundleInfo) Run() (err error) {
	var (
		curdir string
		pbc    *ProjectBuildConfig
	)
	curdir, err = os.Getwd()
	if err != nil {
		d.ErrorExit("%s", err)
	}

	pbc, err = LoadProjectBuildFromDirectory(curdir)
	if err != nil {
		d.ErrorExit("%v", err)
	}

	if d.ShowName && d.ShowVersion {
		fmt.Printf("%s-%s", pbc.Name, pbc.Version)
	} else if d.ShowVersion {
		fmt.Printf("%s", pbc.Version)
	} else {
		fmt.Printf("%s", pbc.Name)
	}
	return
}
