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
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"

	"github.com/docker/go-connections/nat"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"snz1.cn/snz1dp/snz1dpctl/docker"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

const (
	haproxyConfigFileName       = "haproxy.yaml"
	defaultHaproxyContainerName = "haproxy"
)

// StartHaproxy -
type StartHaproxy struct {
	BaseAction
	GenerateHaproxyCfg bool
	ForcePullImage     bool
	LoadImageLocal     bool
}

// AddProxyService -
type AddProxyService struct {
	BaseAction
	AccessProxy bool
	SendProxy   bool
	Mode        string
	ServiceName string
	Port        int
	Inter       int
	Rise        int
	Fall        int
	Balance     string
	Backends    []string
}

// AddBackend -
type AddBackend struct {
	BaseAction
	Service     string
	BackendName string
	SendProxy   bool
	IP          string
	Port        int
}

// NewAddBackend -
func NewAddBackend(setting *GlobalSetting) *AddBackend {
	return &AddBackend{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// RemoveBackend -
type RemoveBackend struct {
	BaseAction
	Service string
	IP      string
	Port    int
}

// NewRemoveBackend -
func NewRemoveBackend(setting *GlobalSetting) *RemoveBackend {
	return &RemoveBackend{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// RemoveProxyService -
type RemoveProxyService struct {
	BaseAction
	ServiceName []string
}

// StopHaproxy -
type StopHaproxy struct {
	BaseAction
}

// ListHaproxy -
type ListHaproxy struct {
	BaseAction
}

type backendService struct {
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Port      uint32 `json:"port"`
	SendProxy bool   `json:"send-proxy"`
	Inter     uint32 `json:"inter"`
	Rise      uint32 `json:"rise"`
	Fall      uint32 `json:"fall"`
	Weight    uint32 `json:"weight"`
}

type proxyService struct {
	AccessProxy bool             `json:"access-proxy"`
	Name        string           `json:"name"`
	Mode        string           `json:"mode"`
	Port        uint32           `json:"port"`
	Balance     string           `json:"balance"`
	Backends    []backendService `json:"backends"`
}

type haproxyConfig struct {
	Inner   bool   `json:"-"`
	Name    string `json:"name"`
	Maxconn uint32 `json:"maconn"`
	Ulimit  uint32 `json:"ulimit"`
	Mode    string `json:"mode"`
	Balance string `json:"balance"`
	Stats   struct {
		Port uint32 `json:"port"`
		URI  string `json:"uri"`
		Auth struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"auth"`
	} `json:"stats"`
	Services []proxyService `json:"services"`
}

func (ha *haproxyConfig) addService(sc proxyService) (err error) {
	for _, sv := range ha.Services {
		if sc.Name == sv.Name {
			err = errors.Errorf("name %s existed!", sc.Name)
			return
		} else if sc.Port == sv.Port {
			err = errors.Errorf("port %s existed!", sc.Name)
			return
		}
	}
	ha.Services = append(ha.Services, sc)
	return
}

func (ha *haproxyConfig) removeService(name string) (rved bool) {
	var ss []proxyService
	for _, sv := range ha.Services {
		if name == sv.Name {
			rved = true
			continue
		}
		ss = append(ss, sv)
	}
	ha.Services = ss
	return
}

func (sv *proxyService) addBackend(ba backendService) (err error) {
	var (
		va *backendService
	)
	for i := range sv.Backends {
		va = &(sv.Backends[i])
		if va.Name == ba.Name && ba.Name != "" {
			err = errors.Errorf("existed backend name %s", ba.Name)
			return
		}
		if va.IP == ba.IP && (va.Port == ba.Port || va.Port == 0 && ba.Port == 80 || va.Port == 80 && ba.Port == 0) {
			err = errors.Errorf("existed backend %s:%d", ba.Name, ba.Port)
			return
		}
	}

	sv.Backends = append(sv.Backends, ba)
	return
}

func (sv *proxyService) removeBackend(host string, port uint32) (rved bool) {
	var ss []backendService
	for _, va := range sv.Backends {
		if host == va.IP && (va.Port == port || va.Port == 0 && port == 80 || va.Port == 80 && port == 0) {
			rved = true
			continue
		}
		ss = append(ss, va)
	}
	sv.Backends = ss
	return
}

func (ha *haproxyConfig) addBackend(name string, ba backendService) (err error) {
	var (
		sv *proxyService
	)
	for i := range ha.Services {
		sv = &(ha.Services[i])
		if name == sv.Name {
			return sv.addBackend(ba)
		}
	}
	err = errors.Errorf("not found proxy service %s", name)
	return
}

func (ha *haproxyConfig) removeBackend(name string, host string, port uint32) (err error) {
	var (
		sv *proxyService
	)
	for i := range ha.Services {
		sv = &(ha.Services[i])
		if name == sv.Name {
			if !sv.removeBackend(host, uint32(port)) {
				err = errors.Errorf("not found proxy service %s backend %s:%d", name, host, port)
			}
			return
		}
	}
	err = errors.Errorf("not found proxy service %s", name)
	return
}

func (ha *haproxyConfig) portList() (portlst []string) {
	portlst = append(portlst, fmt.Sprintf("%d:%d", ha.Stats.Port, ha.Stats.Port))
	for _, sv := range ha.Services {
		portlst = append(portlst, fmt.Sprintf("%d:%d", sv.Port, sv.Port))
	}
	return
}

func (ha *haproxyConfig) saveToFile(file string) (err error) {
	var (
		sdata []byte
	)
	sdata, err = yaml.Marshal(ha)
	if err != nil {
		return
	}
	os.MkdirAll(path.Dir(file), os.ModePerm)
	return os.WriteFile(file, sdata, 0644)
}

// NewStartHaproxy -
func NewStartHaproxy(setting *GlobalSetting) *StartHaproxy {
	return &StartHaproxy{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// NewAddProxyService -
func NewAddProxyService(setting *GlobalSetting) *AddProxyService {
	return &AddProxyService{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// NewRemoveProxyService -
func NewRemoveProxyService(setting *GlobalSetting) *RemoveProxyService {
	return &RemoveProxyService{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// NewStopHaproxy -
func NewStopHaproxy(setting *GlobalSetting) *StopHaproxy {
	return &StopHaproxy{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// NewListHaproxy -
func NewListHaproxy(setting *GlobalSetting) *ListHaproxy {
	return &ListHaproxy{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

func loadHaproxyConfig(setting *GlobalSetting) (haconf string, hc *haproxyConfig, err error) {
	var (
		fst     os.FileInfo
		haData  []byte
		confBox *rice.Box
	)
	haconf = path.Join(setting.GetConfigDir(), haproxyConfigFileName)

	if fst, err = os.Stat(haconf); err == nil && !fst.IsDir() {
		haData, err = os.ReadFile(haconf)
		if err != nil {
			return
		}
		hc = new(haproxyConfig)
		err = yaml.Unmarshal(haData, hc)
		if err != nil {
			return
		}
		return
	}

	if fst != nil && fst.IsDir() {
		os.RemoveAll(haconf)
	}

	confBox, err = rice.FindBox("../asset/config")
	if err != nil {
		return
	}

	haData, _ = confBox.Bytes(haproxyConfigFileName)
	hc = new(haproxyConfig)
	hc.Inner = true
	err = yaml.Unmarshal(haData, hc)

	return
}

func (ha *haproxyConfig) verifyHaproxyConfig() (save bool, err error) {
	hc := ha

	if hc.Name == "" {
		hc.Name = defaultHaproxyContainerName
		save = true
	}

	if hc.Mode == "" {
		hc.Mode = "http"
		save = true
	}

	if hc.Maxconn == 0 {
		hc.Maxconn = 8192
		save = true
	}

	if !(hc.Mode == "tcp" || hc.Mode == "http") {
		err = errors.Errorf("error mode %s, only tcp or http", hc.Mode)
		return
	}

	if hc.Balance == "" {
		hc.Balance = "source"
		save = true
	}

	if !(hc.Balance == "source" || hc.Balance == "roundrobin") {
		err = errors.Errorf("error balance %s, only roundrobin or source", hc.Balance)
		return
	}

	if hc.Stats.Port == 0 {
		hc.Stats.Port = 8081
		save = true
	}

	if hc.Stats.Auth.Username == "" {
		hc.Stats.Auth.Username = "haproxy"
		save = true
	}

	for i := range hc.Services {
		v := &(hc.Services[i])
		if v.Mode == "" {
			v.Mode = "http"
			save = true
		}

		if v.Balance == "" {
			v.Balance = "source"
			save = true
		}

		if v.Name == "" {
			v.Name = fmt.Sprintf("%s-%d", v.Name, i)
			save = true
		}

		if !(v.Balance == "source" || v.Balance == "roundrobin") {
			err = errors.Errorf("error balance %s, only roundrobin or source", hc.Balance)
			return
		}

		if v.Port == 0 {
			err = errors.Errorf("%s must have port", v.Name)
			return
		}

		for j := range v.Backends {
			b := &(v.Backends[i])
			if b.Name == "" {
				b.Name = fmt.Sprintf("%s-backend-%d", v.Name, j)
				save = true
			}

			if b.Port == 0 {
				err = errors.Errorf("%s must have port", b.Name)
				return
			}

			if b.Inter == 0 {
				b.Inter = 1500
				save = true
			}

			if b.Rise == 0 {
				b.Rise = 3
				save = true
			}

			if b.Fall == 0 {
				b.Fall = 3
				save = true
			}
		}
	}

	return
}

func (ha *haproxyConfig) generateHaproxyConfig() (cd []byte, err error) {
	hc := ha
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
	tpldata, err = templateBox.String("haproxy.cfg")
	if err != nil {
		return
	}

	tpl, err = template.New("haproxy.cfg").Parse(tpldata)
	if err != nil {
		return
	}
	tbuf = bytes.NewBuffer(nil)
	err = tpl.Execute(tbuf, hc)
	cd = tbuf.Bytes()
	return
}

// Run -
func (s *StartHaproxy) Run() (err error) {
	setting := s.GlobalSetting()
	var (
		haconf           string
		hc               *haproxyConfig
		dc               *client.Client
		ct               types.Container
		sf               bool
		cfgdata          []byte
		config           *container.Config
		hostConfig       *container.HostConfig
		networkingConfig *network.NetworkingConfig
		cc               container.ContainerCreateCreatedBody
		pxcfg            string
		haproxydir       string
		ccid             string
		portmap          nat.PortMap
		portset          nat.PortSet
		spinner          *utils.WaitSpinner
		ic               *InstallConfiguration
		haproxyImage     string
		cfginfo          os.FileInfo
		icfile           string
		img              *types.ImageSummary
		networkID        string
	)

	icfile, ic, err = setting.LoadLocalInstallConfiguration()

	if err != nil {
		s.ErrorExit("load %s error: %v", icfile, err)
		return err
	}

	haconf, hc, err = loadHaproxyConfig(setting)
	if err != nil {
		s.ErrorExit("load config file %s error: %v", haconf, err)
		return
	}

	sf, err = hc.verifyHaproxyConfig()
	if err != nil {
		s.ErrorExit("haproxy config file %s error: %v", haconf, err)
		return
	}

	if sf || hc.Inner {
		err = hc.saveToFile(haconf)
		if err != nil {
			s.ErrorExit("save config file %s error: %v", haconf, err)
			return
		}
	}

	cfgdata, err = hc.generateHaproxyConfig()

	if err != nil {
		s.ErrorExit("%v", err)
		return
	}

	portset, portmap, err = nat.ParsePortSpecs(hc.portList())

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

	haproxyImage = fmt.Sprintf("%s/%s:%s", ic.Snz1dp.Registry.GetDockerRepoURL(), BaseConfig.Haproxy.ImageName, BaseConfig.Haproxy.ImageTag)

	ct, err = docker.ContainerExisted(dc, hc.Name)
	if err == nil {
		if ct.State != "exited" && ct.State != "created" {
			s.ErrorExit("%s is %s, id is %s", hc.Name, ct.State, ct.ID)
			return
		}
		ccid = ct.ID
	}

	img, err = docker.ImageExisted(dc, haproxyImage)
	if s.ForcePullImage || img == nil {
		if s.LoadImageLocal {
			var (
				imageTarfile string = path.Join(setting.GetBundleDir(), fmt.Sprintf("haproxy-%s-IMAGES.tar", BaseConfig.Haproxy.Version))
			)
			spinner = utils.NewSpinner(fmt.Sprintf("load %s image from %s...", haproxyImage, imageTarfile), setting.OutOrStdout())
			_, err = docker.LoadImageFromFile(dc, imageTarfile)
			spinner.Close()
			if err != nil {
				s.ErrorExit("failed: %v", err.Error())
				return
			}
			s.Println("ok!")
		} else {
			var (
				repoUsername, repoPassword string = ic.ResolveImageRepoUserAndPwd(haproxyImage)
			)
			spinner = utils.NewSpinner(fmt.Sprintf("pull %s image...", haproxyImage), setting.OutOrStdout())
			err = docker.PullAndRenameImages(dc, haproxyImage, "", repoUsername, repoPassword, "")
			spinner.Close()
			if err != nil {
				s.ErrorExit("failed: %v", err.Error())
				return err
			}
			s.Println("ok!")
		}
	}

	haproxydir = path.Join(setting.GetBaseDir(), "run", "haproxy")
	os.MkdirAll(haproxydir, os.ModePerm)
	pxcfg = path.Join(haproxydir, "haproxy.cfg")

	cfginfo, err = os.Stat(pxcfg)

	if err != nil || cfginfo.IsDir() {
		os.RemoveAll(pxcfg)
		s.GenerateHaproxyCfg = true
		err = nil
	}

	if s.GenerateHaproxyCfg {
		err = os.WriteFile(pxcfg, cfgdata, os.ModePerm)
		if err != nil {
			s.ErrorExit("save haproxy config file %s error: %v", pxcfg, err)
			return
		}
	}

	if ccid == "" {
		config = &container.Config{
			ExposedPorts: portset,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Image:        haproxyImage,
		}

		hostConfig = &container.HostConfig{
			Binds: []string{
				haproxydir + ":" + "/usr/local/etc/haproxy",
			},
			RestartPolicy: container.RestartPolicy{
				Name: "always",
			},
			PortBindings: portmap,
			Privileged:   true,
		}

		networkingConfig = &network.NetworkingConfig{
			EndpointsConfig: make(map[string]*network.EndpointSettings),
		}

		networkID, err = docker.NetworkExisted(dc, "snz1dp")
		if err != nil {
			networkID, err = docker.CreateNetwork(dc, "snz1dp")
			if err != nil {
				s.ErrorExit("%v", err)
				return err
			}
		}

		networkingConfig.EndpointsConfig["snz1dp"] = &network.EndpointSettings{
			NetworkID: networkID,
		}

		cc, err = dc.ContainerCreate(context.Background(), config, hostConfig, networkingConfig, hc.Name)
		if err != nil {
			s.ErrorExit("create %s error: %v", hc.Name, err)
			return
		}

		ccid = cc.ID
	}

	err = dc.ContainerStart(context.Background(), ccid, types.ContainerStartOptions{})
	if err != nil {
		s.Println("start %s error: %s", hc.Name, err.Error())
		return
	}

	s.Println("start %s success, id is %s", hc.Name, ccid)
	if len(cc.Warnings) > 0 {
		s.Println("warnning: %v", cc.Warnings)
	}

	return
}

// Run -
func (s *StopHaproxy) Run() (err error) {
	setting := s.GlobalSetting()
	var (
		haconf  string
		hc      *haproxyConfig
		dc      *client.Client
		ct      types.Container
		spinner *utils.WaitSpinner
		ccid    string
	)
	haconf, hc, err = loadHaproxyConfig(setting)
	if err != nil {
		s.ErrorExit("load config file %s error: %v", haconf, err)
		return
	}

	dc, err = docker.NewClient()
	if err != nil {
		s.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	ct, err = docker.ContainerExisted(dc, hc.Name)
	if err != nil {
		s.ErrorExit("%s not running", hc.Name)
		return
	}

	ccid = ct.ID

	if ct.State != "exited" && ct.State != "created" {
		spinner = utils.NewSpinner(fmt.Sprintf("stop %s...", hc.Name), setting.OutOrStdout())
		dc.ContainerStop(context.Background(), ccid, nil)
		spinner.Close()
		s.Println("ok!")
	}

	dc.ContainerRemove(context.Background(), ccid, types.ContainerRemoveOptions{
		Force: true,
	})

	return
}

func showHaproxyInfo(s Action) (err error) {
	setting := s.GlobalSetting()

	var (
		haconf     string
		hc         *haproxyConfig
		dc         *client.Client
		ct         types.Container
		bt         *bytes.Buffer
		ap         bool
		haproxydir string
	)

	haproxydir = path.Join(setting.GetBaseDir(), "run", "haproxy")
	haconf, hc, err = loadHaproxyConfig(setting)
	if err != nil {
		s.ErrorExit("load config file %s error: %v", haconf, err)
		return
	}

	dc, err = docker.NewClient()
	if err != nil {
		s.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	s.Println("docker container    : %s", hc.Name)
	s.Println("config file         : %s", haconf)
	s.Println("haproxy run dir     : %s", haproxydir)

	ct, err = docker.ContainerExisted(dc, hc.Name)
	if err != nil {
		s.Println("container state     : %s", "not existed")
		err = nil
	} else {
		s.Println("container state     : %s", ct.State)
	}

	s.Println("statistics port     : %d", hc.Stats.Port)
	s.Println("statistics url      : http://localhost:%d%s", hc.Stats.Port, hc.Stats.URI)
	s.Println("proxy service count : %d", len(hc.Services))

	s.Println("")
	for _, sv := range hc.Services {
		s.Println("%s listen port    : %d", sv.Name, sv.Port)
		s.Println("%s backend count  : %d", sv.Name, len(sv.Backends))
		bt = bytes.NewBuffer(nil)
		ap = false
		for _, bm := range sv.Backends {
			if ap {
				fmt.Fprintf(bt, ", ")
			} else {
				ap = true
			}
			fmt.Fprintf(bt, "%s:%d", bm.IP, bm.Port)
		}
		s.Println("%s backend list   : %s", sv.Name, bt.String())
		s.Println("")
	}

	return
}

// Run -
func (s *ListHaproxy) Run() (err error) {
	return showHaproxyInfo(s)
}

// Run -
func (s *AddProxyService) Run() (err error) {
	setting := s.GlobalSetting()

	var (
		haconf string
		hc     *haproxyConfig
		bds    []backendService
	)

	if s.ServiceName == "" {
		s.ErrorExit("error proxy service name parameter")
		return
	}

	if s.Port <= 0 {
		s.ErrorExit("error proxy service port parameter")
		return
	}

	if s.Mode == "" {
		s.Mode = "http"
	}

	if !(s.Mode == "tcp" || s.Mode == "http") {
		err = errors.Errorf("error mode %s, only tcp or http", s.Mode)
		return
	}

	if s.Balance == "" {
		s.Balance = "source"
	}

	if !(s.Balance == "source" || s.Balance == "roundrobin") {
		err = errors.Errorf("error balance %s, only roundrobin or source", s.Balance)
		return
	}

	bds, _ = func() (ret []backendService, err error) {
		for _, v := range s.Backends {
			var (
				st  int
				sip string
				spt string
			)
			st = strings.LastIndex(v, ":")
			if st >= 0 {
				sip = v[0:st]
				spt = v[st+1:]
			} else {
				sip = v
				spt = "80"
			}

			if sip == "" {
				continue
			}

			st, err = strconv.Atoi(spt)
			if err != nil {
				return
			}
			ret = append(ret, backendService{
				SendProxy: s.SendProxy,
				IP:        sip,
				Port:      uint32(st),
				Inter:     uint32(s.Inter),
				Rise:      uint32(s.Rise),
				Fall:      uint32(s.Fall),
				Weight:    100,
			})
		}
		return
	}()

	haconf, hc, err = loadHaproxyConfig(setting)
	if err != nil {
		s.ErrorExit("load config file %s error: %v", haconf, err)
		return
	}

	if s.Balance == "" {
		s.Balance = hc.Balance
	}

	err = hc.addService(proxyService{
		AccessProxy: s.AccessProxy,
		Name:        s.ServiceName,
		Mode:        s.Mode,
		Balance:     s.Balance,
		Port:        uint32(s.Port),
		Backends:    bds,
	})

	if err != nil {
		s.ErrorExit("%v", err)
		return
	}

	_, err = hc.verifyHaproxyConfig()
	if err != nil {
		s.ErrorExit("%v", err)
		return
	}

	err = hc.saveToFile(haconf)

	if err != nil {
		s.ErrorExit("%v", err)
		return
	}

	s.Println("add haproxy proxy service %s ok!", s.Name)
	s.Println("")

	return showHaproxyInfo(s)
}

// Run -
func (r *RemoveProxyService) Run() (err error) {
	setting := r.GlobalSetting()
	var (
		haconf string
		hc     *haproxyConfig
		rved   bool
	)

	haconf, hc, err = loadHaproxyConfig(setting)
	if err != nil {
		r.ErrorExit("load config file %s error: %v", haconf, err)
		return
	}

	for _, svcName := range r.ServiceName {
		rved = hc.removeService(svcName)
		if rved {
			err = hc.saveToFile(haconf)
			if err != nil {
				r.ErrorExit("save config file %s error: %v", haconf, err)
				return
			}
			r.Println("proxy service %s remove success, you can restart haproxy!", r.Name)
		} else {
			r.Println("not found proxy service %s", r.Name)
		}
	}

	r.Println("")

	return showHaproxyInfo(r)

}

// Run -
func (a *AddBackend) Run() (err error) {
	setting := a.GlobalSetting()
	var (
		haconf string
		hc     *haproxyConfig
	)

	if a.Service == "" {
		a.ErrorExit("error proxy service name parameter")
		return
	}

	haconf, hc, err = loadHaproxyConfig(setting)
	if err != nil {
		a.ErrorExit("load config file %s error: %v", haconf, err)
		return
	}

	err = hc.addBackend(a.Service, backendService{
		Name:      a.BackendName,
		IP:        a.IP,
		Port:      uint32(a.Port),
		SendProxy: a.SendProxy,
	})

	if err != nil {
		a.ErrorExit("%v", err)
		return
	}

	_, err = hc.verifyHaproxyConfig()
	if err != nil {
		a.ErrorExit("%v", err)
		return
	}

	err = hc.saveToFile(haconf)

	if err != nil {
		a.ErrorExit("%v", err)
		return
	}

	a.Println("add proxy service %s backend %s:%d ok!", a.Service, a.IP, a.Port)
	a.Println("")

	return showHaproxyInfo(a)
}

// Run -
func (a *RemoveBackend) Run() (err error) {
	setting := a.GlobalSetting()
	var (
		haconf string
		hc     *haproxyConfig
	)

	if a.Service == "" {
		a.ErrorExit("error proxy service name parameter")
		return
	}

	haconf, hc, err = loadHaproxyConfig(setting)
	if err != nil {
		a.ErrorExit("load config file %s error: %v", haconf, err)
		return
	}

	err = hc.removeBackend(a.Service, a.IP, uint32(a.Port))

	if err != nil {
		a.ErrorExit("%v", err)
		return
	}

	_, err = hc.verifyHaproxyConfig()
	if err != nil {
		a.ErrorExit("%v", err)
		return
	}

	err = hc.saveToFile(haconf)

	if err != nil {
		a.ErrorExit("%v", err)
		return
	}

	a.Println("proxy service %s backend %s:%d removed!", a.Service, a.IP, a.Port)
	a.Println("")
	return showHaproxyInfo(a)
}

// LogHaproxy -
type LogHaproxy struct {
	BaseAction
	Follow     bool
	Since      string
	Tail       string
	Timestamps bool
	Details    bool
	Until      string
}

// NewLogHaproxy -
func NewLogHaproxy(setting *GlobalSetting) *LogHaproxy {
	return &LogHaproxy{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (l *LogHaproxy) Run() (err error) {
	setting := l.GlobalSetting()
	var (
		dc     *client.Client
		haconf string
		hc     *haproxyConfig
		oc     io.ReadCloser
		ct     types.Container
		c      types.ContainerJSON
	)

	haconf, hc, err = loadHaproxyConfig(setting)
	if err != nil {
		l.ErrorExit("load config file %s error: %v", haconf, err)
		return
	}

	dc, err = docker.NewClient()
	if err != nil {
		l.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	ct, err = docker.ContainerExisted(dc, hc.Name)
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
