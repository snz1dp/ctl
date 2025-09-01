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

// ServiceVisit 服务访问
type ServiceVisit func(*kong.Service)

// GetIngressService 获取服务
func GetIngressService(kc *kong.Client, nameOrID string) (svc *kong.Service, err error) {
	svc, err = kc.Services.Get(context.Background(), kong.String(nameOrID))
	return
}

// DeleteIngressService 删除服务
func DeleteIngressService(kc *kong.Client, nameOrID string) (err error) {
	err = kc.Services.Delete(context.Background(), kong.String(nameOrID))
	return
}

// GetIngressServicePath 获取路径
func GetIngressServicePath(svc *kong.Service) (path string) {
	if svc.Path == nil {
		path = "/"
		return
	}
	path = *svc.Path
	return
}

// GetIngressServiceTagList 获取路径
func GetIngressServiceTagList(svc *kong.Service) (tags string) {
	if svc.Tags == nil || len(svc.Tags) == 0 {
		tags = "<None>"
		return
	}
	rtags := []string{}
	for _, t := range svc.Tags {
		rtags = append(rtags, *t)
	}
	tags = strings.Join(rtags, ",")
	return
}

// GetIngressServiceProtocol 获取协议
func GetIngressServiceProtocol(svc *kong.Service) (protocol string) {
	if svc.Protocol == nil {
		return
	}
	protocol = *svc.Protocol
	return
}

// ListIngressServices 列表服务
func ListIngressServices(kc *kong.Client, opt *kong.ListOpt, visit ServiceVisit) (err error) {
	var (
		svcList []*kong.Service
	)
	if opt == nil {
		opt = new(kong.ListOpt)
		opt.Size = 100
	}
	for {
		svcList, opt, err = kc.Services.List(context.Background(), opt)
		if len(svcList) == 0 {
			break
		}
		for _, svc := range svcList {
			visit(svc)
		}
		if opt == nil {
			break
		}
	}
	return
}

// ApplyIngressService 创建服务
func ApplyIngressService(kc *kong.Client, orgSvc *kong.Service, existedUpdate bool) (svc *kong.Service, err error) {
	var (
		svcName *string = orgSvc.ID
	)

	if svcName == nil {
		svcName = orgSvc.Name
	}

	if svc, err = kc.Services.Get(context.Background(), svcName); err != nil && !kong.IsNotFoundErr(err) {
		return
	}

	err = nil

	if svc == nil {
		svc, err = kc.Services.Create(context.Background(), orgSvc)
	} else if existedUpdate {
		if orgSvc.Host != nil {
			svc.Host = orgSvc.Host
		}
		if orgSvc.Port != nil {
			svc.Port = orgSvc.Port
		}
		if len(orgSvc.Tags) > 0 {
			svc.Tags = orgSvc.Tags
		}
		if orgSvc.Protocol != nil {
			svc.Protocol = orgSvc.Protocol
		}
		if orgSvc.ReadTimeout != nil {
			svc.ReadTimeout = orgSvc.ReadTimeout
		}
		if orgSvc.WriteTimeout != nil {
			svc.WriteTimeout = orgSvc.WriteTimeout
		}
		if orgSvc.Path != nil {
			svc.Path = orgSvc.Path
		}
		if orgSvc.Retries != nil {
			svc.Retries = orgSvc.Retries
		}
		svc, err = kc.Services.Update(context.Background(), svc)
	} else {
		err = errors.Errorf("service %s existed: host=%s, port=%d", *svcName, *orgSvc.Host, *orgSvc.Port)
	}

	return
}

// IngressAddService 添加服务
type IngressAddService struct {
	BaseIngressAction
	// 服务名称
	ServiceName string
	// 后台主机
	BackendHost string
	// 后台端口
	BackendPort int
	// 是否覆盖
	Override bool
	// 读超时
	ReadTimeout int
	// 写超时
	WriteTimeout int
	// Service协议
	ServiceProtocol string
	// 服务路径
	ServicePath string
	// 重试次数
	Retries int
	// 标签
	Tags []string
}

// IngressListService 添加服务
type IngressListService struct {
	BaseIngressAction
	ServiceName  []string
	OutputFormat string
	Tags         []string
	MatchAllTags bool
}

// GetNameMap 获取名称
func (i *IngressListService) GetNameMap() (ret map[string]bool) {
	ret = make(map[string]bool)
	for _, name := range i.ServiceName {
		ret[name] = true
	}
	return
}

// IngressDeleteService 删除服务
type IngressDeleteService struct {
	BaseIngressAction
	ServiceNames []string
	Force        bool
}

// NewIngressAddService 添加服务
func NewIngressAddService(setting *GlobalSetting) *IngressAddService {
	return &IngressAddService{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// NewIngressListService 添加服务
func NewIngressListService(setting *GlobalSetting) *IngressListService {
	return &IngressListService{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// NewIngressDeleteService 删除服务
func NewIngressDeleteService(setting *GlobalSetting) *IngressDeleteService {
	return &IngressDeleteService{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// ToKongService 获取Service
func (i *IngressAddService) ToKongService() (svc *kong.Service) {

	svc = &kong.Service{
		Name: kong.String(i.ServiceName),
		Host: kong.String(i.BackendHost),
		Port: kong.Int(i.BackendPort),
	}

	for _, tag := range i.Tags {
		svc.Tags = append(svc.Tags, kong.String(tag))
	}

	if i.ServicePath != "" && i.ServicePath != "/" {
		svc.Path = kong.String(i.ServicePath)
	}

	if i.ServiceProtocol != "" {
		svc.Protocol = kong.String(i.ServiceProtocol)
	}

	return
}

// Run 实现新增服务
func (i *IngressAddService) Run() (err error) {
	var (
		kc  *kong.Client
		svc *kong.Service
	)
	if kc, err = i.CreateKong(nil); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}
	svc = i.ToKongService()
	if svc, err = ApplyIngressService(kc, svc, i.Override); err != nil {
		err = errors.Errorf("create service error: %s", err)
		return
	}

	i.Println("service %s created: id=%s", *svc.Name, *svc.ID)

	return
}

// Run 列表服务
func (i *IngressListService) Run() (err error) {
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

	opt.MatchAllTags = i.MatchAllTags
	names = i.GetNameMap()

	switch i.OutputFormat {
	case "table", "":
		i.Println("%-40s %-30s %-22s %-5s %-8s %-20s %-30s", "ID", "Name", "Host", "Port", "Protocol", "Path", "Tags")

		err = ListIngressServices(kc, opt, func(svc *kong.Service) {
			if len(names) > 0 && !(names[*svc.Name] || names[*svc.ID]) {
				return
			}
			i.Println("%-40s %-30s %-22s %-5d %-8s %-20s %-30s", *svc.ID, *svc.Name, *svc.Host, *svc.Port,
				GetIngressServiceProtocol(svc), GetIngressServicePath(svc), GetIngressServiceTagList(svc))
		})
	case "yaml", "json":
		var (
			svcList []*kong.Service
			svcData []byte
		)
		_ = ListIngressServices(kc, opt, func(svc *kong.Service) {
			if len(names) > 0 && !(names[*svc.Name] || names[*svc.ID]) {
				return
			}
			svcList = append(svcList, svc)
		})
		if i.OutputFormat == "yaml" {
			svcData, err = yaml.Marshal(svcList)
		} else {
			svcData, err = json.Marshal(svcList)
		}
		if err == nil {
			i.Println("%s", string(svcData))
		}
	default:
		err = errors.Errorf("not found output format: %s", i.OutputFormat)
	}

	return
}

// Run 删除服务
func (i *IngressDeleteService) Run() (err error) {
	var (
		setting *GlobalSetting = i.GlobalSetting()
		kc      *kong.Client
		svc     *kong.Service
		cmsg    string
	)
	if kc, err = i.CreateKong(nil); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}

	for _, svcNameOrID := range i.ServiceNames {

		if svc, err = GetIngressService(kc, svcNameOrID); err != nil {
			if kong.IsNotFoundErr(err) {
				continue
			}
			return err
		}
		cmsg = fmt.Sprintf("delete service:\n  id=%s,\n  name=%s\n  host=%s\n  port=%d\nproceed? (y/N)",
			*svc.ID, *svc.Name, *svc.Host, *svc.Port)
		if !i.Force && !utils.Confirm(cmsg, setting.InOrStdin(), setting.OutOrStdout()) {
			i.Println("Cancelled.")
			return
		}

		if err = DeleteIngressService(kc, *svc.ID); err != nil {
			if kong.IsNotFoundErr(err) {
				continue
			}
			return err
		}
	}

	err = nil

	return
}
