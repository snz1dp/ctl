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
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hbagdi/go-kong/kong"
	"github.com/pkg/errors"
	"snz1.cn/snz1dp/snz1dpctl/down"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

const (
	// JWTAuthPlugin -
	JWTAuthPlugin = "jwtauth"
	// SSOAuthPlugin -
	SSOAuthPlugin = "ssoauth"
	// AuthACLPlugin -
	AuthACLPlugin = "authacl"
)

// XeaiClient - 客户端
type XeaiClient struct {
	kongClient *kong.Client
	baseURL    string
	useJWT     bool
}

// XeaiResult - 应答
type XeaiResult struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// XeaiReturn - 返回
type XeaiReturn struct {
	XeaiResult
	Data *map[string]interface{} `json:"data,omitempty"`
}

// SetDefaultLogin - 添加登录页面配置
func (xeai *XeaiClient) SetDefaultLogin(default_url, login_url, logout_url string) (err error) {
	if default_url == "" && login_url == "" && logout_url == "" {
		return
	}

	var (
		requestData  map[string]interface{}
		requestJson  []byte
		httpRequest  *http.Request
		httpResponse *http.Response
		responseBody []byte
		responseData XeaiReturn
		httpClient   *http.Client
		accessToken  string
		retryCount   int = 0
	)

retryRequest:
	if httpRequest, err = http.NewRequest("GET", fmt.Sprintf("%s/setup/login_security_policy", xeai.baseURL), nil); err != nil {
		return
	}

	if xeai.useJWT {
		if accessToken, err = xeai.kongClient.CreateJwtAccessToken(); err != nil {
			return
		}
		httpRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	}

	httpClient = &http.Client{}
	if httpResponse, err = httpClient.Do(httpRequest); err != nil {
		return err
	}

	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode > 400 {

		if httpResponse.StatusCode == 404 && retryCount < 10 {
			retryCount++
			time.Sleep(3 * time.Second)
			goto retryRequest
		}

		err = fmt.Errorf("xeai get login config response error, code = %d", httpResponse.StatusCode)
		return
	}

	if responseBody, err = ioutil.ReadAll(httpResponse.Body); err != nil {
		return
	}

	if err = json.Unmarshal(responseBody, &responseData); err != nil {
		return
	}

	if responseData.Code != 0 {
		err = fmt.Errorf("xeai get login config response error, code = %d, message = %s", responseData.Code, responseData.Message)
		return
	}

	if responseData.Data == nil {
		err = fmt.Errorf("xeai get login config response empty")
	}

	requestData = *responseData.Data
	if default_url != "" {
		requestData["login_default_url"] = default_url
	}

	if logout_url != "" {
		requestData["logout_default_url"] = logout_url
	}

	if login_url != "" {
		requestData["custom_view_url"] = login_url
	}

	if requestJson, err = json.Marshal(requestData); err != nil {
		return
	}

	if httpRequest, err = http.NewRequest("POST", fmt.Sprintf("%s/setup/login_security_policy", xeai.baseURL), bytes.NewBuffer(requestJson)); err != nil {
		return
	}

	if xeai.useJWT {
		if accessToken, err = xeai.kongClient.CreateJwtAccessToken(); err != nil {
			return
		}
		httpRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	}

	httpRequest.Header.Set("Content-Type", "application/json")

	if httpResponse, err = httpClient.Do(httpRequest); err != nil {
		return err
	}

	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode > 400 {
		err = fmt.Errorf("xeai set login page response error, code = %d", httpResponse.StatusCode)
		return
	}

	return
}

// AddLoginConfig - 添加登录页面配置
func (xeai *XeaiClient) AddLoginConfig(clientid string, lc *LoginConfig) (err error) {
	var (
		requestData  map[string]interface{} = map[string]interface{}{}
		requestJson  []byte
		httpClient   *http.Client
		httpRequest  *http.Request
		httpResponse *http.Response
		responseBody []byte
		responseData XeaiResult
		accessToken  string
	)

	requestData["path"] = lc.PageURL
	requestData["login"] = lc.LoginURL
	requestData["only_view"] = lc.OnlyView
	requestData["client_id"] = clientid

	if requestJson, err = json.Marshal(requestData); err != nil {
		return
	}

	if httpRequest, err = http.NewRequest("POST", fmt.Sprintf("%s/setup/path_login_redirect_config", xeai.baseURL), bytes.NewBuffer(requestJson)); err != nil {
		return
	}

	if xeai.useJWT {
		if accessToken, err = xeai.kongClient.CreateJwtAccessToken(); err != nil {
			return
		}
		httpRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	}

	httpRequest.Header.Set("Content-Type", "application/json")

	httpClient = &http.Client{}
	if httpResponse, err = httpClient.Do(httpRequest); err != nil {
		return err
	}

	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode > 400 {
		err = fmt.Errorf("xeai response error, code = %d", httpResponse.StatusCode)
		return
	}

	if responseBody, err = ioutil.ReadAll(httpResponse.Body); err != nil {
		return
	}

	if err = json.Unmarshal(responseBody, &responseData); err != nil {
		return
	}

	if responseData.Code != 0 {
		err = fmt.Errorf("xeai response error, code = %d, message = %s", responseData.Code, responseData.Message)
		return
	}

	return
}

// BaseIngressAction 需要地址的
type BaseIngressAction struct {
	BaseAction
	// 网关管理地址
	GatewayAdminURL string
	// 私钥
	RSAPrivateKey string
	// 密钥ID
	KeyID      string
	kongClient *kong.Client
	xeaiClient *XeaiClient
	useJWT     bool
}

// LoadRSAPrivateKey 加载私钥
func (b *BaseIngressAction) LoadRSAPrivateKey() (rsaKey *rsa.PrivateKey, err error) {
	var (
		setting       *GlobalSetting = b.GlobalSetting()
		rsaKeyContent string
		furl          *url.URL
		currentDir    string
		icdata        []byte
		spinner       *utils.WaitSpinner
	)

	if currentDir, err = os.Getwd(); err != nil {
		err = errors.Errorf("get cwd error: %s", err)
		return
	}

	// 尝试解码内容
	if strings.Index(b.RSAPrivateKey, "-----BEGIN") == 0 {
		rsaKeyContent = strings.ReplaceAll(b.RSAPrivateKey, "\\n", "\n")
	} else {
		if furl, err = url.Parse(b.RSAPrivateKey); err != nil {
			err = errors.Errorf("error url %s: %s", b.RSAPrivateKey, err)
			return
		}

		if furl.Scheme == "" || furl.Scheme == "file" {
			var fromFilePath string = furl.Path
			if !filepath.IsAbs(fromFilePath) {
				fromFilePath = filepath.Join(currentDir, fromFilePath)
				if fromFilePath, err = filepath.Abs(fromFilePath); err != nil {
					err = errors.Errorf("error file path: %s", err)
					return
				}
			}

			if icdata, err = os.ReadFile(fromFilePath); err != nil {
				err = errors.Errorf("read %s error: %s", fromFilePath, err)
				return
			}

		} else {
			spinner = utils.NewSpinner(fmt.Sprintf("download %s...", furl.String()), setting.OutOrStdout())
			var g down.Getter
			if g, err = down.AllProviders().ByScheme(furl.Scheme); err != nil {
				spinner.Close()
				b.ErrorExit("failed: %s", err)
				return
			}

			var fout *bytes.Buffer = bytes.NewBuffer(nil)

			if _, err = g.Get(b.RSAPrivateKey, fout, nil, nil); err != nil {
				spinner.Close()
				b.ErrorExit("failed: %s", err)
				return
			}
			spinner.Close()
			icdata = fout.Bytes()
			b.Println("ok!")

		}
		rsaKeyContent = string(icdata)
	}

	rsaKey, err = utils.DecodePrivateKeyFromPEM(rsaKeyContent)

	return
}

// CreateXeai - 创建Xeai客户端
func (b *BaseIngressAction) CreateXeai(sic *InstallConfiguration) (xc *XeaiClient, err error) {
	if b.xeaiClient != nil {
		xc = b.xeaiClient
		return
	}

	if _, err = b.CreateKong(sic); err != nil {
		return
	}

	var (
		setting *GlobalSetting = b.GlobalSetting()
		ic      *InstallConfiguration
		cf      string
	)

	if sic == nil {
		if cf, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
			err = errors.Errorf("load %s error: %s", cf, err)
			return
		}
	} else {
		ic = sic
	}

	xc = &XeaiClient{}
	xc.kongClient = b.kongClient
	xc.useJWT = b.useJWT

	if b.useJWT {
		var (
			tmpURL  *url.URL
			xeaiURL string
		)
		if tmpURL, err = url.Parse(b.GatewayAdminURL); err != nil {
			return
		}
		if tmpURL.Scheme == "" {
			xeaiURL = "http://"
		} else {
			xeaiURL = tmpURL.Scheme + "://"
		}
		xeaiURL += tmpURL.Hostname()
		if tmpURL.Port() != "" {
			xeaiURL += ":" + tmpURL.Port()
		}
		xeaiURL += "/xeai"

		xc.baseURL = xeaiURL
	} else {
		xc.baseURL = ic.Xeai.Web.GetURL()
	}

	return
}

// CreateKong 创建Kong客户端
func (b *BaseIngressAction) CreateKong(sic *InstallConfiguration) (kc *kong.Client, err error) {
	if b.kongClient != nil {
		kc = b.kongClient
		return
	}

	var (
		setting *GlobalSetting = b.GlobalSetting()
		ic      *InstallConfiguration
		cf      string
		rsaKey  *rsa.PrivateKey
	)

	if sic == nil {
		if cf, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
			err = errors.Errorf("load %s error: %s", cf, err)
			return
		}
	} else {
		ic = sic
	}

	if b.RSAPrivateKey == "" && ic.Appgateway != nil && ic.Appgateway.Jwt != nil {
		b.RSAPrivateKey = ic.Appgateway.Jwt.RSAKey
	}

	if b.KeyID == "" && ic.Appgateway != nil && ic.Appgateway.Jwt != nil {
		b.KeyID = ic.Appgateway.Jwt.Token
		if b.KeyID == "" {
			b.KeyID = "gatewayadmin"
		}
	}

	if b.RSAPrivateKey != "" {
		if b.GatewayAdminURL == "" {
			b.GatewayAdminURL = fmt.Sprintf("%s/gateway/admin", ic.Snz1dp.Ingress.GetBaseWebURL())
		}
		if rsaKey, err = b.LoadRSAPrivateKey(); err != nil {
			err = errors.Errorf("error rsa private key: %s", err)
			return
		}
		kc, err = kong.NewJwtClient(kong.String(b.GatewayAdminURL), nil, b.KeyID, rsaKey, 600)
		b.useJWT = true
	} else {
		if b.GatewayAdminURL == "" {
			b.GatewayAdminURL = ic.Appgateway.Admin.GetURL()
		}
		kc, err = kong.NewClient(kong.String(b.GatewayAdminURL), nil)
		b.useJWT = false
	}

	if err == nil {
		b.kongClient = kc
	}

	return
}

func (s *StandaloneService) getEnvVariablesMap() map[string]string {
	envmap := map[string]string{}
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
		envmap[mk] = mv
	}
	return envmap
}

func (s *StandaloneService) applySnz1dpIngress() (err error) {
	// 没有定义直接退出
	if s.ic.Snz1dp.Ingress.Services == nil && s.ic.Snz1dp.Ingress.Routes == nil {
		return
	}

	var (
		kc     *kong.Client
		envmap map[string]string  = s.getEnvVariablesMap()
		bic    *BaseIngressAction = &BaseIngressAction{
			BaseAction: BaseAction{
				setting: s.action.GlobalSetting(),
			},
		}
		svcmap            map[string]*kong.Service = map[string]*kong.Service{}
		backendService    *BackendService
		backendServiceURL *url.URL
		svc               *kong.Service
		createdService    *kong.Service
		tryCount          int
	)

	if kc, err = bic.CreateKong(s.ic); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}

	if s.ic.Snz1dp.Ingress.Services != nil {
		for _, backendService = range *s.ic.Snz1dp.Ingress.Services {
			if backendServiceURL, err = url.Parse(backendService.URL); err != nil || backendService.Name == "" {
				if backendService.Name == "" {
					fmt.Fprintf(
						s.action.GlobalSetting().ErrOrStderr(),
						"ignore create service, name is empty",
					)
				} else {
					fmt.Fprintf(
						s.action.GlobalSetting().ErrOrStderr(),
						"ignore create service, name is %s, error url is: %s",
						backendService.Name,
						backendService.URL,
					)
				}
				continue
			}

			var (
				servicePort int
				webHostname string
				webRootPath *string
				webProtocol *string
				serviceURL  string
			)

			if backendService.ReadTimeout <= 0 {
				backendService.ReadTimeout = 60000
			}

			if backendService.WriteTimeout <= 0 {
				backendService.WriteTimeout = 60000
			}

			if backendServiceURL.Path == "/" {
				backendServiceURL.Path = ""
			}

			if backendServiceURL.Path != "" {
				webRootPath = kong.String(backendServiceURL.Path)
			}

			if backendServiceURL.Port() != "" {
				if _, err = strconv.Atoi(backendServiceURL.Port()); err != nil {
					fmt.Fprintf(
						s.action.GlobalSetting().ErrOrStderr(),
						"ignore create service, name is %s, error url is: %s",
						backendService.Name,
						backendService.URL,
					)
				}
				continue
			}

			webHostname = backendServiceURL.Hostname()

			if webHostname == "" {
				fmt.Fprintf(
					s.action.GlobalSetting().ErrOrStderr(),
					"ignore create service, name is %s, error url is: %s",
					backendService.Name,
					backendService.URL,
				)
				continue
			}

			if backendServiceURL.Scheme != "" {
				webProtocol = kong.String(backendServiceURL.Scheme)
				if servicePort <= 0 {
					switch backendServiceURL.Scheme {
					case "http":
						servicePort = 80
					case "https":
						servicePort = 443
					default:
						servicePort = 80
					}
				}
			} else if servicePort <= 0 {
				webProtocol = kong.String("http")
				servicePort = 80
			}

			svc = &kong.Service{
				Name:         kong.String(backendService.Name),
				Protocol:     webProtocol,
				Host:         kong.String(webHostname),
				Port:         kong.Int(servicePort),
				ReadTimeout:  kong.Int(backendService.ReadTimeout),
				WriteTimeout: kong.Int(backendService.WriteTimeout),
				Path:         webRootPath,
			}

		recreateService:

			if createdService, err = ApplyIngressService(kc, svc, true); err != nil {

				if kong.IsNotFoundErr(err) && tryCount < 10 {
					time.Sleep(3 * time.Second)
					tryCount++
					goto recreateService
				}

				err = errors.Errorf("create service error, name is %s, error: %s", backendService.Name, err)
				return
			}

			serviceURL = fmt.Sprintf("%s://%s:%d%s", *webProtocol, webHostname, servicePort, backendServiceURL.Path)
			svcmap[serviceURL] = createdService
		}
	}

	err = nil

	if s.ic.Snz1dp.Ingress.Routes == nil {
		return
	}

	tryCount = 0

	var (
		route             *ServiceRoute
		namedSvcMap       map[string]*kong.Service = map[string]*kong.Service{}
		wellbeCreateRoute map[string]*ServiceRoute = map[string]*ServiceRoute{}
		routeServiceMap   map[string]*kong.Service = map[string]*kong.Service{}
	)

	// 遍历路由
	for _, route = range *s.ic.Snz1dp.Ingress.Routes {
		if route.Name == "" {
			fmt.Fprintf(
				s.action.GlobalSetting().ErrOrStderr(),
				"ignore create route, name is empty",
			)
			continue
		}

		if wellbeCreateRoute[route.Name] != nil {
			fmt.Fprintf(
				s.action.GlobalSetting().ErrOrStderr(),
				"ignore create route, name is %s, duplicate name error",
				route.Name,
			)
		}

		if route.Service != "" {
			if svc, err = kc.Services.Get(context.Background(), kong.String(route.Service)); err != nil {
				return
			}
			namedSvcMap[route.Service] = svc
			routeServiceMap[route.Name] = svc
			wellbeCreateRoute[route.Name] = route
			continue
		}

		if backendServiceURL, err = url.Parse(route.ServiceURL); err != nil {
			fmt.Fprintf(
				s.action.GlobalSetting().ErrOrStderr(),
				"ignore create route, name is %s, error service url is: %s",
				route.Name,
				route.ServiceURL,
			)
			continue
		}

		var (
			svcName     string
			servicePort int
			webHostname string
			webRootPath *string
			webProtocol *string
			serviceURL  string
		)

		if route.ReadTimeout <= 0 {
			route.ReadTimeout = 60000
		}

		if route.WriteTimeout <= 0 {
			route.WriteTimeout = 60000
		}

		if backendServiceURL.Path == "/" {
			backendServiceURL.Path = ""
		}

		if backendServiceURL.Path != "" {
			webRootPath = kong.String(backendServiceURL.Path)
		}

		if backendServiceURL.Port() != "" {
			if _, err = strconv.Atoi(backendServiceURL.Port()); err != nil {
				fmt.Fprintf(
					s.action.GlobalSetting().ErrOrStderr(),
					"ignore create route, name is %s, error service url is: %s",
					route.Name,
					route.ServiceURL,
				)
			}
			continue
		}

		webHostname = backendServiceURL.Hostname()

		if webHostname == "" {
			fmt.Fprintf(
				s.action.GlobalSetting().ErrOrStderr(),
				"ignore create route, name is %s, error service url is: %s",
				route.Name,
				route.ServiceURL,
			)
			continue
		}

		if backendServiceURL.Scheme != "" {
			webProtocol = kong.String(backendServiceURL.Scheme)
			if servicePort <= 0 {
				switch backendServiceURL.Scheme {
				case "http":
					servicePort = 80
				case "https":
					servicePort = 443
				default:
					servicePort = 80
				}
			}
		} else if servicePort <= 0 {
			webProtocol = kong.String("http")
			servicePort = 80
		}

		wellbeCreateRoute[route.Name] = route

		serviceURL = fmt.Sprintf("%s://%s:%d%s", *webProtocol, webHostname, servicePort, backendServiceURL.Path)

		if svcmap[serviceURL] == nil {
			svcName = fmt.Sprintf("%s.%d", route.Name, 0)

			svc = &kong.Service{
				Name:         kong.String(svcName),
				Protocol:     webProtocol,
				Host:         kong.String(webHostname),
				Port:         kong.Int(servicePort),
				ReadTimeout:  kong.Int(route.ReadTimeout),
				WriteTimeout: kong.Int(route.ReadTimeout),
				Path:         webRootPath,
			}

		reCreateRouteService:
			if createdService, err = ApplyIngressService(kc, svc, true); err != nil {
				if kong.IsNotFoundErr(err) && tryCount < 10 {
					time.Sleep(3 * time.Second)
					tryCount++
					goto reCreateRouteService
				}
				err = errors.Errorf("create service error, name is %s, error: %s", svcName, err)
				return
			}

			svcmap[serviceURL] = createdService
			routeServiceMap[route.Name] = createdService
		} else {
			routeServiceMap[route.Name] = svcmap[serviceURL]
		}
	}

	err = nil

	for _, route = range wellbeCreateRoute {
		var (
			svc       *kong.Service
			rt        *kong.Route
			rtName    string
			plugin    *kong.Plugin
			aclPlugin *kong.Plugin
			jwtPlugin string = "jwtauth"
			ssoPlugin string = "ssoauth"
			plugins   []*kong.Plugin
			epdHost   []string
		)

		for _, v := range route.Host {
			v = os.Expand(v, func(key string) string {
				return envmap[key]
			})
			epdHost = append(epdHost, v)
		}

		rtName = route.Name
		svc = routeServiceMap[rtName]
		if svc == nil {
			continue
		}

		preserveHost := true

		if route.PreserveHost != nil {
			preserveHost = *route.PreserveHost
		}

		rt = &kong.Route{
			Name:         kong.String(rtName),
			Hosts:        kong.StringSlice(epdHost...),
			StripPath:    kong.Bool(route.StripPath),
			PreserveHost: kong.Bool(preserveHost),
			Service:      svc,
			Paths:        kong.StringSlice(route.Path...),
		}

		if rt, err = ApplyIngressRoute(kc, rt, true); err != nil {
			err = errors.Errorf("apply %s route error: %s", rtName, err)
			return
		}

		if route.AuthMode == "jwt" {
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

			if len(route.Whitelist) > 0 || len(route.Blacklist) > 0 {
				aclPlugin = &kong.Plugin{
					Name:    kong.String(AuthACLPlugin),
					Enabled: kong.Bool(true),
					Route:   rt,
					Config:  kong.Configuration{},
				}

				if len(route.Whitelist) > 0 {
					aclPlugin.Config["whitelist"] = route.Whitelist
				} else if len(route.Blacklist) > 0 {
					aclPlugin.Config["blacklist"] = route.Blacklist
				}
				if _, err = kc.Plugins.Create(context.Background(), aclPlugin); err != nil {
					err = errors.Errorf("apply %s route acl plugin error: %s", rtName, err)
					return
				}
			}
		} else if route.AuthMode == "sso" {
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
					"anonymous": route.MiscAuth,
				},
			}

			if _, err = kc.Plugins.Create(context.Background(), plugin); err != nil {
				err = errors.Errorf("apply %s route sso plugin error: %s", rtName, err)
				return
			}

		} else {
			plugins, err = kc.Plugins.ListAllForRoute(context.Background(), rt.ID)
			if err != nil {
				return err
			}

			for _, plugin = range plugins {
				kc.Plugins.Delete(context.Background(), plugin.ID)
			}

		}

	}

	return
}

// ApplyIngress -
func (s *StandaloneService) ApplyIngress() (err error) {

	// 如果是ingress自身则初始化Snz1dp配置
	if s.ServiceName == "ingress" {
		err = s.applySnz1dpIngress()
		return
	}

	if len(s.Ingress) == 0 {
		return
	}

	var (
		kc     *kong.Client
		envmap map[string]string  = s.getEnvVariablesMap()
		bic    *BaseIngressAction = &BaseIngressAction{
			BaseAction: BaseAction{
				setting: s.action.GlobalSetting(),
			},
		}
		svcmap map[string]*kong.Service = map[string]*kong.Service{}
	)

	var (
		i       int
		ingress *StandaloneIngressConfig
	)

	if kc, err = bic.CreateKong(s.ic); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}

	for i, ingress = range s.Ingress {
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

	for i, ingress = range s.Ingress {
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

			if xc, err = bic.CreateXeai(s.ic); err != nil {
				err = errors.Errorf("create ingress xeai client error: %s", err)
				return
			}

			if err = xc.AddLoginConfig(s.ServiceName, ingress.LoginConfig); err != nil {
				err = errors.Errorf("add ingress login config error: %s", err)
				return
			}

		}

	}

	return
}

// ApplyDefaultURL -
func (s *StandaloneService) ApplyDefaultURL() (err error) {
	var (
		bic *BaseIngressAction = &BaseIngressAction{
			BaseAction: BaseAction{
				setting: s.action.GlobalSetting(),
			},
		}
		xc *XeaiClient
	)
	if xc, err = bic.CreateXeai(s.ic); err != nil {
		return
	}
	err = xc.SetDefaultLogin(
		s.ic.Snz1dp.DefaultURL,
		s.ic.Snz1dp.LoginURL,
		s.ic.Snz1dp.LogoutURL,
	)
	return
}
