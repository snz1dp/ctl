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
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unsafe"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"snz1.cn/snz1dp/snz1dpctl/docker"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

const (
	// DefaultProfileName 配置名称
	DefaultProfileName = "profile"
	// AllBundleName 所有的
	AllBundleName = "all"
)

// StartStandaloneService -
type StartStandaloneService struct {
	BaseAction
	ServiceName    string
	ForcePullImage bool
	LoadImageLocal bool
	EnvVariables   []string
	ExtrasHosts    []string
	GPU            string
	Runtime        string
}

// NewStartStandaloneService -
func NewStartStandaloneService(setting *GlobalSetting) *StartStandaloneService {
	return &StartStandaloneService{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// MergeVolumeFiles -
func MergeVolumeFiles(ic *InstallConfiguration, volumeBinds []string, runFiles map[string]string) (ovs []string, rfs map[string]string, err error) {
	var tfs map[string]string = make(map[string]string)
	rfs = make(map[string]string)
	for k, v := range runFiles {
		if strings.TrimSpace(v) == "" {
			tfs[k] = "e"
			continue
		}
		v, _ = ic.RenderString(v)
		rfs[k] = v
		tfs[k] = "v"
	}

	var (
		ei, fi  int
		lf, ifn string
		efmap   map[string]string = make(map[string]string)
	)

	for _, v := range volumeBinds {
		ei = strings.Index(v, ":")
		lf = v[0:ei]
		ifn = v[(ei + 1):]
		fi = strings.Index(ifn, ":")
		if fi >= 0 {
			ifn = ifn[:fi]
		}

		if ifn == "" {
			continue
		}

		if efmap[ifn] != "" {
			continue
		}

		// 不是内置文件则
		if tfs[lf] == "" {
			ovs = append(ovs, v)
			continue
		}

		// 不存在
		if tfs[lf] == "e" {
			continue
		}
		ovs = append(ovs, v)
	}

	return
}

func (s *StartStandaloneService) mergeEnvVariables(envs []string, ic *InstallConfiguration) (ret []string) {
	var envMap map[string]string = make(map[string]string)

	for _, m := range s.EnvVariables {
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

		if mv == "" {
			envMap[mk] = "(empty)"
		} else {
			envMap[mk] = mv
		}
		ret = append(ret, fmt.Sprintf("%s=%s", mk, mv))
	}

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
				err error
			)

			if tpl, err = template.New("env").Parse(mv); err == nil {
				var tbuf *bytes.Buffer = bytes.NewBuffer(nil)
				if err = tpl.Execute(tbuf, ic); err == nil {
					mv = tbuf.String()
				}
			}

			ret = append(ret, fmt.Sprintf("%s=%s", mk, mv))
			envMap[mk] = mv
		}

	}
	return
}

type mapData map[string]interface{}

func (m *mapData) GetString(name string) (val string) {
	vnames := strings.Split(name, ".")
	var (
		ts interface{}
	)
	ts = *(*map[string]interface{})(unsafe.Pointer(m))
	for _, vname := range vnames {
		switch av := ts.(type) {
		case map[string]interface{}:
			ts = av[vname]
		default:
			return
		}
	}

	switch av := ts.(type) {
	case string:
		val = av
	case int64:
		val = strconv.FormatInt(av, 10)
	case float64:
		val = strconv.FormatFloat(av, 'f', -1, 64)
	}
	return
}

func readImageNameAndTag(fdata []byte, name, tag string) (outname, outtag string, err error) {
	var (
		vdata mapData
	)

	err = yaml.Unmarshal(fdata, &vdata)
	if err != nil {
		return
	}

	outname = vdata.GetString(name)
	outtag = vdata.GetString(tag)
	return
}

// Run -
func (s *StartStandaloneService) Run() (err error) {

	setting := s.GlobalSetting()

	//加载本地安装配置
	configFilePath, ic, err := setting.LoadLocalInstallConfiguration()
	if err != nil {
		s.ErrorExit("load %s error: %v", configFilePath, err)
		return err
	}

	var (
		compNames  []string
		components map[string]Component
	)

	// 核心组件
	if compNames, components, err = ic.GetBundleComponents(false); err != nil {
		s.ErrorExit("load bundle error: %s", err)
		return
	}

	var (
		standaloneComps []Component
		standaloneNames map[Component]string
		saverepo        bool = false
	)

	standaloneNames = make(map[Component]string)
	for _, k := range compNames {
		ccomp := components[k]
		if (s.ServiceName == DefaultProfileName || k == s.ServiceName) && ccomp.IsStandalone() && k != "istio" {
			if ccomp.BeInstall() {
				standaloneComps = append(standaloneComps, ccomp)
				standaloneNames[ccomp] = k
			}
		}
		var ch bool
		if ch, err = ccomp.PreInit(s); err != nil {
			s.ErrorExit("%s\nplease fix %s", err.Error(), configFilePath)
			return err
		} else if ch && !saverepo {
			saverepo = true
		}
	}

	// 保存
	if saverepo {
		err = setting.SaveLocalInstallConfiguration(ic, configFilePath)
		if err != nil {
			s.ErrorExit("save %s error: %s", configFilePath, err)
			return err
		}
		configFilePath, ic, err = setting.LoadLocalInstallConfiguration()
		if err != nil {
			s.ErrorExit("load %s error: %v", configFilePath, err)
			return err
		}
	}

	if len(standaloneComps) == 0 {
		s.ErrorExit("not found %s standalone service!", s.ServiceName)
		return
	}

	ic.Snz1dp.ExternalIP = utils.GetExternalIpv4()

	saverepo = false

	for _, comp := range standaloneComps {
		compName := standaloneNames[comp]
		var (
			standaloneConfig                    *StandaloneConfig
			imageName, imageTag, containerState string
			envVariables                        []string
			extrasHosts                         []string
			volumeBinds                         []string
			runFiles                            map[string]string
			healthcheck                         *container.HealthConfig
			runCmd                              []string
			initCmd                             []string
		)

		standaloneConfig, err = comp.LoadStandaloneConfig()

		// 设置jwt
		ic.Snz1dp.Jwt = comp.GetJwtConfig()

		if err != nil {
			s.ErrorExit("%s-%s load service config error: %s", comp.GetName(), comp.GetVersion(), err)
			return err
		}

		if standaloneConfig.Docker != nil {
			imageName, imageTag = standaloneConfig.Docker.Image, standaloneConfig.Docker.Tag
		} else {
			var vadata string
			vadata, _ = comp.GetConfigValues()
			if vadata, err = ic.RenderString(vadata); err != nil {
				s.ErrorExit("%s-%s render config error: %s", comp.GetName(), comp.GetVersion(), err)
			}
			imageName, imageTag, err = readImageNameAndTag([]byte(vadata), standaloneConfig.Image, standaloneConfig.Tag)
			if err != nil {
				s.ErrorExit("%s-%s docker image error: %s", comp.GetName(), comp.GetVersion(), err)
				return err
			}
		}

		volumeBinds, runFiles, err = MergeVolumeFiles(ic, standaloneConfig.Volumes, standaloneConfig.RunFiles)

		if err != nil {
			s.ErrorExit("%s-%s volumes error: %s", comp.GetName(), comp.GetVersion(), err)
			return err
		}

		extrasHosts = append(standaloneConfig.ExtraHosts, s.ExtrasHosts...)

		envVariables = append(standaloneConfig.Envs, s.EnvVariables...)
		if envVariables, err = ic.RenderEnvVariables(envVariables); err != nil {
			s.ErrorExit("%s-%s env error: %s\nenv: %v", comp.GetName(), comp.GetVersion(), err, envVariables)
			return
		}
		envVariables = s.mergeEnvVariables(envVariables, ic)
		standaloneConfig.Envs = envVariables

		if runCmd, err = ic.RenderStrings(standaloneConfig.Cmd); err != nil {
			s.ErrorExit("%s-%s run command error: %s\ncommand: %v", comp.GetName(), comp.GetVersion(), err, standaloneConfig.Cmd)
			return
		}

		initCmd = standaloneConfig.InitCmd

		if len(initCmd) == 0 && standaloneConfig.InitJob != nil && len(standaloneConfig.InitJob.Command) > 0 {
			initCmd = standaloneConfig.InitJob.Command
		}

		if initCmd, err = ic.RenderStrings(initCmd); err != nil {
			s.ErrorExit("%s-%s init command error: %s\ncommand: %v", comp.GetName(), comp.GetVersion(), err, initCmd)
		}

		healthcheck, err = standaloneConfig.getHealthcheck()
		if err != nil {
			s.ErrorExit("%s-%s health check error: %s", comp.GetName(), comp.GetVersion(), err)
			return err
		}

		gpu := standaloneConfig.GPU
		runtime := standaloneConfig.Runtime

		if s.GPU != "" {
			gpu = s.GPU
		}

		if s.Runtime != "" {
			runtime = s.Runtime
		}

		service := StandaloneService{
			ServiceName:    compName,
			Hostname:       standaloneConfig.Hostname,
			Version:        comp.GetVersion(),
			ImageName:      imageName,
			ImageTag:       imageTag,
			PortBinds:      standaloneConfig.Ports,
			ForcePullImage: s.ForcePullImage,
			LoadImageLocal: s.LoadImageLocal,
			VolumeBinds:    volumeBinds,
			VolumeOwners:   standaloneConfig.Owners,
			EnvVariables:   envVariables,
			Cmd:            runCmd,
			InitCmd:        initCmd,
			RunFiles:       runFiles,
			Healthcheck:    healthcheck,
			Ingress:        standaloneConfig.Ingress,
			GPU:            gpu,
			Runtime:        runtime,
			action:         s,
			ic:             ic,
			ExtraHosts:     extrasHosts,
		}

		if service.ServiceName == "ingress" {
			if containerState, _, _, err = service.State(); err == nil && containerState != "not running" {
				var response []byte
				response, err = service.execCmdAndReturn([]string{
					"bash",
					"-c",
					"echo ${KONG_ADMIN_KEY}",
				})
				if err != nil {
					s.ErrorExit("resolve private key of ingress error: %s", err)
				}
				var responseStr = string(response)
				if comp.GetJwtConfig().PrivateKey != responseStr {
					comp.GetJwtConfig().PrivateKey = responseStr
					comp.GetJwtConfig().RSAKey = strings.ReplaceAll(responseStr, "\\n", "\n")
					saverepo = true
				}
			}
		} else if service.ServiceName == "redis" {
			if containerState, _, _, err = service.State(); err == nil && containerState != "not running" {
				var response []byte
				response, err = service.execCmdAndReturn([]string{
					"sh",
					"-c",
					"echo ${REDIS_PASSWORD}",
				})
				if err != nil {
					s.ErrorExit("resolve redis password error: %s", err)
				}
				var responseStr = string(response)
				responseStr = responseStr[:len(responseStr)-1]
				if ic.Redis.Password != responseStr {
					ic.Redis.Password = responseStr
					ic.Redis.EncodedPassword = ""
					saverepo = true
				}
			}
		} else if service.ServiceName == "postgres" {
			if containerState, _, _, err = service.State(); err == nil && containerState != "not running" {
				var response []byte

				// 获取postgres密码
				response, err = service.execCmdAndReturn([]string{
					"bash",
					"-c",
					"echo ${POSTGRES_PASSWORD}",
				})
				if err != nil {
					s.ErrorExit("resolve postgres password error: %s", err)
				}
				var passwordStr = string(response)

				passwordStr = passwordStr[:len(passwordStr)-1]

				// 获取postgres用户
				response, err = service.execCmdAndReturn([]string{
					"bash",
					"-c",
					"echo ${POSTGRES_USER}",
				})
				if err != nil {
					s.ErrorExit("resolve postgres password error: %s", err)
				}
				var usernameStr = string(response)
				usernameStr = usernameStr[:len(usernameStr)-1]

				if ic.Postgres.Admin.Password != passwordStr || ic.Postgres.Admin.Username != usernameStr {
					ic.Postgres.Admin.Password = passwordStr
					ic.Postgres.Admin.Username = usernameStr
					ic.Postgres.Admin.EncodedPassword = ""
					saverepo = true
				}
			}
		}

		if err = service.Start(); err != nil {
			s.ErrorExit("start %s-%s error: %s", service.ServiceName, service.Version, err)
			return
		}

		if err = service.WaitHealthy(); err != nil {
			s.ErrorExit("wait %s-%s health error: %s", service.ServiceName, service.Version, err)
			return
		}

		if err = service.Init(); err != nil {
			s.ErrorExit("init %s-%s error: %v", service.ServiceName, service.Version, err)
			return
		}

		if ic.Appgateway.infile {
			if err = service.ApplyIngress(); err != nil {
				s.Println("apply %s-%s ingress error: %v", service.ServiceName, service.Version, err)
			}
		}

		if ic.Appgateway.infile {
			if service.ServiceName == "xeai" {
				if err = service.ApplyDefaultURL(); err != nil {
					s.ErrorExit("config %s-%s error: %v", service.ServiceName, service.Version, err)
				}
			}
		}

	}

	ic.Snz1dp.Jwt = nil

	// 保存
	err = setting.SaveLocalInstallConfiguration(ic, configFilePath)
	if err != nil {
		s.ErrorExit("save %s error: %s", configFilePath, err)
		return err
	}

	return
}

// ListStandaloneService -
type ListStandaloneService struct {
	BaseAction
}

// NewListStandaloneService -
func NewListStandaloneService(setting *GlobalSetting) *ListStandaloneService {
	return &ListStandaloneService{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *ListStandaloneService) Run() (err error) {

	setting := s.GlobalSetting()

	//加载本地安装配置
	configFilePath, ic, err := setting.LoadLocalInstallConfiguration()
	if err != nil {
		s.ErrorExit("load %s error: %v", configFilePath, err)
		return err
	}

	var (
		compNames  []string
		components map[string]Component
	)

	// 核心组件
	if compNames, components, err = ic.GetBundleComponents(false); err != nil {
		s.ErrorExit("load bundle error: %s", err)
	}

	var (
		saverepo        bool
		standaloneComps []Component
		standaloneNames map[Component]string
	)

	standaloneNames = make(map[Component]string)

	for _, k := range compNames {
		ccomp := components[k]
		if ccomp.IsStandalone() {
			if ccomp.BeInstall() {
				standaloneComps = append(standaloneComps, ccomp)
				standaloneNames[ccomp] = k
			}
		}

		ch, _ := ccomp.PreInit(s)
		if ch && !saverepo {
			saverepo = true
		}
	}

	// 保存
	if saverepo {
		err = setting.SaveLocalInstallConfiguration(ic, configFilePath)
		if err != nil {
			s.ErrorExit("save %s error: %s", configFilePath, err)
			return err
		}
	}

	if len(standaloneComps) == 0 {
		return
	}

	for _, comp := range standaloneComps {
		compName := standaloneNames[comp]
		service := StandaloneService{
			ServiceName: compName,
			Version:     comp.GetVersion(),
			action:      s,
		}
		var (
			state, cid, listen string
			ports              []string
		)
		state, cid, ports, err = service.State()
		if err != nil {
			s.ErrorExit("%v", err)
			return
		}
		if cid != "" {
			listen = strings.Join(ports, ",")
			if listen == "" {
				listen = "bind disabled"
			}
			s.Println("%s@%s port bind %s is %s, id=%s", compName, comp.GetVersion(), strings.Join(ports, ","), state, cid[:12])
		} else {
			s.Println("%s@%s is %s", compName, comp.GetVersion(), state)
		}
	}
	return
}

// StopStandaloneService -
type StopStandaloneService struct {
	BaseAction
	ServiceName string
	Really      bool
}

// NewStopStandaloneService -
func NewStopStandaloneService(setting *GlobalSetting) *StopStandaloneService {
	return &StopStandaloneService{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *StopStandaloneService) Run() (err error) {
	setting := s.GlobalSetting()

	//加载本地安装配置
	configFilePath, ic, err := setting.LoadLocalInstallConfiguration()
	if err != nil {
		s.ErrorExit("load %s error: %v", configFilePath, err)
		return err
	}

	var (
		compNames  []string
		components map[string]Component
	)

	// 核心组件
	if compNames, components, err = ic.GetBundleComponents(false); err != nil {
		s.ErrorExit("load bundle error: %s", err)
	}

	var (
		saverepo        bool
		standaloneComps []Component
		standaloneNames map[Component]string
		namesLen        int
	)

	standaloneNames = make(map[Component]string)
	for _, k := range compNames {
		ccomp := components[k]

		if (s.ServiceName == AllBundleName || s.ServiceName == DefaultProfileName || k == s.ServiceName) && ccomp.IsStandalone() && k != "istio" {
			if ccomp.BeInstall() || s.ServiceName == AllBundleName {
				standaloneComps = append(standaloneComps, ccomp)
				standaloneNames[ccomp] = k
			}
		}

		ch, _ := ccomp.PreInit(s)
		if ch && !saverepo {
			saverepo = true
		}
	}

	// 保存
	if saverepo {
		err = setting.SaveLocalInstallConfiguration(ic, configFilePath)
		if err != nil {
			s.ErrorExit("save %s error: %s", configFilePath, err)
			return err
		}
	}

	namesLen = len(standaloneComps)

	if namesLen == 0 {
		s.ErrorExit("not found %s standalone service!", s.ServiceName)
		return
	}

	if !s.Really && !utils.Confirm("will stop "+s.ServiceName+" standalone service, proceed? (y/N)", setting.InOrStdin(), setting.OutOrStdout()) {
		s.Println("Cancelled.")
		return nil
	}

	for i := 0; i < namesLen; i++ {
		var comp Component = standaloneComps[namesLen-(i+1)]
		var compName string = standaloneNames[comp]
		service := StandaloneService{
			ServiceName: compName,
			Version:     comp.GetVersion(),
			action:      s,
		}
		_ = service.Stop()
	}

	err = nil

	return
}

// LogStandaloneService -
type LogStandaloneService struct {
	BaseAction
	ServiceName string
	Follow      bool
	Since       string
	Tail        string
	Timestamps  bool
	Details     bool
	Until       string
}

// NewLogStandaloneService -
func NewLogStandaloneService(setting *GlobalSetting) *LogStandaloneService {
	return &LogStandaloneService{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *LogStandaloneService) Run() (err error) {
	setting := s.GlobalSetting()

	//加载本地安装配置
	configFilePath, ic, err := setting.LoadLocalInstallConfiguration()
	if err != nil {
		s.ErrorExit("load %s error: %v", configFilePath, err)
		return err
	}

	var (
		compNames  []string
		components map[string]Component
	)

	// 核心组件
	if compNames, components, err = ic.GetBundleComponents(false); err != nil {
		s.ErrorExit("load bundle error: %s", err)
	}

	var (
		comp Component
	)
	for _, k := range compNames {
		if k == s.ServiceName && components[k].IsStandalone() {
			comp = components[k]
			break
		}
	}

	if comp == nil {
		s.ErrorExit("not found %s standalone service!", s.ServiceName)
		return
	}

	service := StandaloneService{
		ServiceName: s.ServiceName,
		Version:     comp.GetVersion(),
		action:      s,
		Follow:      s.Follow,
		Since:       s.Since,
		Tail:        s.Tail,
		Timestamps:  s.Timestamps,
		Details:     s.Details,
		Until:       s.Until,
	}

	return service.Log()
}

// CleanStandaloneService -
type CleanStandaloneService struct {
	BaseAction
	ServiceName string
	Really      bool
}

// NewCleanStandaloneService -
func NewCleanStandaloneService(setting *GlobalSetting) *CleanStandaloneService {
	return &CleanStandaloneService{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *CleanStandaloneService) Run() (err error) {
	setting := s.GlobalSetting()

	stopService := NewStopStandaloneService(setting)
	stopService.ServiceName = s.ServiceName
	stopService.Really = s.Really
	stopService.Run()

	//加载本地安装配置
	configFilePath, ic, err := setting.LoadLocalInstallConfiguration()
	if err != nil {
		s.ErrorExit("load %s error: %v", configFilePath, err)
		return err
	}

	var (
		compNames  []string
		components map[string]Component
	)

	// 核心组件
	if compNames, components, err = ic.GetBundleComponents(false); err != nil {
		s.ErrorExit("load bundle error: %s", err)
	}

	var (
		saverepo        bool
		standaloneComps []Component
		standaloneNames map[Component]string
		namesLen        int
	)

	standaloneNames = make(map[Component]string)

	for _, k := range compNames {
		ccomp := components[k]
		if (s.ServiceName == AllBundleName || s.ServiceName == DefaultProfileName || k == s.ServiceName) && components[k].IsStandalone() && k != "istio" {
			if ccomp.BeInstall() || s.ServiceName == AllBundleName {
				standaloneComps = append(standaloneComps, ccomp)
				standaloneNames[ccomp] = k
			}
		}

		ch, _ := ccomp.PreInit(s)
		if ch && !saverepo {
			saverepo = true
		}
	}

	// 保存
	if saverepo {
		err = setting.SaveLocalInstallConfiguration(ic, configFilePath)
		if err != nil {
			s.ErrorExit("save %s error: %s", configFilePath, err)
			return err
		}
	}

	namesLen = len(standaloneComps)

	if namesLen == 0 {
		return
	}

	if !s.Really && !utils.Confirm("will clean "+s.ServiceName+" standalone service, proceed? (y/N)", setting.InOrStdin(), setting.OutOrStdout()) {
		s.Println("Cancelled.")
		return nil
	}

	for i := 0; i < namesLen; i++ {
		var comp Component = standaloneComps[namesLen-(i+1)]
		compName := standaloneNames[comp]
		service := StandaloneService{
			ServiceName: compName,
			Version:     comp.GetVersion(),
			action:      s,
		}
		service.Clean()
	}

	err = nil

	return
}

// StandaloneService -
type StandaloneService struct {
	Hostname       string
	BaseDir        string
	ServiceName    string
	Version        string
	ImageName      string
	ImageTag       string
	ForcePullImage bool
	LoadImageLocal bool
	PortBinds      []string
	VolumeBinds    []string
	VolumeOwners   []string
	EnvVariables   []string
	NetworkLinks   []string
	Cmd            []string
	InitCmd        []string
	RunFiles       map[string]string
	ExtraHosts     []string

	Healthcheck *container.HealthConfig
	Ingress     []*StandaloneIngressConfig

	Follow     bool
	Since      string
	Tail       string
	Timestamps bool
	Details    bool
	Until      string
	GPU        string
	Runtime    string

	ic *InstallConfiguration

	action Action
}

func (c *StandaloneConfig) getHealthcheck() (ckf *container.HealthConfig, err error) {
	if c.HealthCheck == nil {
		return
	}

	var (
		interval, timeout, period time.Duration
		commands                  []string
		si                        int
		envMap                    map[string]string
	)

	envMap = make(map[string]string)

	for _, v := range c.Envs {
		si = strings.Index(v, "=")
		if si <= 0 {
			continue
		}
		envMap[v[0:si]] = v[si+1:]
	}

	if c.HealthCheck.URL != "" {
		if tmpurl, tmperr := url.Parse(c.HealthCheck.URL); tmperr != nil || !tmpurl.IsAbs() {
			if len(c.Ingress) > 0 {
				commands = []string{
					"CMD",
					"curl",
					"-k",
					"localhost:" + strconv.Itoa(c.Ingress[0].GetBackendPort()) + c.HealthCheck.URL,
				}
			} else {
				commands = []string{
					"CMD",
					"curl",
					"-k",
					"localhost" + c.HealthCheck.URL,
				}
			}
		} else {
			commands = []string{
				"CMD",
				"curl",
				"-k",
				c.HealthCheck.URL,
			}
		}
	} else {
		for _, v := range c.HealthCheck.Test {
			commands = append(commands, os.Expand(v, func(key string) (ret string) {
				ret = envMap[key]
				return
			}))
		}
		if len(commands) > 0 {
			if commands[0] == "CMD-SHELL" {
				commands = commands[1:]
			}
			if commands[0] != "CMD" {
				commands = append([]string{"CMD"}, commands...)
			}
		} else {
			commands = []string{"CMD", "echo", "1"}
		}
	}

	interval, err = time.ParseDuration(c.HealthCheck.Interval)
	if err != nil {
		return
	}

	timeout, err = time.ParseDuration(c.HealthCheck.Timeout)
	if err != nil {
		return
	}

	period, err = time.ParseDuration(c.HealthCheck.StartPeriod)
	if err != nil {
		return
	}

	ckf = &container.HealthConfig{
		Test:        commands,
		Interval:    interval,
		Timeout:     timeout,
		StartPeriod: period,
		Retries:     c.HealthCheck.Retries,
	}
	return
}

// Start 启动
func (s *StandaloneService) Start() (err error) {
	setting := s.action.GlobalSetting()

	var (
		dc               *client.Client
		ct               types.Container
		ccid             string
		imageName        string
		img              *types.ImageSummary
		spinner          *utils.WaitSpinner
		config           *container.Config
		hostConfig       *container.HostConfig
		networkingConfig *network.NetworkingConfig
		portmap          nat.PortMap
		portset          nat.PortSet
		cc               container.ContainerCreateCreatedBody
		volumeBinds      []string
		runDir           string
		containerName    string
		networkID        string
		istoolbox        bool
		ic               *InstallConfiguration
		icfile           string
		runfiles         map[string]bool   = map[string]bool{}
		volumeOnwers     map[string]string = map[string]string{}
	)

	icfile, ic, err = setting.LoadLocalInstallConfiguration()
	if err != nil {
		s.action.ErrorExit("load %s error: %s", icfile, err)
		return err
	}

	dc, err = docker.NewClient()
	if err != nil {
		s.action.ErrorExit("%s", err)
		return
	}
	defer dc.Close()

	istoolbox, err = docker.IsDockerInToolBox(dc)
	if err != nil {
		s.action.ErrorExit("%s", err)
		return
	}

	containerName = s.ServiceName

	if s.BaseDir != "" {
		runDir = s.BaseDir
	} else {
		runDir = path.Join(setting.GetBaseDir(), "run", s.ServiceName)
	}

	if istoolbox {
		s.action.Println("WARN: please add share ~/.snz1dp/run to /home/docker/run in docker-machine and sudo mount -t vboxsf /home/docker/run")
	}

	for k := range s.RunFiles {
		runfiles[k] = true
	}

	for _, v := range s.VolumeOwners {
		ownerArray := strings.Split(v, "=")
		if len(ownerArray) == 2 {
			volumeOnwers[ownerArray[0]] = ownerArray[1]
		}
	}

	for _, v := range s.VolumeBinds {
		firstLetter := strings.Index(v, ":")
		if firstLetter < 0 {
			continue
		}
		if v[0:1] == "/" || len(v) > 1 && v[0:2] == "./" || len(v) > 2 && v[0:3] == "../" {
			volumeBinds = append(volumeBinds, v)
		} else {
			if istoolbox {
				ist := strings.Index(v, ":")
				if ist < 0 {
					continue
				}
				pv := v[:(ist)]
				lv := v[(ist + 1):]
				if runfiles[pv] {
					v = fmt.Sprintf("/home/docker/run/%s/%s:%s", s.ServiceName, pv, lv)
					s.action.Println("WARN: %s-%s use in docker-machone volume `/home/docker/run/%s/%s` for `%s`", s.ServiceName, s.Version, s.ServiceName, pv, lv)
				} else {
					v = fmt.Sprintf("/home/docker/data/%s/%s:%s", s.ServiceName, pv, lv)
				}
			} else {
				volumeName := v[0:firstLetter]
				v = fmt.Sprintf("%s/%s", runDir, v)
				vDir := fmt.Sprintf("%s/%s", runDir, volumeName)
				if volumeOnwers[volumeName] != "" && runtime.GOOS == "linux" {
					if !utils.FileExists(vDir) {
						os.MkdirAll(vDir, os.ModePerm)
						ownerArray := strings.Split(volumeOnwers[volumeName], ":")
						ownerUserId, _ := strconv.Atoi(ownerArray[0])
						ownerGroupId := 0
						if len(ownerArray) > 1 {
							ownerGroupId, _ = strconv.Atoi(ownerArray[1])
						}
						err = os.Chown(vDir, ownerUserId, ownerGroupId)
						if err != nil {
							s.action.Println("WARN: %s", vDir, err)
						}
					}
				}
			}
			volumeBinds = append(volumeBinds, v)
		}
	}

	ct, err = docker.ContainerExisted(dc, containerName)
	if err == nil {
		if ct.State != "exited" && ct.State != "created" {
			s.action.Println("%s-%s is %s, id=%s", s.ServiceName, s.Version, ct.State, ct.ID)
			return
		}
		ccid = ct.ID
	} else {
		os.MkdirAll(runDir, os.ModePerm)
		var tmp []byte
		for k, v := range s.RunFiles {
			fname := path.Join(runDir, k)
			var fst os.FileInfo
			fst, err = os.Stat(fname)
			if err != nil || fst.IsDir() {
				os.RemoveAll(fname)
			}
			if strings.HasPrefix(v, "base64://") {
				tmp, err = utils.Decode(v[9:])
				if err != nil {
					s.action.ErrorExit("%s", err)
					continue
				}
			} else {
				tmp = []byte(v)
			}
			err = os.WriteFile(fname, tmp, os.ModePerm)
			if err != nil {
				s.action.ErrorExit("%s", err)
				return
			}
		}
	}

	imageName = fmt.Sprintf("%s:%s", s.ImageName, s.ImageTag)

	img, err = docker.ImageExisted(dc, imageName)
	if s.ForcePullImage || img == nil {
		if s.LoadImageLocal {
			var imageTarFile = path.Join(setting.GetBundleDir(), fmt.Sprintf("%s-%s-IMAGES.tar", s.ServiceName, s.Version))
			spinner = utils.NewSpinner(fmt.Sprintf("load %s image from %s...", imageName, imageTarFile), setting.OutOrStdout())
			_, err = docker.LoadImageFromFile(dc, imageTarFile)
			spinner.Close()
			if err != nil {
				s.action.ErrorExit("failed: %v", err.Error())
				return
			}
			s.action.Println("ok!")
		} else {
			var repoUsername, repoPassword string = ic.ResolveImageRepoUserAndPwd(imageName)
			spinner = utils.NewSpinner(fmt.Sprintf("pull %s image...", imageName), setting.OutOrStdout())
			err = docker.PullAndRenameImages(dc, imageName, "", repoUsername, repoPassword, "")
			spinner.Close()
			if err != nil {
				s.action.ErrorExit("failed: %v", err.Error())
				return err
			}
			s.action.Println("ok!")
		}
	}

	if ccid == "" {

		portset, portmap, err = nat.ParsePortSpecs(s.PortBinds)

		if err != nil {
			s.action.ErrorExit("%v", err)
			return
		}

		envs := s.EnvVariables
		if s.GPU != "" {
			envs = append(envs, "NVIDIA_VISIBLE_DEVICES="+s.GPU)
		}

		config = &container.Config{
			Hostname:     s.Hostname,
			ExposedPorts: portset,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Image:        imageName,
			Env:          envs,
			Cmd:          s.Cmd,
			Healthcheck:  s.Healthcheck,
		}

		hostConfig = &container.HostConfig{
			Binds: volumeBinds,
			RestartPolicy: container.RestartPolicy{
				Name: "always",
			},
			PortBindings: portmap,
			Privileged:   true,
			ExtraHosts:   s.ExtraHosts,
			Runtime:      s.Runtime,
		}

		networkingConfig = &network.NetworkingConfig{
			EndpointsConfig: make(map[string]*network.EndpointSettings),
		}

		networkID, err = docker.NetworkExisted(dc, "snz1dp")
		if err != nil {
			networkID, err = docker.CreateNetwork(dc, "snz1dp")
			if err != nil {
				s.action.ErrorExit("%v", err)
				return err
			}
		}

		networkingConfig.EndpointsConfig["snz1dp"] = &network.EndpointSettings{
			Links:     s.NetworkLinks,
			NetworkID: networkID,
		}

		cc, err = dc.ContainerCreate(context.Background(), config, hostConfig, networkingConfig, containerName)
		if err != nil {
			s.action.ErrorExit("create %s@%s error: %v", s.ServiceName, s.Version, err)
			return
		}

		ccid = cc.ID
	}

	err = dc.ContainerStart(context.Background(), ccid, types.ContainerStartOptions{})
	if err != nil {
		s.action.Println("start %s@%s error: %s", s.ServiceName, s.Version, err.Error())
		return
	}

	s.action.Println("start %s@%s success, id=%s", s.ServiceName, s.Version, ccid[:12])
	if len(cc.Warnings) > 0 {
		s.action.Println("warnning: %v", cc.Warnings)
	}

	return
}

// Existed 已存在
func (s *StandaloneService) execCmdAndReturn(cmd []string) (resp []byte, err error) {
	var (
		dc            *client.Client
		ct            types.Container
		containerName string
		response      types.IDResponse
		execID        string
		execResp      types.HijackedResponse
	)

	containerName = s.ServiceName

	dc, err = docker.NewClient()
	if err != nil {
		s.action.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	ct, err = docker.ContainerExisted(dc, containerName)
	if err != nil {
		err = nil
		return
	}

	response, err = dc.ContainerExecCreate(context.Background(), ct.ID, types.ExecConfig{
		Privileged:   false,
		AttachStdout: true,
		Cmd:          cmd,
	})

	if err != nil {
		return
	}

	execID = response.ID
	if execID == "" {
		err = errors.Errorf("exec ID empty")
		return
	}

	execResp, err = dc.ContainerExecAttach(context.Background(), execID, types.ExecStartCheck{
		Detach: false,
		Tty:    false,
	})

	if err != nil {
		return
	}

	defer execResp.Close()

	resp, err = ioutil.ReadAll(execResp.Reader)
	if err != nil {
		return
	}
	resp = resp[8:]

	return
}

// Existed 已存在
func (s *StandaloneService) ExecCmd(cmd []string, interactive, detach, tty bool, env []string) (err error) {
	var (
		dc            *client.Client
		ct            types.Container
		containerName string
		response      types.IDResponse
		execID        string
		execResp      types.HijackedResponse
	)

	containerName = s.ServiceName

	dc, err = docker.NewClient()
	if err != nil {
		s.action.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	ct, err = docker.ContainerExisted(dc, containerName)
	if err != nil {
		return
	}

	response, err = dc.ContainerExecCreate(context.Background(), ct.ID, types.ExecConfig{
		Privileged:   false,
		AttachStdin:  interactive,
		AttachStdout: tty,
		AttachStderr: tty,
		Cmd:          cmd,
		Tty:          tty,
		Detach:       detach,
		Env:          env,
	})

	if err != nil {
		return
	}

	if detach {
		return
	}

	execID = response.ID
	if execID == "" {
		err = errors.Errorf("exec ID empty")
		return
	}

	execResp, err = dc.ContainerExecAttach(context.Background(), execID, types.ExecStartCheck{
		Detach: detach,
		Tty:    tty,
	})

	if err != nil {
		return
	}

	defer execResp.Close()

	setting := s.action.GlobalSetting()
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)
		errCh <- func() error {
			streamer := hijackedIOStreamer{
				inputStream:  io.ReadCloser(setting.InOrStdin()),
				outputStream: setting.OutOrStdout(),
				errorStream:  setting.ErrOrStderr(),
				resp:         execResp,
				tty:          tty,
				detachKeys:   "",
			}

			return streamer.stream(context.Background())
		}()
	}()

	if err = <-errCh; err != nil {
		err = errors.Errorf("Error hijack: %s", err)
		return
	}

	resp, err := dc.ContainerExecInspect(context.Background(), execID)
	if err != nil {
		err = errors.Errorf("%s", err)
		return
	}
	status := resp.ExitCode
	if status != 0 {
		err = errors.Errorf("status is %d", status)
	} else {
		err = nil
	}
	return
}

// Stop 停止
func (s *StandaloneService) Stop() (err error) {
	setting := s.action.GlobalSetting()
	var (
		dc            *client.Client
		ct            types.Container
		spinner       *utils.WaitSpinner
		ccid          string
		containerName string
	)

	containerName = s.ServiceName

	dc, err = docker.NewClient()
	if err != nil {
		s.action.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	ct, err = docker.ContainerExisted(dc, containerName)
	if err != nil {
		s.action.Println("%s-%s not running", s.ServiceName, s.Version)
		err = nil
		return
	}

	ccid = ct.ID

	if ct.State != "exited" && ct.State != "created" {
		spinner = utils.NewSpinner(fmt.Sprintf("stop %s-%s...", s.ServiceName, s.Version), setting.OutOrStdout())
		dc.ContainerStop(context.Background(), ccid, nil)
		spinner.Close()
		s.action.Println("ok!")
	}

	dc.ContainerRemove(context.Background(), ccid, types.ContainerRemoveOptions{
		Force: true,
	})

	return
}

// WaitHealthy -
func (s *StandaloneService) WaitHealthy() (err error) {
	setting := s.action.GlobalSetting()
	var (
		dc            *client.Client
		ct            types.Container
		containerName string
		spinner       *utils.WaitSpinner
		cj            types.ContainerJSON
	)
	dc, err = docker.NewClient()
	if err != nil {
		s.action.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	containerName = s.ServiceName
	spinner = utils.NewSpinner(fmt.Sprintf("wait %s-%s service healthy...", s.ServiceName, s.Version), setting.OutOrStdout())

	for {
		ct, err = docker.ContainerExisted(dc, containerName)
		if err != nil {
			spinner.Close()
			s.action.Println("failed: %v", err)
			return
		}
		cj, err = dc.ContainerInspect(context.Background(), ct.ID)
		if err != nil {
			spinner.Close()
			s.action.Println("failed: %v", err)
			return
		}
		if cj.State.Health == nil || cj.State.Health.Status == types.Healthy {
			break
		}
		time.Sleep(3 * time.Second)
	}

	spinner.Close()
	s.action.Println("ok!")

	return
}

// Init 状态
func (s *StandaloneService) Init() (err error) {
	if len(s.InitCmd) == 0 {
		return
	}
	setting := s.action.GlobalSetting()

	var (
		spinner       *utils.WaitSpinner
		dc            *client.Client
		ct            types.Container
		containerName string
		response      types.IDResponse
		execID        string
		execResp      types.HijackedResponse
		resp          []byte
	)
	dc, err = docker.NewClient()
	if err != nil {
		return
	}
	defer dc.Close()

	containerName = s.ServiceName

	ct, err = docker.ContainerExisted(dc, containerName)
	if err != nil {
		return
	}

	spinner = utils.NewSpinner(fmt.Sprintf("execute %s-%s init command...", s.ServiceName, s.Version), setting.OutOrStdout())

	response, err = dc.ContainerExecCreate(context.Background(), ct.ID, types.ExecConfig{
		Privileged:   false,
		Tty:          false,
		AttachStdout: true,
		Cmd:          s.InitCmd,
		Detach:       false,
	})

	if err != nil {
		spinner.Close()
		s.action.Println("failed: %v", err)
		return
	}

	execID = response.ID
	if execID == "" {
		spinner.Close()
		s.action.Println("failed: %v", err)
		err = errors.Errorf("exec ID empty")
		return
	}

	execResp, err = dc.ContainerExecAttach(context.Background(), execID, types.ExecStartCheck{
		Detach: false,
		Tty:    false,
	})

	if err != nil {
		spinner.Close()
		s.action.Println("failed: %v", err)
		return
	}

	defer execResp.Close()

	resp, err = ioutil.ReadAll(execResp.Reader)

	if err != nil {
		spinner.Close()
		s.action.Println("failed: %v", err)
		return
	}
	spinner.Close()
	if len(resp) > 8 {
		s.action.Println("ok:")
		resp = resp[8:]
		s.action.Println("%s", string(resp))
	} else {
		s.action.Println("failed!")
	}

	return
}

// State 状态
func (s *StandaloneService) State() (state string, cid string, ports []string, err error) {
	var (
		dc            *client.Client
		ct            types.Container
		containerName string
	)
	dc, err = docker.NewClient()
	if err != nil {
		s.action.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	containerName = s.ServiceName

	ct, err = docker.ContainerExisted(dc, containerName)
	if err != nil {
		err = nil
		state = "not running"
		return
	}

	state = ct.State
	cid = ct.ID
	inports := map[uint16]bool{}
	for _, port := range ct.Ports {
		if inports[port.PublicPort] {
			continue
		}
		inports[port.PublicPort] = true
		ports = append(ports, fmt.Sprintf("%d:%d", port.PublicPort, port.PrivatePort))
	}
	return
}

// Clean 显示日志
func (s *StandaloneService) Clean() (err error) {
	setting := s.action.GlobalSetting()
	var (
		dc            *client.Client
		ct            types.Container
		containerName string
		ccid          string
		spinner       *utils.WaitSpinner
		runDir        string
	)
	dc, err = docker.NewClient()
	if err != nil {
		s.action.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	containerName = s.ServiceName

	ct, err = docker.ContainerExisted(dc, containerName)
	if err == nil {
		ccid = ct.ID

		if ct.State != "exited" && ct.State != "created" {
			spinner = utils.NewSpinner(fmt.Sprintf("stop %s-%s...", s.ServiceName, s.Version), setting.OutOrStdout())
			dc.ContainerStop(context.Background(), ccid, nil)
			spinner.Close()
			s.action.Println("ok!")
		}

		dc.ContainerRemove(context.Background(), ccid, types.ContainerRemoveOptions{
			Force: true,
		})
	}
	err = nil

	if s.BaseDir != "" {
		runDir = s.BaseDir
	} else {
		runDir = path.Join(setting.GetBaseDir(), "run", s.ServiceName)
	}

	os.RemoveAll(runDir)
	s.action.Println("clean %s-%s run files ok!", s.ServiceName, s.Version)
	return
}

// Log 显示日志
func (s *StandaloneService) Log() (err error) {
	setting := s.action.GlobalSetting()
	var (
		dc            *client.Client
		oc            io.ReadCloser
		ct            types.Container
		containerName string
		c             types.ContainerJSON
	)

	dc, err = docker.NewClient()
	if err != nil {
		s.action.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	containerName = s.ServiceName

	ct, err = docker.ContainerExisted(dc, containerName)
	if err != nil {
		s.action.ErrorExit("%v", err)
		return
	}

	c, err = dc.ContainerInspect(context.Background(), ct.ID)
	if err != nil {
		s.action.ErrorExit("%v", err)
		return
	}

	oc, err = dc.ContainerLogs(context.Background(), ct.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     s.Follow,
		Since:      s.Since,
		Tail:       s.Tail,
		Timestamps: s.Timestamps,
		Details:    s.Details,
		Until:      s.Until,
	})

	if err != nil {
		s.action.ErrorExit("%v", err)
		return
	}

	defer oc.Close()

	if c.Config.Tty {
		_, err = io.Copy(setting.OutOrStdout(), oc)
	} else {
		_, err = stdcopy.StdCopy(setting.OutOrStdout(), setting.ErrOrStderr(), oc)
	}

	return
}

type ExecStandaloneService struct {
	BaseAction
	ServiceName string
	Detach      bool
	Interactive bool
	Tty         bool
	Cmd         []string
	Env         []string
}

// NewStandaloneService -
func NewExecStandaloneService(setting *GlobalSetting) *ExecStandaloneService {
	return &ExecStandaloneService{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *ExecStandaloneService) Run() (err error) {
	setting := s.GlobalSetting()

	//加载本地安装配置
	configFilePath, ic, err := setting.LoadLocalInstallConfiguration()
	if err != nil {
		s.ErrorExit("load %s error: %v", configFilePath, err)
		return err
	}

	var (
		compNames  []string
		components map[string]Component
	)

	// 核心组件
	if compNames, components, err = ic.GetBundleComponents(false); err != nil {
		s.ErrorExit("load bundle error: %s", err)
	}

	var (
		comp Component
	)
	for _, k := range compNames {
		if k == s.ServiceName && components[k].IsStandalone() {
			comp = components[k]
			break
		}
	}

	if comp == nil {
		s.ErrorExit("not found %s standalone service!", s.ServiceName)
		return
	}

	service := StandaloneService{
		ServiceName: s.ServiceName,
		Version:     comp.GetVersion(),
		action:      s,
	}

	err = service.ExecCmd(s.Cmd, s.Interactive, s.Detach, s.Tty, s.Env)
	return
}
