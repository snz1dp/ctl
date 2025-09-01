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
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/hbagdi/go-kong/kong"
	"github.com/pkg/errors"
)

var (
	ErrorNotFoundRunConfigFile error = errors.Errorf("not found run config file: RUN.yaml")
)

// IngressApply 添加服务
type IngressApply struct {
	BaseIngressAction
	ComponentFile string
	ServiceName   string
}

// NewIngressApply 添加服务
func NewIngressApply(setting *GlobalSetting) *IngressApply {
	return &IngressApply{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// Run 实现新增服务
func (s *IngressApply) Run() (err error) {
	var (
		bdfile       string
		pbcfst       os.FileInfo
		stdconf      StandaloneConfig
		pcdata       []byte
		backendports map[uint64]bool = make(map[uint64]bool)
	)
	bdfile = s.ComponentFile
	pbcfst, err = os.Stat(bdfile)
	if err != nil || pbcfst.IsDir() {
		bdfile = path.Join(s.ComponentFile, "RUN.yaml")
		pbcfst, err = os.Stat(bdfile)
		if err != nil || pbcfst.IsDir() {
			err = ErrorNotFoundRunConfigFile
			return
		}
	}
	pcdata, err = os.ReadFile(bdfile)
	if err != nil {
		err = errors.Errorf("read %s error:\n%v", pbcfst.Name(), err)
		return
	}

	if err = yaml.Unmarshal(pcdata, &stdconf); err != nil {
		err = errors.Errorf("read %s error:\n%v", pbcfst.Name(), err)
		return
	}

	for _, v := range stdconf.Ports {
		var (
			vst         int = strings.Index(v, ":")
			backendport string
			portval     uint64
			vtp         int
		)
		if vst < 0 {
			backendport = v
		} else {
			backendport = v[vst+1:]
		}

		vtp = strings.Index(backendport, "/")
		if vtp > 0 {
			backendport = backendport[:vtp]
		}

		portval, err = strconv.ParseUint(backendport, 10, 32)
		if err != nil {
			err = errors.Errorf("error service ports: %s", v)
			return
		}
		backendports[portval] = true
	}

	for _, v := range stdconf.Ingress {
		if v.BackendPort == 0 {
			v.BackendPort = 80
		}
	}

	var (
		kc      *kong.Client
		svcmap  map[string]*kong.Service = map[string]*kong.Service{}
		envmap  map[string]string        = map[string]string{}
		ingress *StandaloneIngressConfig
		i       int
	)

	for _, m := range stdconf.Envs {
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
		envmap[mk] = mv
	}

	if kc, err = s.CreateKong(nil); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}

	for i, ingress = range stdconf.Ingress {
		var (
			svcName      string
			backendPort  int
			webRootPath  *string
			schemaName   string
			serviceURL   string
			readTimeout  = ingress.ReadTimeout
			writeTimeout = ingress.WriteTimeout
			svc          *kong.Service
		)
		if ingress.BackendPort > 0 {
			backendPort = ingress.BackendPort
		} else if ingress.Protocol == "http" || ingress.Protocol == "" {
			backendPort = 80
		} else if ingress.Protocol == "https" {
			backendPort = 443
		} else {
			err = errors.Errorf("%s service must be set backend port", ingress.Protocol)
			return
		}
		if ingress.Protocol == "" {
			schemaName = "http"
		}

		if readTimeout == 0 {
			readTimeout = 60000
		}

		if writeTimeout == 0 {
			writeTimeout = 60000
		}

		if ingress.WebRoot == "/" {
			ingress.WebRoot = ""
		}

		if ingress.WebRoot != "" {
			webRootPath = kong.String(ingress.WebRoot)
		}

		serviceURL = fmt.Sprintf("%s://%s:%d%s", schemaName, s.ServiceName, backendPort, ingress.WebRoot)
		if svcmap[serviceURL] != nil {
			continue
		}
		svcName = fmt.Sprintf("%s.%d", s.ServiceName, len(svcmap))

		svc = &kong.Service{
			Name:         kong.String(svcName),
			Protocol:     kong.String(schemaName),
			Host:         kong.String(s.ServiceName),
			Port:         kong.Int(ingress.BackendPort),
			ReadTimeout:  kong.Int(readTimeout),
			WriteTimeout: kong.Int(writeTimeout),
			Path:         webRootPath,
		}

		if svc, err = ApplyIngressService(kc, svc, true); err != nil {
			err = errors.Errorf("apply %s service error: %s", svcName, err)
			return
		}

		svcmap[serviceURL] = svc
	}

	for i, ingress = range stdconf.Ingress {
		var (
			svc         *kong.Service
			rt          *kong.Route
			rtName      string
			plugin      *kong.Plugin
			aclPlugin   *kong.Plugin
			jwtPlugin   string = "jwtauth"
			ssoPlugin   string = "ssoauth"
			plugins     []*kong.Plugin
			epdHost     []string
			backendPort int
			webRootPath string = ingress.WebRoot
			schemaName  string
			serviceURL  string
		)

		for _, v := range ingress.Host {
			v = os.Expand(v, func(key string) string {
				return envmap[key]
			})
			epdHost = append(epdHost, v)
		}

		if ingress.BackendPort > 0 {
			backendPort = ingress.BackendPort
		} else if ingress.Protocol == "http" || ingress.Protocol == "" {
			backendPort = 80
		} else if ingress.Protocol == "https" {
			backendPort = 443
		} else {
			err = errors.Errorf("%s service must be set backend port", ingress.Protocol)
			return
		}
		if ingress.Protocol == "" {
			schemaName = "http"
		}
		webRootPath = ingress.WebRoot
		if webRootPath == "/" {
			webRootPath = ""
		}
		serviceURL = fmt.Sprintf("%s://%s:%d%s", schemaName, s.ServiceName, backendPort, webRootPath)
		svc = svcmap[serviceURL]

		preserveHost := true
		if ingress.PreserveHost != nil {
			preserveHost = *ingress.PreserveHost
		}

		if len(ingress.JWTAuth) > 0 {

			// 添加路由
			rtName = fmt.Sprintf("%s.%d.jwtauth", s.ServiceName, i)
			rt = &kong.Route{
				Name:         kong.String(rtName),
				Hosts:        kong.StringSlice(epdHost...),
				StripPath:    kong.Bool(ingress.StripPath),
				PreserveHost: kong.Bool(preserveHost),
				Service:      svc,
				Paths:        kong.StringSlice(ingress.JWTAuth...),
			}

			if rt, err = ApplyIngressRoute(kc, rt, true); err != nil {
				err = errors.Errorf("apply %s route error: %s", rtName, err)
				return
			}

			if plugins, err = kc.Plugins.ListAllForRoute(context.Background(), rt.ID); err != nil {
				err = errors.Errorf("list %s route plugins error: %s", rtName, err)
				return err
			}

			for _, plugin = range plugins {
				kc.Plugins.Delete(context.Background(), plugin.ID)
			}

			// 配置插件
			plugin = &kong.Plugin{
				Name:    kong.String(jwtPlugin),
				Enabled: kong.Bool(true),
				Route:   rt,
			}

			if _, err = kc.Plugins.Create(context.Background(), plugin); err != nil {
				err = errors.Errorf("apply %s route jwt plugin error: %s", rtName, err)
				return
			}

			if len(ingress.Whitelist) > 0 || len(ingress.Blacklist) > 0 {
				aclPlugin = &kong.Plugin{
					Name:    kong.String(AuthACLPlugin),
					Enabled: kong.Bool(true),
					Route:   rt,
					Config:  kong.Configuration{},
				}

				if len(ingress.Whitelist) > 0 {
					aclPlugin.Config["whitelist"] = ingress.Whitelist
				} else if len(ingress.Blacklist) > 0 {
					aclPlugin.Config["blacklist"] = ingress.Blacklist
				}
				if _, err = kc.Plugins.Create(context.Background(), aclPlugin); err != nil {
					err = errors.Errorf("apply %s route acl plugin error: %s", rtName, err)
					return
				}
			}

		}

		if len(ingress.SSOAuth) > 0 {
			rtName = fmt.Sprintf("%s.%d.ssoauth", s.ServiceName, i)
			rt = &kong.Route{
				Name:         kong.String(rtName),
				Hosts:        kong.StringSlice(epdHost...),
				StripPath:    kong.Bool(ingress.StripPath),
				PreserveHost: kong.Bool(preserveHost),
				Service:      svc,
				Paths:        kong.StringSlice(ingress.SSOAuth...),
			}

			if rt, err = ApplyIngressRoute(kc, rt, true); err != nil {
				err = errors.Errorf("apply %s route error: %s", rtName, err)
				return
			}

			if plugins, err = kc.Plugins.ListAllForRoute(context.Background(), rt.ID); err != nil {
				err = errors.Errorf("list %s route plugins error: %s", rtName, err)
				return err
			}

			for _, plugin = range plugins {
				kc.Plugins.Delete(context.Background(), plugin.ID)
			}

			plugin = &kong.Plugin{
				Name:    kong.String(ssoPlugin),
				Enabled: kong.Bool(true),
				Route:   rt,
				Config: kong.Configuration{
					"anonymous": ingress.MiscAuth,
				},
			}

			if _, err = kc.Plugins.Create(context.Background(), plugin); err != nil {
				err = errors.Errorf("apply %s route sso plugin error: %s", rtName, err)
				return
			}

		}

		if len(ingress.Anonymous) > 0 {
			rtName = fmt.Sprintf("%s.%d.anonymous", s.ServiceName, i)
			rt = &kong.Route{
				Name:         kong.String(rtName),
				Hosts:        kong.StringSlice(epdHost...),
				StripPath:    kong.Bool(ingress.StripPath),
				PreserveHost: kong.Bool(preserveHost),
				Service:      svc,
				Paths:        kong.StringSlice(ingress.Anonymous...),
			}

			if rt, err = ApplyIngressRoute(kc, rt, true); err != nil {
				err = errors.Errorf("apply %s route error: %s", rtName, err)
				return
			}

			plugins, err = kc.Plugins.ListAllForRoute(context.Background(), rt.ID)
			if err != nil {
				return err
			}

			for _, plugin = range plugins {
				kc.Plugins.Delete(context.Background(), plugin.ID)
			}

		}

		// 添加自定义登录接口
		if ingress.LoginConfig != nil {
			var (
				xc *XeaiClient
			)

			if xc, err = s.CreateXeai(nil); err != nil {
				err = errors.Errorf("create ingress xeai client error: %s", err)
				return
			}

			if err = xc.AddLoginConfig(s.ServiceName, ingress.LoginConfig); err != nil {
				err = errors.Errorf("add ingress login config error: %s", err)
				return
			}
		}

	}

	err = nil
	s.Println("config %s ingress success.", s.ServiceName)

	return
}
