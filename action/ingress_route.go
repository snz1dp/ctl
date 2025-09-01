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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/hbagdi/go-kong/kong"
	"github.com/pkg/errors"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

// RouteWrapper 路由封装
type RouteWrapper struct {
	kong.Route
	Plugins []*kong.Plugin `json:"plugin,omitempty"`
}

// RouteVisit 服务访问
type RouteVisit func(*kong.Route, []*kong.Plugin)

// GetRouteMethodList 方法列表
func GetRouteMethodList(route *kong.Route) (lsr string) {
	methods := []string{}
	for _, m := range route.Methods {
		methods = append(methods, *m)
	}
	lsr = strings.Join(methods, ",")
	if lsr == "" {
		lsr = "<All>"
	}
	return
}

// GetIngressRouteTagList 获取路径
func GetIngressRouteTagList(route *kong.Route) (tags string) {
	if route.Tags == nil || len(route.Tags) == 0 {
		tags = "<None>"
		return
	}
	rtags := []string{}
	for _, t := range route.Tags {
		rtags = append(rtags, *t)
	}
	tags = strings.Join(rtags, ",")
	return
}

// GetRoutePathList 方法列表
func GetRoutePathList(route *kong.Route) (lsr string) {
	paths := []string{}
	for _, m := range route.Paths {
		paths = append(paths, *m)
	}
	lsr = strings.Join(paths, ",")
	return
}

// GetRouteProtocolList 方法列表
func GetRouteProtocolList(route *kong.Route) (lsr string) {
	protocols := []string{}
	for _, m := range route.Protocols {
		protocols = append(protocols, *m)
	}
	lsr = strings.Join(protocols, ",")
	return
}

// GetRouteSNIList 获取主机名
func GetRouteSNIList(route *kong.Route) (lsr string) {
	snis := []string{}
	for _, m := range route.SNIs {
		snis = append(snis, *m)
	}
	lsr = strings.Join(snis, ",")
	return
}

// GetIngressRoute 获取路由
func GetIngressRoute(kc *kong.Client, nameOrID string) (route *kong.Route, err error) {
	route, err = kc.Routes.Get(context.Background(), kong.String(nameOrID))
	return
}

// DeleteIngressRoute 删除路由
func DeleteIngressRoute(kc *kong.Client, nameOrID string) (err error) {
	err = kc.Routes.Delete(context.Background(), kong.String(nameOrID))
	return
}

// DeleteIngressPlugin 删除路由
func DeleteIngressPlugin(kc *kong.Client, plugin ...*kong.Plugin) (err error) {
	for _, p := range plugin {
		if err = kc.Plugins.Delete(context.Background(), p.ID); err != nil {
			return
		}
	}
	return
}

// GetIngressRoutePlugins 获取路由插件
func GetIngressRoutePlugins(kc *kong.Client, route *kong.Route) (plugins []*kong.Plugin, err error) {
	plugins, err = kc.Plugins.ListAllForRoute(context.Background(), route.ID)
	return
}

// ListIngressRoutes 列表服务
func ListIngressRoutes(kc *kong.Client, opt *kong.ListOpt, visit RouteVisit) (err error) {
	var (
		routeList  []*kong.Route
		pluginList []*kong.Plugin
	)
	if opt == nil {
		opt = new(kong.ListOpt)
		opt.Size = 100
	}
	for {
		routeList, opt, err = kc.Routes.List(context.Background(), opt)
		if len(routeList) == 0 {
			break
		}
		for _, route := range routeList {
			if pluginList, err = GetIngressRoutePlugins(kc, route); err != nil {
				return
			}
			if route.Service, err = GetIngressService(kc, *route.Service.ID); err != nil {
				return
			}
			visit(route, pluginList)
		}
		if opt == nil {
			break
		}
	}
	return
}

// ApplyIngressRoute 创建路由
func ApplyIngressRoute(kc *kong.Client, orgRoute *kong.Route, existedUpdate bool) (route *kong.Route, err error) {
	var (
		routeNameOrID *string = orgRoute.ID
	)

	if routeNameOrID == nil {
		routeNameOrID = orgRoute.Name
	}

	if route, err = kc.Routes.Get(context.Background(), routeNameOrID); err != nil && !kong.IsNotFoundErr(err) {
		return
	}

	err = nil

	if route == nil {
		route, err = kc.Routes.Create(context.Background(), orgRoute)
	} else if existedUpdate {
		if orgRoute.Name != nil {
			route.Name = orgRoute.Name
		}
		if len(orgRoute.Hosts) > 0 {
			route.Hosts = orgRoute.Hosts
		}
		if len(orgRoute.Methods) > 0 {
			route.Methods = orgRoute.Methods
		}
		if len(orgRoute.Tags) > 0 {
			route.Tags = orgRoute.Tags
		}
		if len(orgRoute.Paths) > 0 {
			route.Paths = orgRoute.Paths
		}
		if orgRoute.StripPath != nil {
			route.StripPath = orgRoute.StripPath
		}
		if len(orgRoute.SNIs) > 0 {
			route.SNIs = orgRoute.SNIs
		}
		if len(orgRoute.Sources) > 0 {
			route.Sources = orgRoute.Sources
		}
		if len(orgRoute.Destinations) > 0 {
			route.Destinations = orgRoute.Destinations
		}
		if len(orgRoute.Headers) > 0 {
			route.Headers = orgRoute.Headers
		}

		route, err = kc.Routes.Update(context.Background(), route)
	} else {
		err = errors.Errorf("route %s existed", *routeNameOrID)
	}

	return
}

// IngressAddRoute 添加路由
type IngressAddRoute struct {
	IngressAddService
	// 路由名称
	RouteName string
	// 对外域名
	Domain []string
	// 路径
	Path []string
	// 协议
	Protocol []string
	// 是否保护域名（不向后传递主机名)
	PreserveHost bool
	// 请求后台时是否剥离对外路由地址
	StripPath bool
	// 认证模式
	AuthType string
	// 允许的分组
	AllowGroup []string
	// 不允许的分组
	DenyGroup []string
	// 允许匿名
	AllowAnonymous bool
}

// ToKongRoute 转路由
func (i *IngressAddRoute) ToKongRoute() (route *kong.Route) {
	route = &kong.Route{
		Name: kong.String(i.RouteName),
	}
	for _, h := range i.Domain {
		route.Hosts = append(route.Hosts, kong.String(h))
	}

	for _, p := range i.Path {
		route.Paths = append(route.Paths, kong.String(p))
	}

	for _, p := range i.Protocol {
		route.Protocols = append(route.Protocols, kong.String(p))
	}

	route.PreserveHost = kong.Bool(i.PreserveHost)
	route.StripPath = kong.Bool(i.StripPath)

	for _, tag := range i.Tags {
		route.Tags = append(route.Tags, kong.String(tag))
	}

	return
}

// IngressListRoute 列表路由
type IngressListRoute struct {
	BaseIngressAction
	RouteName    []string
	OutputFormat string
	Tags         []string
	MatchAllTags bool
}

// GetNameMap 获取名称
func (i *IngressListRoute) GetNameMap() (ret map[string]bool) {
	ret = make(map[string]bool)
	for _, name := range i.RouteName {
		ret[name] = true
	}
	return
}

// IngressDeleteRoute 删除服务
type IngressDeleteRoute struct {
	BaseIngressAction
	RouteNames []string
	Force      bool
}

// NewIngressAddRoute 添加路由
func NewIngressAddRoute(setting *GlobalSetting) *IngressAddRoute {
	return &IngressAddRoute{
		IngressAddService: IngressAddService{
			BaseIngressAction: BaseIngressAction{
				BaseAction: BaseAction{
					setting: setting,
				},
			},
		},
	}
}

// NewIngressListRoute 列表路由
func NewIngressListRoute(setting *GlobalSetting) *IngressListRoute {
	return &IngressListRoute{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// NewIngressDeleteRoute 删除服务
func NewIngressDeleteRoute(setting *GlobalSetting) *IngressDeleteRoute {
	return &IngressDeleteRoute{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// Run 实现新增路由
func (i *IngressAddRoute) Run() (err error) {
	var (
		kc      *kong.Client
		svc     *kong.Service
		route   *kong.Route
		plugins []*kong.Plugin
	)

	if kc, err = i.CreateKong(nil); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}

	if i.ServiceName == "" {
		i.ServiceName = i.RouteName
	}

	if svc, err = GetIngressService(kc, i.ServiceName); err != nil && !kong.IsNotFoundErr(err) {
		return
	}

	if svc == nil {
		if i.BackendHost == "" {
			err = errors.Errorf("not found service or backend host")
			return
		}

		if svc, err = ApplyIngressService(kc, i.ToKongService(), i.Override); err != nil {
			err = errors.Errorf("create service error: %s", err)
			return
		}
	}

	route = i.ToKongRoute()
	route.Service = svc

	if route, err = ApplyIngressRoute(kc, route, i.Override); err != nil {
		err = errors.Errorf("create route error: %s", err)
		return
	}

	if plugins, err = GetIngressRoutePlugins(kc, route); err != nil {
		err = errors.Errorf("load route %s plugin error: %s", *route.Name, err)
		return
	}

	switch i.AuthType {
	case "", "anonymous":
		var (
			weldelplugins []*kong.Plugin
		)
		for _, p := range plugins {
			if *p.Name == JWTAuthPlugin || *p.Name == SSOAuthPlugin || *p.Name == AuthACLPlugin {
				weldelplugins = append(weldelplugins, p)
			}
		}
		DeleteIngressPlugin(kc, weldelplugins...)
	case "jwt", "app":
		var (
			aclPlugin, jwtPlugin, ssoPlugin *kong.Plugin
		)
		for _, p := range plugins {
			if *p.Name == JWTAuthPlugin {
				jwtPlugin = p
			} else if *p.Name == AuthACLPlugin {
				aclPlugin = p
			} else if *p.Name == SSOAuthPlugin {
				ssoPlugin = p
			}
		}

		if ssoPlugin != nil {
			if err = kc.Plugins.Delete(context.Background(), ssoPlugin.ID); err != nil {
				return
			}
		}

		if jwtPlugin == nil {
			jwtPlugin = &kong.Plugin{
				Name:    kong.String(JWTAuthPlugin),
				Enabled: kong.Bool(true),
				Route:   route,
			}
			if _, err = kc.Plugins.Create(context.Background(), jwtPlugin); err != nil {
				return
			}
		} else {
			jwtPlugin.Enabled = kong.Bool(true)
			if _, err = kc.Plugins.Update(context.Background(), jwtPlugin); err != nil {
				return
			}
		}

		if len(i.AllowGroup) > 0 || len(i.DenyGroup) > 0 {

			if aclPlugin == nil {
				aclPlugin = &kong.Plugin{
					Name:    kong.String(AuthACLPlugin),
					Enabled: kong.Bool(true),
					Route:   route,
					Config:  kong.Configuration{},
				}

				if len(i.AllowGroup) > 0 {
					aclPlugin.Config["whitelist"] = i.AllowGroup
				} else if len(i.DenyGroup) > 0 {
					aclPlugin.Config["blacklist"] = i.DenyGroup
				}

				if _, err = kc.Plugins.Create(context.Background(), aclPlugin); err != nil {
					return
				}
			} else {
				aclPlugin.Enabled = kong.Bool(true)
				if len(i.AllowGroup) > 0 {
					aclPlugin.Config["whitelist"] = i.AllowGroup
				} else if len(i.DenyGroup) > 0 {
					aclPlugin.Config["blacklist"] = i.DenyGroup
				}

				if _, err = kc.Plugins.Update(context.Background(), aclPlugin); err != nil {
					return
				}
			}
		} else if aclPlugin != nil {
			err = DeleteIngressPlugin(kc, aclPlugin)
			if err != nil {
				return
			}
		}

	case "sso", "user":
		var (
			aclPlugin, jwtPlugin, ssoPlugin *kong.Plugin
		)
		for _, p := range plugins {
			if *p.Name == JWTAuthPlugin {
				jwtPlugin = p
			} else if *p.Name == AuthACLPlugin {
				aclPlugin = p
			} else if *p.Name == SSOAuthPlugin {
				ssoPlugin = p
			}
		}

		if aclPlugin != nil {
			if err = kc.Plugins.Delete(context.Background(), aclPlugin.ID); err != nil {
				return
			}
		}

		if jwtPlugin != nil {
			if err = kc.Plugins.Delete(context.Background(), jwtPlugin.ID); err != nil {
				return
			}
		}

		if ssoPlugin == nil {
			ssoPlugin = &kong.Plugin{
				Name:    kong.String(SSOAuthPlugin),
				Enabled: kong.Bool(true),
				Route:   route,
				Config:  kong.Configuration{},
			}
			if i.AllowAnonymous {
				ssoPlugin.Config["anonymous"] = true
			} else {
				ssoPlugin.Config["anonymous"] = false
			}
			if _, err = kc.Plugins.Create(context.Background(), ssoPlugin); err != nil {
				return
			}
		} else {
			ssoPlugin.Enabled = kong.Bool(true)
			if ssoPlugin.Config == nil {
				ssoPlugin.Config = kong.Configuration{}
			}
			if i.AllowAnonymous {
				ssoPlugin.Config["anonymous"] = true
			} else {
				ssoPlugin.Config["anonymous"] = false
			}
			if _, err = kc.Plugins.Update(context.Background(), ssoPlugin); err != nil {
				return
			}
		}
	default:
		err = errors.Errorf("auth-mode only suport app or user")
		return
	}

	s := NewIngressListRoute(i.setting)
	s.RouteName = []string{*route.ID}
	if i.GatewayAdminURL != "" {
		s.GatewayAdminURL = i.GatewayAdminURL
	}
	err = s.Run()

	return
}

// Run 列表路由
func (i *IngressListRoute) Run() (err error) {
	var (
		kc    *kong.Client
		opt   *kong.ListOpt = &kong.ListOpt{Size: 100}
		names map[string]bool
	)
	if kc, err = i.CreateKong(nil); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}

	for _, tag := range i.Tags {
		opt.Tags = append(opt.Tags, kong.String(tag))
	}

	names = i.GetNameMap()
	opt.MatchAllTags = i.MatchAllTags

	switch i.OutputFormat {
	case "table", "":
		i.Println("%-38s %-18s %-14s %-26s %-15s %-30s %-30s", "ID", "Name", "Authentication", "Service", "Hit Methods", "Hit Paths", "Tags")

		err = ListIngressRoutes(kc, opt, func(route *kong.Route, plugins []*kong.Plugin) {
			var (
				authMode string
			)
			if len(names) > 0 && !(names[*route.Name] || names[*route.ID]) {
				return
			}
			for _, p := range plugins {
				if *p.Name == JWTAuthPlugin {
					authMode = "app"
				} else if *p.Name == SSOAuthPlugin {
					authMode = "user"
				}
			}
			i.Println("%-38s %-18s %-14s %-26s %-15s %-30s %-30s", *route.ID, *route.Name,
				authMode, *route.Service.Name, GetRouteMethodList(route),
				GetRoutePathList(route), GetIngressRouteTagList(route))
		})

	case "yaml", "json":
		var (
			objList   []*RouteWrapper
			routeData []byte
		)
		_ = ListIngressRoutes(kc, opt, func(route *kong.Route, plugins []*kong.Plugin) {
			if len(names) > 0 && !(names[*route.Name] || names[*route.ID]) {
				return
			}
			objList = append(objList, &RouteWrapper{Route: *route, Plugins: plugins})
		})
		if i.OutputFormat == "yaml" {
			routeData, err = yaml.Marshal(objList)
		} else {
			routeData, err = json.Marshal(objList)
		}
		if err == nil {
			i.Println("%s", string(routeData))
		}
	default:
		err = errors.Errorf("not found output format: %s", i.OutputFormat)
	}
	return
}

// Run 删除路由
func (i *IngressDeleteRoute) Run() (err error) {
	var (
		setting *GlobalSetting = i.GlobalSetting()
		kc      *kong.Client
		route   *kong.Route
		cmsg    string
	)
	if kc, err = i.CreateKong(nil); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}

	for _, nameOrID := range i.RouteNames {

		if route, err = GetIngressRoute(kc, nameOrID); err != nil {
			if kong.IsNotFoundErr(err) {
				continue
			}
			return err
		}
		cmsg = fmt.Sprintf("delete route:\n  id=%s,\n  name=%s\n  service=%s\n  path=%s\nproceed? (y/N)",
			*route.ID, *route.Name, *route.Service.ID, GetRoutePathList(route))
		if !i.Force && !utils.Confirm(cmsg, setting.InOrStdin(), setting.OutOrStdout()) {
			i.Println("Cancelled.")
			return
		}

		if err = DeleteIngressRoute(kc, *route.ID); err != nil {
			if kong.IsNotFoundErr(err) {
				continue
			}
			return err
		}
	}

	err = nil

	return
}
