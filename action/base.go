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

	rice "github.com/GeertJohan/go.rice"
	"github.com/ghodss/yaml"
	"github.com/wonderivan/logger"
)

var (
	// BaseConfig - 基本配置
	BaseConfig StaticBaseConfig
	// ArchetypeCatalog -
	ArchetypeCatalog []byte
)

// HealthCheckConfig - 心跳检测配置
type HealthCheckConfig struct {
	URL  string   `json:"url,omitempty"`
	Test []string `json:"test,omitempty"`

	Interval    string `json:"interval,omitempty"` // Interval is the time to wait between checks.
	Timeout     string `json:"timeout,omitempty"`  // Timeout is the time to wait before considering the check to have hung.
	StartPeriod string `json:"period,omitempty"`   // The start period for the container to initialize before the retries starts to count down.

	Retries int `json:"retries,omitempty"`
}

// LoginConfig - 登录页面配置
type LoginConfig struct {
	PageURL  string `json:"path,omitempty"` // 页面地址
	LoginURL string `json:"page,omitempty"` // 登录页面
	OnlyView bool   `json:"view,omitempty"` // 是否仅视图
}

// StandaloneIngressConfig - 独立运行发布配置
type StandaloneIngressConfig struct {
	Protocol     string       `json:"protocol,omitempty"` //协议，默认http
	Host         []string     `json:"host,omitempty"`     //路由主机
	ReadTimeout  int          `json:"read-timeout,omitempty"`
	WriteTimeout int          `json:"write-timeout,omitempty"`
	BackendPort  int          `json:"backend-port,omitempty"` //后台主机端口
	PreserveHost *bool        `json:"preserve-host,omitempty"`
	StripPath    bool         `json:"strip-path,omitempty"`
	WebRoot      string       `json:"webroot,omitempty"` // 后台路径
	LoginConfig  *LoginConfig `json:"login,omitempty"`
	JWTAuth      []string     `json:"jwt,omitempty"`
	MiscAuth     bool         `json:"misc,omitempty"`
	SSOAuth      []string     `json:"sso,omitempty"`
	Anonymous    []string     `json:"anonymous,omitempty"`
	Whitelist    []string     `json:"whitelist,omitempty"`
	Blacklist    []string     `json:"blacklist,omitempty"`
}

// GetBackendPort - 获取独立运行后端地址
func (s *StandaloneIngressConfig) GetBackendPort() (val int) {
	val = s.BackendPort
	if val <= 0 {
		val = 80
	}
	return
}

// StandaloneDocker - 独立镜像配置
type StandaloneDocker struct {
	Image string `json:"image,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

// InitJobConfig
type InitJobConfig struct {
	Command     []string `json:"command,omitempty"`
	DockerImage string   `json:"image,omitempty"`
}

// StandaloneConfig - 独立运行配置
type StandaloneConfig struct {
	Kind        string                     `json:"kind,omitempty"` // 默认无
	Image       string                     `json:"image,omitempty"`
	Tag         string                     `json:"tag,omitempty"`
	Docker      *StandaloneDocker          `json:"docker,omitempty"`
	Hostname    string                     `json:"host,omitempty"`
	Ports       []string                   `json:"ports,omitempty"`
	Volumes     []string                   `json:"volumes,omitempty"`
	Owners      []string                   `json:"owners,omitempty"`
	Envs        []string                   `json:"envs,omitempty"`
	ExtraHosts  []string                   `json:"hosts,omitempty"`
	Cmd         []string                   `json:"command,omitempty"`
	RunFiles    map[string]string          `json:"files,omitempty"`
	HealthCheck *HealthCheckConfig         `json:"healthcheck,omitempty"`
	Ingress     []*StandaloneIngressConfig `json:"ingress,omitempty"`
	InitCmd     []string                   `json:"init,omitempty"`
	InitJob     *InitJobConfig             `json:"initjob,omitempty"`
	Depends     []string                   `json:"depends,omitempty"`
	Platform    []string                   `json:"platform,omitempty"`
	GPU         string                     `json:"gpu,omitempty"`
	Runtime     string                     `json:"runtime,omitempty"`
}

// ChartConfig - Helm包配置
type ChartConfig struct {
	APIVersion  string `json:"apiVersion,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Version     string `json:"version,omitempty"`
	AppVersion  string `json:"appVersion,omitempty"`
}

// JwtSecretConfig JWT配置
type JwtSecretConfig struct {
	// 授权令牌
	Token string `json:"token,omitempty"`
	// 私钥
	PrivateKey string `json:"-"`
	// 存储的私钥
	RSAKey string `json:"rsakey,omitempty"`
	inline bool
}

// StaticBaseConfig - 静态基本配置
type StaticBaseConfig struct {
	Banner  string `json:"banner"`
	Website string `json:"website"`
	NewHint string `json:"newhint"`
	Snz1dp  struct {
		Server struct {
			URL string `json:"url"`
			Git string `json:"git"`
		} `json:"server"`

		Download struct {
			Env struct {
				Arg string `json:"arg"`
			} `json:"env"`
			Default string `json:"default"`
		} `json:"download"`

		Ctl struct {
			Name   string `json:"name"`
			Pinger string `json:"pinger"`
			URL    string `json:"url"`
			Home   struct {
				Env struct {
					Arg string `json:"arg"`
				} `json:"env"`
				Default string `json:"default"`
			} `json:"home"`
		} `json:"ctl"`

		Docker struct {
			URL    string `json:"url"`
			Secure bool   `json:"secure"`
		} `json:"docker"`

		Helm struct {
			Name        string `json:"name"`
			URL         string `json:"url"`
			PushPlugin  string `json:"push"`
			PushUserArg string `json:"user_arg"`
			PushPwdArg  string `json:"pwd_arg"`
		} `json:"helm"`

		Maven struct {
			ID      string            `json:"id"`
			URL     string            `json:"url"`
			Mirrors []MavenMirrorItem `json:"mirrors"`
		} `json:"maven"`

		Npm struct {
			ID             string `json:"id"`
			URL            string `json:"url"`
			Private        bool   `json:"private"`
			SassBinarySite struct {
				URL     string `json:"url"`
				ID      string `json:"id"`
				Private bool   `json:"private"`
			} `json:"sass-binary-site"`
		} `json:"npm"`

		K8s struct {
			Image struct {
				Prefix   string `json:"prefix"`
				Filename string `json:"filename"`
			} `json:"image"`
		} `json:"k8s"`

		Sonar struct {
			URL string `json:"url"`
		} `json:"sonar"`
	} `json:"snz1dp"`

	Kubectl struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Version string `json:"version"`
	} `json:"kubectl"`

	Sonar struct {
		URL     string `json:"url"`
		Name    string `json:"name"`
		Version string `json:"version"`
		Suffix  string `json:"suffix"`
	} `json:"sonar"`

	Keepalived struct {
		ImageName string `json:"image"`
		ImageTag  string `json:"tag"`
		Version   string `json:"version"`
	} `json:"keepalived"`

	Haproxy struct {
		ImageName string `json:"image"`
		ImageTag  string `json:"tag"`
		Version   string `json:"version"`
	} `json:"haproxy"`

	MavenWrap struct {
		URL             string `json:"url"`
		DistributionURL string `json:"distribution_url"`
		WrapperURL      string `json:"wrapper_url"`
	} `json:"mvw"`

	Archetype struct {
		Version  string `json:"version"`
		Group    string `json:"group"`
		Artifact string `json:"artifact"`
		URL      string `json:"url"`
	} `json:"archetype"`

	Vueproject struct {
		URL string `json:"url"`
	} `json:"vueproject"`

	Helm struct {
		Name          string `json:"name"`
		URL           string `json:"url"`
		Version       string `json:"version"`
		Suffix        string `json:"suffix"`
		Plugin        string `json:"plugin"`
		WindowsPlugin string `json:"windows-plugin"`
	} `json:"Helm"`

	Node struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Version string `json:"version"`
		Suffix  string `json:"suffix"`
	} `json:"node"`

	Buildx struct {
		Version string `json:"version"`
		Prefix  string `json:"prefix"`
	} `json:"buildx"`

	Runner struct {
		Docker struct {
			Image string `json:"image"`
		} `json:"docker"`
	} `json:"runner"`

	Nvm struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Windows struct {
			Version string `json:"version"`
			Suffix  string `json:"suffix"`
		} `json:"windows"`
	} `json:"nvm"`
}

// Action 操作接口
type Action interface {
	// 全局配置
	GlobalSetting() *GlobalSetting
	// 动作名称
	Name() string
	// 执行操作
	Run() error
	// 输出info日志
	Info(f string, v ...interface{})
	// 输出debug日志
	Debug(f string, v ...interface{})
	// 输出warn日志
	Warn(f string, v ...interface{})
	// 输出Error日志
	Error(f string, v ...interface{})
	// 输出warn日志
	Fatal(f string, v ...interface{})
	// 打印
	Print(v ...interface{}) (n int, err error)
	Printf(f string, v ...interface{}) (n int, err error)
	// 打印行
	Println(f string, v ...interface{}) (n int, err error)
	// 读取输入
	Scanln(v ...interface{}) (int, error)
	Scan(v ...interface{}) (int, error)
	Scanf(f string, v ...interface{}) (int, error)
	// 错误并退出
	ErrorExit(f string, v ...interface{})
}

// Component 组件接口
type Component interface {
	// 安装前
	PreInit(a Action) (bool, error)
	// 安装
	DoInit(a Action) error
	// 更新前
	PreUpgrade(a Action) error
	// 更新
	DoUpgrade(a Action) error
	// 删除
	DoFini(a Action) error
	// 获取名称
	GetName() string
	// 设置名称
	SetName(string)
	GetRealName() string
	SetRealName(string)
	GetRealVersion() string
	SetRealVersion(string)
	// 获取名称与版本
	GetNameWithVersion() string
	// 版本
	GetVersion() string
	// 设置
	SetVersion(string)
	// 是否独立
	IsStandalone() bool
	// 安装配置
	InstallConfiguration() *InstallConfiguration
	SetInstallConfiguration(*InstallConfiguration)
	// 是否安装
	BeInstall() bool
	// 设置安装
	SetInstall()
	// 是否卸载
	UnInstall()
	// 是否扩展
	IsExtras() bool
	// 设置为口占
	SetExtras(bool)
	// 获取组件地址
	GetBundleURL() string
	// 设置组件地址
	SetBundleURL(string)
	// 获取本地地址
	GetLocalFilePath() string
	// 加载配置
	Load(download, force bool) error
	// 下载
	Download(force bool) error
	// 推送镜像至部署仓库
	PushDockerImages(force bool) error
	// 获取镜像名称
	GetDockerImages() (repos [][]string)
	// 加载独立运行配置
	LoadStandaloneConfig() (*StandaloneConfig, error)
	// 加载K8s运行配置
	LoadKubernetesConfig() ([]byte, error)
	// 设置配置
	SetConfigValues(k8s, st string)
	/// 获取配置
	GetConfigValues() (k8s, st string)
	// 获取JWT配置
	GetJwtConfig() *JwtSecretConfig
	// 设置JWT配置
	SetJwtConfig(*JwtSecretConfig)
	// 获取配置
	GetEnvironments() []string
	// 设置环境变量
	SetEnvironments([]string)
	// 获取扩展Hosts
	GetExtrasHosts() []string
	GetVolumes() []string
	// 设置扩展Hosts
	SetExtrasHosts([]string)
	SetVolumes([]string)
	// 获取绑定端口
	GetBindPorts() []string
	SetBindPorts([]string)
	SetBindPortEnable(bool)
	GetBindPortEnable() bool
	SetGPU(string)
	GetGPU() string
	SetRuntime(string)
	GetRuntime() string
	GetCommand() []string
	SetCommand([]string)
	GetHealthcheck() HealthCheckConfig
	SetHealthcheck(HealthCheckConfig)
	SetDockerImage(string)
	GetDockerImage() string
	GetRunFiles() map[string]string
	SetRunFiles(map[string]string)
	// 清理
	ClearData()
}

// BaseAction - 基础操作
type BaseAction struct {
	// 接口
	Action
	// 全局配置
	setting *GlobalSetting
	// 名称
	name string
}

// Name 操作名称
func (a *BaseAction) Name() string {
	return a.name
}

// Info 日志
func (a *BaseAction) Info(f string, v ...interface{}) {
	logger.Info(f, v...)
}

// Debug 日志
func (a *BaseAction) Debug(f string, v ...interface{}) {
	a.setting.Debug(f, v...)
}

// Warn 日志
func (a *BaseAction) Warn(f string, v ...interface{}) {
	logger.Warn(f, v...)
}

// Error 日志
func (a *BaseAction) Error(f string, v ...interface{}) {
	logger.Error(f, v...)
}

// Fatal 日志
func (a *BaseAction) Fatal(f string, v ...interface{}) {
	logger.Fatal(f, v...)
}

// Print 打印
func (a *BaseAction) Print(v ...interface{}) (int, error) {
	return fmt.Fprint(a.setting.OutOrStdout(), v...)
}

// Printf 打印
func (a *BaseAction) Printf(f string, v ...interface{}) (int, error) {
	return fmt.Fprintf(a.setting.OutOrStdout(), f, v...)
}

// Println 打印
func (a *BaseAction) Println(f string, v ...interface{}) (int, error) {
	return a.Printf(f+"\n", v...)
}

// Scan 打印
func (a *BaseAction) Scan(v ...interface{}) (int, error) {
	return fmt.Fscan(a.setting.InOrStdin(), v...)
}

// Scanln 打印
func (a *BaseAction) Scanln(v ...interface{}) (int, error) {
	return fmt.Fscanln(a.setting.InOrStdin(), v...)
}

// Scanf 打印
func (a *BaseAction) Scanf(f string, v ...interface{}) (int, error) {
	return fmt.Fscanf(a.setting.InOrStdin(), f, v...)
}

// ErrorExit 错误并退出
func (a *BaseAction) ErrorExit(f string, v ...interface{}) {
	a.Printf(f+"\n", v...)
	os.Exit(1)
}

// GlobalSetting 获取全局配置
func (a *BaseAction) GlobalSetting() *GlobalSetting {
	return a.setting
}

// InitBase -
func InitBase() {
	var (
		bDatas []byte
		err    error
	)
	confBox, err := rice.FindBox("../asset/config")
	if err != nil {
		panic(err)
	}
	bDatas, err = confBox.Bytes("base.yaml")
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(bDatas, &BaseConfig)
	if err != nil {
		panic(err)
	}

	ArchetypeCatalog, err = confBox.Bytes("archetype-catalog.xml")
	if err != nil {
		panic(err)
	}

}
