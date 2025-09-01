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

	rice "github.com/GeertJohan/go.rice"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"io"
	"net"
	"os"
	"path"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"snz1.cn/snz1dp/snz1dpctl/docker"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

const (
	keepalivedConfigFileName       = "keepalived.yaml"
	defaultKeepavliedContainerName = "keepalived"
)

type keepalivedConfig struct {
	Inner               bool              `json:"-"`                      // 是否内置
	Name                string            `json:"name"`                   // 容器名称
	RouteID             string            `json:"router_id"`              // 不设置默认为Name
	VRRPVersion         uint32            `json:"vrrp_version"`           // 默认为2
	VRRPGarpMasterDelay uint32            `json:"vrrp_garp_master_delay"` // 默认为1
	VRRPMcastGroup4     string            `json:"vrrp_mcast_group4"`      // 默认为224.0.0.18
	VirtualInstances    []virtualInstance `json:"vrrp_instances"`         //
	VRRPCheck           struct {
		IP       string `json:"ip"`       // 检测IP
		Port     uint32 `json:"port"`     // 检测端口
		Timeout  uint32 `json:"timeout"`  // 默认为1秒
		Interval uint32 `json:"interval"` // 默认为3秒
		Fall     uint32 `json:"fall"`     // 默认为3次失败表示宕机
		Rise     uint32 `json:"rise"`     // 默认两次成功表示已启动
	} `json:"vrrp_check"`
}

func (ka *keepalivedConfig) addVirtualInstance(inst virtualInstance) (err error) {
	var (
		iface *net.Interface
	)

	if !(inst.State == "MASTER" || inst.State == "BACKUP") {
		err = errors.Errorf("error virtual instance state!")
		return
	}

	iface, err = net.InterfaceByName(inst.Interface)
	if err != nil || iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
		err = errors.Errorf("interface '%s' not existed or is loopback", inst.Interface)
		return
	}

	for _, v := range ka.VirtualInstances {
		if v.Name == inst.Name && inst.Name != "" {
			err = errors.Errorf("name of virtual instance existed!")
			return
		}
		if v.VirtualIP == inst.VirtualIP {
			err = errors.Errorf("ip of virtual instance existed!")
			return
		}
		if v.VirtualRouterID == inst.VirtualRouterID {
			err = errors.Errorf("router id of virtual instance existed!")
			return
		}
	}
	ka.VirtualInstances = append(ka.VirtualInstances, inst)
	return
}

func (ka *keepalivedConfig) removeVirtualInstance(virip string) (rmd *virtualInstance, err error) {
	var (
		insts []virtualInstance
	)
	for _, v := range ka.VirtualInstances {
		if v.VirtualIP == virip {
			rmd = &v
			continue
		}
		insts = append(insts, v)
	}
	if rmd == nil {
		err = errors.Errorf("not found virtual instance ip: %s", virip)
		return
	}

	ka.VirtualInstances = insts
	return
}

type virtualInstance struct {
	Name            string `json:"name"`              // 服务IP名称
	State           string `json:"state"`             // 当前节点状态
	Interface       string `json:"interface"`         // 绑定的网卡
	VirtualRouterID uint8  `json:"virtual_router_id"` // 虚拟路由实例🆔
	Priority        uint32 `json:"priority"`          // 优先级
	AdvertInterval  uint32 `json:"advert_int"`        // 默认1秒
	AuthPass        string `json:"auth_pass"`         // #认证方式为PASS，只前8位生效
	VirtualIP       string `json:"virtual_ip"`        // 虚拟服务IP
	Subnet          uint8  `json:"subnet"`            // 子网范围(默认24)
}

// StartKeepalived -
type StartKeepalived struct {
	BaseAction
	GenerateConfig bool
	ForcePullImage bool
	LoadImageLocal bool
}

// NewStartKeepalived -
func NewStartKeepalived(setting *GlobalSetting) *StartKeepalived {
	return &StartKeepalived{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// LogKeepalived -
type LogKeepalived struct {
	BaseAction
	Follow     bool
	Since      string
	Tail       string
	Timestamps bool
	Details    bool
	Until      string
}

// NewLogKeepalived -
func NewLogKeepalived(setting *GlobalSetting) *LogKeepalived {
	return &LogKeepalived{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// StopKeepalived -
type StopKeepalived struct {
	BaseAction
}

// NewStopKeepalived -
func NewStopKeepalived(setting *GlobalSetting) *StopKeepalived {
	return &StopKeepalived{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// AddVirtualIP -
type AddVirtualIP struct {
	BaseAction
	InstanceName    string // 服务IP名称
	State           string // 当前节点状态
	Interface       string // 绑定的网卡
	VirtualRouterID uint8  // 虚拟路由实例🆔
	Priority        uint32 // 优先级
	AdvertInterval  uint32 // 默认1秒
	AuthPass        string // #认证方式为PASS，只前8位生效
	VirtualIP       string // 虚拟服务IP
	Subnet          uint8  // 子网范围(默认24)
}

// NewAddVirtualIP -
func NewAddVirtualIP(setting *GlobalSetting) *AddVirtualIP {
	return &AddVirtualIP{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// ListVirtualIP -
type ListVirtualIP struct {
	BaseAction
}

// NewListVirtualIP -
func NewListVirtualIP(setting *GlobalSetting) *ListVirtualIP {
	return &ListVirtualIP{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// RemoveVirtualIP -
type RemoveVirtualIP struct {
	BaseAction
	VirtualIP []string
}

// NewRemoveVirtualIP -
func NewRemoveVirtualIP(setting *GlobalSetting) *RemoveVirtualIP {
	return &RemoveVirtualIP{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// ConfigLiveDetection -
type ConfigLiveDetection struct {
	BaseAction
	IP       string // 检测IP地址，默认127.0.0.1
	Port     uint32 // 检测端口
	Timeout  uint32 // 超时，默认3秒
	Interval uint32 // 间隔，默认3秒
	Fall     uint32 // 错误次数
	Rise     uint32 // 成功次数
}

// NewConfigLiveDetection -
func NewConfigLiveDetection(setting *GlobalSetting) *ConfigLiveDetection {
	return &ConfigLiveDetection{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

func loadKeepalivedConfig(setting *GlobalSetting) (confFile string, kc *keepalivedConfig, err error) {
	var (
		fst     os.FileInfo
		haData  []byte
		confBox *rice.Box
	)
	confFile = path.Join(setting.GetConfigDir(), keepalivedConfigFileName)

	if fst, err = os.Stat(confFile); err == nil && !fst.IsDir() {
		haData, err = os.ReadFile(confFile)
		if err != nil {
			return
		}
		kc = new(keepalivedConfig)
		err = yaml.Unmarshal(haData, kc)
		if err != nil {
			return
		}
		return
	}

	if fst != nil && fst.IsDir() {
		os.RemoveAll(confFile)
	}

	confBox, err = rice.FindBox("../asset/config")
	if err != nil {
		return
	}

	haData, _ = confBox.Bytes(keepalivedConfigFileName)
	kc = new(keepalivedConfig)
	kc.Inner = true
	err = yaml.Unmarshal(haData, kc)

	return
}

func (ka *keepalivedConfig) saveToFile(file string) (err error) {
	var (
		sdata []byte
	)
	sdata, err = yaml.Marshal(ka)
	if err != nil {
		return
	}

	os.MkdirAll(path.Dir(file), os.ModePerm)

	err = os.WriteFile(file, sdata, 0644)
	return
}

func (ka *keepalivedConfig) generateKeepalivedConfig() (cd []byte, shcd []byte, err error) {
	var (
		templateBox *rice.Box
		tpldata     string
		tbuf        *bytes.Buffer
		tpl         *template.Template
	)
	templateBox, err = rice.FindBox("template")
	if err != nil {
		return
	}
	// keepalived.conf
	tpldata, err = templateBox.String("keepalived.conf")
	if err != nil {
		return
	}

	tpl, err = template.New("keepalived.conf").Parse(tpldata)
	if err != nil {
		return
	}

	tbuf = bytes.NewBuffer(nil)
	err = tpl.Execute(tbuf, ka)
	if err != nil {
		return
	}

	cd = tbuf.Bytes()

	// check-svc-alived.sh
	tpldata, err = templateBox.String("check-svc-alived.sh")
	if err != nil {
		return
	}
	tpl, err = template.New("check-svc-alived.sh").Parse(tpldata)
	if err != nil {
		return
	}

	tbuf = bytes.NewBuffer(nil)
	err = tpl.Execute(tbuf, ka)
	if err != nil {
		return
	}

	shcd = tbuf.Bytes()

	return
}

func (ka *keepalivedConfig) verifyKeepalivedConfig() (sf bool, err error) {

	var (
		it    *virtualInstance
		iface *net.Interface
	)

	if ka.Name == "" {
		ka.Name = defaultKeepavliedContainerName
		sf = true
	}

	if ka.RouteID == "" {
		ka.RouteID = ka.Name
		sf = true
	}

	if ka.VRRPVersion == 0 {
		ka.VRRPVersion = 2
		sf = true
	}

	if ka.VRRPMcastGroup4 == "" {
		ka.VRRPMcastGroup4 = "224.0.0.18"
		sf = true
	}

	if ka.VRRPGarpMasterDelay == 0 {
		ka.VRRPGarpMasterDelay = 3
		sf = true
	}

	if ka.VRRPCheck.Interval == 0 {
		ka.VRRPCheck.Interval = 3
		sf = true
	}

	if ka.VRRPCheck.Fall == 0 {
		ka.VRRPCheck.Fall = 3
		sf = true
	}

	if ka.VRRPCheck.Rise == 0 {
		ka.VRRPCheck.Rise = 2
		sf = true
	}

	if ka.VRRPCheck.Timeout == 0 {
		ka.VRRPCheck.Timeout = 3
		sf = true
	}

	if ka.VRRPCheck.IP == "" {
		ka.VRRPCheck.IP = "127.0.0.1"
		sf = true
	}

	if ka.VRRPCheck.Port == 0 {
		err = errors.Errorf("not config live detection port!")
		return
	}

	for i := range ka.VirtualInstances {
		it = &(ka.VirtualInstances[i])
		if it.Name == "" {
			it.Name = fmt.Sprintf("vip%d", i)
			sf = true
		}

		if it.State == "" {
			it.State = "BACKUP"
		}

		if it.Interface == "" {
			err = errors.Errorf("%s(%s) no interface!", it.Name, it.VirtualIP)
			return
		}

		iface, err = net.InterfaceByName(it.Interface)
		if err != nil || iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			err = errors.Errorf("interface %s not existed or it is loopback", it.Interface)
			return
		}

		if it.AdvertInterval == 0 {
			it.AdvertInterval = 3
		}

		if it.AuthPass == "" {
			it.AuthPass = "snz1dp"
		}

		if it.VirtualIP == "" {
			err = errors.Errorf("%s(%s) no virtual ip of service!", it.Name, it.VirtualIP)
			return
		}

		if it.Subnet > 24 {
			it.Subnet = 24
		}

	}

	return
}

// Run -
func (s *StartKeepalived) Run() (err error) {
	setting := s.GlobalSetting()
	var (
		kaconf          string
		icfile          string
		ic              *InstallConfiguration
		kc              *keepalivedConfig
		sf              bool
		cfgdata         []byte
		shdata          []byte
		dc              *client.Client
		keepalivedImage string
		ct              types.Container
		ccid            string
		spinner         *utils.WaitSpinner
		keepaliveddir   string
		pxcfg           string
		alvsh           string
		cfginfo         os.FileInfo
		config          *container.Config
		hostConfig      *container.HostConfig
		cc              container.ContainerCreateCreatedBody
		img             *types.ImageSummary
	)

	icfile, ic, err = setting.LoadLocalInstallConfiguration()
	if err != nil {
		s.ErrorExit("load %s error: %v", icfile, err)
		return err
	}

	kaconf, kc, err = loadKeepalivedConfig(setting)
	if err != nil {
		s.ErrorExit("load config file %s error: %v", kaconf, err)
		return
	}

	sf, err = kc.verifyKeepalivedConfig()
	if err != nil {
		s.ErrorExit("keepalived config file %s error: %v", kaconf, err)
		return
	}

	if sf || kc.Inner {
		err = kc.saveToFile(kaconf)
		if err != nil {
			s.ErrorExit("save config file %s error: %v", kaconf, err)
			return
		}
	}

	cfgdata, shdata, err = kc.generateKeepalivedConfig()

	if err != nil {
		s.ErrorExit("%v", err)
		return
	}

	dc, err = docker.NewClient()
	if err != nil {
		s.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	keepalivedImage = fmt.Sprintf("%s/%s:%s", ic.Snz1dp.Registry.GetDockerRepoURL(), BaseConfig.Keepalived.ImageName, BaseConfig.Keepalived.ImageTag)

	ct, err = docker.ContainerExisted(dc, kc.Name)
	if err == nil {
		if ct.State != "exited" && ct.State != "created" {
			s.ErrorExit("%s is %s, id is %s", kc.Name, ct.State, ct.ID)
			return
		}
		ccid = ct.ID
	}

	img, err = docker.ImageExisted(dc, keepalivedImage)
	if s.ForcePullImage || img == nil {
		if s.LoadImageLocal {
			var imageTarFile = path.Join(setting.GetBundleDir(), fmt.Sprintf("keepalived-%s-IMAGES.tar", BaseConfig.Keepalived.Version))
			spinner = utils.NewSpinner(fmt.Sprintf("load %s image from %s...", keepalivedImage, imageTarFile), setting.OutOrStdout())
			_, err = docker.LoadImageFromFile(dc, imageTarFile)
			spinner.Close()
			if err != nil {
				s.ErrorExit("failed: %v", err.Error())
				return err
			}
			s.Println("ok!")
		} else {
			spinner = utils.NewSpinner(fmt.Sprintf("pull %s image...", keepalivedImage), setting.OutOrStdout())
			var repoUsernae, repoPassword string = ic.ResolveImageRepoUserAndPwd(keepalivedImage)
			err = docker.PullAndRenameImages(dc, keepalivedImage, "", repoUsernae, repoPassword, "")
			spinner.Close()
			if err != nil {
				s.ErrorExit("failed: %v", err.Error())
				return err
			}
			s.Println("ok!")
		}

	}

	keepaliveddir = path.Join(setting.GetBaseDir(), "run", "keepalived")
	os.MkdirAll(keepaliveddir, os.ModePerm)
	pxcfg = path.Join(keepaliveddir, "keepalived.conf")
	alvsh = path.Join(keepaliveddir, "check-svc-alived.sh")

	cfginfo, err = os.Stat(pxcfg)

	if err != nil || cfginfo.IsDir() {
		os.RemoveAll(pxcfg)
		s.GenerateConfig = true
		err = nil
	}

	cfginfo, err = os.Stat(alvsh)
	if err != nil || cfginfo.IsDir() {
		os.RemoveAll(alvsh)
		s.GenerateConfig = true
		err = nil
	}

	if s.GenerateConfig {
		err = os.WriteFile(pxcfg, cfgdata, 0644)
		if err != nil {
			s.ErrorExit("save keepalived config file %s error: %v", pxcfg, err)
			return
		}
		err = os.WriteFile(alvsh, shdata, os.ModePerm)
		if err != nil {
			s.ErrorExit("save keepalived check script %s error: %v", alvsh, err)
			return
		}
	}

	if ccid == "" {
		config = &container.Config{
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Image:        keepalivedImage,
		}

		hostConfig = &container.HostConfig{
			Binds: []string{
				keepaliveddir + ":" + "/etc/keepalived",
			},
			RestartPolicy: container.RestartPolicy{
				Name: "always",
			},
			Privileged:  true,
			NetworkMode: "host",
			CapAdd:      []string{"NET_ADMIN"},
		}

		cc, err = dc.ContainerCreate(context.Background(), config, hostConfig, nil, kc.Name)
		if err != nil {
			s.ErrorExit("create %s error: %v", kc.Name, err)
			return
		}

		ccid = cc.ID
	}

	err = dc.ContainerStart(context.Background(), ccid, types.ContainerStartOptions{})
	if err != nil {
		s.Println("start %s error: %s", kc.Name, err.Error())
		return
	}

	s.Println("start %s success, id is %s", kc.Name, ccid)
	if len(cc.Warnings) > 0 {
		s.Println("warnning: %v", cc.Warnings)
	}

	return
}

// Run -
func (l *LogKeepalived) Run() (err error) {

	setting := l.GlobalSetting()
	var (
		dc     *client.Client
		kaconf string
		kc     *keepalivedConfig
		oc     io.ReadCloser
		ct     types.Container
		c      types.ContainerJSON
	)

	kaconf, kc, err = loadKeepalivedConfig(setting)
	if err != nil {
		l.ErrorExit("load config file %s error: %v", kaconf, err)
		return
	}

	dc, err = docker.NewClient()
	if err != nil {
		l.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	ct, err = docker.ContainerExisted(dc, kc.Name)
	if err != nil {
		l.ErrorExit("%v", err)
		return
	}

	c, err = dc.ContainerInspect(context.Background(), ct.ID)
	if err != nil {
		l.ErrorExit("%v", err)
		return
	}

	oc, err = dc.ContainerLogs(context.Background(), ct.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     l.Follow,
		Since:      l.Since,
		Tail:       l.Tail,
		Timestamps: l.Timestamps,
		Details:    l.Details,
		Until:      l.Until,
	})

	if err != nil {
		l.ErrorExit("%v", err)
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

func showListInfo(s Action) (err error) {
	setting := s.GlobalSetting()
	var (
		kaconf        string
		kc            *keepalivedConfig
		dc            *client.Client
		ct            types.Container
		keepaliveddir string
	)

	keepaliveddir = path.Join(setting.GetBaseDir(), "run", "keepalived")
	kaconf, kc, err = loadKeepalivedConfig(setting)
	if err != nil {
		s.ErrorExit("load config file %s error: %v", kaconf, err)
		return
	}

	dc, err = docker.NewClient()
	if err != nil {
		s.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	s.Println("docker container    : %s", kc.Name)
	s.Println("config file         : %s", kaconf)
	s.Println("keepalived run dir  : %s", keepaliveddir)

	ct, err = docker.ContainerExisted(dc, kc.Name)
	if err != nil {
		s.Println("container state     : %s", "not existed")
		err = nil
	} else {
		s.Println("container state     : %s", ct.State)
	}

	s.Println("alive check ip      : %s", kc.VRRPCheck.IP)
	s.Println("alive check port    : %d", kc.VRRPCheck.Port)
	s.Println("alive check timeout : %d", kc.VRRPCheck.Timeout)
	s.Println("alive success rise  : %d", kc.VRRPCheck.Rise)
	s.Println("alive failed fall   : %d", kc.VRRPCheck.Fall)
	s.Println("virtual ip count    : %d", len(kc.VirtualInstances))

	s.Println("")
	for _, sv := range kc.VirtualInstances {
		s.Println("%s router id       : %d", sv.Name, sv.VirtualRouterID)
		s.Println("%s virtual ip      : %s/%d", sv.Name, sv.VirtualIP, sv.Subnet)
		s.Println("%s interface       : %s", sv.Name, sv.Interface)
		s.Println("%s priority        : %d", sv.Name, sv.Priority)
		s.Println("%s advert interval : %d", sv.Name, sv.AdvertInterval)
		s.Println("%s auth pass       : %s", sv.Name, sv.AuthPass)
		s.Println("")
	}
	return
}

// Run -
func (s *ListVirtualIP) Run() (err error) {
	return showListInfo(s)
}

// Run -
func (s *StopKeepalived) Run() (err error) {
	setting := s.GlobalSetting()
	var (
		kaconf  string
		kc      *keepalivedConfig
		dc      *client.Client
		ct      types.Container
		spinner *utils.WaitSpinner
		ccid    string
	)
	kaconf, kc, err = loadKeepalivedConfig(setting)
	if err != nil {
		s.ErrorExit("load config file %s error: %v", kaconf, err)
		return
	}

	dc, err = docker.NewClient()
	if err != nil {
		s.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	ct, err = docker.ContainerExisted(dc, kc.Name)
	if err != nil {
		s.ErrorExit("%s not running", kc.Name)
		return
	}

	ccid = ct.ID

	if ct.State != "exited" && ct.State != "created" {
		spinner = utils.NewSpinner(fmt.Sprintf("stop %s...", kc.Name), setting.OutOrStdout())
		dc.ContainerStop(context.Background(), ccid, nil)
		spinner.Close()
		s.Println("ok!")
	}

	dc.ContainerRemove(context.Background(), ccid, types.ContainerRemoveOptions{
		Force: true,
	})

	return
}

// Run -
func (s *ConfigLiveDetection) Run() (err error) {
	setting := s.GlobalSetting()

	var (
		kaconf string
		kc     *keepalivedConfig
	)
	kaconf, kc, err = loadKeepalivedConfig(setting)
	if err != nil {
		s.ErrorExit("load config file %s error: %v", kaconf, err)
		return
	}

	if s.IP != "" {
		kc.VRRPCheck.IP = s.IP
	} else if kc.VRRPCheck.IP == "" {
		kc.VRRPCheck.IP = "127.0.0.1"
	}

	if s.Port > 0 {
		kc.VRRPCheck.Port = s.Port
	} else if kc.VRRPCheck.Port == 0 {
		s.ErrorExit("error check port value!")
		return
	}

	kc.VRRPCheck.Timeout = s.Timeout
	kc.VRRPCheck.Interval = s.Interval
	kc.VRRPCheck.Fall = s.Fall
	kc.VRRPCheck.Rise = s.Rise

	err = kc.saveToFile(kaconf)
	if err != nil {
		s.ErrorExit("%v", err)
		return
	}

	s.Println("keepalived config file: %s", kaconf)
	s.Println("keepalived config success saved!")
	s.Println("")

	return showListInfo(s)
}

// Run -
func (s *RemoveVirtualIP) Run() (err error) {
	setting := s.GlobalSetting()
	var (
		kaconf string
		kc     *keepalivedConfig
		rmd    *virtualInstance
	)
	kaconf, kc, err = loadKeepalivedConfig(setting)
	if err != nil {
		s.ErrorExit("load config file %s error: %v", kaconf, err)
		return
	}

	for _, vip := range s.VirtualIP {

		rmd, err = kc.removeVirtualInstance(vip)
		if err != nil {
			s.Println("not found virtual ip %s", vip)
			continue
		}
		err = kc.saveToFile(kaconf)
		if err != nil {
			s.ErrorExit("%v", err)
			return
		}
		s.Println("unmount virtual ip %s on %s ok!", s.VirtualIP, rmd.Interface)
	}
	s.Println("")

	return showListInfo(s)
}

// Run -
func (s *AddVirtualIP) Run() (err error) {
	setting := s.GlobalSetting()
	var (
		kaconf string
		kc     *keepalivedConfig
	)
	kaconf, kc, err = loadKeepalivedConfig(setting)
	if err != nil {
		s.ErrorExit("load config file %s error: %v", kaconf, err)
		return
	}

	if s.State == "" {
		s.ErrorExit("error state value: %s!", s.State)
		return
	}

	if s.Interface == "" {
		s.ErrorExit("error interface value!")
		return
	}

	if s.VirtualIP == "" {
		s.ErrorExit("error virtualip value!")
		return
	}

	if s.AuthPass == "" {
		s.ErrorExit("error auth pass value!")
		return
	}

	if s.VirtualRouterID == 0 {
		s.ErrorExit("error router id, allowed values are 1-255!")
		return
	}

	err = kc.addVirtualInstance(virtualInstance{
		Name:            s.InstanceName,
		State:           s.State,
		Interface:       s.Interface,
		VirtualRouterID: s.VirtualRouterID,
		Priority:        s.Priority,
		AdvertInterval:  s.AdvertInterval,
		AuthPass:        s.AuthPass,
		VirtualIP:       s.VirtualIP,
		Subnet:          s.Subnet,
	})

	if err != nil {
		s.ErrorExit("%v", err)
	}

	err = kc.saveToFile(kaconf)
	if err != nil {
		s.ErrorExit("%v", err)
		return
	}

	s.Println("mount virtual ip %s on %s ok!", s.VirtualIP, s.Interface)
	s.Println("")

	return showListInfo(s)
}

// ListInterface -
type ListInterface struct {
	BaseAction
}

// NewListInterface -
func NewListInterface(setting *GlobalSetting) *ListInterface {
	return &ListInterface{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *ListInterface) Run() (err error) {

	var (
		ifaces []net.Interface
		addrs  []net.Addr
		sbuf   *bytes.Buffer
		aped   bool
	)

	ifaces, err = utils.GetAvaibleInterfaces()
	if err != nil {
		s.ErrorExit("%v", err)
		return
	}

	s.Println("interface count: %d", len(ifaces))
	s.Println("")

	for _, iface := range ifaces {
		addrs, err = iface.Addrs()
		sbuf = bytes.NewBuffer(nil)
		aped = false
		for _, addr := range addrs {
			if aped {
				fmt.Fprintf(sbuf, ", ")
			} else {
				aped = true
			}
			fmt.Fprintf(sbuf, "%s", addr.String())
		}

		s.Println("%s:\n  ether: %s\n  inet: %s", iface.Name, iface.HardwareAddr, sbuf.String())
	}

	return
}
