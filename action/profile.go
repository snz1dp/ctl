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
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
	"github.com/ghodss/yaml"
	"github.com/howeyc/gopass"
	"github.com/pkg/errors"
	helmGetter "helm.sh/helm/v3/pkg/getter"
	helmRepo "helm.sh/helm/v3/pkg/repo"
	"snz1.cn/snz1dp/snz1dpctl/docker"
	"snz1.cn/snz1dp/snz1dpctl/down"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

// ShowProfile -
type ShowProfile struct {
	BaseAction
}

// NewShowProfile -
func NewShowProfile(setting *GlobalSetting) *ShowProfile {
	return &ShowProfile{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *ShowProfile) Run() error {
	setting := s.GlobalSetting()

	// 安装配置文件
	configFilePath, _, err := setting.LoadLocalInstallConfiguration()
	if err != nil {
		s.ErrorExit("%s", err.Error())
		return err
	}
	fin, err := os.Open(configFilePath)
	if err != nil {
		s.ErrorExit("%s", err.Error())
		return err
	}
	defer fin.Close()
	io.Copy(setting.OutOrStdout(), fin)
	return nil
}

// FetchProfile -
type FetchProfile struct {
	BaseAction
}

// NewFetchProfile -
func NewFetchProfile(setting *GlobalSetting) *FetchProfile {
	return &FetchProfile{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (f *FetchProfile) Run() error {

	var (
		err            error
		ic, oldic      *InstallConfiguration
		configFilePath string
	)

	setting := f.GlobalSetting()

	// 初始化Helm操作配置
	InitHelmActionConfig(setting)

	// 检查K8s是否可用
	if err = setting.IsKubernetesReachable(); err != nil {
		f.ErrorExit("%s", err.Error())
		return err
	}

	if oldic, err = setting.IsInitialized(); err != nil {
		f.ErrorExit("snz1dp-%s not install!", utils.Version())
		return err
	}

	configFilePath, ic, _ = setting.LoadLocalInstallConfiguration()
	oldic.Kubernetes = ic.Kubernetes
	if err = setting.SaveLocalInstallConfiguration(oldic, configFilePath); err != nil {
		f.ErrorExit("save profile to %s error: %s", configFilePath, err.Error())
		return err
	}

	f.Println("profile saved in %s", configFilePath)

	return nil
}

// ProfileType -
type ProfileType string

// SetProfile -
type SetProfile struct {
	BaseAction
	ConfigType ProfileType
	Username   string
	Password   string

	DockerUsername string
	DockerPassword string
	DockerRepo     string

	HelmRepo     string
	HelmRepoName string
	HelmPassword string
	HelmUsername string

	MavenRepoID   string
	MavenRepoURL  string
	MavenUsername string
	MavenPassword string

	NpmRepoID      string
	NpmRepoURL     string
	NpmUsername    string
	NpmPassword    string
	SassSiteID     string
	SassBinarySite string
}

var (
	// MinimalProfile -
	MinimalProfile ProfileType = "minimal"
	// NormalProfile -
	NormalProfile ProfileType = "normal"
	// ProductionProfile -
	ProductionProfile ProfileType = "production"
)

// NewSetProfile -
func NewSetProfile(setting *GlobalSetting) *SetProfile {
	return &SetProfile{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

func (s *SetProfile) applyProfile(purl string) (err error) {

	var (
		currentDir     string
		setting        *GlobalSetting = s.GlobalSetting()
		configFilePath string
		oldic, ic      *InstallConfiguration
		furl           *url.URL
		spinner        *utils.WaitSpinner
		icdata         []byte
	)

	if currentDir, err = os.Getwd(); err != nil {
		s.ErrorExit("get cwd error: %s", err)
		return
	}

	if furl, err = url.Parse(purl); err != nil {
		s.ErrorExit("error url %s: %s", purl, err)
		return
	}

	// 安装配置文件
	if configFilePath, oldic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		s.ErrorExit("load %s error: %s", err)
		return
	}

	if furl.Scheme == "" || furl.Scheme == "file" {
		var fromFilePath string = furl.Path
		if !filepath.IsAbs(fromFilePath) {
			fromFilePath = filepath.Join(currentDir, fromFilePath)
			if fromFilePath, err = filepath.Abs(fromFilePath); err != nil {
				s.ErrorExit("error file path: %s", err)
				return
			}
		}

		if icdata, err = os.ReadFile(fromFilePath); err != nil {
			s.ErrorExit("read %s error: %s", fromFilePath, err)
			return
		}

	} else {
		spinner = utils.NewSpinner(fmt.Sprintf("download %s...", furl.String()), setting.OutOrStdout())
		var g down.Getter
		if g, err = down.AllProviders().ByScheme(furl.Scheme); err != nil {
			spinner.Close()
			s.ErrorExit("failed: %s", err)
			return
		}

		var fout *bytes.Buffer = bytes.NewBuffer(nil)

		if _, err = g.Get(purl, fout, nil, nil); err != nil {
			spinner.Close()
			s.ErrorExit("failed: %s", err)
			return
		}
		spinner.Close()

		icdata = fout.Bytes()
		s.Println("ok!")
	}

	if ic, err = LoadInstallConfigurationFromBytes([]byte(icdata)); err != nil {
		s.ErrorExit("file %s format error: %s", purl, err)
		return
	}

	// 内置配置
	ic.inline = oldic.inline

	if err = setting.InitInstallConfiguration(ic); err != nil {
		s.ErrorExit("load %s error: %s", purl, err)
		return
	}

	// 如果老的Jwt已有配置则设置
	if ic.Appgateway.GetJwtConfig() == nil && oldic.Appgateway.infile && oldic.Appgateway.GetJwtConfig() != nil {
		ic.Appgateway.SetJwtConfig(oldic.Appgateway.GetJwtConfig())
	}

	if err = setting.SaveLocalInstallConfiguration(ic, configFilePath); err != nil {
		s.ErrorExit("save %s config to %s error: %s", purl, configFilePath, err)
		return
	}

	s.Println("save %s -> %s ok!", purl, configFilePath)

	if configFilePath, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		s.ErrorExit("load %s error: %s", err)
		return
	}

	if err = s.doLogin(configFilePath, ic); err != nil {
		s.ErrorExit("login %s error: %s", configFilePath, err.Error())
		return
	}

	if _, _, err = ic.GetBundleComponents(false); err != nil {
		s.ErrorExit("load bundle error: %s", err.Error())
		return
	}

	return NewListBundle(s.setting).Run()
}

func (s *SetProfile) doLogin(configFilePath string, ic *InstallConfiguration) (err error) {
	login := NewLoginRegistry(ic.setting)

	login.Username = s.Username
	login.Password = s.Password

	login.DockerRepo = s.DockerRepo
	login.DockerUsername = s.DockerUsername
	login.DockerPassword = s.DockerPassword

	login.HelmRepo = s.HelmRepo
	login.HelmRepoName = s.HelmRepoName
	login.HelmUsername = s.HelmUsername
	login.HelmPassword = s.HelmUsername

	login.MavenRepoID = s.MavenRepoID
	login.MavenRepoURL = s.MavenRepoURL
	login.MavenUsername = s.MavenUsername
	login.MavenPassword = s.MavenPassword

	login.NpmRepoID = s.NpmRepoID
	login.NpmRepoURL = s.NpmRepoURL
	login.NpmUsername = s.NpmUsername
	login.NpmPassword = s.NpmPassword
	login.SassBinarySite = s.SassSiteID
	login.SassBinarySite = s.SassBinarySite

	if err = login.runLogin(configFilePath, ic); err != nil {
		return
	}

	return
}

// Run -
func (s *SetProfile) Run() (err error) {

	setting := s.GlobalSetting()

	var (
		ic             *InstallConfiguration
		configFilePath string
	)

	// 安装配置文件
	if configFilePath, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		s.ErrorExit("load %s error: %s", err)
		return
	}

	if s.ConfigType == "" {
		err = NewListBundle(s.setting).Run()
		return
	}

	switch s.ConfigType {
	case MinimalProfile:
		ic.Postgres.Install = false
		ic.Redis.Install = false
		ic.Appgateway.Install = false
		ic.Confserv.Install = false
		ic.Xeai.Install = false

	case NormalProfile:
		ic.Postgres.Install = true
		ic.Redis.Install = true
		ic.Appgateway.Install = true
		ic.Confserv.Install = false
		ic.Xeai.Install = false

	case ProductionProfile:
		ic.Postgres.Install = true
		ic.Redis.Install = true
		ic.Appgateway.Install = true
		ic.Confserv.Install = true
		ic.Xeai.Install = true

	default:
		return s.applyProfile(string(s.ConfigType))
	}

	if err = s.doLogin(configFilePath, ic); err != nil {
		s.ErrorExit("login %s error: %s", configFilePath, err.Error())
		return
	}

	if configFilePath, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		s.ErrorExit("load %s error: %s", err)
		return
	}

	var (
		compNames  []string
		components map[string]Component
	)

	if compNames, components, err = ic.GetBundleComponents(false); err != nil {
		s.ErrorExit("load bundle error: %s", err.Error())
		return
	}

	for _, v := range compNames {
		comp := components[v]
		comp.PreInit(s)
	}

	if err = setting.SaveLocalInstallConfiguration(ic, configFilePath); err != nil {
		s.ErrorExit("save %s error: %s", configFilePath, err.Error())
		return
	}

	s.Println("profile saved in %s", configFilePath)
	s.Println("")

	err = NewListBundle(s.setting).Run()
	return

}

// LoginRegistry -
type LoginRegistry struct {
	BaseAction

	RepoType string

	Username string
	Password string

	RepoURL string

	DockerUsername string
	DockerPassword string
	DockerRepo     string

	HelmRepo     string
	HelmRepoName string
	HelmPassword string
	HelmUsername string

	MavenRepoID   string
	MavenRepoURL  string
	MavenUsername string
	MavenPassword string

	NpmRepoID      string
	NpmRepoURL     string
	SassSiteID     string
	SassBinarySite string
	NpmUsername    string
	NpmPassword    string

	PromptInput bool
}

// NewLoginRegistry -
func NewLoginRegistry(setting *GlobalSetting) *LoginRegistry {
	return &LoginRegistry{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

func getSimpleType(v interface{}) string {
	switch i := v.(type) {
	case bool:
		return "bool"
	case string:
		return "string"
	default:
		_ = i
		return "other"
	}
}

// Run -
func (s *LoginRegistry) Run() (err error) {
	setting := s.GlobalSetting()

	var (
		configFilePath string
		ic             *InstallConfiguration
	)

	// 安装配置文件
	if configFilePath, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		s.ErrorExit("load %s error: %s", err)
		return
	}

	// 检查更新
	checkNewVersion(setting)

	// 执行登录
	err = s.runLogin(configFilePath, ic)
	return
}

// runLogin
func (s *LoginRegistry) runLogin(configFilePath string, ic *InstallConfiguration) (err error) {
	setting := s.GlobalSetting()

	var (
		dc *client.Client

		serverConfigURL string

		serverfurl *url.URL

		dockerRegistry     *DockerRegistry
		dockerRepoURL      string
		dockerSecure       bool
		dockerRepoUser     string
		dockerRepoPassword string

		helmRegistry     *HelmRegistry
		helmRepoName     string
		helmRepoURL      string
		helmRepoUser     string
		helmRepoPassword string

		mavenServerItem   *MavenServerItem
		mavenRepoID       string
		mavenRepoURL      string
		mavenRepoUser     string
		mavenRepoPassword string
		spinner           *utils.WaitSpinner

		npmServerItem   *NpmRegistry
		npmRepoID       string
		npmRepoURL      string
		npmRepoUser     string
		npmRepoPassword string

		sassSiteItem     *SassBinarySite
		sassSiteID       string
		sassSiteURL      string
		sassSiteUser     string
		sassSitePassword string

		downloadURL string

		dockerLogin   bool
		helmLogin     bool
		mavenLogin    bool
		npmLogin      bool
		sassSiteLogin bool

		saveConfig bool

		configBytes []byte
	)

loginToURL:

	if strings.ToLower(s.RepoType) == "docker" {
		if docker.IsDockerRunning() {
			dockerLogin = true
		} else {
			s.ErrorExit("container server is not running!")
			return
		}
		dockerRepoURL = s.RepoURL
		dockerRepoUser = s.Username
		dockerRepoPassword = s.Password
	} else if strings.ToLower(s.RepoType) == "helm" {
		helmLogin = true
		helmRepoName = s.HelmRepoName
		helmRepoURL = s.RepoURL
		helmRepoUser = s.Username
		helmRepoPassword = s.Password
	} else if strings.ToLower(s.RepoType) == "maven" {
		mavenLogin = true
		mavenRepoID = s.MavenRepoID
		mavenRepoURL = s.RepoURL
		mavenRepoUser = s.Username
		mavenRepoPassword = s.Password
	} else if strings.ToLower(s.RepoType) == "npm" {
		npmLogin = true
		npmRepoID = s.NpmRepoID
		npmRepoURL = s.RepoURL
		npmRepoUser = s.Username
		npmRepoPassword = s.Password
	} else if strings.ToLower(s.RepoType) == "sass-site" {
		sassSiteLogin = true
		sassSiteID = s.SassSiteID
		sassSiteURL = s.RepoURL
		sassSiteUser = s.Username
		sassSitePassword = s.Password
	} else if strings.ToLower(s.RepoType) == "deploy" {
		dockerLogin = true
		dockerRepoURL = s.DockerRepo
		dockerRepoUser = s.Username
		dockerRepoPassword = s.Password

		helmLogin = true
		helmRepoURL = s.HelmRepo
		helmRepoUser = s.Username
		helmRepoPassword = s.Password
	} else if s.RepoType != "" {

		if strings.HasPrefix(s.RepoType, "https://") || strings.HasPrefix(s.RepoType, "http://") {
			serverConfigURL = s.RepoType
		} else {
			serverConfigURL = fmt.Sprintf("https://%s/develop/platform/api/configs", s.RepoType)
		}

		if BaseConfig.Snz1dp.Server.URL != serverConfigURL {
			if serverfurl, err = url.Parse(serverConfigURL); err != nil {
				s.ErrorExit("invalid server url: %s", serverConfigURL)
				return
			}

			spinner = utils.NewSpinner(fmt.Sprintf("fetching config from %s...", serverfurl.String()), setting.OutOrStdout())
			var g down.Getter
			if g, err = down.AllProviders().ByScheme(serverfurl.Scheme); err != nil {
				spinner.Close()
				s.ErrorExit("failed: %s", err)
				return
			}

			var fout *bytes.Buffer = bytes.NewBuffer(nil)
			if _, err = g.Get(serverConfigURL, fout, nil, nil); err != nil {
				spinner.Close()
				s.ErrorExit("download config failed: %s", err)
				return
			}
			spinner.Close()

			configBytes = fout.Bytes()
			s.Println("ok!")

			var configMap map[string]interface{}
			if err = json.Unmarshal(configBytes, &configMap); err != nil || configMap["code"] == nil || configMap["code"].(float64) != 0 || configMap["data"] == nil {
				s.ErrorExit("server config failed: %s", "data error")
				return
			}

			configMap = configMap["data"].(map[string]interface{})

			dockerLogin = true
			helmLogin = true
			mavenLogin = true

			npmLogin = false
			sassSiteLogin = false

			if configMap["nexus_url"] == nil || getSimpleType(configMap["nexus_url"]) != "string" {
				s.ErrorExit("config nexus_url not found or invalid")
				return
			}

			nexusURL := configMap["nexus_url"].(string)
			if !strings.HasPrefix(nexusURL, "/") {
				nexusURL += "/"
			}

			downloadURL = BaseConfig.Snz1dp.Download.Default
			if configMap["download_url"] != nil && getSimpleType(configMap["download_url"]) == "string" {
				downloadURL = configMap["download_url"].(string)
				if downloadURL != "" {
					if !strings.HasSuffix(downloadURL, "/") {
						downloadURL += "/"
					}
					saveConfig = true
				}
			}

			if configMap["dockerhub"] != nil && getSimpleType(configMap["dockerhub"]) == "string" {
				dockerRepoURL = configMap["dockerhub"].(string)
			}

			dockerRepoUser = s.Username
			dockerRepoPassword = s.Password

			if dockerRepoURL == ic.Snz1dp.Registry.URL {
				if dockerRepoUser == "" {
					dockerRepoUser = ic.Snz1dp.Registry.Username
				}

				if dockerRepoPassword == "" {
					dockerRepoPassword = ic.Snz1dp.Registry.Password
				}
			} else if dockerRepoUser == "" || dockerRepoPassword == "" {
				s.PromptInput = true
			}

			if configMap["maven_repo_id"] != nil && getSimpleType(configMap["maven_repo_id"]) == "string" {
				mavenRepoID = configMap["maven_repo_id"].(string)
			}

			if configMap["npm_repo_id"] != nil && getSimpleType(configMap["npm_repo_id"]) == "string" {
				npmRepoID = configMap["npm_repo_id"].(string)
			}

			if configMap["helm_repo_id"] != nil && getSimpleType(configMap["helm_repo_id"]) == "string" {
				helmRepoName = configMap["helm_repo_id"].(string)
			}

			if configMap["npm_sass_id"] != nil && getSimpleType(configMap["npm_sass_id"]) == "string" {
				sassSiteID = configMap["npm_sass_id"].(string)
			}

			helmHostedRepo := "helm-hosted"
			if configMap["hosted_helm_repo"] != nil && getSimpleType(configMap["hosted_helm_repo"]) == "string" {
				helmHostedRepo = configMap["hosted_helm_repo"].(string)
			}

			helmRepoURL = nexusURL + "repository/" + helmHostedRepo + "/"
			helmRepoUser = s.Username
			helmRepoPassword = s.Password

			mavenRepoURL = nexusURL + "repository/maven-public/"
			mavenRepoUser = s.Username
			mavenRepoPassword = s.Password

			if configMap["npm_repo_url"] != nil && getSimpleType(configMap["npm_repo_url"]) == "string" {
				npmRepoURL = configMap["npm_repo_url"].(string)
			}
			npmRepoUser = s.Username
			npmRepoPassword = s.Password
			if configMap["npm_repo_private"] != nil && getSimpleType(configMap["npm_repo_private"]) == "bool" {
				npmLogin = configMap["npm_repo_private"].(bool)
			}

			if configMap["npm_sass_url"] != nil && getSimpleType(configMap["npm_sass_url"]) == "string" {
				sassSiteURL = configMap["npm_sass_url"].(string)
			}
			sassSiteUser = s.Username
			sassSitePassword = s.Password
			if configMap["npm_sass_private"] != nil && getSimpleType(configMap["npm_sass_private"]) == "bool" {
				sassSiteLogin = configMap["npm_sass_private"].(bool)
			}

			if configMap["customer_name"] != nil && getSimpleType(configMap["customer_name"]) == "string" {
				ic.Snz1dp.Organization = configMap["customer_name"].(string)
			}

			runnerImage := BaseConfig.Runner.Docker.Image
			if configMap["runner_image"] != nil && getSimpleType(configMap["runner_image"]) == "string" {
				runnerImage = configMap["runner_image"].(string)
			}

			gitlabURL := BaseConfig.Snz1dp.Server.Git
			if configMap["source_repo"] != nil && getSimpleType(configMap["source_repo"]) == "string" {
				gitlabURL = configMap["source_repo"].(string)
			}

			mavenMirrorId := BaseConfig.Snz1dp.Maven.Mirrors[0].ID
			if configMap["maven_mirror_id"] != nil && getSimpleType(configMap["maven_mirror_id"]) == "string" {
				mavenMirrorId = configMap["maven_mirror_id"].(string)
			}

			mavenMirrorURL := BaseConfig.Snz1dp.Maven.Mirrors[0].URL
			if configMap["maven_mirror_url"] != nil && getSimpleType(configMap["maven_mirror_url"]) == "string" {
				mavenMirrorURL = configMap["maven_mirror_url"].(string)
			}

			mavenMirrorOf := BaseConfig.Snz1dp.Maven.Mirrors[0].MirrorOf
			if configMap["maven_mirror_of"] != nil && getSimpleType(configMap["maven_mirror_of"]) == "string" {
				mavenMirrorOf = configMap["maven_mirror_of"].(string)
			}

			if !strings.HasSuffix(gitlabURL, "/") {
				gitlabURL += "/"
			}

			if helmRepoName != ic.Snz1dp.HelmRepo.Name {
				ic.Snz1dp.HelmRepo.Name = helmRepoName
				saveConfig = true
			}

			if npmRepoID != ic.Snz1dp.NpmRepo.ID {
				ic.Snz1dp.NpmRepo.ID = npmRepoID
				saveConfig = true
			}

			if sassSiteID != ic.Snz1dp.SassSite.ID {
				ic.Snz1dp.SassSite.ID = sassSiteID
				saveConfig = true
			}

			if ic.Snz1dp.RunnerConfig.DockerImage != runnerImage {
				ic.Snz1dp.RunnerConfig.DockerImage = runnerImage
				saveConfig = true
			}

			if ic.Snz1dp.Server.URL != serverConfigURL {
				ic.Snz1dp.Server.URL = serverConfigURL
				saveConfig = true
			}

			if ic.Snz1dp.Server.DownloadPrefix != downloadURL {
				ic.Snz1dp.Server.DownloadPrefix = downloadURL
				ic.GlobalSetting().SetDownloadURL(downloadURL)
				saveConfig = true
			}

			if ic.Snz1dp.Server.GitURL != gitlabURL {
				ic.Snz1dp.Server.GitURL = gitlabURL
				saveConfig = true
			}

			var mirrors = []MavenMirrorItem{}
			mirrors = append([]MavenMirrorItem{},
				MavenMirrorItem{
					ID:       mavenMirrorId,
					Name:     mavenMirrorId,
					URL:      mavenMirrorURL,
					MirrorOf: mavenMirrorOf,
				})
			ic.Snz1dp.MavenRepo.Mirrors = &mirrors
		}
	} else {
		if ic.Snz1dp.Server.URL != BaseConfig.Snz1dp.Server.URL {
			s.RepoType = ic.Snz1dp.Server.URL
			goto loginToURL
		}

		dockerLogin = true
		helmLogin = true
		mavenLogin = true
		npmLogin = true
		sassSiteLogin = true

		dockerRepoURL = s.DockerRepo
		dockerRepoUser = s.Username
		dockerRepoPassword = s.Password

		helmRepoURL = s.HelmRepo
		helmRepoUser = s.Username
		helmRepoPassword = s.Password

		mavenRepoID = s.MavenRepoID
		mavenRepoURL = s.MavenRepoURL
		mavenRepoUser = s.Username
		mavenRepoPassword = s.Password

		npmRepoID = s.NpmRepoID
		npmRepoURL = s.NpmRepoURL
		npmRepoUser = s.Username
		npmRepoPassword = s.Password

		sassSiteID = s.SassSiteID
		sassSiteURL = s.SassBinarySite
		sassSiteUser = s.Username
		sassSitePassword = s.Password
	}

	if dockerLogin {
		if dockerRepoURL == "" {
			dockerRepoURL = ic.Snz1dp.Registry.URL
		} else {
			ic.Snz1dp.Registry.URL = dockerRepoURL
		}

		if dockerRepoURL == "" {
			dockerRepoURL = BaseConfig.Snz1dp.Docker.URL
			ic.Snz1dp.Registry.URL = dockerRepoURL
		}

		if dockerRepoUser == "" {
			dockerRepoUser = s.DockerUsername
		} else {
			dockerRepoUser = strings.TrimSpace(dockerRepoUser)
		}

		if dockerRepoPassword == "" {
			dockerRepoPassword = s.DockerPassword
		}

		if dockerRepoUser == "" && !s.PromptInput {
			dockerRepoUser = ic.Snz1dp.Registry.Username
		}

		if s.PromptInput {
			dockerRepoPassword = ""
		} else if dockerRepoPassword == "" && dockerRepoUser == ic.Snz1dp.Registry.Username && s.DockerUsername == "" && s.Username == "" {
			dockerRepoPassword = ic.Snz1dp.Registry.Password
		}

		if dockerRepoUser == "" {
			for {
				fmt.Fprintf(setting.OutOrStdout(), "image restry %s user: ", dockerRepoURL)
				_, err = fmt.Fscanf(setting.InOrStdin(), "%s", &dockerRepoUser)
				dockerRepoUser = strings.TrimSpace(dockerRepoUser)
				if err != nil || dockerRepoUser == "" {
					continue
				}
				break
			}
		}
		ic.Snz1dp.Registry.Username = dockerRepoUser

		if dockerRepoPassword == "" {
			for {
				var repoPwd []byte
				repoPwd, err = gopass.GetPasswdPrompt(
					fmt.Sprintf("image registry user %s password: ", dockerRepoUser),
					true, setting.InOrStdin().(gopass.FdReader), setting.OutOrStdout())
				if len(repoPwd) == 0 {
					continue
				}
				dockerRepoPassword = string(repoPwd)
				break
			}
		}
		ic.Snz1dp.Registry.Password = dockerRepoPassword

		if s.RepoType == "" {
			s.PromptInput = false

			helmRepoUser = dockerRepoUser
			helmRepoPassword = dockerRepoPassword

			mavenRepoUser = dockerRepoUser
			mavenRepoPassword = dockerRepoPassword

			npmRepoUser = dockerRepoUser
			npmRepoPassword = dockerRepoPassword

			sassSiteUser = dockerRepoUser
			sassSitePassword = dockerRepoPassword

		} else if s.RepoType == "deploy" {
			s.PromptInput = false
			helmRepoUser = dockerRepoUser
			helmRepoPassword = dockerRepoPassword
		} else {
			helmRepoUser = dockerRepoUser
			helmRepoPassword = dockerRepoPassword

			mavenRepoUser = dockerRepoUser
			mavenRepoPassword = dockerRepoPassword

			npmRepoUser = dockerRepoUser
			npmRepoPassword = dockerRepoPassword

			sassSiteUser = dockerRepoUser
			sassSitePassword = dockerRepoPassword
		}
	}

	if helmLogin {
		if helmRepoUser == "" {
			helmRepoUser = s.HelmUsername
		} else {
			helmRepoUser = strings.TrimSpace(helmRepoUser)
		}

		if helmRepoPassword == "" {
			helmRepoPassword = s.HelmPassword
		}

		if helmRepoURL == "" {
			helmRepoURL = ic.Snz1dp.HelmRepo.URL
		} else {
			ic.Snz1dp.HelmRepo.URL = helmRepoURL
		}

		if helmRepoUser == "" && !s.PromptInput {
			helmRepoUser = ic.Snz1dp.HelmRepo.Username
		} else {
			ic.Snz1dp.HelmRepo.Username = helmRepoUser
		}

		if helmRepoPassword == "" && helmRepoUser == ic.Snz1dp.HelmRepo.Username && s.HelmUsername == "" && s.Username == "" {
			helmRepoPassword = ic.Snz1dp.HelmRepo.Password
		} else {
			ic.Snz1dp.HelmRepo.Password = helmRepoPassword
		}

		if helmRepoName == "" {
			helmRepoName = ic.Snz1dp.HelmRepo.Name
		}
		ic.Snz1dp.HelmRepo.Name = helmRepoName

		if helmRepoUser == "" {
			for {
				fmt.Fprintf(setting.OutOrStdout(), "helm repo %s user: ", helmRepoURL)
				_, err = fmt.Fscanf(setting.InOrStdin(), "%s", &helmRepoUser)
				helmRepoUser = strings.TrimSpace(helmRepoUser)
				if err != nil || helmRepoUser == "" {
					continue
				}
				break
			}
		}
		if helmRepoPassword == "" {
			for {
				var repoPwd []byte
				repoPwd, err = gopass.GetPasswdPrompt(
					fmt.Sprintf("helm repo user %s password: ", helmRepoUser),
					true, setting.InOrStdin().(gopass.FdReader), setting.OutOrStdout())
				if len(repoPwd) == 0 {
					continue
				}
				helmRepoPassword = string(repoPwd)
				break
			}
		}
	}

	if mavenLogin {
		if mavenRepoUser == "" {
			mavenRepoUser = s.MavenUsername
		} else {
			mavenRepoUser = strings.TrimSpace(mavenRepoUser)
		}
		if mavenRepoPassword == "" {
			mavenRepoPassword = s.MavenPassword
		} else {
			mavenRepoPassword = strings.TrimSpace(mavenRepoPassword)
		}
		if mavenRepoURL == "" {
			mavenRepoURL = ic.Snz1dp.MavenRepo.URL
		} else {
			ic.Snz1dp.MavenRepo.URL = mavenRepoURL
		}
		if mavenRepoURL == "" {
			mavenRepoURL = BaseConfig.Snz1dp.Maven.URL
			ic.Snz1dp.MavenRepo.URL = mavenRepoURL
		}
		if mavenRepoID == "" {
			mavenRepoID = ic.Snz1dp.MavenRepo.ID
		} else {
			ic.Snz1dp.MavenRepo.ID = mavenRepoID
		}
		if mavenRepoID == "" {
			mavenRepoID = BaseConfig.Snz1dp.Maven.ID
			ic.Snz1dp.MavenRepo.ID = mavenRepoID
		} else {
			ic.Snz1dp.MavenRepo.ID = BaseConfig.Snz1dp.Maven.ID
		}
		if mavenRepoUser == "" && !s.PromptInput {
			mavenRepoUser = ic.Snz1dp.MavenRepo.Username
		} else {
			ic.Snz1dp.MavenRepo.Username = mavenRepoUser
		}
		if mavenRepoPassword == "" && mavenRepoUser == ic.Snz1dp.MavenRepo.Username && s.MavenUsername == "" && s.Username == "" {
			mavenRepoPassword = ic.Snz1dp.MavenRepo.Password
		} else {
			ic.Snz1dp.MavenRepo.Password = mavenRepoPassword
		}

		// 检查是否有Java环境
		jvmVersionCmdpath, jvmVersionCommands := parseCommand([]string{"java", "-version"})
		jvmVersionCmd := exec.CommandContext(context.Background(), jvmVersionCmdpath, jvmVersionCommands...)
		jvmVersionCmd.Stdin = setting.InOrStdin()

		if err = jvmVersionCmd.Run(); err != nil {
			mavenLogin = false
			s.Println("[%s] maven login %s(%s) ignored, not found java!", mavenRepoUser, mavenRepoID, mavenRepoURL)
		} else {
			if mavenRepoUser == "" {
				for {
					fmt.Fprintf(setting.OutOrStdout(), "maven repo %s user: ", mavenRepoURL)
					_, err = fmt.Fscanf(setting.InOrStdin(), "%s", &mavenRepoUser)
					mavenRepoUser = strings.TrimSpace(mavenRepoUser)
					if err != nil || mavenRepoUser == "" {
						continue
					}
					break
				}
			}
			if mavenRepoPassword == "" {
				for {
					var repoPwd []byte
					repoPwd, err = gopass.GetPasswdPrompt(
						fmt.Sprintf("maven repo user %s password: ", mavenRepoUser),
						true, setting.InOrStdin().(gopass.FdReader), setting.OutOrStdout())
					if len(repoPwd) == 0 {
						continue
					}
					mavenRepoPassword = string(repoPwd)
					break
				}
			}

			mavenServerItem = new(MavenServerItem)
			mavenServerItem.ID = mavenRepoID
			mavenServerItem.Username = mavenRepoUser
			mavenServerItem.Password = mavenRepoPassword
			mavenServerItem.URL = mavenRepoURL
		}
	}

	if npmLogin {
		if npmRepoUser == "" {
			npmRepoUser = s.NpmUsername
		} else {
			npmRepoUser = strings.TrimSpace(npmRepoUser)
		}
		if npmRepoPassword == "" {
			npmRepoPassword = s.NpmPassword
		} else {
			npmRepoPassword = strings.TrimSpace(npmRepoPassword)
		}
		if npmRepoURL == "" {
			npmRepoURL = ic.Snz1dp.NpmRepo.URL
		} else {
			ic.Snz1dp.NpmRepo.URL = npmRepoURL
		}
		if npmRepoURL == "" {
			npmRepoURL = BaseConfig.Snz1dp.Npm.URL
			ic.Snz1dp.NpmRepo.URL = npmRepoURL
		}
		if npmRepoID == "" {
			npmRepoID = ic.Snz1dp.NpmRepo.ID
		}
		ic.Snz1dp.NpmRepo.ID = npmRepoID

		if npmRepoID == BaseConfig.Snz1dp.Npm.ID &&
			npmRepoURL == BaseConfig.Snz1dp.Npm.URL {
			npmLogin = *ic.Snz1dp.NpmRepo.Private
		}

		if npmLogin {
			if npmRepoUser == "" && !s.PromptInput {
				npmRepoUser = ic.Snz1dp.NpmRepo.Username
			}
			if npmRepoPassword == "" && npmRepoUser == ic.Snz1dp.NpmRepo.Username && s.NpmUsername == "" && s.Username == "" {
				npmRepoPassword = ic.Snz1dp.NpmRepo.Password
			}
			if npmRepoUser == "" {
				for {
					fmt.Fprintf(setting.OutOrStdout(), "npm repo %s user: ", npmRepoURL)
					_, err = fmt.Fscanf(setting.InOrStdin(), "%s", &npmRepoUser)
					npmRepoUser = strings.TrimSpace(npmRepoUser)
					if err != nil || npmRepoUser == "" {
						continue
					}
					break
				}
			}
			if npmRepoPassword == "" {
				for {
					var repoPwd []byte
					repoPwd, err = gopass.GetPasswdPrompt(
						fmt.Sprintf("npm repo user %s password: ", npmRepoUser),
						true, setting.InOrStdin().(gopass.FdReader), setting.OutOrStdout())
					if len(repoPwd) == 0 {
						continue
					}
					npmRepoPassword = string(repoPwd)
					break
				}
			}
		}

		npmServerItem = new(NpmRegistry)
		npmServerItem.ID = npmRepoID
		npmServerItem.Username = npmRepoUser
		npmServerItem.Password = npmRepoPassword
		npmServerItem.URL = npmRepoURL
		npmServerItem.Private = &npmLogin
	} else {
		npmServerItem = new(NpmRegistry)
		npmServerItem.ID = npmRepoID
		npmServerItem.Username = npmRepoUser
		npmServerItem.Password = npmRepoPassword
		npmServerItem.URL = npmRepoURL
		npmServerItem.Private = new(bool)
		*npmServerItem.Private = false
	}

	if sassSiteLogin {
		if sassSiteUser == "" {
			sassSiteUser = s.NpmUsername
		} else {
			sassSiteUser = strings.TrimSpace(sassSiteUser)
		}
		if sassSitePassword == "" {
			sassSitePassword = s.NpmPassword
		} else {
			sassSitePassword = strings.TrimSpace(sassSitePassword)
		}
		if sassSiteURL == "" {
			sassSiteURL = ic.Snz1dp.SassSite.URL
		} else {
			ic.Snz1dp.SassSite.URL = sassSiteURL
		}
		if sassSiteURL == "" {
			sassSiteURL = BaseConfig.Snz1dp.Npm.SassBinarySite.URL
			ic.Snz1dp.SassSite.URL = sassSiteURL
		}
		if sassSiteID == "" {
			sassSiteID = ic.Snz1dp.SassSite.ID
		}
		if sassSiteID == "" {
			sassSiteID = BaseConfig.Snz1dp.Npm.SassBinarySite.ID
		}

		if sassSiteID == BaseConfig.Snz1dp.Npm.SassBinarySite.ID &&
			sassSiteURL == BaseConfig.Snz1dp.Npm.SassBinarySite.URL {
			sassSiteLogin = *ic.Snz1dp.SassSite.Private
		}

		if sassSiteLogin {
			if sassSiteUser == "" && !s.PromptInput {
				sassSiteUser = ic.Snz1dp.SassSite.Username
			}
			if sassSitePassword == "" && sassSiteUser == ic.Snz1dp.SassSite.Username && s.NpmUsername == "" && s.Username == "" {
				sassSitePassword = ic.Snz1dp.SassSite.Password
			}
			if sassSiteUser == "" {
				for {
					fmt.Fprintf(setting.OutOrStdout(), "npm repo %s user: ", sassSiteURL)
					_, err = fmt.Fscanf(setting.InOrStdin(), "%s", &sassSiteUser)
					sassSiteUser = strings.TrimSpace(sassSiteUser)
					if err != nil || sassSiteUser == "" {
						continue
					}
					break
				}
			}
			if sassSitePassword == "" {
				for {
					var repoPwd []byte
					repoPwd, err = gopass.GetPasswdPrompt(
						fmt.Sprintf("npm repo user %s password: ", sassSiteUser),
						true, setting.InOrStdin().(gopass.FdReader), setting.OutOrStdout())
					if len(repoPwd) == 0 {
						continue
					}
					sassSitePassword = string(repoPwd)
					break
				}
			}
		}

		sassSiteItem = new(SassBinarySite)
		sassSiteItem.ID = sassSiteID
		sassSiteItem.Username = sassSiteUser
		sassSiteItem.Password = sassSitePassword
		sassSiteItem.URL = sassSiteURL
		sassSiteItem.Private = &sassSiteLogin
	} else {
		sassSiteItem = new(SassBinarySite)
		sassSiteItem.ID = sassSiteID
		sassSiteItem.Username = sassSiteUser
		sassSiteItem.Password = sassSitePassword
		sassSiteItem.URL = sassSiteURL
		sassSiteItem.Private = new(bool)
		*sassSiteItem.Private = false
	}

	if dockerLogin {
		if docker.IsDockerRunning() {
			if dc, err = docker.NewClient(); err != nil {
				s.ErrorExit("container server error: %s", err)
				return
			}
		}

		// 构造新的地址
		dockerRepoURL, dockerSecure = BuildDockerRepoURL(dockerRepoURL)

		if dockerRegistry = ic.GetDockerRegistryByURL(dockerRepoURL); dockerRegistry == nil {
			dockerRegistry = NewDockerRegistry(dockerRepoURL, dockerRepoUser, dockerRepoPassword)
			dockerRegistry.sysconfig = false
			dockerRegistry.Secure = new(bool)
			*dockerRegistry.Secure = dockerSecure
		} else {
			var oregistry *DockerRegistry = new(DockerRegistry)
			*oregistry = *dockerRegistry
			dockerRegistry, oregistry = oregistry, dockerRegistry
		}

		dockerRegistry.Username = dockerRepoUser
		dockerRegistry.Password = dockerRepoPassword

		var (
			registryPrefix string = dockerRegistry.GetDockerRepoURL()
			registryURL    string
		)
		if dockerRegistry.IsSecure() {
			registryURL = "https://" + registryPrefix
		} else {
			registryURL = "http://" + registryPrefix
		}

		var (
			build_cmd string
			cmdpath   string
			cmdargs   []string
		)

		build_cmd = "docker"
		cmdargs = []string{
			build_cmd,
			"-v",
		}

		cmdpath, cmdargs = parseCommand(cmdargs)
		buildcmd := exec.CommandContext(context.Background(), cmdpath, cmdargs...)
		if err = buildcmd.Run(); err != nil {
			build_cmd = "podman"
		}

		if docker.IsDockerRunning() {
			if _, err = docker.LoginRegistry(dc, registryURL, dockerRepoUser, dockerRepoPassword); err != nil {
				s.ErrorExit("[%s] %s login %s error: %s", dockerRepoUser, build_cmd, registryURL, err)
				return
			}
			dockerLoginCmd := exec.CommandContext(
				context.Background(),
				build_cmd,
				"login",
				registryURL,
				"--username",
				dockerRepoUser,
				"--password",
				dockerRepoPassword,
			)
			if err = dockerLoginCmd.Run(); err != nil {
				s.ErrorExit("[%s] %s login %s error: %s", dockerRepoUser, build_cmd, registryURL, err)
			}
		} else {
			docker_req, err := http.NewRequest("GET", registryURL+"/v2/", nil)
			if err != nil {
				s.ErrorExit("[%s] %s login %s error: %s", dockerRepoUser, build_cmd, registryURL, err)
			}
			// 用户密码
			docker_req.URL.User = url.UserPassword(dockerRepoUser, dockerRepoPassword)
			// 登录Docker
			docker_rep, err := http.DefaultClient.Do(docker_req)
			if err != nil {
				s.ErrorExit("[%s] %s login %s error: %s", dockerRepoUser, build_cmd, registryURL, err)
			}
			defer docker_rep.Body.Close()
			if docker_rep.StatusCode >= 400 {
				s.ErrorExit("[%s] %s login %s error: %s", dockerRepoUser, build_cmd, registryURL, "invalid user or password")
			}
		}

		ic.Snz1dp.Registry.URL = registryPrefix

		dockerRegistry.URL = dockerRepoURL
		dockerRegistry.Username = dockerRepoUser
		dockerRegistry.Password = dockerRepoPassword

		s.Println("[%s] %s login %s success!", dockerRepoUser, build_cmd, registryPrefix)
	}

	if helmLogin {
		if helmRegistry = ic.GetHelmRegistryByURL(helmRepoURL); helmRegistry == nil {
			helmRegistry = &HelmRegistry{
				Name:     helmRepoName,
				URL:      helmRepoURL,
				Username: helmRepoUser,
				Password: helmRepoPassword,
			}
			helmRegistry.sysconfig = false
		} else {
			var oregistry *HelmRegistry = new(HelmRegistry)
			*oregistry = *helmRegistry
			helmRegistry, oregistry = oregistry, helmRegistry
		}

		if helmRepoName == ic.Snz1dp.HelmRepo.Name {
			ic.Snz1dp.HelmRepo.URL = helmRepoURL
			ic.Snz1dp.HelmRepo.Name = helmRepoName
			saveConfig = true
		}

		oldHelmRepoName := helmRegistry.Name
		helmRegistry.Name = helmRepoName
		helmRegistry.Username = helmRepoUser
		helmRegistry.Password = helmRepoPassword

		// 初始化Helm操作配置
		InitHelmActionConfig(setting)

		c := helmRepo.Entry{
			URL:      helmRepoURL,
			Username: helmRepoUser,
			Password: helmRepoPassword,
			CertFile: "",
			KeyFile:  "",
			CAFile:   "",
		}

		var chartRepo *helmRepo.ChartRepository
		chartRepo, err = helmRepo.NewChartRepository(&c, helmGetter.All(setting.helmSetting))
		_, err = chartRepo.DownloadIndexFile()
		if err != nil {
			s.ErrorExit("[%s] helm login %s(%s) error: %s", helmRepoUser, helmRepoName, helmRepoURL, err)
			return
		}

		helmBinPath, err := resolveHelmBinPath(s, ic)
		if err != nil {
			s.ErrorExit("%s", err)
		}

		helmRepoRemove := exec.CommandContext(
			context.Background(),
			helmBinPath,
			"repo", "remove",
			oldHelmRepoName,
		)
		helmRepoRemove.Run()

		helmRepoAdd := exec.CommandContext(
			context.Background(),
			helmBinPath,
			"repo", "add",
			helmRepoName,
			helmRepoURL,
			"--username",
			helmRepoUser,
			"--password",
			helmRepoPassword,
		)
		var testBuffer bytes.Buffer
		helmRepoAdd.Stdout = &testBuffer
		if err = helmRepoAdd.Run(); err != nil {
			s.ErrorExit("helm repo add error: %s", err)
		}

		s.Println("[%s] helm login %s(%s) success!", helmRepoUser, helmRepoName, helmRepoURL)
	}

	if mavenLogin {
		maven_req, err := http.NewRequest("GET", mavenRepoURL, nil)
		if err != nil {
			s.ErrorExit("[%s] maven login %s(%s) error: %s", mavenRepoUser, mavenRepoID, mavenRepoURL, err)
		}
		// 用户密码
		maven_req.URL.User = url.UserPassword(mavenRepoUser, mavenRepoPassword)
		// 登录Maven
		maven_rep, err := http.DefaultClient.Do(maven_req)
		if err != nil {
			s.ErrorExit("[%s] maven login %s(%s) error: %s", mavenRepoUser, mavenRepoID, mavenRepoURL, err)
		}
		defer maven_rep.Body.Close()
		if maven_rep.StatusCode >= 400 {
			s.ErrorExit("[%s] maven login %s(%s) error: %s", mavenRepoUser, mavenRepoID, mavenRepoURL, "invalid user or password")
		}
		s.Println("[%s] maven login %s(%s) success!", mavenRepoUser, mavenRepoID, mavenRepoURL)
	}

	// 获取NVM命令路径
	nrmPaths, err := resolveNrmPaths(s)
	if err != nil {
		s.ErrorExit("%s", err)
	}

	// 添加npm仓库
	if npmRepoID == "npm" || npmRepoURL == "" || npmRepoID == "node-sass-site" {
		npmRepoURL = "https://registry.npmjs.org/"
		npmLogin = false
	} else if npmRepoID == "yarn" {
		npmRepoURL = "https://registry.yarnpkg.com/"
		npmLogin = false
	} else if npmRepoID == "tencent" {
		npmRepoURL = "https://mirrors.cloud.tencent.com/npm/"
		npmLogin = false
	} else if npmRepoID == "cnpm" {
		npmRepoURL = "https://r.cnpmjs.org/"
		npmLogin = false
	} else if npmRepoID == "taobao" {
		npmRepoURL = "https://registry.npmmirror.com/"
		npmLogin = false
	} else if npmRepoID == "npmMirror" {
		npmRepoURL = "https://skimdb.npmjs.com/registry/"
		npmLogin = false
	} else {
		usename := npmRepoUser
		if usename == "" {
			usename = "anonymous"
		}

		nrmRmCmd := nrmPaths[0]
		nrmRmArgs := nrmPaths[1:]
		nrmRmArgs = append(
			nrmRmArgs,
			"del",
			npmRepoID,
		)

		nrmRmRepo := exec.CommandContext(
			context.Background(),
			nrmRmCmd, nrmRmArgs...,
		)
		nrmRmRepo.Run()

		nrmCmd := nrmPaths[0]
		nrmArgs := nrmPaths[1:]
		nrmArgs = append(
			nrmArgs,
			"add",
			npmRepoID,
			npmRepoURL,
		)

		nrmAddRepo := exec.CommandContext(
			context.Background(),
			nrmCmd, nrmArgs...,
		)

		if err = nrmAddRepo.Run(); err != nil {
			s.ErrorExit("[%s] nrm add %s(%s) error: %s", usename, npmRepoID, npmRepoURL, err)
		}

		s.Println("[%s] nrm add %s(%s) success!", usename, npmRepoID, npmRepoURL)
	}

	// 添加Sass站点
	if sassSiteID != "" && sassSiteURL != "" {
		usename := sassSiteUser
		if usename == "" {
			usename = "anonymous"
		}

		// 添加sass_binary_site配置
		npmPaths, err := resolveNpmPaths(s)
		if err != nil {
			s.ErrorExit("%s", err)
		}
		npmCmd := npmPaths[0]
		npmArgs := npmPaths[1:]
		npmArgs = append(npmArgs,
			"config",
			"set",
			"sass_binary_site",
			sassSiteURL,
		)
		var stdOutBuffer bytes.Buffer
		var errOutBuffer bytes.Buffer

		npmConfigSet := exec.CommandContext(
			context.Background(),
			npmCmd, npmArgs...,
		)
		npmConfigSet.Stdout = &stdOutBuffer
		npmConfigSet.Stderr = &errOutBuffer
		if err = npmConfigSet.Run(); err != nil {
			if !strings.Contains(errOutBuffer.String(), "not a valid npm option") {
				s.ErrorExit("[%s] npm config set sass_binary_site(%s) error: %s", usename, sassSiteURL, errOutBuffer.String())
				return err
			}
		} else {
			s.Println("[%s] npm config set sass_binary_site(%s) success!", usename, sassSiteURL)
		}

		// 删除nrm仓库
		nrmRmCmd := nrmPaths[0]
		nrmRmArgs := nrmPaths[1:]
		nrmRmArgs = append(
			nrmRmArgs,
			"del",
			sassSiteID,
		)

		nrmRmRepo := exec.CommandContext(
			context.Background(),
			nrmRmCmd, nrmRmArgs...,
		)
		nrmRmRepo.Run()

		// 添加nrm仓库
		nrmCmd := nrmPaths[0]
		nrmArgs := nrmPaths[1:]
		nrmArgs = append(
			nrmArgs,
			"add",
			sassSiteID,
			sassSiteURL,
		)

		nrmAddRepo := exec.CommandContext(
			context.Background(),
			nrmCmd, nrmArgs...,
		)

		if err = nrmAddRepo.Run(); err != nil {
			s.ErrorExit("[%s] nrm add %s(%s) error: %s", usename, sassSiteID, sassSiteURL, err)
		}
		s.Println("[%s] nrm add %s(%s) success!", usename, sassSiteID, sassSiteURL)
	}

	// 登录NPM仓库
	if npmLogin {
		npm_req, err := http.NewRequest("GET", npmRepoURL, nil)
		if err != nil {
			s.ErrorExit("[%s] npm login %s(%s) error: %s", npmRepoUser, npmRepoID, npmRepoURL, err)
		}
		// 用户密码
		npm_req.URL.User = url.UserPassword(npmRepoUser, npmRepoPassword)
		// 登录Maven
		npm_rep, err := http.DefaultClient.Do(npm_req)
		if err != nil {
			s.ErrorExit("[%s] npm login %s(%s) error: %s", npmRepoUser, npmRepoID, npmRepoURL, err)
		}
		defer npm_rep.Body.Close()
		if npm_rep.StatusCode >= 400 {
			s.ErrorExit("[%s] npm login %s(%s) error: %s", npmRepoUser, npmRepoID, npmRepoURL, "invalid user or password")
		}

		nrmCmd := nrmPaths[0]
		nrmArgs := nrmPaths[1:]
		nrmArgs = append(
			nrmArgs,
			"login",
			npmRepoID,
			"--username",
			npmRepoUser,
			"--password",
			npmRepoPassword,
		)

		nrmLoginRepo := exec.CommandContext(
			context.Background(),
			nrmCmd, nrmArgs...,
		)

		var stdOutBuffer bytes.Buffer
		nrmLoginRepo.Stdout = &stdOutBuffer
		if err = nrmLoginRepo.Run(); err != nil {
			s.ErrorExit("[%s] nrm login %s(%s) error: %s", npmRepoUser, npmRepoID, npmRepoURL, err)
		}

		nrmStdout := stdOutBuffer.String()
		if strings.Index(nrmStdout, npmRepoID+" success") <= 0 {
			s.ErrorExit("[%s] nrm login %s(%s) error", npmRepoUser, npmRepoID, npmRepoURL)
		}
		stdOutBuffer.Reset()

		nrmArgs = nrmPaths[1:]
		nrmArgs = append(
			nrmArgs,
			"use",
			npmRepoID,
		)
		nrmUseRepo := exec.CommandContext(
			context.Background(),
			nrmCmd, nrmArgs...,
		)
		nrmUseRepo.Stdout = &stdOutBuffer
		if err = nrmUseRepo.Run(); err != nil {
			s.ErrorExit("[%s] nrm use %s(%s) error: %s", npmRepoUser, npmRepoID, npmRepoURL, stdOutBuffer.String())
		}
		s.Println("[%s] nrm login %s(%s) success!", npmRepoUser, npmRepoID, npmRepoURL)
	} else {
		var stdOutBuffer bytes.Buffer
		nrmCmd := nrmPaths[0]
		nrmArgs := nrmPaths[1:]
		nrmArgs = append(
			nrmArgs,
			"use",
			npmRepoID,
		)
		nrmUseRepo := exec.CommandContext(
			context.Background(),
			nrmCmd, nrmArgs...,
		)
		nrmUseRepo.Stdout = &stdOutBuffer
		if err = nrmUseRepo.Run(); err != nil {
			s.ErrorExit("[%s] nrm use %s(%s) error: %s", npmRepoUser, npmRepoID, npmRepoURL, stdOutBuffer.String())
		}
		s.Println("[%s] nrm use %s(%s) success!", npmRepoUser, npmRepoID, npmRepoURL)
	}

	if sassSiteLogin {
		npm_req, err := http.NewRequest("GET", sassSiteURL, nil)
		if err != nil {
			s.ErrorExit("[%s] nrm login %s(%s) error: %s", sassSiteUser, sassSiteID, sassSiteURL, err)
		}
		// 用户密码
		npm_req.URL.User = url.UserPassword(sassSiteUser, sassSitePassword)
		// 登录Maven
		npm_rep, err := http.DefaultClient.Do(npm_req)
		if err != nil {
			s.ErrorExit("[%s] nrm login %s(%s) error: %s", sassSiteUser, sassSiteID, sassSiteURL, err)
		}
		defer npm_rep.Body.Close()
		if npm_rep.StatusCode >= 400 {
			s.ErrorExit("[%s] nrm login %s(%s) error: %s", sassSiteUser, sassSiteID, sassSiteURL, "invalid user or password")
		}

		nrmCmd := nrmPaths[0]
		nrmArgs := nrmPaths[1:]
		nrmArgs = append(
			nrmArgs,
			"login",
			sassSiteID,
			"--username",
			sassSiteUser,
			"--password",
			sassSitePassword,
		)

		nrmLoginRepo := exec.CommandContext(
			context.Background(),
			nrmCmd, nrmArgs...,
		)

		var stdOutBuffer bytes.Buffer
		nrmLoginRepo.Stdout = &stdOutBuffer
		if err = nrmLoginRepo.Run(); err != nil {
			s.ErrorExit("[%s] nrm login %s(%s) error: %s", sassSiteUser, sassSiteID, sassSiteURL, err)
		}

		nrmStdout := stdOutBuffer.String()
		if strings.Index(nrmStdout, sassSiteID+" success") <= 0 {
			s.ErrorExit("[%s] nrm login %s(%s) error", sassSiteUser, sassSiteID, sassSiteURL)
		}
		s.Println("[%s] nrm login %s(%s) success!", sassSiteUser, sassSiteID, sassSiteURL)
	}

	spinner = utils.NewSpinner("save login credentials...", setting.OutOrStdout())

	saveDocker := false
	saveHelm := false
	saveMaven := false
	saveNpm := false
	saveSite := false

	if dockerLogin {
		if saveDocker, err = ic.ApplyDockerRegistry(*dockerRegistry); err != nil {
			spinner.Close()
			s.ErrorExit("%s", err)
		}
	}

	if helmLogin {
		if saveHelm, err = ic.ApplyHelmRegistry(*helmRegistry); err != nil {
			spinner.Close()
			s.Print("failed: ")
			s.ErrorExit("%s", err)
		}
	}

	if mavenLogin {
		if saveMaven, err = ic.ApplyMavenSettings(*mavenServerItem); err != nil {
			spinner.Close()
			s.Print("failed: ")
			s.ErrorExit("%s", err)
		}
	}

	if saveNpm, err = ic.ApplyNpmSettings(*npmServerItem); err != nil {
		spinner.Close()
		s.Print("failed: ")
		s.ErrorExit("%s", err)
	}

	if saveSite, err = ic.ApplySassSiteSettings(*sassSiteItem); err != nil {
		spinner.Close()
		s.Print("failed: ")
		s.ErrorExit("%s", err)
	}

	if saveConfig || saveDocker || saveHelm || saveMaven || saveNpm || saveSite {
		if err = setting.SaveLocalInstallConfiguration(ic, configFilePath); err != nil {
			spinner.Close()
			s.Print("failed: ")
			s.ErrorExit("save %s error: %s", configFilePath, err)
			return
		}
	}

	spinner.Close()
	s.Print("ok!")

	return
}

// InstallBundle -
type InstallBundle struct {
	BaseAction
	Namespace     string
	From          string
	Name          string
	Overlay       bool
	Envs          []string
	HostAlias     []string
	Bind          []string
	RunFiles      []string
	PortDisabled  bool
	Volume        []string
	Command       []string
	Runtime       string
	DockerImage   string
	GPU           string
	HealthCommand []string
	HealthURL     string
}

// NewInstallBundle -
func NewInstallBundle(setting *GlobalSetting) *InstallBundle {
	return &InstallBundle{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (i *InstallBundle) Run() (err error) {
	var (
		currentDir     string
		setting        *GlobalSetting = i.GlobalSetting()
		configFilePath string
		ic             *InstallConfiguration
	)

	if currentDir, err = os.Getwd(); err != nil {
		i.ErrorExit("get cwd error: %s", err)
		return
	}

	//加载本地安装配置
	configFilePath, ic, err = setting.LoadLocalInstallConfiguration()
	if err != nil {
		i.ErrorExit("load %s error: %v", configFilePath, err)
		return
	}

	if err = i.installComponent(currentDir, ic, i.From); err != nil {
		i.ErrorExit("install %s error: %s", i.From, err)
		return
	}

	if err = setting.SaveLocalInstallConfiguration(ic, configFilePath); err != nil {
		i.ErrorExit("save %s error: %s", configFilePath, err)
		return
	}

	i.Println("")
	return NewListBundle(i.setting).Run()

}

// Run -
func (i *InstallBundle) installComponent(currentDir string, ic *InstallConfiguration, fromURL string) (err error) {

	var (
		setting                        *GlobalSetting = ic.GlobalSetting()
		furl                           *url.URL
		bundleFilepath                 string
		spinner                        *utils.WaitSpinner
		tmpBundleDir                   string
		bundleFileName                 string
		bundleFileBase                 string
		installComp                    ExtrasCompoentConfig
		valuesFile, runFile, chartFile string
		stconf                         StandaloneConfig
		chartContext, sumData          []byte
		chartConfig                    ChartConfig
		installBundleFile              string
		bundleMd5File                  string
		oldComponent                   Component
		fst                            os.FileInfo
	)

	// 查看是不是名称定义的组件
	if oldComponent, err = ic.GetBundleComponent(fromURL); err != nil {
		return
	}
	if oldComponent == nil && i.Name != "" {
		if oldComponent, err = ic.GetBundleComponent(i.Name); err != nil {
			return
		}
	}

	if oldComponent != nil {
		if oldComponent.BeInstall() && !i.Overlay {
			err = errors.Errorf("%s@%s bundle already install enabled!", oldComponent.GetName(), oldComponent.GetVersion())
		} else if !oldComponent.BeInstall() {
			oldComponent.SetInstall()
			i.Println("%s@%s bundle install enabled!", oldComponent.GetName(), oldComponent.GetVersion())
		}
		if !i.Overlay {
			return
		}
	}

reget:

	if furl, err = url.Parse(fromURL); err != nil {
		err = errors.Errorf("invalid bundle url: %s", fromURL)
		return
	}

	if tmpBundleDir, err = os.MkdirTemp("", ".snz1dp.*"); err != nil {
		err = errors.Errorf("create temp directory error: %v", err)
		return
	}

	defer os.RemoveAll(tmpBundleDir)

	if furl.Scheme == "" || furl.Scheme == "file" {
		if furl.Scheme == "file" || furl.Scheme == "" && (!strings.Contains(furl.Path, "@") || strings.HasSuffix(furl.Path, ".tgz") || strings.HasSuffix(furl.Path, ".tar.gz")) {
			bundleFilepath = furl.Path
			if !filepath.IsAbs(bundleFilepath) {
				bundleFilepath = filepath.Join(currentDir, bundleFilepath)
				if bundleFilepath, err = filepath.Abs(bundleFilepath); err != nil {
					err = errors.Errorf("error snz1dp bundle path: %v", err)
					return
				}
			}
			installComp.URL = bundleFilepath
			bundleFileName = filepath.Base(bundleFilepath)
			if fst, err = os.Stat(bundleFilepath); err != nil || fst.IsDir() {
				err = errors.Errorf("bundle not existed")
				return
			}
		} else {
			var componentAndVersion []string = strings.Split(fromURL, "@")
			var installChartURL string = ic.Snz1dp.HelmRepo.URL
			if installChartURL[len(installChartURL)-1] != '/' {
				installChartURL += "/"
			}
			if componentAndVersion[0] == "" || componentAndVersion[1] == "" {
				err = errors.Errorf("bundle or version error")
				return
			}
			installChartURL += componentAndVersion[0] + "-" + componentAndVersion[1] + ".tgz"
			fromURL = installChartURL
			goto reget
		}
	} else {
		if err = InitHelmActionConfig(ic.setting); err != nil {
			err = errors.Errorf("init helm config error: %s", err)
			return
		}
		var (
			helmRegistry *HelmRegistry
			bundleURL    string = furl.String()
			idxLastPath  int    = strings.LastIndex(bundleURL, "/")
			chartDigest  string
			chartName    string
			chartVersion string
		)

		// 先分析一下是不是Chart仓库
		if idxLastPath > 0 {
			var charRepo = bundleURL[0:idxLastPath]
			helmRegistry = ic.GetHelmRegistryByURL(charRepo)
		}

		if helmRegistry != nil {
			chartName = bundleURL[idxLastPath+1:]
			idxLastPath = strings.LastIndex(chartName, ".tgz")
			if idxLastPath > 0 {
				chartName = chartName[0:idxLastPath]
				idxLastPath = strings.LastIndex(chartName, "-")
				if idxLastPath > 0 {
					chartVersion = chartName[idxLastPath+1:]
					chartName = chartName[0:idxLastPath]
					bundleURL, chartDigest, err = helmRegistry.ResolveChartURLAndDigest(chartName, chartVersion, helmGetter.All(ic.setting.helmSetting))
					if err != nil {
						err = errors.Errorf("load helm chart error: %s", err)
						return
					}
					if furl, err = url.Parse(bundleURL); err != nil {
						err = errors.Errorf("invalid chart url: %s", bundleURL)
						return
					}
				} else {
					helmRegistry = nil
				}
			} else {
				helmRegistry = nil
			}
		}

		bundleFileName = filepath.Base(furl.Path)
		spinner = utils.NewSpinner(fmt.Sprintf("download %s...", bundleURL), setting.OutOrStdout())
		var downloader *down.BundleDownloader

		if helmRegistry != nil {
			downloader = down.NewBundleDownloader(setting.OutOrStdout(), bundleURL, down.VerifyAlways,
				down.WithBasicAuth(helmRegistry.Username, helmRegistry.Password))
			downloader.SetFileDigest(chartDigest)
		} else {
			downloader = down.NewBundleDownloader(setting.OutOrStdout(), bundleURL, down.VerifyAlways)
		}

		bundleFilepath, err = downloader.Download(tmpBundleDir, bundleFileName)
		spinner.Close()
		if err != nil {
			i.Println("failed: %s", err.Error())
			return
		}
		i.Println("ok!")
		fromURL = furl.String()
		if strings.Index(fromURL, ic.Snz1dp.HelmRepo.URL) != 0 {
			installComp.URL = furl.String()
		}
	}

	bundleFileBase = bundleFileName[:len(bundleFileName)-len(filepath.Ext(bundleFileName))]

	valuesFile = fmt.Sprintf("%s/values.yaml", bundleFileBase)
	runFile = fmt.Sprintf("%s/RUN.yaml", bundleFileBase)
	chartFile = fmt.Sprintf("%s/Chart.yaml", bundleFileBase)
	compExtractDir := path.Join(tmpBundleDir, bundleFileBase)

	if err = ExtractBundleFile(bundleFilepath, valuesFile, tmpBundleDir); err == nil || strings.Contains(err.Error(), "file already exists") {
		var valuesdata []byte
		if valuesdata, err = os.ReadFile(path.Join(compExtractDir, "values.yaml")); err != nil {
			err = errors.Errorf("error format snz1dp bundle file: no values.yaml defined!")
			return
		}
		installComp.SetValuesConfig(string(valuesdata))
	} else {
		err = errors.Errorf("error format snz1dp bundle file: no values.yaml defined!")
		return
	}

	if err = ExtractBundleFile(bundleFilepath, chartFile, tmpBundleDir); err == nil || strings.Contains(err.Error(), "file already exists") {
		if chartContext, err = os.ReadFile(path.Join(compExtractDir, "Chart.yaml")); err != nil {
			err = errors.Errorf("error format snz1dp bundle file: no Chart.yaml defined!")
			return err
		}
	} else {
		err = errors.Errorf("error format snz1dp bundle file: no Chart.yaml defined!")
		return
	}

	if err = ExtractBundleFile(bundleFilepath, runFile, tmpBundleDir); err == nil || strings.Contains(err.Error(), "file already exists") {
		var rundata []byte
		if rundata, err = os.ReadFile(path.Join(compExtractDir, "RUN.yaml")); err != nil {
			err = errors.Errorf("error format snz1dp bundle file: no RUN.yaml defined!")
			return
		}
		installComp.SetStandalone(string(rundata))
	} else {
		err = errors.Errorf("error format snz1dp bundle file: no RUN.yaml defined!")
		return
	}

	if err = yaml.Unmarshal(chartContext, &chartConfig); err != nil {
		err = errors.Errorf("error format snz1dp bundle file, error Chart.yaml: %s", err)
		return
	}

	if err = yaml.Unmarshal([]byte(installComp.GetStandalone()), &stconf); err != nil {
		err = errors.Errorf("error format snz1dp bundle file, error RUN.yaml: %s", err)
		return
	}

	if oldComponent, err = ic.GetBundleComponent(chartConfig.Name); err != nil {
		err = errors.Errorf("load %s bundle error: %s", chartConfig.Name, err)
		return
	}

	if i.Name != "" && i.Name != chartConfig.Name {
		if oldComponent, err = ic.GetBundleComponent(i.Name); err != nil {
			err = errors.Errorf("load %s bundle error: %s", i.Name, err)
			return
		}
	}

	if oldComponent != nil && oldComponent.BeInstall() && utils.CompareStrVer(oldComponent.GetVersion(), chartConfig.AppVersion) >= 0 && !i.Overlay {
		err = errors.Errorf("%s@%s bundle already install enabled!", oldComponent.GetName(), oldComponent.GetVersion())
		return
	} else if oldComponent != nil && !oldComponent.IsExtras() { // 内部组件
		oldComponent.SetInstall()
		oldComponent.SetConfigValues(installComp.GetValuesConfig(), installComp.GetStandalone())
		oldComponent.SetEnvironments(i.Envs)
		oldComponent.SetExtrasHosts(i.HostAlias)
		oldComponent.SetBindPortEnable(!i.PortDisabled)
		oldComponent.SetVolumes(i.Volume)
		oldComponent.SetBindPorts(i.Bind)
		oldComponent.SetCommand(i.Command)
		oldComponent.SetGPU(i.GPU)
		oldComponent.SetRuntime(i.Runtime)
		oldComponent.SetDockerImage(i.DockerImage)

		if len(i.HealthCommand) > 0 || i.HealthURL != "" {
			healthCheck := installComp.GetHealthcheck()
			if len(i.HealthCommand) > 0 {
				healthCheck.Test = i.HealthCommand
			}
			if i.HealthURL != "" {
				healthCheck.URL = i.HealthURL
			}
			oldComponent.SetHealthcheck(healthCheck)
		}

		if len(i.RunFiles) > 0 {
			fromCommandFiles := map[string]string{}
			for _, runfile := range i.RunFiles {
				fileNameIdx := strings.Index(runfile, "=")
				if fileNameIdx > 0 {
					runfileName := runfile[:fileNameIdx]
					runfileContent := runfile[fileNameIdx+1:]
					var (
						fdata   []byte
						fbase64 string
					)
					if fdata, err = os.ReadFile(runfileContent); err != nil {
						err = errors.Errorf("read file '%s' error: %s", runfileContent, err)
						return
					}
					var b64 = base64.StdEncoding
					fbase64 = b64.EncodeToString(fdata)
					fromCommandFiles[runfileName] = "base64://" + fbase64
				}
			}
			oldComponent.SetRunFiles(fromCommandFiles)
		}

		if oldComponent.GetVersion() != chartConfig.AppVersion {
			oldversion := oldComponent.GetVersion()
			oldComponent.SetVersion(chartConfig.AppVersion)
			i.Println("%s bundle upgrade %s to %s!", oldComponent.GetName(), oldversion, chartConfig.AppVersion)
		} else {
			i.Println("%s@%s bundle upgrade config!", oldComponent.GetName(), oldComponent.GetVersion())
		}
		return
	}

	installComp.Name = chartConfig.Name
	installComp.Version = chartConfig.AppVersion
	installComp.Namespace = i.Namespace
	installComp.Install = true

	installBundleFile = filepath.Join(setting.GetBundleDir(), fmt.Sprintf("%s-%s.tgz", installComp.Name, installComp.Version))
	bundleMd5File = filepath.Join(setting.GetBundleDir(), fmt.Sprintf("%s-%s.tgz.sha256", installComp.Name, installComp.Version))

	if bundleFilepath != installBundleFile {
		if err = utils.CopyFile(bundleFilepath, installBundleFile); err != nil {
			err = errors.Errorf("save %s error: %s", installBundleFile, err)
			return
		}
	}

	if i.Name == "" || i.Name == chartConfig.Name {
		installComp.URL = ""
	} else if i.Name != "" && i.Name != chartConfig.Name {
		installComp.Name = i.Name
		installComp.URL = fromURL
	}

	if sumData, err = utils.FileChecksum(installBundleFile, sha256.New()); err != nil {
		err = errors.Errorf("sha256sum %s error: %s", installBundleFile, err)
		return
	}

	// 写入SHA256校验
	if err = os.WriteFile(bundleMd5File, []byte(fmt.Sprintf("%s %s", hex.EncodeToString(sumData), filepath.Base(installBundleFile))), 0644); err != nil {
		err = errors.Errorf("save %s error: %s", bundleMd5File, err)
		return
	}

	installComp.SetConfigValues(installComp.GetValuesConfig(), installComp.GetStandalone())
	installComp.SetEnvironments(i.Envs)
	installComp.SetExtrasHosts(i.HostAlias)
	installComp.SetVolumes(i.Volume)
	installComp.SetBindPortEnable(!i.PortDisabled)
	installComp.SetBindPorts(i.Bind)
	installComp.SetCommand(i.Command)
	installComp.SetGPU(i.GPU)
	installComp.SetRuntime(i.Runtime)
	installComp.SetDockerImage(i.DockerImage)

	if len(i.RunFiles) > 0 {
		fromCommandFiles := map[string]string{}
		for _, runfile := range i.RunFiles {
			fileNameIdx := strings.Index(runfile, "=")
			if fileNameIdx > 0 {
				runfileName := runfile[:fileNameIdx]
				runfileContent := runfile[fileNameIdx+1:]
				var (
					fdata   []byte
					fbase64 string
				)
				if fdata, err = os.ReadFile(runfileContent); err != nil {
					err = errors.Errorf("read file '%s' error: %s", runfileContent, err)
					return
				}
				var b64 = base64.StdEncoding
				fbase64 = b64.EncodeToString(fdata)
				fromCommandFiles[runfileName] = "base64://" + fbase64
			}
		}
		installComp.SetRunFiles(fromCommandFiles)
	}

	if len(i.HealthCommand) > 0 || i.HealthURL != "" {
		healthCheck := installComp.GetHealthcheck()
		if len(i.HealthCommand) > 0 {
			healthCheck.Test = i.HealthCommand
		}
		if i.HealthURL != "" {
			healthCheck.URL = i.HealthURL
		}
		installComp.SetHealthcheck(healthCheck)
	}

	if err = ic.ApplyExtrasComponent(installComp); err != nil {
		return
	}

	i.Println("")

	return
}

// RemoveBundle -
type RemoveBundle struct {
	BaseAction
	Name []string
}

// NewRemoveBundle -
func NewRemoveBundle(setting *GlobalSetting) *RemoveBundle {
	return &RemoveBundle{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (r *RemoveBundle) Run() (err error) {
	var (
		setting        *GlobalSetting = r.GlobalSetting()
		configFilePath string
		ic             *InstallConfiguration
		removed        bool
	)

	//加载本地安装配置
	configFilePath, ic, err = setting.LoadLocalInstallConfiguration()
	if err != nil {
		r.ErrorExit("load %s error: %s", configFilePath, err)
		return err
	}

	for _, v := range r.Name {
		if err = r.removeComponent(ic, v); err != nil {
			r.Println("%s", err)
			err = nil
		} else if !removed {
			removed = true
		}
	}

	if err = setting.SaveLocalInstallConfiguration(ic, configFilePath); err != nil {
		r.ErrorExit("save %s error: %s", configFilePath, err)
		return
	}

	r.Println("")
	return NewListBundle(r.setting).Run()

}

// Run -
func (r *RemoveBundle) removeComponent(ic *InstallConfiguration, compoName string) (err error) {

	var (
		installComp Component
	)

	if installComp, err = ic.GetBundleComponent(compoName); err != nil {
		r.ErrorExit("load %s bundle error: %s", compoName, err)
		return
	}

	if installComp == nil {
		err = errors.Errorf("not found %s bundle!", compoName)
		return
	}

	if !installComp.BeInstall() {
		err = errors.Errorf("%s@%s bundle already install disabled!", installComp.GetName(), installComp.GetVersion())
		return
	}

	installComp.UnInstall()
	r.Println("%s@%s bundle install disabled!", installComp.GetName(), installComp.GetVersion())
	return
}

// ListBundle -
type ListBundle struct {
	BaseAction
}

// NewListBundle -
func NewListBundle(setting *GlobalSetting) *ListBundle {
	return &ListBundle{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (l *ListBundle) Run() (err error) {

	var (
		setting    *GlobalSetting = l.GlobalSetting()
		compNames  []string
		components map[string]Component
	)

	//加载本地安装配置
	configFilePath, ic, err := setting.LoadLocalInstallConfiguration()
	if err != nil {
		l.ErrorExit("load %s error: %s", configFilePath, err)
		return err
	}

	if compNames, components, err = ic.GetBundleComponents(false); err != nil {
		l.ErrorExit("load bundle error: %s", err)
		return
	}

	for _, k := range compNames {
		comp := components[k]
		instState := "✔"
		instTitle := "install enabled"
		if !comp.BeInstall() {
			instState = "✖"
			instTitle = "install disabled"
		}
		switch comp.(type) {
		case *ExtrasCompoentConfig:
			l.Println("%s [apps] %s@%s %s", instState, comp.GetName(), comp.GetVersion(), instTitle)
		default:
			l.Println("%s [core] %s@%s %s", instState, comp.GetName(), comp.GetVersion(), instTitle)
		}
	}

	l.Println("")

	return
}

// ExportProfile 导出配置
type ExportProfile struct {
	BaseAction
	OutputFile     string
	OutputPassword string
	PlainPassword  bool
	DetailConfig   bool
	LoginConfig    bool
}

// NewExportProfile 新建
func NewExportProfile(setting *GlobalSetting) *ExportProfile {
	return &ExportProfile{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run 运行
func (e *ExportProfile) Run() (err error) {
	var (
		ic         *InstallConfiguration
		setting    *GlobalSetting = e.GlobalSetting()
		icData     []byte
		encodePass bool
	)

	// 安装配置文件
	if _, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		e.ErrorExit("load %s error: %s", err)
		return
	}

	if e.PlainPassword {
		encodePass = false
	} else {
		encodePass = true
	}

	ic.GetBundleComponents(false)

	if icData, err = ic.Export(encodePass, e.DetailConfig, e.LoginConfig, e.OutputPassword); err != nil {
		e.ErrorExit("export config error: %s", err)
		return
	}

	if e.OutputFile == "" {
		e.Println("%s", string(icData))
		return
	}

	if err = os.WriteFile(e.OutputFile, icData, 0644); err != nil {
		e.ErrorExit("write %s error: %s", e.OutputFile, err)
		return
	}

	e.Println("export to %s ok!", e.OutputFile)

	return
}

// 导出JWT
type JwtExport struct {
	BaseAction
}

// NewJwtExport 新建
func NewJwtExport(setting *GlobalSetting) *JwtExport {
	return &JwtExport{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run 运行
func (e *JwtExport) Run() (err error) {
	var (
		ic      *InstallConfiguration
		setting *GlobalSetting = e.GlobalSetting()
		ingress Component
	)

	// 安装配置文件
	if _, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		e.ErrorExit("load config error: %s", err)
		return
	}

	ic.GetBundleComponents(false)
	if ingress, err = ic.GetBundleComponent("ingress"); err != nil {
		return
	}

	e.Println("jwt_token   = %s", ingress.GetJwtConfig().Token)
	e.Println("private_key = %s", ingress.GetJwtConfig().PrivateKey)

	return
}
