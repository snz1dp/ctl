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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	helmAction "helm.sh/helm/v3/pkg/action"
	chartLoader "helm.sh/helm/v3/pkg/chart/loader"
	helmCli "helm.sh/helm/v3/pkg/cli"
	helmValues "helm.sh/helm/v3/pkg/cli/values"
	chartDownloader "helm.sh/helm/v3/pkg/downloader"
	helmGetter "helm.sh/helm/v3/pkg/getter"
	helmRelease "helm.sh/helm/v3/pkg/release"
	"k8s.io/client-go/kubernetes"
	"snz1.cn/snz1dp/snz1dpctl/kube"
	"snz1.cn/snz1dp/snz1dpctl/storage"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

// StartHelmBin -
type StartHelmBin struct {
	BaseAction
	Args []string
}

// NewStartKubectl -
func NewStartHelmBin(setting *GlobalSetting) *StartHelmBin {
	return &StartHelmBin{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

func resolveHelmBinPath(s Action, ic *InstallConfiguration) (mkfpath string, err error) {
	setting := s.GlobalSetting()
	var (
		gzfpath string
	)

	mkfpath, err = helmBinFileExisted(setting.GetBinDir())
	if err != nil {
		if gzfpath, err = downloadHelmBin(s, setting.OutOrStdout(), false); err != nil {
			return
		}
		if err = unarchiveHelmBin(gzfpath, setting.GetBinDir()); err != nil {
			return
		}
	}

	helmBinDir := getHelmBinDir(setting.GetBinDir())
	if err = utils.FixBinSearchPath(helmBinDir); err != nil {
		return
	}

	var pluginListBuffer bytes.Buffer
	helmPluginList := exec.CommandContext(context.Background(), mkfpath, "plugin", "list")
	helmPluginList.Stdout = &pluginListBuffer
	if err = helmPluginList.Run(); err != nil {
		return
	}

	pluginListStdout := pluginListBuffer.String()
	pushPluginURL := ic.Snz1dp.Server.GitURL
	if !strings.HasSuffix(pushPluginURL, "/") {
		pushPluginURL += "/"
	}
	switch runtime.GOOS {
	case "windows":
		pushPluginURL += BaseConfig.Helm.WindowsPlugin
	default:
		pushPluginURL += BaseConfig.Helm.Plugin
	}

	if !strings.Contains(pluginListStdout, BaseConfig.Snz1dp.Helm.PushPlugin) {
		// 检查是否安装了Git
		if !utils.IsGitInstalled() {
			s.ErrorExit("install helm plugin %s error: %s", BaseConfig.Snz1dp.Helm.PushPlugin, "git not install")
		}

		// 本地目录
		pluginLocalDir := path.Join(setting.GetBinDir(), "helm-nexus-push")
		os.RemoveAll(pluginLocalDir)

		// 克隆git到本地
		pluginListBuffer.Reset()
		helmClone := exec.CommandContext(context.Background(), "git", "clone", pushPluginURL, pluginLocalDir)
		helmClone.Stdout = &pluginListBuffer

		if err = helmClone.Run(); err != nil {
			s.ErrorExit("git clone plugin repo(%s) error: %s\n%s", pushPluginURL, err, pluginListBuffer.String())
		}

		// 安装本地插件
		pluginListBuffer.Reset()
		helmPluginInstall := exec.CommandContext(context.Background(), mkfpath, "plugin", "install", pluginLocalDir)
		helmPluginInstall.Stdout = &pluginListBuffer
		if err = helmPluginInstall.Run(); err != nil {
			s.ErrorExit("install helm plugin %s error: %s\n%s", BaseConfig.Snz1dp.Helm.PushPlugin, err, pluginListBuffer.String())
		}
		pluginInstallStdout := pluginListBuffer.String()
		if !(strings.Index(pluginInstallStdout, "plugin already exists") != 0 ||
			strings.Index(pluginInstallStdout, "Installed plugin") != 0) {
			s.ErrorExit("install helm plugin %s error: %s\n%s", BaseConfig.Snz1dp.Helm.PushPlugin, pluginInstallStdout)
		}
	}
	return
}

// Run -
func (s *StartHelmBin) Run() (err error) {
	setting := s.GlobalSetting()

	var (
		mkfpath string
		ic      *InstallConfiguration
	)

	setting.InitLogger(BaseConfig.Helm.Name)

	if _, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		s.ErrorExit("%s", err.Error())
		return
	}

	mkfpath, err = resolveHelmBinPath(s, ic)
	if err != nil {
		s.ErrorExit("%s", err.Error())
	}

	helmBinArgs := s.Args
	helmBin := exec.CommandContext(context.Background(), mkfpath, helmBinArgs...)
	helmBin.Stdin = setting.InOrStdin()
	helmBin.Stdout = setting.OutOrStdout()
	helmBin.Stderr = setting.ErrOrStderr()
	if err = helmBin.Run(); err != nil {
		s.ErrorExit("%v", err)
		return
	}
	return
}

// InitHelmActionConfig - 初始化Helm配置
func InitHelmActionConfig(setting *GlobalSetting) (err error) {
	if setting.helmSetting != nil {
		return
	}
	var (
		ic *InstallConfiguration = setting.installConfig
	)
	if ic == nil {
		// 加载本地配置
		_, ic, err = setting.LoadLocalInstallConfiguration()
		if err != nil {
			return
		}
	}

	// 覆盖k8sconfig配置
	setting.ApplyInstallConfig(ic)

	// 设置helm配置
	os.Setenv("HELM_NAMESPACE", setting.Namespace())
	if setting.IsDebug() {
		os.Setenv("HELM_DEBUG", "true")
	}

	helmSetting := helmCli.New()
	helmActionConfig := new(helmAction.Configuration)
	helmDriver := ""

	if err = helmActionConfig.Init(setting.RESTClientGetter(), setting.Namespace(), helmDriver, setting.Debug); err != nil {
		return
	}

	setting.helmConfig = helmActionConfig
	setting.helmSetting = helmSetting
	setting.kubeClient = helmActionConfig.KubeClient

	lazyClient := &lazyClient{
		namespace: setting.Namespace(),
		clientFn:  setting.KubernetesClientSet,
	}
	setting.storage = storage.Init(newSecretClient(lazyClient), setting.Debug)
	return
}

// HelmComponent Helm组件
type HelmComponent struct {
	name    string
	chart   string
	config  string
	setting *GlobalSetting
	values  []string
}

// NewHelmComponent 新建helm组件
func NewHelmComponent(setting *GlobalSetting, name string, chart string, configfile string, stringValues []string) *HelmComponent {
	return &HelmComponent{
		name:    name,
		chart:   chart,
		config:  configfile,
		setting: setting,
		values:  stringValues,
	}
}

// Existed 是否存在
func (c *HelmComponent) Existed() (*helmRelease.Release, error) {
	return c.setting.helmConfig.Releases.Last(c.name)
}

// UnInstall 卸载组件
func (c *HelmComponent) UnInstall() (*helmRelease.UninstallReleaseResponse, error) {
	helmClient := helmAction.NewUninstall(c.setting.helmConfig)
	return helmClient.Run(c.name)
}

// Upgrade 安装
func (c *HelmComponent) Upgrade() error {
	helmClient := helmAction.NewUpgrade(c.setting.helmConfig)
	helmClient.Wait = true

	cp, err := helmClient.ChartPathOptions.LocateChart(c.chart, c.setting.helmSetting)
	if err != nil {
		return err
	}

	c.setting.Debug("CHART PATH: %s\n", cp)
	valueOpts := &helmValues.Options{
		ValueFiles: []string{
			c.config,
		},
		StringValues: c.values,
	}

	p := helmGetter.All(c.setting.helmSetting)
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return err
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := chartLoader.Load(cp)
	if err != nil {
		return err
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := helmAction.CheckDependencies(chartRequested, req); err != nil {
			return err
		}
	}

	helmClient.Namespace = c.setting.Namespace()
	_, err = helmClient.Run(c.name, chartRequested, vals)
	if strings.Index(err.Error(), "with kind Job") > 0 {
		err = nil
	}

	return err
}

// Template 安装
func (c *HelmComponent) Template(ns string) (*helmRelease.Release, error) {
	helmClient := helmAction.NewInstall(c.setting.helmConfig)
	helmClient.Wait = true
	helmClient.ReleaseName = c.name
	helmClient.DryRun = true
	helmClient.ClientOnly = true

	cp, err := helmClient.ChartPathOptions.LocateChart(c.chart, c.setting.helmSetting)
	if err != nil {
		return nil, err
	}

	c.setting.Debug("CHART PATH: %s\n", cp)
	valueOpts := &helmValues.Options{
		ValueFiles: []string{
			c.config,
		},
		StringValues: c.values,
	}

	p := helmGetter.All(c.setting.helmSetting)
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return nil, err
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := chartLoader.Load(cp)
	if err != nil {
		return nil, err
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := helmAction.CheckDependencies(chartRequested, req); err != nil {
			if helmClient.DependencyUpdate {
				man := &chartDownloader.Manager{
					Out:              c.setting.OutOrStdout(),
					ChartPath:        cp,
					Keyring:          helmClient.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: c.setting.helmSetting.RepositoryConfig,
					RepositoryCache:  c.setting.helmSetting.RepositoryCache,
					Debug:            c.setting.IsDebug(),
				}
				if err := man.Update(); err != nil {
					return nil, err
				}
				// Reload the chart with the updated Chart.lock file.
				if chartRequested, err = chartLoader.Load(cp); err != nil {
					return nil, errors.Wrap(err, "failed reloading chart after repo update")
				}
			} else {
				return nil, err
			}
		}
	}

	helmClient.Namespace = ns
	return helmClient.Run(chartRequested, vals)
}

// Install 安装
func (c *HelmComponent) Install() (*helmRelease.Release, error) {
	helmClient := helmAction.NewInstall(c.setting.helmConfig)
	helmClient.Wait = true
	helmClient.ReleaseName = c.name

	cp, err := helmClient.ChartPathOptions.LocateChart(c.chart, c.setting.helmSetting)
	if err != nil {
		return nil, err
	}

	c.setting.Debug("CHART PATH: %s\n", cp)
	valueOpts := &helmValues.Options{
		ValueFiles: []string{
			c.config,
		},
		StringValues: c.values,
	}

	p := helmGetter.All(c.setting.helmSetting)
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return nil, err
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := chartLoader.Load(cp)
	if err != nil {
		return nil, err
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := helmAction.CheckDependencies(chartRequested, req); err != nil {
			if helmClient.DependencyUpdate {
				man := &chartDownloader.Manager{
					Out:              c.setting.OutOrStdout(),
					ChartPath:        cp,
					Keyring:          helmClient.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: c.setting.helmSetting.RepositoryConfig,
					RepositoryCache:  c.setting.helmSetting.RepositoryCache,
					Debug:            c.setting.IsDebug(),
				}
				if err := man.Update(); err != nil {
					return nil, err
				}
				// Reload the chart with the updated Chart.lock file.
				if chartRequested, err = chartLoader.Load(cp); err != nil {
					return nil, errors.Wrap(err, "failed reloading chart after repo update")
				}
			} else {
				return nil, err
			}
		}
	}

	helmClient.Namespace = c.setting.Namespace()
	return helmClient.Run(chartRequested, vals)
}

// UnInstallHelmComponent 卸载helm组件
func UnInstallHelmComponent(comp Component) (rp *helmRelease.UninstallReleaseResponse, err error) {
	var (
		ic                          *InstallConfiguration = comp.InstallConfiguration()
		setting                     *GlobalSetting        = ic.GlobalSetting()
		chartPath, configFile, msgp string
		hc                          *HelmComponent
		spinner                     *utils.WaitSpinner
		rl                          *helmRelease.Release
		client                      *kubernetes.Clientset
		chartDir, defaultFile       string
		defaultValue                string
		defaultData                 []byte
	)
	if !comp.BeInstall() {
		setting.Println("%s-%s not install!", comp.GetName(), comp.GetVersion())
		return
	}

	chartPath = path.Join(setting.GetBundleDir(), comp.GetNameWithVersion())
	chartDir = path.Join(setting.GetBundleDir(), comp.GetNameWithVersion())
	os.RemoveAll(chartDir)
	defer os.RemoveAll(chartDir)

	if err = UnarchiveBundle(chartPath, setting.GetBundleDir()); err != nil {
		err = errors.Errorf("unarchive %s error: %s", chartPath, err)
		return
	}

	defaultFile = path.Join(chartDir, "values.yaml")
	if defaultData, err = os.ReadFile(defaultFile); err != nil {
		err = errors.Errorf("read %s error: %s", defaultFile, err)
		return
	}

	// 设置JWT配置
	ic.Snz1dp.Jwt = comp.GetJwtConfig()

	if defaultValue, err = ic.RenderString(string(defaultData)); err != nil {
		err = errors.Errorf("config %s error: %s", defaultFile, err)
		return
	}

	if err = os.WriteFile(defaultFile, []byte(defaultValue), 0644); err != nil {
		err = errors.Errorf("write %s error: %s", defaultFile, err)
		return
	}

	configFile = path.Join(setting.GetConfigDir(), comp.GetName()+"-kubernetes.yaml")
	hc = NewHelmComponent(setting, comp.GetName(), chartDir, configFile, []string{})

	msgp = fmt.Sprintf("uninstall %s-%s...", comp.GetName(), comp.GetVersion())
	spinner = utils.NewSpinner(msgp, setting.OutOrStdout())
	rl, err = hc.Existed()
	if rl == nil {
		client, err = setting.KubernetesClientSet()
		if err != nil {
			spinner.Close()
			setting.Println("failed: %s", err)
			return
		}
		kube.WaitComponentNotPods(client, setting.Namespace(), comp.GetName(), 0, setting.Debug)
		spinner.Close()
		setting.Println("ok!")
		return
	}

	rp, err = hc.UnInstall()
	if err != nil {
		spinner.Close()
		setting.Println(fmt.Sprintf("failed: %s", err))
		return
	}

	client, err = setting.KubernetesClientSet()
	if err != nil {
		spinner.Close()
		setting.Println(fmt.Sprintf("failed: %s", err))
		return
	}

	kube.WaitComponentNotPods(client, setting.Namespace(), comp.GetName(), 0, setting.Debug)
	spinner.Close()
	setting.Println("ok!")

	return rp, err
}

// InstallHelmComponent 安装helm组件
func InstallHelmComponent(comp Component) (rl *helmRelease.Release, err error) {

	var (
		ic                                       *InstallConfiguration = comp.InstallConfiguration()
		setting                                  *GlobalSetting        = ic.GlobalSetting()
		msgp, chartPath, configFile, defaultFile string
		spinner                                  *utils.WaitSpinner
		confData, defaultData                    []byte
		hc                                       *HelmComponent
		tmpBundleDir                             string
		confValues, defaultValue                 string
		chartDir                                 string
	)

	if !comp.BeInstall() {
		setting.Println("ignored %s-%s install!", comp.GetName(), comp.GetVersion())
		return nil, nil
	}

	msgp = fmt.Sprintf("install %s-%s...", comp.GetName(), comp.GetVersion())
	spinner = utils.NewSpinner(msgp, setting.OutOrStdout())

	chartPath = path.Join(setting.GetBundleDir(), comp.GetNameWithVersion()+".tgz")
	chartDir = path.Join(setting.GetBundleDir(), comp.GetNameWithVersion())
	os.RemoveAll(chartDir)
	defer os.RemoveAll(chartDir)

	if err = UnarchiveBundle(chartPath, setting.GetBundleDir()); err != nil {
		err = errors.Errorf("unarchive %s error: %s", chartPath, err)
		return
	}

	defaultFile = path.Join(chartDir, "values.yaml")
	if defaultData, err = os.ReadFile(defaultFile); err != nil {
		err = errors.Errorf("read %s error: %s", defaultFile, err)
		return
	}

	// 设置JWT配置
	ic.Snz1dp.Jwt = comp.GetJwtConfig()

	if defaultValue, err = ic.RenderString(string(defaultData)); err != nil {
		err = errors.Errorf("config %s error: %s", defaultFile, err)
		return
	}

	if strings.Contains(defaultValue, BaseConfig.Snz1dp.Docker.URL) {
		defaultValue = strings.ReplaceAll(defaultValue,
			BaseConfig.Snz1dp.Docker.URL,
			ic.Snz1dp.Registry.GetDockerRepoURL(),
		)
	}

	if err = os.WriteFile(defaultFile, []byte(defaultValue), 0644); err != nil {
		err = errors.Errorf("write %s error: %s", defaultFile, err)
		return
	}

	if confData, err = comp.LoadKubernetesConfig(); err != nil {
		return
	}
	confValues = string(confData)

	if tmpBundleDir, err = ioutil.TempDir("", ".snz1dp.*"); err != nil {
		err = errors.Errorf("create temp directory error: %s", err)
		return
	}

	// 替换镜像名称
	if strings.Contains(confValues, BaseConfig.Snz1dp.Docker.URL) {
		confValues = strings.ReplaceAll(confValues,
			BaseConfig.Snz1dp.Docker.URL,
			ic.Snz1dp.Registry.GetDockerRepoURL(),
		)
	}

	defer os.RemoveAll(tmpBundleDir)
	configFile = path.Join(tmpBundleDir, comp.GetName()+"-kubernetes.yaml")
	if err = os.WriteFile(configFile, []byte(confValues), 0644); err != nil {
		err = errors.Errorf("write %s error: %s", configFile, err)
		return
	}

	var overlayValues = []string{
		"imagePullSecrets[0].name=" + ic.Snz1dp.Registry.GetK8sImagePullSecretName(),
	}

	for _, v := range comp.GetEnvironments() {
		overlayValues = append(overlayValues, "env."+v)
	}

	for i, v := range comp.GetExtrasHosts() {
		var extrasHostPrefix string = fmt.Sprintf("extrasHosts[%d].ip", i)
		var extrasHostArray []string = strings.Split(v, ":")
		if len(extrasHostArray) < 2 {
			continue
		}
		overlayValues = append(overlayValues, extrasHostPrefix+".ip="+extrasHostArray[1])
		overlayValues = append(overlayValues, extrasHostPrefix+".hostnames[0]="+extrasHostArray[0])
	}

	hc = NewHelmComponent(setting, comp.GetName(), chartDir, configFile, overlayValues)

	rl, err = hc.Existed()
	if rl != nil {
		err = hc.Upgrade()
		if err != nil {
			spinner.Close()
			setting.Println(fmt.Sprintf("failed: %s", err))
			return
		}

		var client *kubernetes.Clientset

		client, err = setting.KubernetesClientSet()
		if err != nil {
			spinner.Close()
			setting.Println(fmt.Sprintf("failed: %s", err))
			return
		}

		if err = kube.WaitDeploymentAvaible(client, setting.Namespace(), comp.GetName(), 0, setting.Debug); err != nil {
			err = kube.WaitStatefulSetAvaible(client, setting.Namespace(), comp.GetName(), 0, setting.Debug)
		}

		err = nil

		spinner.Close()
		setting.Println("ok!")
		return
	}

	rl, err = hc.Install()
	if err != nil {
		spinner.Close()
		setting.Println(fmt.Sprintf("failed: %s", err))
		return
	}

	spinner.Close()
	setting.Println("ok!")
	return rl, nil
}

// CreateHelmChart chart
type CreateHelmChart struct {
	BaseAction
}

// NewCreateHelmChart 创建chart
func NewCreateHelmChart(setting *GlobalSetting) *CreateHelmChart {
	return &CreateHelmChart{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (c *CreateHelmChart) Run() (err error) {
	var (
		curdir   string
		pbc      *ProjectBuildConfig
		chart    *HelmChart
		chartdir string
	)
	curdir, err = os.Getwd()
	if err != nil {
		c.ErrorExit("%s", err)
	}

	pbc, err = LoadProjectBuildFromDirectory(curdir)
	if err != nil {
		c.ErrorExit("%s", err)
	}

	chart = &HelmChart{
		Version:         pbc.GetVersion(),
		Name:            pbc.Name,
		Description:     pbc.Description,
		Type:            pbc.Catelog,
		ImageRepository: pbc.Docker.Image,
		ImageTag:        pbc.Version,
		Service:         pbc.Service,
	}

	chartdir, err = CreateChart(path.Join(curdir, "chart"), chart)
	if err != nil {
		c.ErrorExit("%s", err)
	} else {
		c.Println("helm chart created at %s", chartdir)
	}

	return
}

// CreateK8sDeployYaml deploy.yaml
type CreateK8sDeployYaml struct {
	BaseAction
	Force bool
}

// NewCreateK8sDeployYaml 创建deploy.yaml
func NewCreateK8sDeployYaml(setting *GlobalSetting) *CreateK8sDeployYaml {
	return &CreateK8sDeployYaml{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (c *CreateK8sDeployYaml) Run() (err error) {
	var (
		setting        *GlobalSetting = c.GlobalSetting()
		ic             *InstallConfiguration
		curdir         string
		pbc            *ProjectBuildConfig
		hc             *HelmComponent
		bundleTmpDir   string
		bundleFileDir  string
		bundleFilePath string
		valuesdata     []byte
		confgValues    string
		hrl            *helmRelease.Release
		outfile        string
		locfile        string
	)
	curdir, err = os.Getwd()
	if err != nil {
		c.ErrorExit("%s", err)
	}

	outfile = path.Join(curdir, "deploy.yaml")

	if _, err = os.Stat(outfile); err == nil {
		if !c.Force && !utils.Confirm(fmt.Sprintf("file %s existed, proceed? (y/N)", outfile), setting.InOrStdin(), setting.OutOrStdout()) {
			c.Println("Cancelled.")
			return
		}
		os.RemoveAll(outfile)
	}

	pbc, err = LoadProjectBuildFromDirectory(curdir)
	if err != nil {
		c.ErrorExit("%s", err)
		return
	}
	// 加载本地配置
	_, ic, err = setting.LoadLocalInstallConfiguration()
	if err != nil {
		c.ErrorExit("%s", err)
		return
	}

	pbc.action = c

	if err = pbc.Package(false); err != nil {
		return
	}

	bundleTmpDir = path.Join(curdir, "out", "temp")
	bundleFileDir = path.Join(bundleTmpDir, fmt.Sprintf("%s-%s", pbc.Name, pbc.GetVersion()))
	bundleFilePath = path.Join(curdir, "out", fmt.Sprintf("%s-%s.tgz", pbc.Name, pbc.GetVersion()))
	locfile = path.Join(bundleFileDir, "values.yaml")

	os.RemoveAll(bundleTmpDir)
	defer os.RemoveAll(bundleTmpDir)
	os.MkdirAll(bundleTmpDir, os.ModePerm)

	// 加载values.yaml
	if err = UnarchiveBundle(bundleFilePath, bundleTmpDir); err != nil {
		c.ErrorExit("load bundle error: %s", err)
		return
	}

	if valuesdata, err = os.ReadFile(locfile); err != nil {
		c.ErrorExit("write config error: %s", err)
		return
	}

	// 设置jwt
	ic.Snz1dp.Jwt = pbc.GetJwtConfig()

	if confgValues, err = ic.RenderString(string(valuesdata)); err != nil {
		c.ErrorExit("config BUILD.yaml or chart/values.yaml error: %s", err)
		return
	}

	if err = os.WriteFile(locfile, []byte(confgValues), 0644); err != nil {
		c.ErrorExit("%s", err)
		return
	}

	if err = InitHelmActionConfig(setting); err != nil {
		os.RemoveAll(bundleTmpDir)
		c.ErrorExit("%s", err)
		return
	}

	hc = NewHelmComponent(setting, pbc.Name, bundleFileDir, locfile, []string{})

	if hrl, err = hc.Template(DefaultAppNS); err != nil {
		c.ErrorExit("create error: %s", err)
		return
	}

	valuesdata = []byte(hrl.Manifest)

	if err = os.WriteFile(outfile, valuesdata, 0644); err != nil {
		c.ErrorExit("save %s error: %s", err)
		return
	}

	c.Println("save %s ok!", outfile)

	return
}

func getHelmBinFile(bindir string) string {
	fcname := fmt.Sprintf("%s-%s/%s", runtime.GOOS, runtime.GOARCH, BaseConfig.Helm.Name)
	switch runtime.GOOS {
	case "windows":
		fcname = fcname + ".exe"
	}
	return path.Join(bindir, fcname)
}

func getHelmBinDir(bindir string) string {
	fcname := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	return path.Join(bindir, fcname)
}

func helmBinFileExisted(bindir string) (kf string, err error) {
	kf = getHelmBinFile(bindir)
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

func unarchiveHelmBin(gzfpath string, binbasedir string) error {
	var (
		err error
	)
	if err = UnarchiveBundle(gzfpath, binbasedir); err != nil {
		return err
	}
	return nil
}
