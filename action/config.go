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
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	rice "github.com/GeertJohan/go.rice"
	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/docker/client"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/wonderivan/logger"
	helmAction "helm.sh/helm/v3/pkg/action"
	helmCli "helm.sh/helm/v3/pkg/cli"
	helmGetter "helm.sh/helm/v3/pkg/getter"
	helmKube "helm.sh/helm/v3/pkg/kube"
	helmRepo "helm.sh/helm/v3/pkg/repo"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"snz1.cn/snz1dp/snz1dpctl/docker"
	"snz1.cn/snz1dp/snz1dpctl/down"
	"snz1.cn/snz1dp/snz1dpctl/storage"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

const (
	// DefaultSnz1dpNS 缺省的名字空间
	DefaultSnz1dpNS = "snz1dp-system"
	// DefaultAppNS 缺省的应用名字空间
	DefaultAppNS = "snz1dp-app"
	// DefaultStorageClass 缺省的持久化卷类别
	DefaultStorageClass = "hostpath"
	// Snz1dpRegistrySecretName -
	Snz1dpRegistrySecretName = "snz1dp-registry-secret"
)

// ComponentConfig -
type ComponentConfig struct {
	Standalone   *string            `json:"standalone,omitempty"`
	Kubernetes   *string            `json:"kubernetes,omitempty"`
	Envs         *[]string          `json:"envs,omitempty"`
	HostPortMode *string            `json:"port,omitempty"`  // 是否绑定绑定, disabled 表示不绑定
	HostPortBind *[]string          `json:"bind,omitempty"`  // 绑定配置，不设置则使用默认配置
	ExtraHosts   *[]string          `json:"hosts,omitempty"` // 配置的主机名
	Volumes      *[]string          `json:"volumes,omitempty"`
	Command      *[]string          `json:"command,omitempty"`
	GPU          *string            `json:"gpu,omitempty"`
	Runtime      *string            `json:"runtime,omitempty"`
	HealthCheck  *HealthCheckConfig `json:"healthcheck,omitempty"`
	DockerImage  *string            `json:"image,omitempty"`
	RunFiles     *map[string]string `json:"files,omitempty"`
}

// BaseVersionConfig - 版本
type BaseVersionConfig struct {
	Component `json:"-"`
	// 版本，omitempty表示为空时节点依然存在
	Version string `json:"version"`
	// 内置配置
	InlineConfig *ComponentConfig `json:"config,omitempty"`
	// JWT配置
	Jwt *JwtSecretConfig `json:"jwt,omitempty"`
	// 独立运行配置
	standalone string
	// K8s运行配置
	valuesConfig string
	// 镜像列表
	dockerImages []string
	// 安装
	Install bool `json:"install"`
	// 安装配置
	install *InstallConfiguration
	// 名称
	name string
	// 是否扩展
	extras bool
	// 是否在文件中
	infile bool
	// 获取URL
	bundleurl   string
	realname    string
	realversion string
}

// SetConfigValues -
func (b *BaseVersionConfig) SetConfigValues(k8s, st string) {
	b.InlineConfig = new(ComponentConfig)
	b.InlineConfig.Kubernetes = new(string)
	*b.InlineConfig.Kubernetes = k8s
	b.InlineConfig.Standalone = new(string)
	*b.InlineConfig.Standalone = st
}

// GetConfigValues -
func (b *BaseVersionConfig) GetConfigValues() (k8s, st string) {
	if b.InlineConfig == nil {
		return
	}
	if b.InlineConfig.Kubernetes != nil {
		k8s = *b.InlineConfig.Kubernetes
	}
	if b.InlineConfig.Standalone != nil {
		st = *b.InlineConfig.Standalone
	}
	return
}

// IsExtras 是否扩展
func (b *BaseVersionConfig) IsExtras() bool {
	return b.extras
}

// SetExtras 是否扩展
func (b *BaseVersionConfig) SetExtras(val bool) {
	b.extras = val
}

// ClearData 清理数据
func (b *BaseVersionConfig) ClearData() {
	if b == nil {
		return
	}
	if b.Jwt != nil && b.Jwt.inline {
		b.Jwt = nil
	}
	if b.InlineConfig != nil && b.InlineConfig.Kubernetes != nil && *b.InlineConfig.Kubernetes == b.valuesConfig {
		b.InlineConfig.Kubernetes = nil
	}
	if b.InlineConfig != nil && b.InlineConfig.Standalone != nil && *b.InlineConfig.Standalone == b.standalone {
		b.InlineConfig.Standalone = nil
	}
	if b.InlineConfig != nil && b.InlineConfig.Envs != nil && len(*b.InlineConfig.Envs) == 0 {
		b.InlineConfig.Envs = nil
	}
	if b.InlineConfig != nil && b.InlineConfig.ExtraHosts != nil && len(*b.InlineConfig.ExtraHosts) == 0 {
		b.InlineConfig.ExtraHosts = nil
	}
	if b.InlineConfig != nil && b.InlineConfig.Volumes != nil && len(*b.InlineConfig.Volumes) == 0 {
		b.InlineConfig.Volumes = nil
	}
	if b.InlineConfig != nil && b.InlineConfig.HostPortBind != nil && len(*b.InlineConfig.HostPortBind) == 0 {
		b.InlineConfig.HostPortBind = nil
	}
	if b.InlineConfig != nil && b.InlineConfig.HostPortMode != nil && *b.InlineConfig.HostPortMode == "" {
		b.InlineConfig.HostPortMode = nil
	}
	if b.InlineConfig != nil && b.InlineConfig.GPU != nil && *b.InlineConfig.GPU == "" {
		b.InlineConfig.GPU = nil
	}
	if b.InlineConfig != nil && b.InlineConfig.Runtime != nil && *b.InlineConfig.Runtime == "" {
		b.InlineConfig.Runtime = nil
	}
	if b.InlineConfig != nil && b.InlineConfig.DockerImage != nil && *b.InlineConfig.DockerImage == "" {
		b.InlineConfig.DockerImage = nil
	}
	if b.InlineConfig != nil && b.InlineConfig.Command != nil && len(*b.InlineConfig.Command) == 0 {
		b.InlineConfig.Command = nil
	}
	if b.InlineConfig != nil && b.InlineConfig.HealthCheck != nil && b.InlineConfig.HealthCheck.URL == "" && b.InlineConfig.HealthCheck.Interval == "" &&
		b.InlineConfig.HealthCheck.Timeout == "" && b.InlineConfig.HealthCheck.Retries == 0 && b.InlineConfig.HealthCheck.StartPeriod == "" {
		b.InlineConfig.HealthCheck = nil
	}
	if b.InlineConfig != nil && b.InlineConfig.RunFiles != nil && len(*b.InlineConfig.RunFiles) == 0 {
		b.InlineConfig.RunFiles = nil
	}
	if b.InlineConfig != nil && b.InlineConfig.Kubernetes == nil && b.InlineConfig.Standalone == nil &&
		b.InlineConfig.Envs == nil && b.InlineConfig.ExtraHosts == nil && b.InlineConfig.HostPortBind == nil &&
		b.InlineConfig.HostPortMode == nil && b.InlineConfig.GPU == nil && b.InlineConfig.Runtime == nil &&
		b.InlineConfig.Command == nil && b.InlineConfig.HealthCheck == nil && b.InlineConfig.DockerImage == nil &&
		b.InlineConfig.RunFiles == nil && b.InlineConfig.Volumes == nil {
		b.InlineConfig = nil
	}
}

// ParseDockerImageName -
func ParseDockerImageName(imgName string) (repoURL, shortName string) {
	var urlStart int = strings.Index(imgName, "/")
	if urlStart < 0 {
		repoURL = "registry-1.docker.io/v2/"
		shortName = imgName
		return
	}

	repoURL = imgName[:urlStart]
	if !strings.ContainsAny(repoURL, ".:") {
		repoURL = "registry-1.docker.io/v2/"
		shortName = imgName
	} else {
		shortName = imgName[urlStart+1:]
	}

	return
}

// LoadStandaloneConfig -
func (b *BaseVersionConfig) LoadStandaloneConfig() (stconf *StandaloneConfig, err error) {

	compName := b.GetName()
	ic := b.InstallConfiguration()
	setting := ic.GlobalSetting()

	var (
		cvdata   string
		cdata    []byte
		confType string
		confPath string
	)

	if b.InlineConfig != nil && b.InlineConfig.Standalone != nil && *b.InlineConfig.Standalone != "" {
		cdata = []byte(*b.InlineConfig.Standalone)
		confPath = fmt.Sprintf("[%s inline config]", compName)
	} else {
		confType = "file "
		confPath = path.Join(setting.GetConfigDir(), fmt.Sprintf("%s-standalone.yaml", compName))
		cdata, err = os.ReadFile(confPath)
		if err != nil {
			err = errors.Errorf("read %s error: %s", confPath, err)
			return
		}
	}

	// 设置JWT配置
	ic.Snz1dp.Jwt = b.GetJwtConfig()

	if cvdata, err = ic.RenderString(string(cdata)); err != nil {
		err = errors.Errorf("render %s error: %s", cdata, err)
		return
	}

	cdata = []byte(cvdata)
	stconf = new(StandaloneConfig)
	if err = yaml.Unmarshal(cdata, stconf); err != nil {
		stconf = nil
		err = errors.Errorf("%s%s error: %s\n%s", confType, confPath, err, cvdata)
		return
	}

	if b.InlineConfig.Envs != nil {
		stconf.Envs = append(stconf.Envs, *b.InlineConfig.Envs...)
	}

	if b.InlineConfig.ExtraHosts != nil {
		stconf.ExtraHosts = append(stconf.ExtraHosts, *b.InlineConfig.ExtraHosts...)
	}

	if b.InlineConfig.HostPortMode != nil && *b.InlineConfig.HostPortMode == "disabled" {
		stconf.Ports = []string{}
	} else if b.InlineConfig.HostPortBind != nil && len(*b.InlineConfig.HostPortBind) > 0 {
		stconf.Ports = []string{}
		stconf.Ports = append(stconf.Ports, *b.InlineConfig.HostPortBind...)
	}

	if b.InlineConfig.GPU != nil && *(b.InlineConfig.GPU) != "" {
		if *(b.InlineConfig.GPU) != "disabled" {
			stconf.GPU = *b.InlineConfig.GPU
		} else {
			stconf.GPU = ""
		}
	}

	if b.InlineConfig.Runtime != nil && *(b.InlineConfig.Runtime) != "" {
		stconf.Runtime = *b.InlineConfig.Runtime
	}

	if b.InlineConfig.Command != nil && len(*b.InlineConfig.Command) > 0 {
		stconf.Cmd = *b.InlineConfig.Command
	}

	if b.InlineConfig.HealthCheck != nil {
		if stconf.HealthCheck == nil {
			stconf.HealthCheck = new(HealthCheckConfig)
			stconf.HealthCheck.Interval = "10"
			stconf.HealthCheck.Timeout = "10s"
			stconf.HealthCheck.Retries = 30
			stconf.HealthCheck.StartPeriod = "60s"
		}
		if b.InlineConfig.HealthCheck.Interval != "" {
			stconf.HealthCheck.Interval = b.InlineConfig.HealthCheck.Interval
		}
		if b.InlineConfig.HealthCheck.Timeout != "" {
			stconf.HealthCheck.Timeout = b.InlineConfig.HealthCheck.Timeout
		}
		if b.InlineConfig.HealthCheck.Retries > 0 {
			stconf.HealthCheck.Retries = b.InlineConfig.HealthCheck.Retries
		}
		if b.InlineConfig.HealthCheck.StartPeriod != "" {
			stconf.HealthCheck.StartPeriod = b.InlineConfig.HealthCheck.StartPeriod
		}
		if b.InlineConfig.HealthCheck.URL != "" {
			stconf.HealthCheck.URL = b.InlineConfig.HealthCheck.URL
		}
		if len(b.InlineConfig.HealthCheck.Test) > 0 {
			stconf.HealthCheck.Test = b.InlineConfig.HealthCheck.Test
		}
	}

	if len(stconf.Volumes) > 0 || len(b.GetVolumes()) > 0 {
		newVolumes := []string{}
		existedMap := map[string]bool{}
		if len(b.GetVolumes()) > 0 {
			for _, newval := range b.GetVolumes() {
				splitted_array := strings.Split(newval, ":")
				if len(splitted_array) >= 2 {
					existedMap[splitted_array[1]] = true
					newVolumes = append(newVolumes, newval)
				}
			}
		}
		if len(stconf.Volumes) > 0 {
			for _, volume := range stconf.Volumes {
				splited_array := strings.Split(volume, ":")
				if len(splited_array) >= 2 && !existedMap[splited_array[1]] {
					newVolumes = append(newVolumes, volume)
				}
			}
		}
		stconf.Volumes = newVolumes
	}

	if len(stconf.RunFiles) > 0 || len(b.GetRunFiles()) > 0 {
		newRunFiles := map[string]string{}
		for k, v := range stconf.RunFiles {
			newRunFiles[k] = v
		}
		for k, v := range b.GetRunFiles() {
			newRunFiles[k] = v
		}
		stconf.RunFiles = newRunFiles
	}

	if b.InlineConfig.DockerImage != nil && *b.InlineConfig.DockerImage != "" {
		splitIdx := strings.Index(*b.InlineConfig.DockerImage, ":")
		var imageName, imageTag string
		if splitIdx > 0 {
			imageName = (*b.InlineConfig.DockerImage)[:splitIdx]
			imageTag = (*b.InlineConfig.DockerImage)[splitIdx+1:]
		} else {
			imageName = *b.InlineConfig.DockerImage
			imageTag = "latest"
		}
		if stconf.Docker == nil {
			stconf.Docker = new(StandaloneDocker)
		}
		stconf.Docker.Image = imageName
		stconf.Docker.Tag = imageTag
	}

	return
}

// GetEnvironments 获取环境变量配置
func (b *BaseVersionConfig) GetEnvironments() (ret []string) {
	if b.InlineConfig == nil || b.InlineConfig.Envs == nil {
		return
	}
	ret = *b.InlineConfig.Envs
	return
}

// SetEnvironments 设置环境变量配置
func (b *BaseVersionConfig) SetEnvironments(in []string) {
	if b.InlineConfig == nil {
		b.InlineConfig = new(ComponentConfig)
	}
	if b.InlineConfig.Envs == nil && len(in) > 0 {
		b.InlineConfig.Envs = new([]string)
		*b.InlineConfig.Envs = in
	} else {
		b.InlineConfig.Envs = nil
	}
}

// GetExtrasHosts
func (b *BaseVersionConfig) GetExtrasHosts() (ret []string) {
	if b.InlineConfig == nil || b.InlineConfig.ExtraHosts == nil {
		return
	}
	ret = *b.InlineConfig.ExtraHosts
	return
}

// GetRuntime
func (b *BaseVersionConfig) GetRuntime() (ret string) {
	if b.InlineConfig == nil || b.InlineConfig.Runtime == nil {
		return
	}
	ret = *b.InlineConfig.Runtime
	return
}

// SetRuntime
func (b *BaseVersionConfig) SetRuntime(in string) {
	if b.InlineConfig == nil {
		b.InlineConfig = new(ComponentConfig)
	}
	if b.InlineConfig.Runtime == nil && in != "" {
		b.InlineConfig.Runtime = new(string)
		*b.InlineConfig.Runtime = in
	} else {
		b.InlineConfig.Runtime = nil
	}
}

func (b *BaseVersionConfig) GetCommand() (ret []string) {
	if b.InlineConfig == nil || b.InlineConfig.Command == nil {
		return
	}
	ret = *b.InlineConfig.Command
	return
}

func (b *BaseVersionConfig) SetCommand(in []string) {
	if b.InlineConfig == nil {
		b.InlineConfig = new(ComponentConfig)
	}
	if b.InlineConfig.Command == nil && len(in) > 0 {
		b.InlineConfig.Command = new([]string)
		*b.InlineConfig.Command = in
	} else {
		b.InlineConfig.Command = nil
	}
}

func (b *BaseVersionConfig) GetHealthcheck() (ret HealthCheckConfig) {
	if b.InlineConfig == nil || b.InlineConfig.HealthCheck == nil {
		return
	}
	ret = *b.InlineConfig.HealthCheck
	return
}

func (b *BaseVersionConfig) SetHealthcheck(in HealthCheckConfig) {
	if b.InlineConfig == nil {
		b.InlineConfig = new(ComponentConfig)
	}

	if len(in.Test) > 0 || in.URL != "" || in.Interval != "" || in.Timeout != "" || in.Retries > 0 || in.StartPeriod != "" {
		b.InlineConfig.HealthCheck = new(HealthCheckConfig)
		if in.URL != "" {
			b.InlineConfig.HealthCheck.URL = in.URL
		}
		if in.Interval != "" {
			b.InlineConfig.HealthCheck.Interval = in.Interval
		}
		if in.Timeout != "" {
			b.InlineConfig.HealthCheck.Timeout = in.Timeout
		}
		if in.Retries > 0 {
			b.InlineConfig.HealthCheck.Retries = in.Retries
		}
		if in.StartPeriod != "" {
			b.InlineConfig.HealthCheck.StartPeriod = in.StartPeriod
		}
		if len(in.Test) > 0 {
			b.InlineConfig.HealthCheck.Test = in.Test
		}
	} else {
		b.InlineConfig.HealthCheck = nil
	}
}

func (b *BaseVersionConfig) SetDockerImage(in string) {
	if b.InlineConfig == nil {
		b.InlineConfig = new(ComponentConfig)
	}
	if b.InlineConfig.DockerImage == nil && in != "" {
		b.InlineConfig.DockerImage = new(string)
		*b.InlineConfig.DockerImage = in
	} else {
		b.InlineConfig.DockerImage = nil
	}
}

func (b *BaseVersionConfig) GetDockerImage() string {
	if b.InlineConfig == nil || b.InlineConfig.DockerImage == nil {
		return ""
	}
	return *b.InlineConfig.DockerImage
}

func (b *BaseVersionConfig) GetRunFiles() map[string]string {
	if b.InlineConfig == nil || b.InlineConfig.RunFiles == nil {
		return map[string]string{}
	}
	return *b.InlineConfig.RunFiles
}

func (b *BaseVersionConfig) SetRunFiles(in map[string]string) {
	if b.InlineConfig == nil {
		b.InlineConfig = new(ComponentConfig)
	}
	if b.InlineConfig.RunFiles == nil && len(in) > 0 {
		b.InlineConfig.RunFiles = new(map[string]string)
		*b.InlineConfig.RunFiles = in
	} else {
		b.InlineConfig.RunFiles = nil
	}
}

// GetGPU
func (b *BaseVersionConfig) GetGPU() (ret string) {
	if b.InlineConfig == nil || b.InlineConfig.GPU == nil {
		return
	}
	ret = *b.InlineConfig.GPU
	return
}

// SetGPU
func (b *BaseVersionConfig) SetGPU(in string) {
	if b.InlineConfig == nil {
		b.InlineConfig = new(ComponentConfig)
	}
	if b.InlineConfig.GPU == nil && in != "" {
		b.InlineConfig.GPU = new(string)
		*b.InlineConfig.GPU = in
	} else {
		b.InlineConfig.GPU = nil
	}
}

func (b *BaseVersionConfig) GetVolumes() (ret []string) {
	if b.InlineConfig == nil || b.InlineConfig.Volumes == nil {
		return
	}
	ret = *b.InlineConfig.Volumes
	return
}

// GetBindPorts -
func (b *BaseVersionConfig) GetBindPorts() (ports []string) {
	if b.InlineConfig == nil || b.InlineConfig.HostPortBind == nil {
		return
	}
	ports = *b.InlineConfig.HostPortBind
	return
}

// SetBindPorts -
func (b *BaseVersionConfig) SetBindPorts(in []string) {
	if b.InlineConfig == nil {
		b.InlineConfig = new(ComponentConfig)
	}
	if b.InlineConfig.HostPortBind == nil && len(in) > 0 {
		b.InlineConfig.HostPortBind = new([]string)
		*b.InlineConfig.HostPortBind = in
	} else {
		b.InlineConfig.HostPortBind = nil
	}
}

// GetBindPortEnable -
func (b *BaseVersionConfig) GetBindPortEnable() bool {
	if b.InlineConfig == nil || b.InlineConfig.HostPortMode == nil {
		return true
	}

	if strings.Compare(*b.InlineConfig.HostPortMode, "disabled") == 0 {
		return false
	}

	return false
}

// SetBindPortEnable -
func (b *BaseVersionConfig) SetBindPortEnable(val bool) {
	if b.InlineConfig == nil {
		b.InlineConfig = new(ComponentConfig)
	}

	if val {
		b.InlineConfig.HostPortMode = nil
	} else {
		b.InlineConfig.HostPortMode = new(string)
		*b.InlineConfig.HostPortMode = "disabled"
	}

}

// SetExtrasHosts
func (b *BaseVersionConfig) SetExtrasHosts(in []string) {
	if b.InlineConfig == nil {
		b.InlineConfig = new(ComponentConfig)
	}
	if b.InlineConfig.ExtraHosts == nil && len(in) > 0 {
		b.InlineConfig.ExtraHosts = new([]string)
		*b.InlineConfig.ExtraHosts = in
	} else {
		b.InlineConfig.ExtraHosts = nil
	}
}

func (b *BaseVersionConfig) SetVolumes(in []string) {
	if b.InlineConfig == nil {
		b.InlineConfig = new(ComponentConfig)
	}
	if b.InlineConfig.Volumes == nil && len(in) > 0 {
		b.InlineConfig.Volumes = new([]string)
		*b.InlineConfig.Volumes = in
	} else {
		b.InlineConfig.Volumes = nil
	}
}

// LoadKubernetesConfig -
func (b *BaseVersionConfig) LoadKubernetesConfig() (cdata []byte, err error) {

	compName := b.GetName()
	ic := b.InstallConfiguration()
	setting := ic.GlobalSetting()

	var (
		cvdata   string
		confPath string
	)

	if b.InlineConfig != nil && b.InlineConfig.Kubernetes != nil && *b.InlineConfig.Kubernetes != "" {
		cdata = []byte(*b.InlineConfig.Kubernetes)
	} else {
		confPath = path.Join(setting.GetConfigDir(), fmt.Sprintf("%s-kubernetes.yaml", compName))
		cdata, err = os.ReadFile(confPath)
		if err != nil {
			err = errors.Errorf("read %s error: %s", confPath, err)
			return
		}
	}

	// 设置JWT配置
	ic.Snz1dp.Jwt = b.GetJwtConfig()

	if cvdata, err = ic.RenderString(string(cdata)); err != nil {
		err = errors.Errorf("render %s error: %s", cdata, err)
		return
	}
	cdata = []byte(cvdata)
	return
}

// GetDockerImages 获取镜像仓库
func (b *BaseVersionConfig) GetDockerImages() (repos [][]string) {
	for _, dmg := range b.dockerImages {
		r, v := ParseDockerImageName(dmg)
		repos = append(repos, []string{r, v})
	}
	return
}

// PushDockerImages 推送镜像至仓库
func (b *BaseVersionConfig) PushDockerImages(force bool) (err error) {
	var (
		dc       *client.Client
		newImage string
	)

	if dc, err = docker.NewClient(); err != nil {
		return
	}

	defer dc.Close()

	for _, dmg := range b.dockerImages {
		r, v := ParseDockerImageName(dmg)
		sr := b.install.Snz1dp.Registry.GetDockerRepoURL()
		if sr != r && sr != BaseConfig.Snz1dp.Docker.URL {
			newImage = sr + "/" + v
			if _, err = docker.ImageExisted(dc, newImage); err != nil || force {
				if err = docker.TagImage(dc, dmg, newImage); err != nil {
					return
				}
				var (
					repoUser, repoPwd string = b.install.ResolveImageRepoUserAndPwd(newImage)
				)
				if err = docker.PushImage(dc, newImage, repoUser, repoPwd, ""); err != nil {
					return
				}
			}
		}
	}

	return
}

// Download 下载组件
func (b *BaseVersionConfig) Download(force bool) (err error) {
	bundleDir := b.install.setting.GetBundleDir()
	os.MkdirAll(bundleDir, os.ModePerm)

	if err = b.install.downloadBundles(true, []string{b.GetName()}, map[string]Component{b.GetName(): b}, force, false, false, ""); err != nil {
		return
	}

	return
}

// RenderEnvVariables 合并便利那个
func (c *InstallConfiguration) RenderEnvVariables(envs []string) (ret []string, err error) {
	externalIPV4 := utils.GetExternalIpv4()
	ret = append(ret, fmt.Sprintf("SNZ1DP_EXIP=%s", externalIPV4))
	ret = append(ret, fmt.Sprintf("SNZ1DP_HOST=%s", c.Snz1dp.Ingress.Host))
	ret = append(ret, fmt.Sprintf("SNZ1DP_BASE_URL=%s", c.Snz1dp.Ingress.GetBaseWebURL()))

	var envMap map[string]string = make(map[string]string)

	envMap["SNZ1DP_EXIP"] = externalIPV4
	envMap["SNZ1DP_HOST"] = c.Snz1dp.Ingress.Host
	envMap["SNZ1DP_BASE_URL"] = c.Snz1dp.Ingress.GetBaseWebURL()

	for i := range envs {
		m := envs[len(envs)-(i+1)]
		var (
			est    int
			mk, mv string
		)

		est = strings.Index(m, "=")
		if est < 0 {
			mk = m
		} else {
			mk = m[0:est]
			mv = m[est+1:]
		}

		if envMap[mk] == "" {
			mv = os.Expand(mv, func(key string) string {
				return envMap[key]
			})

			var (
				tpl *template.Template
			)

			if mv == "" {
				envMap[mk] = "(nil)"
				ret = append(ret, fmt.Sprintf("%s=%s", mk, mv))
				continue
			}

			if mv[0] == '"' && mv[len(mv)-1] == '"' {
				mv = mv[1:]
				mv = mv[:len(mv)-1]
			}

			if mv == "" {
				envMap[mk] = "(nil)"
				ret = append(ret, fmt.Sprintf("%s=%s", mk, mv))
				continue
			}

			mv = strings.ReplaceAll(mv, "[[{", "{{")
			mv = strings.ReplaceAll(mv, "}]]", "}}")

			if tpl, err = template.New("env").Parse(mv); err != nil {
				err = errors.Errorf("%s=%v invalid", mk, mv)
				return
			}

			var tbuf *bytes.Buffer = bytes.NewBuffer(nil)
			if err = tpl.Execute(tbuf, c); err != nil {
				err = errors.Errorf("%s=%v invalid", mk, mv)
				return
			}

			mv = tbuf.String()
			ret = append(ret, fmt.Sprintf("%s=%s", mk, mv))
			if mv == "" {
				envMap[mk] = "(nil)"
			} else {
				envMap[mk] = mv
			}
		}

	}
	return
}

// Load 加载缺省配置
func (b *BaseVersionConfig) Load(download, force bool) (err error) {
	bundleDir := b.install.setting.GetBundleDir()
	os.MkdirAll(bundleDir, os.ModePerm)

	var (
		comp           Component = b
		valuesFilename           = "values.yaml"
	)

	k := b.GetName()

	bundleFilepath := path.Join(bundleDir, fmt.Sprintf("%s-%s.tgz", k, comp.GetVersion()))

	if download {
		// 下载
		if err = comp.Download(force); err != nil {
			return
		}
	} else {
		var (
			bundleURL     string = comp.GetBundleURL()
			localFilepath string = comp.GetLocalFilePath()
			idxLastPath   int    = strings.LastIndex(comp.GetBundleURL(), "/")
			chartName     string = comp.GetName()
			chartVersion  string = comp.GetVersion()
		)

		if runtime.GOOS == "windows" {
			bundleURL = strings.ReplaceAll(comp.GetBundleURL(), "\\", "/")
			idxLastPath = strings.LastIndex(comp.GetBundleURL(), "/")
		}

		if localFilepath == "" {
			chartName = bundleURL[idxLastPath+1:]
			idxLastPath = strings.LastIndex(chartName, ".tgz")
			if idxLastPath > 0 {
				chartName = chartName[0:idxLastPath]
				idxLastPath = strings.LastIndex(chartName, "-")
				if idxLastPath > 0 {
					tsx := strings.LastIndex(chartName[0:idxLastPath], "-")
					if tsx > 0 && regexp.MustCompile(`\d+.\d+.\d+`).MatchString(chartName[tsx:idxLastPath]) {
						chartVersion = chartName[tsx+1:]
						chartName = chartName[0:tsx]
					} else {
						chartVersion = chartName[idxLastPath+1:]
						chartName = chartName[:idxLastPath]
					}
				} else {
					chartVersion = comp.GetVersion()
				}
			} else {
				chartName = comp.GetName()
				chartVersion = comp.GetVersion()
			}
			comp.SetRealName(chartName)
			comp.SetRealVersion(chartVersion)
		}
		comp.SetRealName(chartName)
		comp.SetRealVersion(chartVersion)
	}

	compExtractDir := path.Join(bundleDir, fmt.Sprintf("%s-%s", comp.GetRealName(), comp.GetRealVersion()))
	os.RemoveAll(compExtractDir)

	valuesFile := fmt.Sprintf("%s-%s/%s", comp.GetRealName(), comp.GetRealVersion(), valuesFilename)

	runFile := fmt.Sprintf("%s-%s/RUN.yaml", comp.GetRealName(), comp.GetRealVersion())

	// 退出后删除
	defer os.RemoveAll(compExtractDir)

	if err = ExtractBundleFile(bundleFilepath, valuesFile, bundleDir); err == nil || strings.Contains(err.Error(), "file already exists") {
		var valuesdata []byte
		if valuesdata, err = os.ReadFile(path.Join(compExtractDir, valuesFilename)); err != nil {
			return
		}
		b.valuesConfig = string(valuesdata)
	} else {
		return
	}

	if err = ExtractBundleFile(bundleFilepath, runFile, bundleDir); err == nil || strings.Contains(err.Error(), "file already exists") {
		var rundata []byte
		if rundata, err = os.ReadFile(path.Join(compExtractDir, "RUN.yaml")); err == nil {
			b.standalone = string(rundata)
		}
	}

	if err = ExtractBundleFile(bundleFilepath, runFile, bundleDir); err == nil || strings.Contains(err.Error(), "file already exists") {
		var imageData []byte
		if imageData, err = os.ReadFile(path.Join(compExtractDir, "IMAGES")); err == nil {
			flines := strings.Split(strings.ReplaceAll(string(imageData), "\r\n", "\n"), "\n")
			b.dockerImages = []string{}
			for _, fline := range flines {
				fImageNames := strings.Split(fline, " ")
				if len(fImageNames) == 0 {
					continue
				}
				imageName := fImageNames[len(fImageNames)-1]
				if imageName == "" {
					continue
				}
				b.dockerImages = append(b.dockerImages, imageName)
			}
		}
	}

	err = nil

	if b.InlineConfig == nil {
		b.InlineConfig = new(ComponentConfig)
	}

	if b.InlineConfig.Kubernetes == nil || *b.InlineConfig.Kubernetes == "" {
		b.InlineConfig.Kubernetes = new(string)
		*b.InlineConfig.Kubernetes = b.valuesConfig
	}

	if b.InlineConfig.Standalone == nil || *b.InlineConfig.Standalone == "" {
		b.InlineConfig.Standalone = new(string)
		*b.InlineConfig.Standalone = b.standalone
	}

	if b.InlineConfig.Envs == nil {
		b.InlineConfig.Envs = new([]string)
	}

	// 如果是Ingress
	if b.Jwt != nil && b.Jwt.RSAKey != "" {
		if _, err = utils.DecodePrivateKeyFromPEM(b.Jwt.RSAKey); err != nil {
			err = errors.Errorf("%s-%s jwt rsa private key error: %s", k, comp.GetVersion(), err)
			return
		}
		b.Jwt.PrivateKey = strings.ReplaceAll(b.Jwt.RSAKey, "\n", "\\n")
	} else if k == "ingress" && comp.BeInstall() {
		var rsaPrivateKey *rsa.PrivateKey
		if rsaPrivateKey, err = utils.GenerateRSAKeyPair(1024); err != nil {
			err = errors.Errorf("%s-%s generate rsa private key error: %s", k, comp.GetVersion(), err)
			return
		}
		if b.Jwt == nil {
			b.Jwt = new(JwtSecretConfig)
		}
		if b.Jwt.RSAKey, err = utils.RSAPrivateKeyToPem(rsaPrivateKey); err != nil {
			err = errors.Errorf("%s-%s export rsa private key to pem error: %s", k, comp.GetVersion(), err)
			return
		}
		b.Jwt.PrivateKey = strings.ReplaceAll(b.Jwt.RSAKey, "\n", "\\n")
		if b.Jwt.Token == "" {
			b.Jwt.Token = "gatewayadmin"
		}
	} else {
		b.Jwt = new(JwtSecretConfig)
		b.Jwt.inline = true
	}

	return
}

// GetJwtConfig 获取JWT配置
func (b *BaseVersionConfig) GetJwtConfig() *JwtSecretConfig {
	return b.Jwt
}

// SetJwtConfig 设置JWT配置
func (b *BaseVersionConfig) SetJwtConfig(jwt *JwtSecretConfig) {
	if jwt == nil {
		b.Jwt = nil
		return
	}

	b.Jwt = new(JwtSecretConfig)
	*b.Jwt = *jwt
}

// SetInstallConfiguration 设置上级配置
func (b *BaseVersionConfig) SetInstallConfiguration(ic *InstallConfiguration) {
	b.install = ic
}

// GetVersion -
func (b *BaseVersionConfig) GetVersion() string {
	return b.Version
}

// SetVersion -
func (b *BaseVersionConfig) SetVersion(val string) {
	b.Version = val
}

// SetName -
func (b *BaseVersionConfig) SetName(val string) {
	b.name = val
}

// GetName -
func (b *BaseVersionConfig) GetName() (val string) {
	val = b.name
	return
}

// SetRealName -
func (b *BaseVersionConfig) SetRealName(val string) {
	b.realname = val
}

// GetName -
func (b *BaseVersionConfig) GetRealName() (val string) {
	val = b.realname
	return
}

// SetRealVersion -
func (b *BaseVersionConfig) SetRealVersion(val string) {
	b.realversion = val
}

// GetName -
func (b *BaseVersionConfig) GetRealVersion() (val string) {
	val = b.realversion
	return
}

// GetNameWithVersion -
func (b *BaseVersionConfig) GetNameWithVersion() (val string) {
	val = fmt.Sprintf("%s-%s", b.GetName(), b.GetVersion())
	return
}

// InstallConfiguration -
func (b *BaseVersionConfig) InstallConfiguration() *InstallConfiguration {
	return b.install
}

// IsStandalone -
func (b *BaseVersionConfig) IsStandalone() bool {
	return b.standalone != ""
}

// SetStandalone - 设置允许独立运行
func (b *BaseVersionConfig) SetStandalone(val string) {
	b.standalone = val
}

// GetStandalone - 获取运行配置
func (b *BaseVersionConfig) GetStandalone() (val string) {
	val = b.standalone
	return
}

// GetValuesConfig 获取values.yaml配置
func (b *BaseVersionConfig) GetValuesConfig() (val string) {
	val = b.valuesConfig
	return
}

// SetValuesConfig 设置values.yaml配置
func (b *BaseVersionConfig) SetValuesConfig(val string) {
	b.valuesConfig = val
}

// BeInstall -
func (b *BaseVersionConfig) BeInstall() bool {
	return b.Install
}

// SetInstall -
func (b *BaseVersionConfig) SetInstall() {
	b.Install = true
	b.infile = true
}

// UnInstall -
func (b *BaseVersionConfig) UnInstall() {
	b.Install = false
}

// 后台服务
type BackendService struct {
	// 服务名称
	Name string `json:"name,omitempty"`
	// 服务地址
	URL string `json:"url,omitempty"`
	// 重试次数
	Retries string `json:"retries,omitempty"`
	// 读取超时实际
	ReadTimeout int `json:"read-timeout,omitempty"`
	// 读取超时实际
	WriteTimeout int `json:"write-timeout,omitempty"`
	// 标签
	Tags []string `json:"tags,omitempty"`
}

// 服务路由
type ServiceRoute struct {
	// 路由名称
	Name string `json:"name,omitempty"`
	// 路由主机名
	Host []string `json:"host,omitempty"`
	// 路由地址
	Path []string `json:"path,omitempty"`
	// 路由协议
	Protocol []string `json:"protocol,omitempty"`
	// 服务名称
	Service string `json:"service,omitempty"`
	// 服务地址
	ServiceURL string `json:"service-url,omitempty"`
	// 是否保留主机名
	PreserveHost *bool `json:"preserve-host,omitempty"`
	// 是否剥离路由地址
	StripPath bool `json:"strip-path,omitempty"`
	// 读取超时实际
	ReadTimeout int `json:"read-timeout,omitempty"`
	// 读取超时实际
	WriteTimeout int `json:"write-timeout,omitempty"`
	// 标签
	Tags []string `json:"tags,omitempty"`
	// 认证模式
	AuthMode  string   `json:"auth-mode,omitempty"`
	MiscAuth  bool     `json:"misc,omitempty"`
	Whitelist []string `json:"whitelist,omitempty"`
	Blacklist []string `json:"blacklist,omitempty"`
}

// Snz1dpIngressConfig - 对外服务配置
type Snz1dpIngressConfig struct {
	// 管理控制台对外提供的主机名或IP
	Host string `json:"host,omitempty"`
	// 端口
	Port *uint32 `json:"port,omitempty"`
	// 管理控制台对外提供访问的协议
	Protocol string `json:"protocol,omitempty"`
	// 基本地址
	WebBaseURL string `json:"-"`
	// 服务定义
	Services *[]*BackendService `json:"services,omitempty"`
	// 服务定义
	Routes *[]*ServiceRoute `json:"routes,omitempty"`
}

// GetBaseWebURL 获取基本Web地址
func (i *Snz1dpIngressConfig) GetBaseWebURL() string {
	if i.WebBaseURL != "" {
		return i.WebBaseURL
	}

	var bf bytes.Buffer
	if i.Protocol == "" {
		bf.WriteString("http")
	} else {
		bf.WriteString(i.Protocol)
	}
	bf.WriteString("://")
	if i.Host == "" {
		bf.WriteString("localhost")
	} else {
		bf.WriteString(i.Host)
	}
	if i.Protocol == "http" && i.Port != nil && *i.Port != 80 ||
		i.Protocol == "https" && i.Port != nil && *i.Port != 443 {
		bf.WriteString(":")
		bf.WriteString(strconv.FormatUint(uint64(*i.Port), 10))
	}
	i.WebBaseURL = bf.String()
	return i.WebBaseURL
}

// UserInfoConfig - 用户名密码配置
type UserInfoConfig struct {
	// 用户名
	Username string `json:"username,omitempty"`
	// 密码
	Password string `json:"password,omitempty"`
	// 加密密码
	EncodedPassword string `json:"encoded_password,omitempty"`
	// 邮箱
	Email string `json:"email,omitempty"`
	// 访问令牌
	AccessToken string `json:"access_token,omitempty"`
}

// KubeConfig k8s配置
type KubeConfig struct {
	Config       string `json:"config,omitempty"`
	Context      string `json:"context,omitempty"`
	Storageclass string `json:"storageclass,omitempty"`
	Token        string `json:"token,omitempty"`
	Apiserver    string `json:"server,omitempty"`
}

// DockerRegistry -
type DockerRegistry struct {

	// 仓库地址
	URL string `json:"url,omitempty"`

	// 用户名
	Username string `json:"username,omitempty"`

	// 密码
	Password string `json:"password,omitempty"`

	// 加密密码
	EncodedPassword string `json:"encoded_password,omitempty"`

	// 是否HTTPS，默认为https
	Secure    *bool `json:"secure,omitempty"`
	repoid    string
	sysconfig bool
}

// GetID 获取Repo🆔
func (d *DockerRegistry) GetID() (repoid string) {
	if d.repoid == "" {
		h := md5.New()
		h.Write([]byte(d.GetDockerRepoURL()))
		d.repoid = hex.EncodeToString(h.Sum(nil))
	}
	repoid = d.repoid
	return
}

// HelmRegistry
type HelmRegistry struct {
	// 仓库地址
	URL string `json:"url,omitempty"`

	// 用户名
	Username string `json:"username,omitempty"`

	// 密码
	Password string `json:"password,omitempty"`

	// 加密密码
	EncodedPassword string `json:"encoded_password,omitempty"`

	// 仓库名称
	Name string `json:"name,omitempty"`

	repoid    string
	sysconfig bool
}

// GetID 获取Repo🆔
func (d *HelmRegistry) GetID() (repoid string) {
	if d.repoid == "" {
		h := md5.New()
		h.Write([]byte(d.URL))
		d.repoid = hex.EncodeToString(h.Sum(nil))
	}
	repoid = d.repoid
	return
}

// MavenRegistry
type MavenRegistry struct {
	ID              string `json:"id,omitempty"`
	URL             string `json:"url,omitempty"`
	Username        string `json:"username,omitempty"`
	Password        string `json:"password,omitempty"`
	EncodedPassword string `json:"encoded_password,omitempty"`

	Mirrors *[]MavenMirrorItem `json:"mirrors,omitempty"`
}

// NpmRegistry
type NpmRegistry struct {
	ID              string `json:"id,omitempty"`
	URL             string `json:"url,omitempty"`
	Private         *bool  `json:"private,omitempty"`
	Username        string `json:"username,omitempty"`
	Password        string `json:"password,omitempty"`
	EncodedPassword string `json:"encoded_password,omitempty"`
}

// SassBinarySite
type SassBinarySite struct {
	ID              string `json:"id,omitempty"`
	URL             string `json:"url,omitempty"`
	Private         *bool  `json:"private,omitempty"`
	Username        string `json:"username,omitempty"`
	Password        string `json:"password,omitempty"`
	EncodedPassword string `json:"encoded_password,omitempty"`
}

func (d *HelmRegistry) ListChartVersions(getters helmGetter.Providers) (charts map[string]helmRepo.ChartVersions, err error) {
	// Download and write the index file to a temporary location
	buf := make([]byte, 20)
	rand.Read(buf)
	name := strings.ReplaceAll(base64.StdEncoding.EncodeToString(buf), "/", "-")

	c := helmRepo.Entry{
		URL:      d.URL,
		Username: d.Username,
		Password: d.Password,
		CertFile: "",
		KeyFile:  "",
		CAFile:   "",
		Name:     name,
	}
	r, err := helmRepo.NewChartRepository(&c, getters)
	if err != nil {
		return
	}
	idx, err := r.DownloadIndexFile()
	if err != nil {
		err = errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", d.URL)
		return
	}

	// Read the index file for the repository to get chart information and return chart URL
	repoIndex, err := helmRepo.LoadIndexFile(idx)
	if err != nil {
		return
	}

	repoIndex.SortEntries()
	charts = repoIndex.Entries
	return
}

// FindChartComponent
func (d *HelmRegistry) ResolveChartURLAndDigest(chartName, chartVersion string, getters helmGetter.Providers) (string, string, error) {

	// Download and write the index file to a temporary location
	buf := make([]byte, 20)
	rand.Read(buf)
	name := strings.ReplaceAll(base64.StdEncoding.EncodeToString(buf), "/", "-")

	c := helmRepo.Entry{
		URL:      d.URL,
		Username: d.Username,
		Password: d.Password,
		CertFile: "",
		KeyFile:  "",
		CAFile:   "",
		Name:     name,
	}
	r, err := helmRepo.NewChartRepository(&c, getters)
	if err != nil {
		return "", "", err
	}
	idx, err := r.DownloadIndexFile()
	if err != nil {
		return "", "", errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", d.URL)
	}

	// Read the index file for the repository to get chart information and return chart URL
	repoIndex, err := helmRepo.LoadIndexFile(idx)
	if err != nil {
		return "", "", err
	}

	errMsg := fmt.Sprintf("chart %q", chartName)
	if chartVersion != "" {
		errMsg = fmt.Sprintf("%s version %q", errMsg, chartVersion)
	}
	cv, err := repoIndex.Get(chartName, chartVersion)
	if err != nil {
		return "", "", errors.Errorf("%s not found in %s repository", errMsg, d.URL)
	}

	if len(cv.URLs) == 0 {
		return "", "", errors.Errorf("%s has no downloadable URLs", errMsg)
	}

	chartURL := cv.URLs[0]

	absoluteChartURL, err := helmRepo.ResolveReferenceURL(d.URL, chartURL)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to make chart URL absolute")
	}
	return absoluteChartURL, cv.Digest, nil
}

// BuildDockerRepoURL -
func BuildDockerRepoURL(url string) (prefix string, secure bool) {
	var ks int
	if ks = strings.Index(url, "https://"); ks == 0 {
		secure = true
		prefix = url[8:]
	} else if ks = strings.Index(url, "http://"); ks == 0 {
		prefix = url[7:]
		secure = false
	} else {
		prefix = url
		secure = true
	}
	prefix = strings.TrimSuffix(prefix, "/")
	return
}

// NewDockerRegistry -
func NewDockerRegistry(dockerUrl, u, p string) (d *DockerRegistry) {
	d = &DockerRegistry{
		URL:      dockerUrl,
		Username: u,
		Password: p,
	}
	var secure bool
	if d.Secure == nil {
		d.Secure = new(bool)
	}
	_, secure = BuildDockerRepoURL(d.URL)
	*d.Secure = secure
	return
}

// GetDockerRepoURL 仓库URL
func (d *DockerRegistry) GetDockerRepoURL() (prefix string) {
	var (
		secure  bool
		repourl string
	)
	if d.Secure == nil {
		d.Secure = new(bool)
	}
	repourl = d.URL
	if repourl == "" {
		repourl = d.URL
	}
	prefix, secure = BuildDockerRepoURL(repourl)
	*d.Secure = secure
	return
}

// GetK8sImagePullSecretName 获取K8s镜像拉取密钥名称
func (d *DockerRegistry) GetK8sImagePullSecretName() string {
	return fmt.Sprintf("snz1dp-%s", d.GetID())
}

// IsSecure 是否HTTPS
func (d *DockerRegistry) IsSecure() (secure bool) {
	d.GetDockerRepoURL()
	secure = *d.Secure
	return
}

type Snz1dpServer struct {
	URL            string `json:"config_url,omitempty"`
	DownloadPrefix string `json:"download_url,omitempty"`
	GitURL         string `json:"git_url,omitempty"`
}

func (s *Snz1dpServer) GetApiPrefix() string {
	return strings.TrimSuffix(s.URL, "/configs")
}

type RunnerConfig struct {
	DockerImage string `json:"docker_image,omitempty"`
}

// Snz1dpConfig - snz1dp配置
type Snz1dpConfig struct {

	// 版本
	Version string `json:"version"`

	// 安装的名字空间
	Namespace string `json:"namespace"`

	// 对外服务配置
	Ingress Snz1dpIngressConfig `json:"ingress"`

	// 安装默认时区
	Timezone string `json:"timezone"`

	// 组织名称
	Organization string `json:"organization"`

	// 管理员信息
	Admin UserInfoConfig `json:"admin,omitempty"`

	// 服务器地址
	Server *Snz1dpServer `json:"dpserver,omitempty"`

	// 仓库
	Registry *DockerRegistry `json:"registry,omitempty"`

	// 组件仓库
	HelmRepo *HelmRegistry `json:"helmrepo,omitempty"`

	// Maven仓库配置
	MavenRepo *MavenRegistry `json:"mvnrepo,omitempty"`

	// Npm仓库配置
	NpmRepo *NpmRegistry `json:"npmrepo,omitempty"`

	// SassSite仓库
	SassSite *SassBinarySite `json:"sassrepo,omitempty"`

	// 缺省跳转地址
	DefaultURL string `json:"default,omitempty"`

	// 注销跳转地址
	LogoutURL string `json:"logout,omitempty"`

	// 登录地址
	LoginURL string `json:"login,omitempty"`

	// 执行器配置
	RunnerConfig *RunnerConfig `json:"runner,omitempty"`

	// JWT配置
	Jwt        *JwtSecretConfig `json:"-"`
	ExternalIP string           `json:"-"`
}

// PostgresConfig - PGSQL配置
type PostgresConfig struct {
	// 可选安装
	BaseVersionConfig
	// 用户信息
	Admin UserInfoConfig `json:"admin,omitempty"`
	// 主机名
	Host string `json:"host"`
	// 端口
	Port *uint32 `json:"port"`
}

// RedisConfig - Redis配置
type RedisConfig struct {
	// 可选安装
	BaseVersionConfig
	// 主机名
	Host string `json:"host"`
	// 端口
	Port *uint32 `json:"port"`
	// 访问密码
	Password string `json:"password,omitempty"`
	// 加密密码
	EncodedPassword string `json:"encoded_password,omitempty"`
}

// WebConfig - Web定义
type WebConfig struct {
	Host     string  `json:"host,omitempty"`
	Port     *uint32 `json:"port,omitempty"`
	Webroot  string  `json:"webroot,omitempty"`
	Protocol string  `json:"protocol,omitempty"`
}

// GetURL 获取URL
func (w *WebConfig) GetURL() (url string) {
	buf := bytes.NewBuffer(nil)
	if w.Protocol == "" {
		fmt.Fprintf(buf, "http://")
		fmt.Fprintf(buf, "%s", w.Host)
		if w.Port != nil && *w.Port != 80 {
			fmt.Fprintf(buf, ":%d", *w.Port)
		}
	} else {
		fmt.Fprintf(buf, "%s://", w.Protocol)
		fmt.Fprintf(buf, "%s", w.Host)
		if w.Protocol == "http" || w.Protocol == "https" {
			if w.Protocol == "http" && (w.Port != nil && *w.Port != 80) ||
				w.Protocol == "https" && (w.Port != nil && *w.Port != 443) {
				fmt.Fprintf(buf, ":%d", *w.Port)
			}
		} else {
			fmt.Fprintf(buf, ":%d", *w.Port)
		}
	}
	fmt.Fprintf(buf, "%s", w.Webroot)
	url = buf.String()
	return
}

// AppgatewayConfig 网关配置
type AppgatewayConfig struct {
	BaseVersionConfig
	Admin WebConfig `json:"admin"`
	Web   WebConfig `json:"web"`
}

// ConfservConfig 配置服务
type ConfservConfig struct {
	BaseVersionConfig
	Web WebConfig `json:"web,omitempty"`
}

// XeaiConfig 单点服务
type XeaiConfig struct {
	BaseVersionConfig
	Web WebConfig `json:"web,omitempty"`
}

// ExtrasCompoentConfig -
type ExtrasCompoentConfig struct {
	BaseVersionConfig
	// 名称
	Name string `json:"name"`
	// 组织
	Group string `json:"group,omitempty"`
	// URL
	URL string `json:"url,omitempty"`
	// HELM仓库
	REPO string `json:"repo,omitempty"`
	// 安装名字空间，默认：snz1dp-app
	Namespace string `json:"namespace,omitempty"`
}

// GetBundleURL 获取组件地址
func (b *BaseVersionConfig) GetBundleURL() string {
	return b.bundleurl
}

// SetBundleURL 设置组件地址
func (b *BaseVersionConfig) SetBundleURL(url string) {
	b.bundleurl = url
}

// GetLocalFilePath - 获取本地文件
func (b *BaseVersionConfig) GetLocalFilePath() (localpath string) {
	var (
		furl       *url.URL
		err        error
		currentDir string
	)

	if furl, err = url.Parse(b.bundleurl); err != nil {
		localpath = b.bundleurl
		return
	}

	currentDir, _ = os.Getwd()

	if furl.Scheme == "" || furl.Scheme == "file" || (furl.Scheme != "http" && furl.Scheme != "https") {
		localpath = furl.Path
		if !filepath.IsAbs(localpath) {
			localpath = filepath.Join(currentDir, localpath)
			if localpath, err = filepath.Abs(localpath); err != nil {
				return
			}
		}
	}

	return
}

// EncryptionConfig - 加密配置
type EncryptionConfig struct {
	// 加密密码
	Password string `json:"password,omitempty"`
	packpass string
	// 内置
	inline bool
}

// DockerConfigWrapper -
type DockerConfigWrapper struct {
	Registry []*DockerRegistry `json:"registry"`
}

type HelmConfigWrapper struct {
	Registry []*HelmRegistry `json:"registry"`
}

// PipelineRunner - 流水线执行器
type PipelineRunner struct {
	ID            string   `json:"id"`
	Secret        string   `json:"secret,omitempty"`
	EncodedSecret string   `json:"encoded_secret,omitempty"`
	ServerURL     string   `json:"server_url,omitempty"`
	WorkDir       string   `json:"work_dir,omitempty"`
	DockerImage   string   `json:"docker_image,omitempty"`
	ExtrasHosts   []string `json:"hosts,omitempty"`
	Envs          []string `json:"envs,omitempty"`
}

// InstallConfiguration - 安装配置
type InstallConfiguration struct {
	Home       string                  `json:"-"`
	Version    uint32                  `json:"version,omitempty"`
	Platform   string                  `json:"platform,omitempty"`
	Kubernetes KubeConfig              `json:"kubernetes"`
	Encryption *EncryptionConfig       `json:"encryption,omitempty"`
	Snz1dp     Snz1dpConfig            `json:"snz1dp"`
	Postgres   *PostgresConfig         `json:"postgres,omitempty"`
	Redis      *RedisConfig            `json:"redis,omitempty"`
	Appgateway *AppgatewayConfig       `json:"appgateway,omitempty"`
	Confserv   *ConfservConfig         `json:"confserv,omitempty"`
	Xeai       *XeaiConfig             `json:"xeai,omitempty"`
	Extras     []*ExtrasCompoentConfig `json:"apps,omitempty"`
	Values     map[string]interface{}  `json:"values,omitempty"`
	Docker     *DockerConfigWrapper    `json:"docker,omitempty"`
	HelmRepo   *HelmConfigWrapper      `json:"helm,omitempty"`
	Runner     []*PipelineRunner       `json:"runner,omitempty"`

	Local struct {
		IP   string
		Name string
	} `json:"-"`

	dockerRegistryUrlMap *map[string]*DockerRegistry
	helmRegistryUrlMap   *map[string]*HelmRegistry
	helmRegistryNameMap  *map[string]*HelmRegistry

	setting *GlobalSetting
	loaded  bool
	inline  *InstallConfiguration
}

// GlobalSetting 获取全局配置
func (c *InstallConfiguration) GlobalSetting() *GlobalSetting {
	return c.setting
}

// GetDockerRegistryByURL 根据URL获取Docker镜像地址
func (c *InstallConfiguration) GetDockerRegistryByURL(url string) (registry *DockerRegistry) {
	url, secure := BuildDockerRepoURL(url)
	if registry = c.GetDockerRegistryUrlMap()[url]; registry == nil {
		return
	}
	if registry.IsSecure() != secure {
		registry = nil
	}
	return
}

func (c *InstallConfiguration) GetHelmRegistryByURL(url string) (registry *HelmRegistry) {
	if registry = c.GetHelmRegistryUrlMap()[url]; registry == nil {
		return
	}
	return
}

func (c *InstallConfiguration) GetHelmRegistryByName(name string) (registry *HelmRegistry) {
	if registry = c.GetHelmRegistryNameMap()[name]; registry == nil {
		return
	}
	return
}

// ResolveImageRepoUserAndPwd 获取镜像仓库密钥
func (c *InstallConfiguration) ResolveImageRepoUserAndPwd(imgName string) (repoUsername, repoPassword string) {
	var (
		repoURL  string
		registry *DockerRegistry
	)

	repoURL, _ = ParseDockerImageName(imgName)

	if registry = c.GetDockerRegistryByURL(repoURL); registry == nil {
		return
	}

	repoUsername = registry.Username
	repoPassword = registry.Password

	return
}

// GetDockerRegistryMap 获取Docker镜像仓库映射
func (c *InstallConfiguration) GetDockerRegistryUrlMap() (ret map[string]*DockerRegistry) {
	if c.dockerRegistryUrlMap == nil {
		c.dockerRegistryUrlMap = &map[string]*DockerRegistry{}
		for _, v := range c.Docker.Registry {
			(*c.dockerRegistryUrlMap)[v.URL] = v
		}

		elog := bytes.NewBuffer(nil)
		dockerConfig := cliconfig.LoadDefaultConfigFile(elog)

		creds, _ := dockerConfig.GetAllCredentials()
		for k, auth := range creds {
			if auth.Username == "" || auth.Password == "" {
				continue
			}
			var registry *DockerRegistry = new(DockerRegistry)
			imageprefix, secure := BuildDockerRepoURL(k)
			registry.URL = imageprefix
			registry.Username = auth.Username
			registry.Password = auth.Password
			registry.Secure = &secure
			registry.sysconfig = true
			if (*c.dockerRegistryUrlMap)[registry.GetDockerRepoURL()] == nil {
				(*c.dockerRegistryUrlMap)[registry.GetDockerRepoURL()] = registry
			}
		}

		if (*c.dockerRegistryUrlMap)[c.Snz1dp.Registry.GetDockerRepoURL()] == nil {
			(*c.dockerRegistryUrlMap)[c.Snz1dp.Registry.GetDockerRepoURL()] = c.Snz1dp.Registry
		} else {
			if (*c.dockerRegistryUrlMap)[c.Snz1dp.Registry.GetDockerRepoURL()].Username == "" ||
				c.Snz1dp.Registry.Username != "" {
				(*c.dockerRegistryUrlMap)[c.Snz1dp.Registry.GetDockerRepoURL()].Username = c.Snz1dp.Registry.Username
			}
			if (*c.dockerRegistryUrlMap)[c.Snz1dp.Registry.GetDockerRepoURL()].Password == "" ||
				c.Snz1dp.Registry.Password != "" {
				(*c.dockerRegistryUrlMap)[c.Snz1dp.Registry.GetDockerRepoURL()].Password = c.Snz1dp.Registry.Password
			}
		}
	}
	ret = *c.dockerRegistryUrlMap
	return
}

// GetHelmRegistryUrlMap 获取Helm仓库映射
func (c *InstallConfiguration) GetHelmRegistryUrlMap() (ret map[string]*HelmRegistry) {
	if c.helmRegistryUrlMap == nil {
		c.helmRegistryUrlMap = &map[string]*HelmRegistry{}
		for _, v := range c.HelmRepo.Registry {
			(*c.helmRegistryUrlMap)[v.URL] = v
		}

		if c.setting != nil {
			InitHelmActionConfig(c.setting)
			repoFile, err := helmRepo.LoadFile(c.setting.helmSetting.RepositoryConfig)

			if err == nil {
				for _, repo := range repoFile.Repositories {
					if repo.Username == "" || repo.Password == "" {
						continue
					}
					var repoURL = repo.URL
					repoURL = strings.TrimSuffix(repoURL, "/")
					var registry *HelmRegistry = new(HelmRegistry)
					registry.Name = repo.Name
					registry.URL = repoURL
					registry.Username = repo.Username
					registry.Password = repo.Password
					registry.sysconfig = true
					if (*c.helmRegistryUrlMap)[repoURL] == nil {
						(*c.helmRegistryUrlMap)[repoURL] = registry
					}
				}
			}
		}

		if (*c.helmRegistryUrlMap)[c.Snz1dp.HelmRepo.URL] == nil {
			(*c.helmRegistryUrlMap)[c.Snz1dp.HelmRepo.URL] = c.Snz1dp.HelmRepo
		} else {
			if (*c.helmRegistryUrlMap)[c.Snz1dp.HelmRepo.URL].Username == "" {
				(*c.helmRegistryUrlMap)[c.Snz1dp.HelmRepo.URL].Username = c.Snz1dp.HelmRepo.Username
			}
			if (*c.helmRegistryUrlMap)[c.Snz1dp.HelmRepo.URL].Password == "" {
				(*c.helmRegistryUrlMap)[c.Snz1dp.HelmRepo.URL].Password = c.Snz1dp.HelmRepo.Password
			}
		}
	}
	ret = *c.helmRegistryUrlMap
	return
}

// GetHelmRegistryUrlMap 获取Helm仓库映射
func (c *InstallConfiguration) GetHelmRegistryNameMap() (ret map[string]*HelmRegistry) {
	if c.helmRegistryNameMap == nil {
		c.helmRegistryNameMap = &map[string]*HelmRegistry{}
		for _, v := range c.HelmRepo.Registry {
			(*c.helmRegistryNameMap)[v.Name] = v
		}

		if c.setting != nil {
			InitHelmActionConfig(c.setting)
			repoFile, err := helmRepo.LoadFile(c.setting.helmSetting.RepositoryConfig)

			if err == nil {
				for _, repo := range repoFile.Repositories {
					if repo.Username == "" || repo.Password == "" {
						continue
					}
					var repoURL = repo.URL
					repoURL = strings.TrimSuffix(repoURL, "/")
					var registry *HelmRegistry = new(HelmRegistry)
					registry.Name = repo.Name
					registry.URL = repoURL
					registry.Username = repo.Username
					registry.Password = repo.Password
					registry.sysconfig = true
					if (*c.helmRegistryNameMap)[repo.Name] == nil {
						(*c.helmRegistryNameMap)[repo.Name] = registry
					}
				}
			}
		}

		if (*c.helmRegistryNameMap)[c.Snz1dp.HelmRepo.Name] == nil {
			(*c.helmRegistryNameMap)[c.Snz1dp.HelmRepo.Name] = c.Snz1dp.HelmRepo
		} else {
			if (*c.helmRegistryNameMap)[c.Snz1dp.HelmRepo.Name].Username == "" {
				(*c.helmRegistryNameMap)[c.Snz1dp.HelmRepo.Name].Username = c.Snz1dp.HelmRepo.Username
			}
			if (*c.helmRegistryNameMap)[c.Snz1dp.HelmRepo.Name].Password == "" {
				(*c.helmRegistryNameMap)[c.Snz1dp.HelmRepo.Name].Password = c.Snz1dp.HelmRepo.Password
			}
		}
	}
	ret = *c.helmRegistryNameMap
	return
}

// GetK8sNamespaces 获取中台与应用安装的名字空间
func (c *InstallConfiguration) GetK8sNamespaces() (ns []string) {
	namespaces := map[string]bool{}
	namespaces[c.Snz1dp.Namespace] = true
	for _, v := range c.Extras {
		namespaces[v.Namespace] = true
	}

	for k := range namespaces {
		ns = append(ns, k)
	}

	return
}

// ApplyDockerRegistry 添加或更新镜像仓库
func (c *InstallConfiguration) ApplyDockerRegistry(registry DockerRegistry) (changed bool, err error) {
	var (
		old *DockerRegistry
	)

	if old = c.GetDockerRegistryByURL(registry.URL); old != nil {
		old.Username = registry.Username
		old.Password = registry.Password
		old.Secure = registry.Secure
		changed = true
	} else {
		c.dockerRegistryUrlMap = nil
		c.Docker.Registry = append(c.Docker.Registry, &registry)
		changed = true
	}
	return
}

func (c *InstallConfiguration) ApplyNpmSettings(npmRegistry NpmRegistry) (changed bool, err error) {
	if npmRegistry.Private == nil {
		npmRegistry.Private = new(bool)
	}
	if *npmRegistry.Private != *c.Snz1dp.NpmRepo.Private {
		c.Snz1dp.NpmRepo.Private = npmRegistry.Private
		changed = true
	}

	if !changed && c.Snz1dp.NpmRepo.ID == npmRegistry.ID &&
		c.Snz1dp.NpmRepo.URL == npmRegistry.URL &&
		c.Snz1dp.NpmRepo.Username == npmRegistry.Username &&
		c.Snz1dp.NpmRepo.Password == npmRegistry.Password {
		return
	}

	c.Snz1dp.NpmRepo.ID = npmRegistry.ID
	c.Snz1dp.NpmRepo.URL = npmRegistry.URL
	c.Snz1dp.NpmRepo.Username = npmRegistry.Username

	if *c.Snz1dp.NpmRepo.Private {
		c.Snz1dp.NpmRepo.Password = npmRegistry.Password
	}
	return
}

func (c *InstallConfiguration) ApplySassSiteSettings(sassSite SassBinarySite) (changed bool, err error) {
	if sassSite.Private == nil {
		sassSite.Private = new(bool)
	}
	if *sassSite.Private != *c.Snz1dp.SassSite.Private {
		c.Snz1dp.SassSite.Private = sassSite.Private
		changed = true
	}

	if !changed && c.Snz1dp.SassSite.ID == sassSite.ID &&
		c.Snz1dp.SassSite.URL == sassSite.URL &&
		c.Snz1dp.SassSite.Username == sassSite.Username &&
		c.Snz1dp.SassSite.Password == sassSite.Password {
		return
	}

	c.Snz1dp.SassSite.ID = sassSite.ID
	c.Snz1dp.SassSite.URL = sassSite.URL
	c.Snz1dp.SassSite.Username = sassSite.Username

	if *c.Snz1dp.SassSite.Private {
		c.Snz1dp.SassSite.Password = sassSite.Password
	}
	return
}

func (c *InstallConfiguration) ApplyHelmRegistry(registry HelmRegistry) (changed bool, err error) {
	var (
		old *HelmRegistry
	)

	if old = c.GetHelmRegistryByURL(registry.URL); old != nil {
		if old.Name == registry.Name &&
			old.Username == registry.Username &&
			old.Password == registry.Password {
			return
		}
		old.Name = registry.Name
		old.Username = registry.Username
		old.Password = registry.Password
		changed = true
	} else {
		c.helmRegistryUrlMap = nil
		c.HelmRepo.Registry = append(c.HelmRepo.Registry, &registry)
		changed = true
	}
	return
}

// GetBundleComponent 获取组件
func (c *InstallConfiguration) GetBundleComponent(name string) (bundle Component, err error) {
	var compMap map[string]Component
	if _, compMap, err = c.GetBundleComponents(false); err != nil {
		return
	}
	bundle = compMap[name]
	if bundle == nil {
		return
	}
	bundle.Load(true, false)
	return
}

// ApplyExtrasComponent 添加组件
func (c *InstallConfiguration) ApplyExtrasComponent(extras ExtrasCompoentConfig) (err error) {
	if _, err = c.GetBundleComponent(extras.Name); err != nil {
		return
	}
	newcomp := new(ExtrasCompoentConfig)
	*newcomp = extras

	newcomp.SetConfigValues(extras.GetValuesConfig(), extras.GetStandalone())
	newcomp.SetEnvironments(extras.GetEnvironments())
	newcomp.SetExtrasHosts(extras.GetExtrasHosts())
	newcomp.SetVolumes(extras.GetVolumes())
	newcomp.SetBindPortEnable(extras.GetBindPortEnable())
	newcomp.SetBindPorts(extras.GetBindPorts())
	newcomp.SetDockerImage(extras.GetDockerImage())
	newcomp.SetRunFiles(extras.GetRunFiles())

	if extras.GetRuntime() != "" {
		newcomp.SetRuntime(extras.GetRuntime())
	}
	if extras.GetGPU() != "" {
		newcomp.SetGPU(extras.GetGPU())
	}
	if len(extras.GetCommand()) > 0 {
		newcomp.SetCommand(extras.GetCommand())
	}

	healthCheckConfig := extras.GetHealthcheck()
	if len(healthCheckConfig.Test) > 0 || healthCheckConfig.URL != "" {
		newcomp.SetHealthcheck(healthCheckConfig)
	}

	var wellbe_append bool = true
	for i, v := range c.Extras {
		if v.Name == extras.Name {
			c.Extras[i] = newcomp
			wellbe_append = false
			break
		}
	}
	if wellbe_append {
		c.Extras = append(c.Extras, newcomp)
	}
	return
}

func (c *InstallConfiguration) ApplyMavenSettings(mavenServer MavenServerItem) (changed bool, err error) {
	maven_setting_home, err := utils.ExpandUserDir("~/.m2")
	if err != nil {
		err = errors.Errorf("error resolve maven directory: %s", err)
		return
	}

	var (
		setting      *GlobalSetting = c.GlobalSetting()
		mvnwURL      string
		furl         *url.URL
		mvnwFileName string
		mvnwFilepath string
		tmpFilePath  string
		bundleDir    string = setting.GetBundleDir()
		dodown       bool   = false
		fst          os.FileInfo
	)

	if fst, err = os.Stat(maven_setting_home); err == nil {
		if !fst.IsDir() {
			err = errors.Errorf("m2 of user directory %s not existed", maven_setting_home)
			return
		}
	} else if err = os.MkdirAll(maven_setting_home, os.ModePerm); err != nil {
		err = errors.Errorf("create directory %s error: %s", maven_setting_home, err)
		return
	}

	mvnwURL = c.GlobalSetting().ResolveDownloadURL(BaseConfig.MavenWrap.URL)
	if furl, err = url.Parse(mvnwURL); err != nil {
		err = errors.Errorf("mvnw url %s error: %s", mvnwURL, err)
		return
	}

	mvnwFileName = filepath.Base(furl.Path)
	mvnwFilepath = path.Join(bundleDir, mvnwFileName)

	os.MkdirAll(bundleDir, os.ModePerm)
	if fst, err = os.Stat(mvnwFilepath); err == nil && !fst.IsDir() {
		sumData, err := utils.FileChecksum(mvnwFilepath, sha256.New())
		fcksum := hex.EncodeToString(sumData)
		dodown = err != nil || down.VerifyBundle(mvnwFilepath+".sha256", fcksum) != nil
	} else {
		if fst != nil {
			os.RemoveAll(mvnwFilepath)
		}
		dodown = true
	}

	if dodown {
		mvnwFilepath, err = down.NewBundleDownloader(setting.OutOrStdout(), furl.String(), down.VerifyAlways).Download(
			setting.GetBundleDir(), mvnwFileName)

		if err != nil {
			err = errors.Errorf("download %s error: %s", furl.String(), err.Error())
			return
		}
	}

	tmpFilePath = strings.TrimSuffix(mvnwFilepath, ".tgz")
	os.RemoveAll(tmpFilePath)
	defer os.RemoveAll(tmpFilePath)

	if err = UnarchiveBundle(mvnwFilepath, setting.GetBundleDir()); err != nil {
		err = errors.Errorf("unzip maven wrapper bundle eror: %s", err.Error())
		return
	}

	var fixPack = 1
	switch runtime.GOOS {
	case "windows":
		fixPack = 2
	}

	// mvnw命令
	mvncmd := path.Join(tmpFilePath, "mvnw")
	wrapperConfigFile := path.Join(tmpFilePath, ".mvn", "wrapper", "maven-wrapper.properties")
	distributionUrl := setting.ResolveDownloadURL(BaseConfig.MavenWrap.DistributionURL)
	wrapperUrl := setting.ResolveDownloadURL(BaseConfig.MavenWrap.WrapperURL)
	tempConfig := fmt.Sprintf("distributionUrl=%s\nwrapperUrl=%s", distributionUrl, wrapperUrl)
	if err = os.WriteFile(wrapperConfigFile, []byte(tempConfig), 0664); err != nil {
		err = errors.Errorf("write maven-wrapper.properties error: %s", err)
		return
	}

	// 检查Maven版本
	mvnVersionCommands := []string{
		mvncmd,
		"--version",
	}

	var temp_buffer bytes.Buffer
	mvnVersionCmdpath, mvnVersionCommands := parseCommand(mvnVersionCommands)
	mvnVersionCmd := exec.CommandContext(context.Background(), mvnVersionCmdpath, mvnVersionCommands...)
	mvnVersionCmd.Dir = tmpFilePath
	mvnVersionCmd.Stdin = setting.InOrStdin()
	mvnVersionCmd.Stdout = &temp_buffer
	mvnVersionCmd.Stderr = setting.ErrOrStderr()

	if err = mvnVersionCmd.Run(); err != nil {
		err = errors.Errorf("check or resolve maven version error:\n%s", temp_buffer.String())
		return
	}

	// Maven配置文件
	setting_xml := path.Join(maven_setting_home, "settings.xml")
	settings_security_xml := path.Join(maven_setting_home, "settings-security.xml")
	if !utils.FileExists(settings_security_xml) {

		commands := []string{
			mvncmd,
			"--encrypt-master-password",
			utils.RandString(16),
		}

		temp_buffer.Reset()
		cmdpath, commands := parseCommand(commands)
		execCmd := exec.CommandContext(context.Background(), cmdpath, commands...)
		execCmd.Dir = tmpFilePath
		execCmd.Stdin = setting.InOrStdin()
		execCmd.Stdout = &temp_buffer
		execCmd.Stderr = setting.ErrOrStderr()

		if err = execCmd.Run(); err != nil {
			err = errors.Errorf("maven encrypt master password error:\n%s", temp_buffer.String())
			return
		}

		var xml_file *os.File
		if xml_file, err = os.OpenFile(settings_security_xml, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
			err = errors.Errorf("save %s error: %s", settings_security_xml, err)
			return
		}
		defer xml_file.Close()

		fmt.Fprintln(xml_file, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
		fmt.Fprintln(xml_file, "<settingsSecurity>")
		fmt.Fprint(xml_file, "  <master>")
		xml_file.Write(temp_buffer.Bytes()[:temp_buffer.Len()-fixPack])
		fmt.Fprintln(xml_file, "</master>")
		fmt.Fprintln(xml_file, "</settingsSecurity>")
	}

	maven_settings := &MavenSettings{}
	setting_bytes, err := os.ReadFile(setting_xml)
	if err == nil {
		xml.Unmarshal(setting_bytes, maven_settings)
	}

	var maven_item *MavenServerItem = nil
	if maven_settings.Servers != nil {
		for i := 0; i < len(maven_settings.Servers.Items); i++ {
			if maven_settings.Servers.Items[i].ID == mavenServer.ID {
				maven_item = maven_settings.Servers.Items[i]
				break
			}
		}
	} else {
		maven_settings.Servers = new(MavenServers)
		maven_settings.Servers.Items = make([]*MavenServerItem, 0)
	}

	if c.Snz1dp.MavenRepo.Mirrors != nil && len(*c.Snz1dp.MavenRepo.Mirrors) > 0 {
		if maven_settings.Mirrors == nil {
			maven_settings.Mirrors = new(MavenMirrors)
			maven_settings.Mirrors.Items = make([]*MavenMirrorItem, 0)
		}
		for i := 0; i < len(*c.Snz1dp.MavenRepo.Mirrors); i++ {
			var mirror_item *MavenMirrorItem
			for j := 0; j < len(maven_settings.Mirrors.Items); j++ {
				if maven_settings.Mirrors.Items[j].ID == (*c.Snz1dp.MavenRepo.Mirrors)[i].ID {
					mirror_item = maven_settings.Mirrors.Items[j]
					break
				}
			}
			if mirror_item == nil {
				mirror_item = new(MavenMirrorItem)
				mirror_item.ID = (*c.Snz1dp.MavenRepo.Mirrors)[i].ID
				maven_settings.Mirrors.Items = append(maven_settings.Mirrors.Items, mirror_item)
			}
			mirror_item.MirrorOf = (*c.Snz1dp.MavenRepo.Mirrors)[i].MirrorOf
			mirror_item.Name = (*c.Snz1dp.MavenRepo.Mirrors)[i].Name
			mirror_item.URL = (*c.Snz1dp.MavenRepo.Mirrors)[i].URL
			if mirror_item.Name == "" {
				mirror_item.Name = mirror_item.ID
			}
		}
	}

	commands := []string{
		mvncmd,
		"--encrypt-password",
		mavenServer.Password,
	}

	temp_buffer.Reset()
	cmdpath, commands := parseCommand(commands)
	execCmd := exec.CommandContext(context.Background(), cmdpath, commands...)
	execCmd.Dir = tmpFilePath
	execCmd.Stdin = setting.InOrStdin()
	execCmd.Stdout = &temp_buffer
	execCmd.Stderr = setting.ErrOrStderr()

	if err = execCmd.Run(); err != nil {
		err = errors.Errorf("maven encrypt password error:\n%s", temp_buffer.String())
		return
	}

	mavenServerPassword := string(temp_buffer.Bytes()[:temp_buffer.Len()-fixPack])
	if maven_item == nil {
		maven_settings.Servers.Items = append(maven_settings.Servers.Items, &MavenServerItem{
			ID:       mavenServer.ID,
			Username: mavenServer.Username,
			Password: mavenServerPassword,
		})
	} else if maven_item.Username != mavenServer.Username || maven_item.Password != mavenServerPassword {
		maven_item.Username = mavenServer.Username
		maven_item.Password = mavenServerPassword
	}

	if c.Snz1dp.MavenRepo.ID != mavenServer.ID ||
		c.Snz1dp.MavenRepo.URL != mavenServer.URL ||
		c.Snz1dp.MavenRepo.Username != mavenServer.Username ||
		c.Snz1dp.MavenRepo.Password != mavenServer.Password {
		c.Snz1dp.MavenRepo.ID = mavenServer.ID
		c.Snz1dp.MavenRepo.URL = mavenServer.URL
		c.Snz1dp.MavenRepo.Username = mavenServer.Username
		c.Snz1dp.MavenRepo.Password = mavenServer.Password
		changed = true
	}

	temp_buffer.Reset()
	if setting_bytes, err = xml.MarshalIndent(maven_settings, "", "  "); err != nil {
		err = errors.Errorf("open %s error: %s", setting_xml, err)
		return
	}
	fmt.Fprintln(&temp_buffer, "<?xml version=\"1.0\"?>")
	temp_buffer.Write(setting_bytes)

	var xml_file *os.File
	if xml_file, err = os.OpenFile(setting_xml, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		err = errors.Errorf("open %s error: %s", setting_xml, err)
		return
	}
	defer xml_file.Close()
	if _, err = temp_buffer.WriteTo(xml_file); err != nil {
		err = errors.Errorf("save %s error: %s", setting_xml, err)
		return
	}
	return
}

// RenderStrings 模板
func (c *InstallConfiguration) RenderStrings(ss []string) (ret []string, err error) {

	var (
		tpl *template.Template
	)

	for _, v := range ss {
		v = strings.ReplaceAll(v, "[[{", "{{")
		v = strings.ReplaceAll(v, "}]]", "}}")

		if tpl, err = template.New("render").Parse(v); err != nil {
			return
		}
		var tbuf *bytes.Buffer = bytes.NewBuffer(nil)
		if err = tpl.Execute(tbuf, c); err != nil {
			return
		}
		ret = append(ret, tbuf.String())
	}

	return
}

// RenderString 模板
func (c *InstallConfiguration) RenderString(v string) (ret string, err error) {

	var (
		tpl *template.Template
	)

	v = strings.ReplaceAll(v, "[[{", "{{")
	v = strings.ReplaceAll(v, "}]]", "}}")

	if tpl, err = template.New("render").Parse(v); err != nil {
		return
	}

	var tbuf *bytes.Buffer = bytes.NewBuffer(nil)
	if err = tpl.Execute(tbuf, c); err != nil {
		return
	}

	ret = tbuf.String()

	return
}

// GetBundleComponents 组件列表
func (c *InstallConfiguration) GetBundleComponents(loadDisabled bool) (compNames []string, compMap map[string]Component, err error) {
	comps := []Component{
		c.Postgres,
		c.Redis,
		c.Appgateway,
		c.Confserv,
		c.Xeai,
	}

	compMap = make(map[string]Component)

	for _, v := range comps {
		compNames = append(compNames, v.GetName())
		compMap[v.GetName()] = v
		if strings.HasPrefix(c.Snz1dp.HelmRepo.URL, "/") {
			v.SetBundleURL(c.Snz1dp.HelmRepo.URL + fmt.Sprintf("%s-%s.tgz", v.GetName(), v.GetVersion()))
		} else {
			v.SetBundleURL(c.Snz1dp.HelmRepo.URL + "/" + fmt.Sprintf("%s-%s.tgz", v.GetName(), v.GetVersion()))
		}
	}

	for _, v := range c.Extras {
		if compMap[v.Name] != nil {
			continue
		}
		v.name = v.Name
		v.SetExtras(true)
		if v.URL == "" {
			if strings.HasPrefix(c.Snz1dp.HelmRepo.URL, "/") {
				v.SetBundleURL(c.Snz1dp.HelmRepo.URL + fmt.Sprintf("%s-%s.tgz", v.GetName(), v.GetVersion()))
			} else {
				v.SetBundleURL(c.Snz1dp.HelmRepo.URL + "/" + fmt.Sprintf("%s-%s.tgz", v.GetName(), v.GetVersion()))
			}
		} else {
			v.SetBundleURL(v.URL)
		}
		compNames = append(compNames, v.Name)
		compMap[v.Name] = v
	}

	if c.loaded {
		return
	}

	for _, k := range compNames {
		comp := compMap[k]
		if !(comp.BeInstall() || loadDisabled) {
			comp.Load(false, false)
			continue
		}
		if err = comp.Load(true, false); err != nil {
			return
		}
	}

	c.loaded = true

	return
}

// CopyComponent -
func CopyComponent(c Component, o Component) Component {
	ic := c.InstallConfiguration()

	switch o.(type) {
	case *PostgresConfig:
		*c.(*PostgresConfig) = *o.(*PostgresConfig)
	case *RedisConfig:
		*c.(*RedisConfig) = *o.(*RedisConfig)
	case *AppgatewayConfig:
		*c.(*AppgatewayConfig) = *o.(*AppgatewayConfig)
	case *ConfservConfig:
		*c.(*ConfservConfig) = *o.(*ConfservConfig)
	case *XeaiConfig:
		*c.(*XeaiConfig) = *o.(*XeaiConfig)
	case *ExtrasCompoentConfig:
		*c.(*ExtrasCompoentConfig) = *o.(*ExtrasCompoentConfig)
	default:
		//TODO: 实现更多的
	}

	if o.GetJwtConfig() != nil {
		c.SetJwtConfig(o.GetJwtConfig())
	}

	c.SetConfigValues(o.GetConfigValues())
	c.SetEnvironments(o.GetEnvironments())
	c.SetExtrasHosts(o.GetExtrasHosts())
	c.SetGPU(o.GetGPU())
	c.SetRuntime(o.GetRuntime())
	c.SetVolumes(o.GetVolumes())
	c.SetBindPorts(o.GetBindPorts())
	c.SetBindPortEnable(o.GetBindPortEnable())
	c.SetCommand(o.GetCommand())
	c.SetDockerImage(o.GetDockerImage())
	c.SetRunFiles(o.GetRunFiles())

	healthCheckConfig := o.GetHealthcheck()
	if len(healthCheckConfig.Test) > 0 || healthCheckConfig.URL != "" {
		c.SetHealthcheck(healthCheckConfig)
	}

	if ic != nil {
		c.SetInstallConfiguration(ic)
	}

	return c
}

// Apply 使用旧的数据覆盖配置
func (c *InstallConfiguration) Apply(o *InstallConfiguration) {

	c.inline = o.inline
	c.Values = o.Values

	if o.Encryption != nil {
		c.Encryption = new(EncryptionConfig)
		*c.Encryption = *o.Encryption
	}

	c.Kubernetes.Storageclass = o.Kubernetes.Storageclass
	c.Snz1dp = o.Snz1dp

	c.Docker = new(DockerConfigWrapper)
	*c.Docker = *o.Docker

	c.HelmRepo = new(HelmConfigWrapper)
	*c.HelmRepo = *o.HelmRepo

	c.Postgres = new(PostgresConfig)
	CopyComponent(c.Postgres, o.Postgres)

	c.Redis = new(RedisConfig)
	CopyComponent(c.Redis, o.Redis)

	c.Appgateway = new(AppgatewayConfig)
	CopyComponent(c.Appgateway, o.Appgateway)

	c.Confserv = new(ConfservConfig)
	CopyComponent(c.Confserv, o.Confserv)

	c.Xeai = new(XeaiConfig)
	CopyComponent(c.Xeai, o.Xeai)

	c.Postgres.install = c
	c.Redis.install = c
	c.Appgateway.install = c
	c.Confserv.install = c
	c.Xeai.install = c

	c.Extras = []*ExtrasCompoentConfig{}
	for _, v := range o.Extras {
		nc := new(ExtrasCompoentConfig)
		nc = CopyComponent(nc, v).(*ExtrasCompoentConfig)
		nc.install = c
		c.Extras = append(c.Extras, nc)
	}

	c.Runner = []*PipelineRunner{}
	for _, v := range o.Runner {
		nc := new(PipelineRunner)
		*nc = *v
		c.Runner = append(c.Runner, nc)
	}

	if o.dockerRegistryUrlMap != nil {
		c.dockerRegistryUrlMap = &map[string]*DockerRegistry{}
		for k, v := range *o.dockerRegistryUrlMap {
			var val *DockerRegistry = new(DockerRegistry)
			*val = *v
			(*c.dockerRegistryUrlMap)[k] = val
		}
	}

	if o.helmRegistryUrlMap != nil {
		c.helmRegistryUrlMap = &map[string]*HelmRegistry{}
		for k, v := range *o.helmRegistryUrlMap {
			var val *HelmRegistry = new(HelmRegistry)
			*val = *v
			(*c.helmRegistryUrlMap)[k] = val
		}
	}

}

// ToYaml 转成yaml
func (c *InstallConfiguration) ToYaml() ([]byte, error) {
	return yaml.Marshal(c)
}

// LoadInstallConfigurationFromBytes - 加载配置字节
func LoadInstallConfigurationFromBytes(y []byte) (*InstallConfiguration, error) {
	var InstallConfiguration *InstallConfiguration = &InstallConfiguration{}
	err := yaml.Unmarshal(y, &InstallConfiguration)
	if err != nil {
		return nil, errors.Wrap(err, "error format yaml")
	}
	return InstallConfiguration, nil
}

// LoadInstallConfigurationFromFile - 加载配置文件
func LoadInstallConfigurationFromFile(path string) (*InstallConfiguration, error) {
	y, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "read config file %s failed", path)
	}
	return LoadInstallConfigurationFromBytes(y)
}

// GlobalSetting - 全局配置
type GlobalSetting struct {
	startTime     time.Time
	startDir      string
	namespace     string
	configOnce    sync.Once
	config        genericclioptions.RESTClientGetter
	storage       *storage.ConfigStorage
	helmConfig    *helmAction.Configuration
	helmSetting   *helmCli.EnvSettings
	installConfig *InstallConfiguration
	kubeClient    helmKube.Interface
	in            io.ReadCloser
	out           io.Writer
	err           io.Writer
	args          []string
	kubeConfig    string
	kubeContext   string
	kubeToken     string
	kubeAPIServer string
	StorageClass  string
	debug         bool
	baseDir       string
	dirExpand     bool
	logFile       string
	offline       bool
	downloadURL   string
}

// IsOffline 网络已离线
func (s *GlobalSetting) IsOffline() bool {
	return s.offline
}

// SetDownloadURL 设置下载地址
func (s *GlobalSetting) SetDownloadURL(url string) {
	s.downloadURL = url
}

// ResolveDownloadURL 获取下载地址
func (s *GlobalSetting) ResolveDownloadURL(downloadURL string) (retURL string) {
	defaultDownloadPrefix := os.Getenv(BaseConfig.Snz1dp.Download.Env.Arg)
	if defaultDownloadPrefix == "" {
		defaultDownloadPrefix = s.downloadURL
		if defaultDownloadPrefix == "" {
			defaultDownloadPrefix = BaseConfig.Snz1dp.Download.Default
		}
	}

	if !strings.HasSuffix(defaultDownloadPrefix, "/") {
		defaultDownloadPrefix = defaultDownloadPrefix + "/"
	}

	if defaultDownloadPrefix != BaseConfig.Snz1dp.Download.Default &&
		strings.Index(downloadURL, BaseConfig.Snz1dp.Download.Default) == 0 {
		retURL = defaultDownloadPrefix + downloadURL[len(BaseConfig.Snz1dp.Download.Default):]
	} else {
		retURL = downloadURL
	}
	return
}

// AddFlags binds flags to the given flagset.
func (s *GlobalSetting) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&s.debug, "debug", s.debug, "enable verbose output")
	// fs.BoolVar(&s.offline, "offline", s.offline, "suppose the network is offline")
	fs.StringVar(&s.baseDir, "base-dir", "", BaseConfig.Snz1dp.Ctl.Name+" home base directory, $"+BaseConfig.Snz1dp.Ctl.Home.Env.Arg+" or "+BaseConfig.Snz1dp.Ctl.Home.Default)
}

// GetBaseDir -
func (s *GlobalSetting) GetBaseDir() string {
	if s.baseDir == "" {
		s.baseDir = os.Getenv(BaseConfig.Snz1dp.Ctl.Home.Env.Arg)
		if s.baseDir == "" {
			s.baseDir, _ = utils.ExpandUserDir(BaseConfig.Snz1dp.Ctl.Home.Default)
		} else {
			s.baseDir, _ = utils.ExpandUserDir(s.baseDir)
			s.dirExpand = true
		}
	} else if !s.dirExpand {
		s.baseDir, _ = utils.ExpandUserDir(s.baseDir)
		s.dirExpand = true
	}
	return s.baseDir
}

// GetBinDir -
func (s *GlobalSetting) GetBinDir() string {
	return path.Join(s.GetBaseDir(), "bin")
}

// GetConfigDir -
func (s *GlobalSetting) GetConfigDir() string {
	return path.Join(s.GetBaseDir(), "config")
}

// GetBundleDir -
func (s *GlobalSetting) GetBundleDir() string {
	return path.Join(s.GetBaseDir(), "bundle")
}

// GetLogDir -
func (s *GlobalSetting) GetLogDir() string {
	return path.Join(s.GetBaseDir(), "logs")
}

// GetConfigFilePath -
func (s *GlobalSetting) GetConfigFilePath() string {
	return path.Join(s.GetConfigDir(), utils.InstallConfigurationFileName)
}

// AddK8sFlags -
func (s *GlobalSetting) AddK8sFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&s.namespace, "namespace", "n", s.Namespace(), "namespace scope for this request")
	fs.StringVar(&s.kubeConfig, "kubeconfig", "~/.kube/config", "path to the kubeconfig file")
	fs.StringVar(&s.kubeContext, "kube-context", s.kubeContext, "name of the kubeconfig context to use")
	fs.StringVar(&s.kubeToken, "kube-token", s.kubeToken, "bearer token used for authentication")
	fs.StringVar(&s.kubeAPIServer, "kube-apiserver", s.kubeAPIServer, "the address and the port for the Kubernetes API server")
}

// Namespace - Get snz1dp namespace
func (s *GlobalSetting) Namespace() string {
	if s.namespace != "" {
		return s.namespace
	}
	return DefaultSnz1dpNS
}

// Debug 调试信息
func (s *GlobalSetting) Debug(f string, v ...interface{}) {
	if s.debug {
		logger.Debug(f, v)
	}
}

// IsDebug 是否调试中台
func (s *GlobalSetting) IsDebug() bool {
	return s.debug
}

// KubeClient -
func (s *GlobalSetting) KubeClient() helmKube.Interface {
	return s.kubeClient
}

// RESTClientGetter - RESTClientGetter gets the kubeconfig from EnvSettings
func (s *GlobalSetting) RESTClientGetter() genericclioptions.RESTClientGetter {
	s.configOnce.Do(func() {
		clientConfig := helmKube.GetConfig(s.kubeConfig, s.kubeContext, s.namespace)
		if s.kubeToken != "" {
			clientConfig.BearerToken = &s.kubeToken
		}
		if s.kubeAPIServer != "" {
			clientConfig.APIServer = &s.kubeAPIServer
		}

		s.config = clientConfig
	})
	return s.config
}

// NewGlobalSetting - 新建一个全局配置
func NewGlobalSetting(startDir string, in io.ReadCloser, out io.Writer, err io.Writer, args []string, startTime time.Time) *GlobalSetting {
	setting := GlobalSetting{
		startDir:     startDir,
		startTime:    startTime,
		namespace:    DefaultSnz1dpNS,
		StorageClass: DefaultStorageClass,
		kubeConfig:   utils.KubeConfigFile,
		in:           in,
		out:          out,
		args:         args,
		debug:        false,
		logFile:      "stdout",
	}
	logger.Disable()
	return &setting
}

// StartTime -
func (s *GlobalSetting) StartTime() time.Time {
	return s.startTime
}

// StartDir -
func (s *GlobalSetting) StartDir() string {
	return s.startDir
}

// KubeAPIServer -
func (s *GlobalSetting) KubeAPIServer() string {
	return s.kubeAPIServer
}

// KubeConfig -
func (s *GlobalSetting) KubeConfig() string {
	return s.kubeConfig
}

// KubeContext -
func (s *GlobalSetting) KubeContext() string {
	return s.kubeContext
}

// KubeToken -
func (s *GlobalSetting) KubeToken() string {
	return s.kubeToken
}

// InputArgs -
func (s *GlobalSetting) InputArgs() []string {
	return s.args
}

// InOrStdin -
func (s *GlobalSetting) InOrStdin() io.ReadCloser {
	if s.in == nil {
		return os.Stdin
	}
	return s.in
}

// OutOrStdout -
func (s *GlobalSetting) OutOrStdout() io.Writer {
	if s.in == nil {
		return os.Stdout
	}
	return s.out
}

// ErrOrStderr -
func (s *GlobalSetting) ErrOrStderr() io.Writer {
	if s.err == nil {
		return os.Stderr
	}
	return s.err
}

// Print 打印
func (s *GlobalSetting) Print(v ...interface{}) (int, error) {
	return fmt.Fprint(s.OutOrStdout(), v...)
}

// Printf 打印
func (s *GlobalSetting) Printf(f string, v ...interface{}) (int, error) {
	return fmt.Fprintf(s.OutOrStdout(), f, v...)
}

// Println 打印
func (s *GlobalSetting) Println(f string, v ...interface{}) (int, error) {
	return s.Printf(f+"\n", v...)
}

// Scan 打印
func (s *GlobalSetting) Scan(v ...interface{}) (int, error) {
	return fmt.Fscan(s.InOrStdin(), v...)
}

// Scanln 打印
func (s *GlobalSetting) Scanln(v ...interface{}) (int, error) {
	return fmt.Fscanln(s.InOrStdin(), v...)
}

// Scanf 打印
func (s *GlobalSetting) Scanf(f string, v ...interface{}) (int, error) {
	return fmt.Fscanf(s.InOrStdin(), f, v...)
}

func formatDate(t time.Time) string {
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	buf := make([]byte, 14)
	buf[0] = byte((year/1000)%10) + '0'
	buf[1] = byte((year/100)%10) + '0'
	buf[2] = byte((year/10)%10) + '0'
	buf[3] = byte(year%10) + '0'
	buf[4] = byte((month)/10) + '0'
	buf[5] = byte((month)%10) + '0'
	buf[6] = byte((day)/10) + '0'
	buf[7] = byte((day)%10) + '0'
	buf[8] = byte((hour)/10) + '0'
	buf[9] = byte((hour)%10) + '0'
	buf[10] = byte((minute)/10) + '0'
	buf[11] = byte((minute)%10) + '0'
	buf[12] = byte((second)/10) + '0'
	buf[13] = byte((second)%10) + '0'
	return string(buf)
}

// InitLogger -
func (s *GlobalSetting) InitLogger(scope string) {
	logdir := path.Join(s.GetBaseDir(), "logs")
	if _, err := os.Stat(logdir); err != nil {
		os.MkdirAll(logdir, os.ModePerm)
	}
	s.logFile = path.Join(logdir, formatDate(s.StartTime())+"."+scope+".log")
	logger.Cfg(s.logFile)
}

// LogFile -
func (s *GlobalSetting) LogFile() string {
	return s.logFile
}

// IsInitialized 是否已经初始化
func (s *GlobalSetting) IsInitialized() (*InstallConfiguration, error) {
	configBytes, err := s.storage.Get()
	if err != nil {
		return nil, err
	}

	var old *InstallConfiguration

	old, err = LoadInstallConfigurationFromBytes(configBytes)
	if err != nil {
		return nil, err
	}

	return old, nil
}

// KubernetesClientSet 客户
func (s *GlobalSetting) KubernetesClientSet() (*kubernetes.Clientset, error) {
	kc := s.kubeClient.(*helmKube.Client)
	return kc.Factory.KubernetesClientSet()
}

// IsKubernetesReachable -
func (s *GlobalSetting) IsKubernetesReachable() error {
	return s.kubeClient.IsReachable()
}

// ApplyInstallConfig 设置为安装配置
func (s *GlobalSetting) ApplyInstallConfig(ic *InstallConfiguration) {

	if ic.Snz1dp.Namespace == "" {
		ic.Snz1dp.Namespace = s.Namespace()
	} else {
		s.namespace = ic.Snz1dp.Namespace
	}

	if ic.Kubernetes.Apiserver == "" {
		ic.Kubernetes.Apiserver = s.kubeAPIServer
	} else {
		s.kubeAPIServer = ic.Kubernetes.Apiserver
	}

	if ic.Kubernetes.Config == "" {
		ic.Kubernetes.Config = s.kubeConfig
	} else {
		s.kubeConfig = ic.Kubernetes.Config
	}

	if ic.Kubernetes.Context == "" {
		ic.Kubernetes.Context = s.kubeContext
	} else {
		s.kubeContext = ic.Kubernetes.Context
	}

	if ic.Kubernetes.Token == "" {
		ic.Kubernetes.Token = s.kubeToken
	} else {
		s.kubeToken = ic.Kubernetes.Token
	}

	if ic.Kubernetes.Storageclass == "" {
		ic.Kubernetes.Storageclass = s.StorageClass
	} else {
		s.StorageClass = ic.Kubernetes.Storageclass
	}

	if s.kubeConfig == "" {
		s.kubeConfig = "~/.kube/config"
	}

	if s.kubeConfig[:1] == "~" {
		s.kubeConfig, _ = utils.ExpandUserDir(s.kubeConfig)
	}

}

// InitInstallConfiguration 获取本地
func (s *GlobalSetting) InitInstallConfiguration(ic *InstallConfiguration) (err error) {

	ic.setting = s
	s.installConfig = ic

	// 版本为0
	if ic.Version == 0 {
		ic.Version = 1
	}

	// 工具版本
	if ic.Snz1dp.Version == "" {
		ic.Snz1dp.Version = utils.Version()
	}

	// 获取本地IP地址
	ic.Local.IP = utils.GetExternalIpv4()
	ic.Local.Name, _ = os.Hostname()
	if ic.Local.Name == "" {
		ic.Local.Name = "localhost"
	}

	// 服务器地址
	if ic.Snz1dp.Server == nil {
		ic.Snz1dp.Server = new(Snz1dpServer)
		ic.Snz1dp.Server.URL = BaseConfig.Snz1dp.Server.URL
		ic.Snz1dp.Server.DownloadPrefix = BaseConfig.Snz1dp.Download.Default
	}

	if ic.Snz1dp.Server.URL == "" {
		ic.Snz1dp.Server.URL = BaseConfig.Snz1dp.Server.URL
	}

	if ic.Snz1dp.Server.DownloadPrefix == "" {
		ic.Snz1dp.Server.DownloadPrefix = BaseConfig.Snz1dp.Download.Default
	}

	if !strings.HasSuffix(ic.Snz1dp.Server.DownloadPrefix, "/") {
		ic.Snz1dp.Server.DownloadPrefix += "/"
	}

	if tempDownloadURL := os.Getenv(BaseConfig.Snz1dp.Download.Env.Arg); tempDownloadURL != "" {
		if !strings.HasSuffix(tempDownloadURL, "/") {
			tempDownloadURL += "/"
		}
		s.downloadURL = tempDownloadURL
	} else {
		s.downloadURL = ic.Snz1dp.Server.DownloadPrefix
	}

	if ic.Snz1dp.Server.GitURL == "" {
		ic.Snz1dp.Server.GitURL = BaseConfig.Snz1dp.Server.Git
	}

	if !strings.HasSuffix(ic.Snz1dp.Server.GitURL, "/") {
		ic.Snz1dp.Server.GitURL += "/"
	}

	// 配置
	if ic.Docker == nil {
		ic.Docker = new(DockerConfigWrapper)
	}

	if ic.HelmRepo == nil {
		ic.HelmRepo = new(HelmConfigWrapper)
	}

	// 主路径
	ic.Home = s.GetBaseDir()

	ic.Snz1dp.Ingress.GetBaseWebURL()

	if ic.Postgres == nil {
		ic.Postgres = new(PostgresConfig)
		CopyComponent(ic.Postgres, ic.inline.Postgres)
		ic.Postgres.infile = false
	} else {
		ic.Postgres.infile = true
	}

	if ic.Redis == nil {
		ic.Redis = new(RedisConfig)
		CopyComponent(ic.Redis, ic.inline.Redis)
		ic.Redis.infile = false
	} else {
		ic.Redis.infile = true
	}

	if ic.Appgateway == nil {
		ic.Appgateway = new(AppgatewayConfig)
		CopyComponent(ic.Appgateway, ic.inline.Appgateway)
		ic.Appgateway.infile = false
	} else {
		ic.Appgateway.infile = true
	}

	if ic.Confserv == nil {
		ic.Confserv = new(ConfservConfig)
		CopyComponent(ic.Confserv, ic.inline.Confserv)
		ic.Confserv.infile = false
	} else {
		ic.Confserv.infile = true
	}

	if ic.Xeai == nil {
		ic.Xeai = new(XeaiConfig)
		CopyComponent(ic.Xeai, ic.inline.Xeai)
		ic.Xeai.infile = false
	} else {
		ic.Xeai.infile = true
	}

	ic.Postgres.install = ic
	ic.Redis.install = ic
	ic.Appgateway.install = ic
	ic.Confserv.install = ic
	ic.Xeai.install = ic

	ic.Postgres.SetName("postgres")
	ic.Redis.SetName("redis")
	ic.Appgateway.SetName("ingress")
	ic.Confserv.SetName("confserv")
	ic.Xeai.SetName("xeai")

	for _, v := range ic.Extras {
		v.install = ic
		v.SetName(v.Name)
	}

	if ic.Snz1dp.Ingress.Port == nil {
		ic.Snz1dp.Ingress.Port = new(uint32)
		*ic.Snz1dp.Ingress.Port = 80
	} else if *ic.Snz1dp.Ingress.Port == 0 {
		*ic.Snz1dp.Ingress.Port = 80
	}

	if ic.Snz1dp.Ingress.Protocol == "" {
		ic.Snz1dp.Ingress.Protocol = "http"
	}

	if ic.Snz1dp.Ingress.Host == "" {
		ic.Snz1dp.Ingress.Host = "localhost"
	}

	// 初始化镜像仓库
	if ic.Snz1dp.Registry == nil {
		ic.Snz1dp.Registry = new(DockerRegistry)
	}

	if ic.Snz1dp.Registry.URL == "" {
		ic.Snz1dp.Registry.URL = BaseConfig.Snz1dp.Docker.URL
		ic.Snz1dp.Registry.Secure = &BaseConfig.Snz1dp.Docker.Secure
	}

	// 初始化Helm仓库
	if ic.Snz1dp.HelmRepo == nil {
		ic.Snz1dp.HelmRepo = new(HelmRegistry)
	}

	if ic.Snz1dp.HelmRepo.Name == "" {
		ic.Snz1dp.HelmRepo.Name = BaseConfig.Snz1dp.Helm.Name
	}

	if ic.Snz1dp.HelmRepo.URL == "" {
		ic.Snz1dp.HelmRepo.URL = BaseConfig.Snz1dp.Helm.URL
	}

	if ic.Snz1dp.HelmRepo.Username == "" {
		ic.Snz1dp.HelmRepo.Username = ic.Snz1dp.Registry.Username
	}

	if ic.Snz1dp.HelmRepo.Password == "" && ic.Snz1dp.HelmRepo.EncodedPassword == "" {
		if ic.Snz1dp.Registry.Password != "" {
			ic.Snz1dp.HelmRepo.Password = ic.Snz1dp.Registry.Password
			ic.Snz1dp.HelmRepo.EncodedPassword = ""
		} else {
			ic.Snz1dp.HelmRepo.EncodedPassword = ic.Snz1dp.Registry.EncodedPassword
		}
	}

	// 初始化Maven仓库
	if ic.Snz1dp.MavenRepo == nil {
		ic.Snz1dp.MavenRepo = new(MavenRegistry)
	}

	if ic.Snz1dp.MavenRepo.Mirrors == nil {
		ic.Snz1dp.MavenRepo.Mirrors = &[]MavenMirrorItem{}
	}

	if ic.Snz1dp.MavenRepo.ID == "" {
		ic.Snz1dp.MavenRepo.ID = BaseConfig.Snz1dp.Maven.ID
	}

	if ic.Snz1dp.MavenRepo.URL == "" {
		ic.Snz1dp.MavenRepo.URL = BaseConfig.Snz1dp.Maven.URL
	}

	if ic.Snz1dp.MavenRepo.Username == "" {
		ic.Snz1dp.MavenRepo.Username = ic.Snz1dp.Registry.Username
	}

	if ic.Snz1dp.MavenRepo.Password == "" && ic.Snz1dp.MavenRepo.EncodedPassword == "" {
		if ic.Snz1dp.Registry.Password != "" {
			ic.Snz1dp.MavenRepo.Password = ic.Snz1dp.Registry.Password
			ic.Snz1dp.MavenRepo.EncodedPassword = ""
		} else {
			ic.Snz1dp.MavenRepo.EncodedPassword = ic.Snz1dp.Registry.EncodedPassword
		}
	}

	if len(*ic.Snz1dp.MavenRepo.Mirrors) == 0 && len(BaseConfig.Snz1dp.Maven.Mirrors) > 0 {
		*ic.Snz1dp.MavenRepo.Mirrors = BaseConfig.Snz1dp.Maven.Mirrors[0:]
		for i := 0; i < len(*ic.Snz1dp.MavenRepo.Mirrors); i++ {
			if (*ic.Snz1dp.MavenRepo.Mirrors)[i].Name == "" {
				(*ic.Snz1dp.MavenRepo.Mirrors)[i].Name = (*ic.Snz1dp.MavenRepo.Mirrors)[i].ID
			}
		}
	}

	// 初始化Npm仓库
	if ic.Snz1dp.NpmRepo == nil {
		ic.Snz1dp.NpmRepo = new(NpmRegistry)
	}

	if ic.Snz1dp.NpmRepo.ID == "" {
		ic.Snz1dp.NpmRepo.ID = BaseConfig.Snz1dp.Npm.ID
	}

	if ic.Snz1dp.NpmRepo.URL == "" {
		ic.Snz1dp.NpmRepo.URL = BaseConfig.Snz1dp.Npm.URL
	}

	if ic.Snz1dp.NpmRepo.Username == "" {
		ic.Snz1dp.NpmRepo.Username = ic.Snz1dp.Registry.Username
	}

	if ic.Snz1dp.NpmRepo.Private == nil {
		ic.Snz1dp.NpmRepo.Private = new(bool)
		*ic.Snz1dp.NpmRepo.Private = BaseConfig.Snz1dp.Npm.Private
	}

	if *ic.Snz1dp.NpmRepo.Private {
		if ic.Snz1dp.NpmRepo.Password == "" && ic.Snz1dp.NpmRepo.EncodedPassword == "" {
			if ic.Snz1dp.Registry.Password != "" {
				ic.Snz1dp.NpmRepo.Password = ic.Snz1dp.Registry.Password
				ic.Snz1dp.NpmRepo.EncodedPassword = ""
			} else {
				ic.Snz1dp.NpmRepo.EncodedPassword = ic.Snz1dp.Registry.EncodedPassword
			}
		}
	} else {
		ic.Snz1dp.NpmRepo.Password = ""
		ic.Snz1dp.NpmRepo.EncodedPassword = ""
	}

	// 初始化SassSite
	if ic.Snz1dp.SassSite == nil {
		ic.Snz1dp.SassSite = new(SassBinarySite)
	}

	if ic.Snz1dp.SassSite.ID == "" {
		ic.Snz1dp.SassSite.ID = BaseConfig.Snz1dp.Npm.SassBinarySite.ID
	}

	if ic.Snz1dp.SassSite.URL == "" {
		ic.Snz1dp.SassSite.URL = BaseConfig.Snz1dp.Npm.SassBinarySite.URL
	}

	if ic.Snz1dp.SassSite.Username == "" {
		ic.Snz1dp.SassSite.Username = ic.Snz1dp.Registry.Username
	}

	if ic.Snz1dp.SassSite.Private == nil {
		ic.Snz1dp.SassSite.Private = new(bool)
		*ic.Snz1dp.SassSite.Private = BaseConfig.Snz1dp.Npm.SassBinarySite.Private
	}

	if *ic.Snz1dp.SassSite.Private {
		if ic.Snz1dp.SassSite.Password == "" && ic.Snz1dp.SassSite.EncodedPassword == "" {
			if ic.Snz1dp.Registry.Password != "" {
				ic.Snz1dp.SassSite.Password = ic.Snz1dp.Registry.Password
				ic.Snz1dp.SassSite.EncodedPassword = ""
			} else {
				ic.Snz1dp.SassSite.EncodedPassword = ic.Snz1dp.Registry.EncodedPassword
			}
		}
	} else {
		ic.Snz1dp.SassSite.Password = ""
		ic.Snz1dp.SassSite.EncodedPassword = ""
	}

	// 解密
	if ic.Encryption == nil {
		ic.Encryption = new(EncryptionConfig)
		ic.Encryption.inline = true
	}

	if ic.Encryption.Password == "" {
		ic.Encryption.Password, _ = encryptRegistryPasssword(normalAesKey)
		ic.Encryption.packpass = normalAesKey
		ic.Encryption.inline = true
	} else {
		var (
			temppass string
			pclen    int
		)

		if temppass, err = decryptRegistryPassword(ic.Encryption.Password); err != nil {
			err = errors.Errorf("config password encryption error: %s", ic.Encryption.Password)
			return
		}

		pclen = len(temppass) - 32
		if pclen >= 0 {
			ic.Encryption.packpass = temppass[0:32]
		} else if pclen < 0 {
			ic.Encryption.packpass = temppass
			pclen = 32 - len(temppass)
			for i := 0; i < pclen; i++ {
				ic.Encryption.packpass += " "
			}
		}
	}

	if ic.Snz1dp.Admin.EncodedPassword != "" {
		ic.Snz1dp.Admin.Password, err = ic.decryptNormalPassword(ic.Snz1dp.Admin.EncodedPassword)
		if err != nil {
			return
		}
		ic.Snz1dp.Admin.EncodedPassword = ""
	}

	if ic.Snz1dp.Registry.EncodedPassword != "" {
		ic.Snz1dp.Registry.Password, err = decryptRegistryPassword(ic.Snz1dp.Registry.EncodedPassword)
		if err != nil {
			return
		}
		ic.Snz1dp.Registry.EncodedPassword = ""
	}

	if ic.Snz1dp.HelmRepo.EncodedPassword != "" {
		ic.Snz1dp.HelmRepo.Password, err = decryptRegistryPassword(ic.Snz1dp.HelmRepo.EncodedPassword)
		if err != nil {
			return
		}
		ic.Snz1dp.HelmRepo.EncodedPassword = ""
	}

	if ic.Snz1dp.MavenRepo.EncodedPassword != "" {
		ic.Snz1dp.MavenRepo.Password, err = decryptRegistryPassword(ic.Snz1dp.MavenRepo.EncodedPassword)
		if err != nil {
			return
		}
		ic.Snz1dp.MavenRepo.EncodedPassword = ""
	}

	if ic.Snz1dp.NpmRepo.EncodedPassword != "" {
		ic.Snz1dp.NpmRepo.Password, err = decryptRegistryPassword(ic.Snz1dp.NpmRepo.EncodedPassword)
		if err != nil {
			return
		}
		ic.Snz1dp.NpmRepo.EncodedPassword = ""
	}

	if ic.Snz1dp.SassSite.EncodedPassword != "" {
		ic.Snz1dp.SassSite.Password, err = decryptRegistryPassword(ic.Snz1dp.SassSite.EncodedPassword)
		if err != nil {
			return
		}
		ic.Snz1dp.SassSite.EncodedPassword = ""
	}

	if ic.Postgres.Admin.EncodedPassword != "" {
		ic.Postgres.Admin.Password, err = ic.decryptNormalPassword(ic.Postgres.Admin.EncodedPassword)
		if err != nil {
			return
		}
		ic.Postgres.Admin.EncodedPassword = ""
	}

	if ic.Redis.EncodedPassword != "" {
		ic.Redis.Password, err = ic.decryptNormalPassword(ic.Redis.EncodedPassword)
		if err != nil {
			return
		}
		ic.Redis.EncodedPassword = ""
	}

	for _, v := range ic.Docker.Registry {
		if v.EncodedPassword != "" {
			v.EncodedPassword, err = decryptRegistryPassword(v.EncodedPassword)
			if err != nil {
				v.EncodedPassword = ""
			}
		}
		v.EncodedPassword = ""
	}

	if ic.Snz1dp.Namespace == "" {
		ic.Snz1dp.Namespace = s.Namespace()
	}

	if ic.Kubernetes.Apiserver == "" {
		ic.Kubernetes.Apiserver = s.kubeAPIServer
	}

	if ic.Kubernetes.Config == "" {
		ic.Kubernetes.Config = s.kubeConfig
	}

	if ic.Snz1dp.RunnerConfig == nil {
		ic.Snz1dp.RunnerConfig = new(RunnerConfig)
		ic.Snz1dp.RunnerConfig.DockerImage = BaseConfig.Runner.Docker.Image
	}

	serverApiURL := ic.Snz1dp.Server.GetApiPrefix()

	for _, v := range ic.Runner {
		if v.ID == "" {
			continue
		}

		if v.ServerURL == "" {
			v.ServerURL = serverApiURL
		}

		if v.EncodedSecret != "" {
			v.Secret, err = ic.decryptNormalPassword(v.EncodedSecret)
			if err != nil {
				return
			}
			v.EncodedSecret = ""
		}

		if v.DockerImage == "" {
			v.DockerImage = ic.Snz1dp.RunnerConfig.DockerImage
		}

	}

	return

}

// LoadLocalInstallConfiguration 获取本地
func (s *GlobalSetting) LoadLocalInstallConfiguration() (configFilePath string, ic *InstallConfiguration, err error) {
	var (
		fst os.FileInfo
	)

	var (
		inline    *InstallConfiguration
		icData    []byte
		confBox   *rice.Box
		saveLocal bool
	)

	// 加载内置缺省的
	confBox, _ = rice.FindBox("../asset/config")
	icData, _ = confBox.Bytes(utils.InstallConfigurationFileName)
	if inline, err = LoadInstallConfigurationFromBytes(icData); err != nil {
		return configFilePath, nil, err
	}

	if err = s.InitInstallConfiguration(inline); err != nil {
		return
	}

	// 安装配置文件
	configFilePath = path.Join(s.GetConfigDir(), utils.InstallConfigurationFileName)
	if fst, err = os.Stat(configFilePath); err != nil || fst.IsDir() {
		os.RemoveAll(configFilePath)
		saveLocal = true
		ic = new(InstallConfiguration)
		ic.Apply(inline)
	} else if ic, err = LoadInstallConfigurationFromFile(configFilePath); err != nil {
		saveLocal = true
		ic = new(InstallConfiguration)
		ic.Apply(inline)
	}

	ic.inline = inline

	if saveLocal {
		if err = s.SaveLocalInstallConfiguration(ic, configFilePath); err != nil {
			return configFilePath, ic, err
		}
	}

	if err = s.InitInstallConfiguration(ic); err != nil {
		return
	}

	return
}

// Export 保存安装配置至本地配置文件
func (c *InstallConfiguration) Export(encodepass, detailconf, loginconfig bool, outPassword string) (icData []byte, err error) {

	var (
		nic InstallConfiguration
	)

	// 保持内容
	nic.Kubernetes = c.Kubernetes
	nic.setting = c.setting

	nic.Apply(c)

	if outPassword != "" {
		encodepass = true
		nic.Encryption.Password = outPassword
		nic.Encryption.inline = false
	}

	if !nic.Postgres.infile {
		nic.Postgres = nil
	} else if !detailconf {
		nic.Postgres.ClearData()
	}

	if !nic.Redis.infile {
		nic.Redis = nil
	} else if !detailconf {
		nic.Redis.ClearData()
	}

	if !nic.Appgateway.infile {
		nic.Appgateway = nil
	} else if !detailconf {
		nic.Appgateway.ClearData()
	}

	if !nic.Confserv.infile {
		nic.Confserv = nil
	} else if !detailconf {
		nic.Confserv.ClearData()
	}

	if !nic.Xeai.infile {
		nic.Xeai = nil
	} else if !detailconf {
		nic.Xeai.ClearData()
	}

	err = nil
	if nic.Snz1dp.Admin.Password != "" && encodepass {
		nic.Snz1dp.Admin.EncodedPassword, err = nic.encryptNormalPasssword(nic.Snz1dp.Admin.Password)
		if err != nil {
			return
		}
		nic.Snz1dp.Admin.Password = ""
	}

	if nic.Snz1dp.Registry.Password != "" && encodepass {
		nic.Snz1dp.Registry.EncodedPassword, err = encryptRegistryPasssword(nic.Snz1dp.Registry.Password)
		if err != nil {
			return
		}
		nic.Snz1dp.Registry.Password = ""
	}

	if nic.Snz1dp.HelmRepo.Password != "" && encodepass {
		nic.Snz1dp.HelmRepo.EncodedPassword, err = encryptRegistryPasssword(nic.Snz1dp.HelmRepo.Password)
		if err != nil {
			return
		}
		nic.Snz1dp.HelmRepo.Password = ""
	}

	if nic.Snz1dp.MavenRepo.Password != "" && encodepass {
		nic.Snz1dp.MavenRepo.EncodedPassword, err = encryptRegistryPasssword(nic.Snz1dp.MavenRepo.Password)
		if err != nil {
			return
		}
		nic.Snz1dp.MavenRepo.Password = ""
	}

	if len(*nic.Snz1dp.MavenRepo.Mirrors) > 0 && len(*nic.Snz1dp.MavenRepo.Mirrors) == len(BaseConfig.Snz1dp.Maven.Mirrors) {
		alleq := true
		for i := len(*nic.Snz1dp.MavenRepo.Mirrors); i < len(*nic.Snz1dp.MavenRepo.Mirrors); i++ {
			inbase := false
			for j := len(BaseConfig.Snz1dp.Maven.Mirrors); j < len(BaseConfig.Snz1dp.Maven.Mirrors); j++ {
				if (*nic.Snz1dp.MavenRepo.Mirrors)[i].ID == BaseConfig.Snz1dp.Maven.Mirrors[j].ID &&
					(*nic.Snz1dp.MavenRepo.Mirrors)[i].URL == BaseConfig.Snz1dp.Maven.Mirrors[j].URL &&
					(*nic.Snz1dp.MavenRepo.Mirrors)[i].MirrorOf == BaseConfig.Snz1dp.Maven.Mirrors[j].MirrorOf {
					inbase = true
					break
				}
			}
			if !inbase {
				alleq = false
				break
			}
		}
		if alleq {
			nic.Snz1dp.MavenRepo.Mirrors = nil
		}
	}

	if nic.Snz1dp.MavenRepo.Mirrors != nil {
		for i := len(*nic.Snz1dp.MavenRepo.Mirrors); i < len(*nic.Snz1dp.MavenRepo.Mirrors); i++ {
			if (*nic.Snz1dp.MavenRepo.Mirrors)[i].ID == (*nic.Snz1dp.MavenRepo.Mirrors)[i].Name {
				(*nic.Snz1dp.MavenRepo.Mirrors)[i].Name = ""
			}
		}
	}

	if nic.Snz1dp.NpmRepo.Password != "" && encodepass {
		nic.Snz1dp.NpmRepo.EncodedPassword, err = encryptRegistryPasssword(nic.Snz1dp.NpmRepo.Password)
		if err != nil {
			return
		}
		nic.Snz1dp.NpmRepo.Password = ""
	}

	if nic.Snz1dp.SassSite.Password != "" && encodepass {
		nic.Snz1dp.SassSite.EncodedPassword, err = encryptRegistryPasssword(nic.Snz1dp.SassSite.Password)
		if err != nil {
			return
		}
		nic.Snz1dp.SassSite.Password = ""
	}

	if nic.Postgres != nil && nic.Postgres.Admin.Password != "" && encodepass {
		nic.Postgres.Admin.EncodedPassword, err = nic.encryptNormalPasssword(nic.Postgres.Admin.Password)
		if err != nil {
			return
		}
		nic.Postgres.Admin.Password = ""
	}

	if nic.Redis != nil && nic.Redis.Password != "" && encodepass {
		nic.Redis.EncodedPassword, err = nic.encryptNormalPasssword(nic.Redis.Password)
		if err != nil {
			return
		}
		nic.Redis.Password = ""
	}

	nic.Docker.Registry = []*DockerRegistry{}
	regMap := nic.GetDockerRegistryUrlMap()

	for k, v := range regMap {
		if v.Password != "" {
			v.EncodedPassword, err = encryptRegistryPasssword(v.Password)
			if err != nil {
				return
			}
		}
		if encodepass {
			v.Password = ""
		}
		if k == nic.Snz1dp.Registry.GetDockerRepoURL() {
			nic.Snz1dp.Registry.Username = v.Username
			nic.Snz1dp.Registry.Password = v.Password
			if encodepass {
				nic.Snz1dp.Registry.EncodedPassword = v.EncodedPassword
			}
			continue
		}
		if v.sysconfig {
			continue
		}
		nic.Docker.Registry = append(nic.Docker.Registry, v)
	}

	if len(nic.Docker.Registry) == 0 {
		nic.Docker = nil
	}

	nic.HelmRepo.Registry = []*HelmRegistry{}
	regHelmMap := nic.GetHelmRegistryUrlMap()

	for k, v := range regHelmMap {
		if v.Password != "" {
			v.EncodedPassword, err = encryptRegistryPasssword(v.Password)
			if err != nil {
				return
			}
		}
		if encodepass {
			v.Password = ""
		}
		if k == nic.Snz1dp.HelmRepo.URL {
			nic.Snz1dp.HelmRepo.Username = v.Username
			nic.Snz1dp.HelmRepo.Password = v.Password
			if encodepass {
				nic.Snz1dp.HelmRepo.EncodedPassword = v.EncodedPassword
			}
			continue
		}
		if v.sysconfig {
			continue
		}
		nic.HelmRepo.Registry = append(nic.HelmRepo.Registry, v)
	}

	if len(nic.HelmRepo.Registry) == 0 {
		nic.HelmRepo = nil
	}

	if nic.Snz1dp.Registry.URL == BaseConfig.Snz1dp.Docker.URL {
		nic.Snz1dp.Registry.URL = ""
	}

	if nic.Snz1dp.HelmRepo.Name == BaseConfig.Snz1dp.Helm.Name {
		nic.Snz1dp.HelmRepo.Name = ""
	}

	if nic.Snz1dp.HelmRepo.URL == BaseConfig.Snz1dp.Helm.URL {
		nic.Snz1dp.HelmRepo.URL = ""
	}

	if nic.Snz1dp.MavenRepo.URL == BaseConfig.Snz1dp.Maven.URL {
		nic.Snz1dp.MavenRepo.URL = ""
	}

	if nic.Snz1dp.MavenRepo.ID == BaseConfig.Snz1dp.Maven.ID {
		nic.Snz1dp.MavenRepo.ID = ""
	}

	if !*nic.Snz1dp.NpmRepo.Private {
		nic.Snz1dp.NpmRepo.Username = ""
		nic.Snz1dp.NpmRepo.Password = ""
	}

	if nic.Snz1dp.NpmRepo.URL == BaseConfig.Snz1dp.Npm.URL {
		nic.Snz1dp.NpmRepo.URL = ""
	}

	if nic.Snz1dp.NpmRepo.ID == BaseConfig.Snz1dp.Npm.ID {
		nic.Snz1dp.NpmRepo.ID = ""
	}

	if *nic.Snz1dp.NpmRepo.Private == BaseConfig.Snz1dp.Npm.Private {
		nic.Snz1dp.NpmRepo.Private = nil
	}

	if !*nic.Snz1dp.SassSite.Private {
		nic.Snz1dp.SassSite.Username = ""
		nic.Snz1dp.SassSite.Password = ""
	}

	if nic.Snz1dp.SassSite.URL == BaseConfig.Snz1dp.Npm.SassBinarySite.URL {
		nic.Snz1dp.SassSite.URL = ""
	}

	if nic.Snz1dp.SassSite.ID == BaseConfig.Snz1dp.Npm.SassBinarySite.ID {
		nic.Snz1dp.SassSite.ID = ""
	}

	if *nic.Snz1dp.SassSite.Private == BaseConfig.Snz1dp.Npm.SassBinarySite.Private {
		nic.Snz1dp.SassSite.Private = nil
	}

	if !loginconfig {
		nic.Snz1dp.Registry.Username = ""
		nic.Snz1dp.Registry.Password = ""
		nic.Snz1dp.Registry.EncodedPassword = ""

		nic.Snz1dp.HelmRepo.Username = ""
		nic.Snz1dp.HelmRepo.Password = ""
		nic.Snz1dp.HelmRepo.EncodedPassword = ""

		nic.Snz1dp.MavenRepo.Username = ""
		nic.Snz1dp.MavenRepo.Password = ""
		nic.Snz1dp.MavenRepo.EncodedPassword = ""

		nic.Snz1dp.NpmRepo.Username = ""
		nic.Snz1dp.NpmRepo.Password = ""
		nic.Snz1dp.NpmRepo.EncodedPassword = ""

		nic.Snz1dp.SassSite.Username = ""
		nic.Snz1dp.SassSite.Password = ""
		nic.Snz1dp.SassSite.EncodedPassword = ""
	}

	if nic.Snz1dp.Registry.Username != "" &&
		nic.Snz1dp.HelmRepo.Username != "" &&
		nic.Snz1dp.Registry.Username == nic.Snz1dp.HelmRepo.Username {
		nic.Snz1dp.HelmRepo.Username = ""
	}

	if nic.Snz1dp.Registry.EncodedPassword != "" &&
		nic.Snz1dp.HelmRepo.EncodedPassword != "" &&
		nic.Snz1dp.Registry.EncodedPassword == nic.Snz1dp.HelmRepo.EncodedPassword {
		nic.Snz1dp.HelmRepo.EncodedPassword = ""
	}

	if nic.Snz1dp.Registry.Username != "" &&
		nic.Snz1dp.MavenRepo.Username != "" &&
		nic.Snz1dp.Registry.Username == nic.Snz1dp.MavenRepo.Username {
		nic.Snz1dp.MavenRepo.Username = ""
	}

	if nic.Snz1dp.Registry.EncodedPassword != "" &&
		nic.Snz1dp.MavenRepo.EncodedPassword != "" &&
		nic.Snz1dp.Registry.EncodedPassword == nic.Snz1dp.MavenRepo.EncodedPassword {
		nic.Snz1dp.MavenRepo.EncodedPassword = ""
	}

	if nic.Snz1dp.Registry.Username != "" &&
		nic.Snz1dp.NpmRepo.Username != "" &&
		nic.Snz1dp.Registry.Username == nic.Snz1dp.NpmRepo.Username {
		nic.Snz1dp.NpmRepo.Username = ""
	}

	if nic.Snz1dp.NpmRepo.Username == "" &&
		nic.Snz1dp.Registry.EncodedPassword != "" &&
		nic.Snz1dp.NpmRepo.EncodedPassword != "" &&
		nic.Snz1dp.Registry.EncodedPassword == nic.Snz1dp.NpmRepo.EncodedPassword {
		nic.Snz1dp.NpmRepo.EncodedPassword = ""
	}

	if nic.Snz1dp.SassSite.Username == "" &&
		nic.Snz1dp.Registry.EncodedPassword != "" &&
		nic.Snz1dp.SassSite.EncodedPassword != "" &&
		nic.Snz1dp.Registry.EncodedPassword == nic.Snz1dp.SassSite.EncodedPassword {
		nic.Snz1dp.SassSite.EncodedPassword = ""
	}

	if nic.Snz1dp.Registry.URL == "" &&
		nic.Snz1dp.Registry.Username == "" &&
		(nic.Snz1dp.Registry.EncodedPassword == "" && encodepass ||
			nic.Snz1dp.Registry.Password == "") {
		nic.Snz1dp.Registry = nil
	}

	if nic.Snz1dp.HelmRepo.URL == "" &&
		nic.Snz1dp.HelmRepo.Username == "" &&
		(nic.Snz1dp.HelmRepo.EncodedPassword == "" && encodepass ||
			nic.Snz1dp.HelmRepo.Password == "") {
		nic.Snz1dp.HelmRepo = nil
	}

	if nic.Snz1dp.MavenRepo.ID == "" &&
		nic.Snz1dp.MavenRepo.URL == "" &&
		nic.Snz1dp.MavenRepo.Username == "" &&
		(nic.Snz1dp.MavenRepo.EncodedPassword == "" && encodepass ||
			nic.Snz1dp.MavenRepo.Password == "") &&
		nic.Snz1dp.MavenRepo.Mirrors == nil {
		nic.Snz1dp.MavenRepo = nil
	}

	if nic.Snz1dp.NpmRepo.ID == "" &&
		nic.Snz1dp.NpmRepo.URL == "" &&
		nic.Snz1dp.NpmRepo.Username == "" &&
		nic.Snz1dp.NpmRepo.Private == nil &&
		(nic.Snz1dp.NpmRepo.EncodedPassword == "" && encodepass ||
			nic.Snz1dp.NpmRepo.Password == "") {
		nic.Snz1dp.NpmRepo = nil
	}

	if nic.Snz1dp.SassSite.ID == "" &&
		nic.Snz1dp.SassSite.URL == "" &&
		nic.Snz1dp.SassSite.Username == "" &&
		nic.Snz1dp.SassSite.Private == nil &&
		(nic.Snz1dp.SassSite.EncodedPassword == "" && encodepass ||
			nic.Snz1dp.SassSite.Password == "") {
		nic.Snz1dp.SassSite = nil
	}

	var apiServerURL string
	if nic.Snz1dp.Server.URL != "" {
		apiServerURL = nic.Snz1dp.Server.GetApiPrefix()
	}

	if nic.Snz1dp.Server.URL == BaseConfig.Snz1dp.Server.URL {
		nic.Snz1dp.Server.URL = ""
	}

	if nic.Snz1dp.Server.DownloadPrefix == BaseConfig.Snz1dp.Download.Default {
		nic.Snz1dp.Server.DownloadPrefix = ""
	}

	if nic.Snz1dp.Server.GitURL == BaseConfig.Snz1dp.Server.Git {
		nic.Snz1dp.Server.GitURL = ""
	}

	if nic.Snz1dp.Server.URL == "" && nic.Snz1dp.Server.DownloadPrefix == "" {
		nic.Snz1dp.Server = nil
	}

	if nic.Snz1dp.RunnerConfig.DockerImage == BaseConfig.Runner.Docker.Image {
		nic.Snz1dp.RunnerConfig = nil
	}

	for _, v := range nic.Runner {
		if encodepass {
			if v.EncodedSecret, err = nic.encryptNormalPasssword(v.Secret); err != nil {
				return
			}
			v.Secret = ""
		} else {
			v.EncodedSecret = ""
		}
		if v.ServerURL == apiServerURL {
			v.ServerURL = ""
		}
		if v.DockerImage == BaseConfig.Runner.Docker.Image {
			v.DockerImage = ""
		}
	}

	if !detailconf {
		for _, v := range nic.Extras {
			v.ClearData()
		}
	}

	if nic.Encryption == nil || nic.Encryption.inline {
		nic.Encryption = nil
	} else if nic.Encryption.Password, err = encryptRegistryPasssword(nic.Encryption.packpass); err != nil {
		return
	}

	icData, err = nic.ToYaml()

	return
}

// SaveLocalInstallConfiguration 保存安装配置至本地配置文件
func (s *GlobalSetting) SaveLocalInstallConfiguration(oic *InstallConfiguration, cfname string) (err error) {

	var (
		ic InstallConfiguration
	)

	// 保持内容
	ic.Kubernetes = oic.Kubernetes
	ic.setting = oic.setting
	ic.Apply(oic)

	cfdir := path.Dir(cfname)

	if _, err = os.Stat(cfdir); err != nil {
		os.MkdirAll(cfdir, os.ModePerm)
	}

	if !ic.Postgres.infile {
		ic.Postgres = nil
	} else if ic.Postgres.Jwt != nil && ic.Postgres.Jwt.inline {
		ic.Postgres.Jwt = nil
	}

	if !ic.Redis.infile {
		ic.Redis = nil
	} else if ic.Redis.Jwt != nil && ic.Redis.Jwt.inline {
		ic.Redis.Jwt = nil
	}

	if !ic.Appgateway.infile {
		ic.Appgateway = nil
	} else if ic.Appgateway.Jwt != nil && ic.Appgateway.Jwt.inline {
		ic.Appgateway.Jwt = nil
	}

	if !ic.Confserv.infile {
		ic.Confserv = nil
	} else if ic.Confserv.Jwt != nil && ic.Confserv.Jwt.inline {
		ic.Confserv.Jwt = nil
	}

	if !ic.Xeai.infile {
		ic.Xeai = nil
	} else if ic.Xeai.Jwt != nil && ic.Xeai.Jwt.inline {
		ic.Xeai.Jwt = nil
	}

	if ic.Postgres != nil {
		ic.Postgres.ClearData()
	}

	if ic.Redis != nil {
		ic.Redis.ClearData()
	}

	if ic.Appgateway != nil {
		ic.Appgateway.ClearData()
	}

	if ic.Confserv != nil {
		ic.Confserv.ClearData()
	}

	if ic.Xeai != nil {
		ic.Xeai.ClearData()
	}

	for _, v := range ic.Extras {
		if (strings.HasPrefix(v.GetBundleURL(), ic.Snz1dp.HelmRepo.URL) || strings.HasPrefix(v.URL, ic.Snz1dp.HelmRepo.URL)) && v.GetName() == v.GetRealName() && v.GetVersion() == v.GetRealVersion() {
			v.URL = ""
		}
		v.ClearData()
	}

	err = nil
	if ic.Snz1dp.Admin.Password != "" {
		ic.Snz1dp.Admin.EncodedPassword, err = ic.encryptNormalPasssword(ic.Snz1dp.Admin.Password)
		if err != nil {
			return
		}
		ic.Snz1dp.Admin.Password = ""
	}

	if ic.Snz1dp.Registry.Password != "" {
		ic.Snz1dp.Registry.EncodedPassword, err = encryptRegistryPasssword(ic.Snz1dp.Registry.Password)
		if err != nil {
			return
		}
		ic.Snz1dp.Registry.Password = ""
	}

	if ic.Snz1dp.HelmRepo.Password != "" {
		ic.Snz1dp.HelmRepo.EncodedPassword, err = encryptRegistryPasssword(ic.Snz1dp.HelmRepo.Password)
		if err != nil {
			return
		}
		ic.Snz1dp.HelmRepo.Password = ""
	}

	if ic.Snz1dp.MavenRepo.Password != "" {
		if ic.Snz1dp.MavenRepo.EncodedPassword, err = encryptRegistryPasssword(ic.Snz1dp.MavenRepo.Password); err != nil {
			return
		}
		ic.Snz1dp.MavenRepo.Password = ""
	}

	if !*ic.Snz1dp.NpmRepo.Private {
		ic.Snz1dp.NpmRepo.Password = ""
	}

	if ic.Snz1dp.NpmRepo.Password != "" {
		if ic.Snz1dp.NpmRepo.EncodedPassword, err = encryptRegistryPasssword(ic.Snz1dp.NpmRepo.Password); err != nil {
			return
		}
		ic.Snz1dp.NpmRepo.Password = ""
	}

	if !*ic.Snz1dp.SassSite.Private {
		ic.Snz1dp.SassSite.Password = ""
	}

	if ic.Snz1dp.SassSite.Password != "" {
		if ic.Snz1dp.SassSite.EncodedPassword, err = encryptRegistryPasssword(ic.Snz1dp.SassSite.Password); err != nil {
			return
		}
		ic.Snz1dp.SassSite.Password = ""
	}

	if len(*ic.Snz1dp.MavenRepo.Mirrors) > 0 && len(*ic.Snz1dp.MavenRepo.Mirrors) == len(BaseConfig.Snz1dp.Maven.Mirrors) {
		alleq := true
		for i := len(*ic.Snz1dp.MavenRepo.Mirrors); i < len(*ic.Snz1dp.MavenRepo.Mirrors); i++ {
			inbase := false
			for j := len(BaseConfig.Snz1dp.Maven.Mirrors); j < len(BaseConfig.Snz1dp.Maven.Mirrors); j++ {
				if (*ic.Snz1dp.MavenRepo.Mirrors)[i].ID == BaseConfig.Snz1dp.Maven.Mirrors[j].ID &&
					(*ic.Snz1dp.MavenRepo.Mirrors)[i].URL == BaseConfig.Snz1dp.Maven.Mirrors[j].URL &&
					(*ic.Snz1dp.MavenRepo.Mirrors)[i].MirrorOf == BaseConfig.Snz1dp.Maven.Mirrors[j].MirrorOf {
					inbase = true
					break
				}
			}
			if !inbase {
				alleq = false
				break
			}
		}
		if alleq {
			ic.Snz1dp.MavenRepo.Mirrors = nil
		}
	}

	if ic.Snz1dp.MavenRepo.Mirrors != nil {
		for i := len(*ic.Snz1dp.MavenRepo.Mirrors); i < len(*ic.Snz1dp.MavenRepo.Mirrors); i++ {
			if (*ic.Snz1dp.MavenRepo.Mirrors)[i].ID == (*ic.Snz1dp.MavenRepo.Mirrors)[i].Name {
				(*ic.Snz1dp.MavenRepo.Mirrors)[i].Name = ""
			}
		}
	}

	if ic.Postgres != nil && ic.Postgres.Admin.Password != "" {
		ic.Postgres.Admin.EncodedPassword, err = ic.encryptNormalPasssword(ic.Postgres.Admin.Password)
		if err != nil {
			return
		}
		ic.Postgres.Admin.Password = ""
	}

	if ic.Redis != nil && ic.Redis.Password != "" {
		ic.Redis.EncodedPassword, err = ic.encryptNormalPasssword(ic.Redis.Password)
		if err != nil {
			return
		}
		ic.Redis.Password = ""
	}

	ic.Docker.Registry = []*DockerRegistry{}
	regMap := ic.GetDockerRegistryUrlMap()

	for k, v := range regMap {
		if v.Password != "" {
			if v.EncodedPassword, err = encryptRegistryPasssword(v.Password); err != nil {
				return
			}
		}
		v.Password = ""
		if k == ic.Snz1dp.Registry.GetDockerRepoURL() {
			ic.Snz1dp.Registry.URL = v.URL
			ic.Snz1dp.Registry.Username = v.Username
			ic.Snz1dp.Registry.Secure = v.Secure
			ic.Snz1dp.Registry.Password = v.Password
			ic.Snz1dp.Registry.EncodedPassword = v.EncodedPassword
			continue
		}
		if v.sysconfig {
			continue
		}
		ic.Docker.Registry = append(ic.Docker.Registry, v)
	}

	ic.HelmRepo.Registry = []*HelmRegistry{}
	helmRepoMap := ic.GetHelmRegistryUrlMap()
	for k, v := range helmRepoMap {
		if v.Password != "" {
			if v.EncodedPassword, err = encryptRegistryPasssword(v.Password); err != nil {
				return
			}
		}
		v.Password = ""
		if k == ic.Snz1dp.HelmRepo.URL {
			ic.Snz1dp.HelmRepo.Name = v.Name
			ic.Snz1dp.HelmRepo.Username = v.Username
			ic.Snz1dp.HelmRepo.EncodedPassword = v.EncodedPassword
			ic.Snz1dp.HelmRepo.Password = v.Password
			continue
		}
		if v.sysconfig {
			continue
		}
		ic.HelmRepo.Registry = append(ic.HelmRepo.Registry, v)
	}

	if ic.Snz1dp.Registry.URL == BaseConfig.Snz1dp.Docker.URL {
		ic.Snz1dp.Registry.URL = ""
	}

	if ic.Snz1dp.HelmRepo.Name == BaseConfig.Snz1dp.Helm.Name {
		ic.Snz1dp.HelmRepo.Name = ""
	}

	if ic.Snz1dp.HelmRepo.URL == BaseConfig.Snz1dp.Helm.URL {
		ic.Snz1dp.HelmRepo.URL = ""
	}

	if ic.Snz1dp.MavenRepo.URL == BaseConfig.Snz1dp.Maven.URL {
		ic.Snz1dp.MavenRepo.URL = ""
	}

	if ic.Snz1dp.MavenRepo.ID == BaseConfig.Snz1dp.Maven.ID {
		ic.Snz1dp.MavenRepo.ID = ""
	}

	if *ic.Snz1dp.NpmRepo.Private == BaseConfig.Snz1dp.Npm.Private {
		ic.Snz1dp.NpmRepo.Private = nil
	}

	if ic.Snz1dp.NpmRepo.URL == BaseConfig.Snz1dp.Npm.URL {
		ic.Snz1dp.NpmRepo.URL = ""
	}

	if ic.Snz1dp.NpmRepo.ID == BaseConfig.Snz1dp.Npm.ID {
		ic.Snz1dp.NpmRepo.ID = ""
	}

	if *ic.Snz1dp.SassSite.Private == BaseConfig.Snz1dp.Npm.SassBinarySite.Private {
		ic.Snz1dp.SassSite.Private = nil
	}

	if ic.Snz1dp.SassSite.URL == BaseConfig.Snz1dp.Npm.SassBinarySite.URL {
		ic.Snz1dp.SassSite.URL = ""
	}

	if ic.Snz1dp.SassSite.ID == BaseConfig.Snz1dp.Npm.SassBinarySite.ID {
		ic.Snz1dp.SassSite.ID = ""
	}

	if ic.Snz1dp.Registry.Username != "" &&
		ic.Snz1dp.HelmRepo.Username != "" &&
		ic.Snz1dp.Registry.Username == ic.Snz1dp.HelmRepo.Username {
		ic.Snz1dp.HelmRepo.Username = ""
	}

	if ic.Snz1dp.Registry.EncodedPassword != "" &&
		ic.Snz1dp.HelmRepo.EncodedPassword != "" &&
		ic.Snz1dp.Registry.EncodedPassword == ic.Snz1dp.HelmRepo.EncodedPassword {
		ic.Snz1dp.HelmRepo.EncodedPassword = ""
	}

	if ic.Snz1dp.Registry.Username != "" &&
		ic.Snz1dp.MavenRepo.Username != "" &&
		ic.Snz1dp.Registry.Username == ic.Snz1dp.MavenRepo.Username {
		ic.Snz1dp.MavenRepo.Username = ""
	}

	if ic.Snz1dp.Registry.EncodedPassword != "" &&
		ic.Snz1dp.MavenRepo.EncodedPassword != "" &&
		ic.Snz1dp.Registry.EncodedPassword == ic.Snz1dp.MavenRepo.EncodedPassword {
		ic.Snz1dp.MavenRepo.EncodedPassword = ""
	}

	if ic.Snz1dp.Registry.Username != "" &&
		ic.Snz1dp.NpmRepo.Username != "" &&
		ic.Snz1dp.Registry.Username == ic.Snz1dp.NpmRepo.Username {
		ic.Snz1dp.NpmRepo.Username = ""
	}

	if ic.Snz1dp.Registry.Username != "" &&
		ic.Snz1dp.SassSite.Username != "" &&
		ic.Snz1dp.Registry.Username == ic.Snz1dp.SassSite.Username {
		ic.Snz1dp.SassSite.Username = ""
	}

	if ic.Snz1dp.NpmRepo.Username == "" &&
		ic.Snz1dp.Registry.EncodedPassword != "" &&
		ic.Snz1dp.NpmRepo.EncodedPassword != "" &&
		ic.Snz1dp.Registry.EncodedPassword == ic.Snz1dp.NpmRepo.EncodedPassword {
		ic.Snz1dp.NpmRepo.EncodedPassword = ""
	}

	if ic.Snz1dp.SassSite.Username == "" &&
		ic.Snz1dp.Registry.EncodedPassword != "" &&
		ic.Snz1dp.SassSite.EncodedPassword != "" &&
		ic.Snz1dp.Registry.EncodedPassword == ic.Snz1dp.SassSite.EncodedPassword {
		ic.Snz1dp.SassSite.EncodedPassword = ""
	}

	if ic.Snz1dp.Registry.URL == "" &&
		ic.Snz1dp.Registry.Username == "" &&
		ic.Snz1dp.Registry.EncodedPassword == "" {
		ic.Snz1dp.Registry = nil
	}

	if ic.Snz1dp.HelmRepo.URL == "" &&
		ic.Snz1dp.HelmRepo.Username == "" &&
		ic.Snz1dp.HelmRepo.EncodedPassword == "" {
		ic.Snz1dp.HelmRepo = nil
	}

	if ic.Snz1dp.MavenRepo.ID == "" &&
		ic.Snz1dp.MavenRepo.URL == "" &&
		ic.Snz1dp.MavenRepo.Username == "" &&
		ic.Snz1dp.MavenRepo.EncodedPassword == "" &&
		ic.Snz1dp.MavenRepo.Mirrors == nil {
		ic.Snz1dp.MavenRepo = nil
	}

	if ic.Snz1dp.NpmRepo.Private == nil &&
		ic.Snz1dp.NpmRepo.ID == "" &&
		ic.Snz1dp.NpmRepo.URL == "" &&
		ic.Snz1dp.NpmRepo.Username == "" &&
		ic.Snz1dp.NpmRepo.EncodedPassword == "" {
		ic.Snz1dp.NpmRepo = nil
	}

	if ic.Snz1dp.SassSite.Private == nil &&
		ic.Snz1dp.SassSite.ID == "" &&
		ic.Snz1dp.SassSite.URL == "" &&
		ic.Snz1dp.SassSite.Username == "" &&
		ic.Snz1dp.SassSite.EncodedPassword == "" {
		ic.Snz1dp.SassSite = nil
	}

	var apiServerURL string
	if ic.Snz1dp.Server.URL != "" {
		apiServerURL = ic.Snz1dp.Server.GetApiPrefix()
	}

	if ic.Snz1dp.Server.URL == BaseConfig.Snz1dp.Server.URL {
		ic.Snz1dp.Server.URL = ""
	}

	if ic.Snz1dp.Server.DownloadPrefix == BaseConfig.Snz1dp.Download.Default {
		ic.Snz1dp.Server.DownloadPrefix = ""
	}

	if ic.Snz1dp.Server.GitURL == BaseConfig.Snz1dp.Server.Git {
		ic.Snz1dp.Server.GitURL = ""
	}

	if ic.Snz1dp.Server.URL == "" && ic.Snz1dp.Server.DownloadPrefix == "" {
		ic.Snz1dp.Server = nil
	}

	for _, v := range ic.Runner {
		if v.EncodedSecret, err = ic.encryptNormalPasssword(v.Secret); err != nil {
			return
		}
		v.Secret = ""
		if v.ServerURL == apiServerURL {
			v.ServerURL = ""
		}
		if v.DockerImage == ic.Snz1dp.RunnerConfig.DockerImage {
			v.DockerImage = ""
		}
	}

	if ic.Snz1dp.RunnerConfig.DockerImage == BaseConfig.Runner.Docker.Image {
		ic.Snz1dp.RunnerConfig = nil
	}

	if ic.Encryption == nil || ic.Encryption.inline {
		ic.Encryption = nil
	} else if ic.Encryption.Password, err = encryptRegistryPasssword(ic.Encryption.packpass); err != nil {
		return
	}

	if len(ic.Docker.Registry) == 0 {
		ic.Docker = nil
	}

	if len(ic.HelmRepo.Registry) == 0 {
		ic.HelmRepo = nil
	}

	var icData, readedData []byte
	icData, err = ic.ToYaml()
	if err != nil {
		return
	}
	var (
		fst os.FileInfo
		bak bool = true
	)
	if fst, err = os.Stat(cfname); err != nil || fst.IsDir() {
		if os.IsNotExist(err) {
			bak = false
		} else {
			if fst.IsDir() {
				err = fmt.Errorf("file %s is a directory", cfname)
			}
			return
		}
	}
	if bak {
		if readedData, err = os.ReadFile(cfname); err != nil {
			return
		}
		if !bytes.Equal(readedData, icData) {
			if err = utils.CopyFile(cfname, fmt.Sprintf("%s.%s", cfname, formatDate(time.Now()))); err != nil {
				return
			}
		}
	}
	if err = os.WriteFile(cfname, icData, 0644); err != nil {
		return
	}
	return
}
