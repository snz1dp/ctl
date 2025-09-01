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

// UpstreamWrapper 负载封装
type UpstreamWrapper struct {
	kong.Upstream
	Targets []*kong.Target `json:"target,omitempty"`
}

// UpstreamVisit 负载访问
type UpstreamVisit func(*kong.Upstream, []*kong.Target)

// GetIngressUpstream 获取负载
func GetIngressUpstream(kc *kong.Client, nameOrID string) (upm *kong.Upstream, err error) {
	upm, err = kc.Upstreams.Get(context.Background(), kong.String(nameOrID))
	return
}

// GetIngressUpstreamTagList 获取路径
func GetIngressUpstreamTagList(upm *kong.Upstream) (tags string) {
	if upm.Tags == nil || len(upm.Tags) == 0 {
		tags = "<None>"
		return
	}
	rtags := []string{}
	for _, t := range upm.Tags {
		rtags = append(rtags, *t)
	}
	tags = strings.Join(rtags, ",")
	return
}

// GetIngressUpstreamTargetList 获取目标
func GetIngressUpstreamTargetList(targets []*kong.Target) (ts string) {
	if len(targets) == 0 {
		ts = "<None>"
		return
	}
	rts := []string{}
	for _, t := range targets {
		rts = append(rts, fmt.Sprintf("%s(%d)", *t.Target, *t.Weight))
	}
	ts = strings.Join(rts, ",")
	return
}

// GetIngressUpstreamTargets 获取列表
func GetIngressUpstreamTargets(kc *kong.Client, upm *kong.Upstream) (targets []*kong.Target, err error) {
	targets, err = kc.Targets.ListAll(context.Background(), upm.ID)
	return
}

// DeleteIngressUpstream 删除负载
func DeleteIngressUpstream(kc *kong.Client, nameOrID string) (err error) {
	err = kc.Upstreams.Delete(context.Background(), kong.String(nameOrID))
	return
}

// ListIngressUpstreams 列表负载
func ListIngressUpstreams(kc *kong.Client, opt *kong.ListOpt, visit UpstreamVisit) (err error) {
	var (
		upmList    []*kong.Upstream
		upmTargets []*kong.Target
	)
	if opt == nil {
		opt = new(kong.ListOpt)
		opt.Size = 100
	}
	for {
		upmList, opt, err = kc.Upstreams.List(context.Background(), opt)
		if len(upmList) == 0 {
			break
		}
		for _, upm := range upmList {
			if upmTargets, err = GetIngressUpstreamTargets(kc, upm); err != nil {
				return
			}
			visit(upm, upmTargets)
		}
		if opt == nil {
			break
		}
	}
	return
}

// ApplyIngressUpstreamTarget 添加负载主机
func ApplyIngressUpstreamTarget(kc *kong.Client, upm *kong.Upstream, orgTarget *kong.Target, existedUpdate bool) (target *kong.Target, err error) {
	var (
		upmTargets []*kong.Target
	)
	if upmTargets, err = kc.Targets.ListAll(context.Background(), upm.ID); err != nil {
		return
	}
	for _, t := range upmTargets {
		if *t.Target == *orgTarget.Target {
			target = t
		}
	}
	if target == nil {
		orgTarget.Upstream = upm
		target, err = kc.Targets.Create(context.Background(), upm.ID, orgTarget)
	} else if existedUpdate {
		if orgTarget.Weight != nil {
			target.Weight = orgTarget.Weight
		}
		kc.Targets.Delete(context.Background(), upm.ID, target.ID)
		target.ID = nil
		target, err = kc.Targets.Create(context.Background(), upm.ID, orgTarget)
	}
	return
}

// ApplyIngressUpstream 创建负载
func ApplyIngressUpstream(kc *kong.Client, orgUpm *kong.Upstream, existedUpdate bool) (upm *kong.Upstream, err error) {
	var (
		upmName *string = orgUpm.ID
	)

	if upmName == nil {
		upmName = orgUpm.Name
	}

	if upm, err = kc.Upstreams.Get(context.Background(), upmName); err != nil && !kong.IsNotFoundErr(err) {
		return
	}

	err = nil

	if upm == nil {
		upm, err = kc.Upstreams.Create(context.Background(), orgUpm)
	} else if existedUpdate {
		if orgUpm.Algorithm != nil {
			upm.Algorithm = orgUpm.Algorithm
		}
		if orgUpm.Slots != nil {
			upm.Slots = orgUpm.Slots
		}
		if orgUpm.HostHeader != nil {
			upm.HostHeader = orgUpm.HostHeader
		}
		if orgUpm.Healthchecks != nil {
			upm.Healthchecks = orgUpm.Healthchecks
		}
		upm, err = kc.Upstreams.Update(context.Background(), upm)
	} else {
		err = errors.Errorf("upstream %s existed: id=%s, algorithm=%s, slots=%d", *upmName, *upm.ID, *upm.Algorithm, *upm.Slots)
	}

	return
}

// IngressListUpstream 列表
type IngressListUpstream struct {
	BaseIngressAction
	UpstreamName []string
	OutputFormat string
	Tags         []string
	MatchAllTags bool
}

// GetNameMap 获取名称
func (i *IngressListUpstream) GetNameMap() (ret map[string]bool) {
	ret = make(map[string]bool)
	for _, name := range i.UpstreamName {
		ret[name] = true
	}
	return
}

// NewIngressListUpstream 列表
func NewIngressListUpstream(setting *GlobalSetting) *IngressListUpstream {
	return &IngressListUpstream{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// IngressAddUpstream 添加负载
type IngressAddUpstream struct {
	BaseIngressAction
	UpstreamName           string
	HealthPath             string
	HealthInterval         int
	UnhealthyTimeouts      int
	Algorithm              string
	Override               bool
	HTTPSVerifyCertificate bool
	Slots                  int
	Timeout                int
	HTTPFailures           int
	TCPFailures            int
	Successes              int
	Tags                   []string
}

// ToKongUpstream 转Upstream
func (i *IngressAddUpstream) ToKongUpstream() (upm *kong.Upstream) {
	upm = &kong.Upstream{
		Name:  kong.String(i.UpstreamName),
		Slots: kong.Int(i.Slots),
		Healthchecks: &kong.Healthcheck{
			Active: &kong.ActiveHealthcheck{
				Healthy: &kong.Healthy{
					HTTPStatuses: []int{200, 302, 401, 403, 404},
					Successes:    kong.Int(i.Successes),
					Interval:     kong.Int(i.HealthInterval),
				},
				Unhealthy: &kong.Unhealthy{
					HTTPFailures: kong.Int(i.HTTPFailures),
					TCPFailures:  kong.Int(i.TCPFailures),
					Interval:     kong.Int(i.HealthInterval),
					Timeouts:     kong.Int(i.UnhealthyTimeouts),
					HTTPStatuses: []int{429, 500, 501, 502, 503, 504, 505},
				},
				HTTPSVerifyCertificate: kong.Bool(i.HTTPSVerifyCertificate),
				Timeout:                kong.Int(i.Timeout),
			},
			Passive: &kong.PassiveHealthcheck{
				Healthy: &kong.Healthy{
					HTTPStatuses: []int{200, 201, 202, 203, 204, 205, 206,
						207, 208, 226, 300, 301, 302, 303, 304, 305, 306, 307,
						308, 401, 403, 404},
					Successes: kong.Int(1),
				},
				Unhealthy: &kong.Unhealthy{
					HTTPFailures: kong.Int(i.HTTPFailures),
					TCPFailures:  kong.Int(i.TCPFailures),
					HTTPStatuses: []int{429, 500, 502, 503},
					Timeouts:     kong.Int(i.UnhealthyTimeouts),
				},
			},
		},
	}
	if i.HealthPath != "" {
		upm.Healthchecks.Active.HTTPPath = kong.String(i.HealthPath)
	}
	return
}

// IngressDeleteUpstream 删除服务
type IngressDeleteUpstream struct {
	BaseIngressAction
	UpstreamName []string
	Force        bool
}

// NewIngressAddUpstream 添加负载
func NewIngressAddUpstream(setting *GlobalSetting) *IngressAddUpstream {
	return &IngressAddUpstream{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// NewIngressDeleteUpstream 添加负载
func NewIngressDeleteUpstream(setting *GlobalSetting) *IngressDeleteUpstream {
	return &IngressDeleteUpstream{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// Run 实现新增负载
func (i *IngressAddUpstream) Run() (err error) {
	var (
		kc  *kong.Client
		upm *kong.Upstream
	)

	if kc, err = i.CreateKong(nil); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}

	upm = i.ToKongUpstream()

	if upm, err = ApplyIngressUpstream(kc, upm, i.Override); err != nil {
		err = errors.Errorf("create upstream error: %s", err)
		return
	}

	i.Println("upstream %s created: id=%s", *upm.Name, *upm.ID)

	return
}

// Run 实现删除负载
func (i *IngressDeleteUpstream) Run() (err error) {
	var (
		setting *GlobalSetting = i.GlobalSetting()
		kc      *kong.Client
		upm     *kong.Upstream
		cmsg    string
	)
	if kc, err = i.CreateKong(nil); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}

	for _, nameOrID := range i.UpstreamName {

		if upm, err = GetIngressUpstream(kc, nameOrID); err != nil {
			if kong.IsNotFoundErr(err) {
				continue
			}
			return err
		}
		cmsg = fmt.Sprintf("delete upstream:\n  id=%s,\n  name=%s\n  algorithm=%s\n  slots=%d\nproceed? (y/N)",
			*upm.ID, *upm.Name, *upm.Algorithm, *upm.Slots)
		if !i.Force && !utils.Confirm(cmsg, setting.InOrStdin(), setting.OutOrStdout()) {
			i.Println("Cancelled.")
			return
		}

		if err = DeleteIngressUpstream(kc, *upm.ID); err != nil {
			if kong.IsNotFoundErr(err) {
				continue
			}
			return err
		}
	}

	err = nil

	return
}

// Run 实现列表负载
func (i *IngressListUpstream) Run() (err error) {
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
		i.Println("%-38s %-26s %-15s %-8s %-50s %-30s", "ID", "Name", "Algorithm", "Slots", "Targets", "Tags")

		if err = ListIngressUpstreams(kc, opt, func(upm *kong.Upstream, targets []*kong.Target) {
			if len(names) > 0 && !(names[*upm.Name] || names[*upm.ID]) {
				return
			}
			i.Println("%-38s %-26s %-15s %-8d %-50s %-30s", *upm.ID, *upm.Name,
				*upm.Algorithm, *upm.Slots, GetIngressUpstreamTargetList(targets), GetIngressUpstreamTagList(upm))
		}); err != nil {
			return
		}

	case "yaml", "json":
		var (
			upmList []*UpstreamWrapper
			upmData []byte
		)
		if err = ListIngressUpstreams(kc, opt, func(upm *kong.Upstream, targets []*kong.Target) {
			if len(names) > 0 && !(names[*upm.Name] || names[*upm.ID]) {
				return
			}
			upmList = append(upmList, &UpstreamWrapper{Upstream: *upm, Targets: targets})
		}); err != nil {
			return
		}

		if i.OutputFormat == "yaml" {
			upmData, err = yaml.Marshal(upmList)
		} else {
			upmData, err = json.Marshal(upmList)
		}
		if err == nil {
			i.Println("%s", string(upmData))
		}
	default:
		err = errors.Errorf("not found output format: %s", i.OutputFormat)
	}
	return
}

// IngressAddUpstreamTarget 添加目标主机
type IngressAddUpstreamTarget struct {
	BaseIngressAction
	UpstreamName string
	Target       []string
	Weight       int
	Override     bool
}

// IngressRemoveUpstreamTarget 删除目标主机
type IngressRemoveUpstreamTarget struct {
	BaseIngressAction
	UpstreamName string
	Target       []string
	Force        bool
}

// NewIngressAddUpstreamTarget 添加目标主机
func NewIngressAddUpstreamTarget(setting *GlobalSetting) *IngressAddUpstreamTarget {
	return &IngressAddUpstreamTarget{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// NewIngressRemoveUpstreamTarget 添加目标主机
func NewIngressRemoveUpstreamTarget(setting *GlobalSetting) *IngressRemoveUpstreamTarget {
	return &IngressRemoveUpstreamTarget{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// ToUpstreamTarget 转Target
func (i *IngressAddUpstreamTarget) ToUpstreamTarget() (targets []*kong.Target) {
	for _, target := range i.Target {
		targets = append(targets, &kong.Target{
			Target: kong.String(target),
			Weight: kong.Int(i.Weight),
		})
	}
	return
}

// Run 实现新增负载主机
func (i *IngressAddUpstreamTarget) Run() (err error) {
	var (
		kc      *kong.Client
		upm     *kong.Upstream
		targets []*kong.Target = i.ToUpstreamTarget()
	)
	if kc, err = i.CreateKong(nil); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}

	if upm, err = GetIngressUpstream(kc, i.UpstreamName); err != nil {
		err = errors.Errorf("not found upstream %s", i.UpstreamName)
		return
	}

	for _, target := range targets {
		if _, err = ApplyIngressUpstreamTarget(kc, upm, target, i.Override); err != nil {
			err = errors.Errorf("upsream %s target existed: %s", i.UpstreamName, *target.Target)
			return
		}
	}

	s := NewIngressListUpstream(i.setting)
	s.UpstreamName = append(s.UpstreamName, i.UpstreamName)
	err = s.Run()
	return
}

// Run 实现删除
func (i *IngressRemoveUpstreamTarget) Run() (err error) {
	var (
		setting                 *GlobalSetting = i.GlobalSetting()
		kc                      *kong.Client
		upm                     *kong.Upstream
		upmTargets, wDelTargets []*kong.Target
		cmsg                    string
		targetData              map[string]bool = make(map[string]bool)
	)
	if kc, err = i.CreateKong(nil); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}

	if upm, err = GetIngressUpstream(kc, i.UpstreamName); err != nil {
		err = errors.Errorf("not found upstream %s", i.UpstreamName)
		return
	}

	if upmTargets, err = GetIngressUpstreamTargets(kc, upm); err != nil {
		err = errors.Errorf("get upstream %s targets error: %s", *upm.Name, err)
		return
	}

	for _, tname := range i.Target {
		targetData[tname] = true
	}

	cmsg = fmt.Sprintf("delete upstream %s targets:\n", *upm.Name)

	for _, upmTarget := range upmTargets {
		if !targetData[*upmTarget.Target] {
			continue
		}
		wDelTargets = append(wDelTargets, upmTarget)
		cmsg += fmt.Sprintf("  %s(%d)\n", *upmTarget.Target, *upmTarget.Weight)
	}

	cmsg += "proceed? (y/N)"

	if len(wDelTargets) > 0 {
		if !i.Force && !utils.Confirm(cmsg, setting.InOrStdin(), setting.OutOrStdout()) {
			i.Println("Cancelled.")
			return
		}

		for _, upmTarget := range wDelTargets {
			if err = kc.Targets.Delete(context.Background(), upm.ID, upmTarget.ID); err != nil {
				err = errors.Errorf("delete upstream %s target %s error: %s", *upm.Name, *upmTarget.Target, err)
				return
			}
		}
	}

	s := NewIngressListUpstream(i.setting)
	s.UpstreamName = append(s.UpstreamName, i.UpstreamName)
	err = s.Run()

	return
}
