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
	"io"

	"github.com/gosuri/uitable"
	helmAction "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli/output"
	"helm.sh/helm/v3/pkg/release"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

// ListK8sRelease - 列出所有已安装组件动作
type ListK8sRelease struct {
	BaseAction
	OutputFormat  string
	AllNamespaces bool
	// All ignores the limit/offset
	All bool
	// Limit is the number of items to return per Run()
	Limit int
	// Offset is the starting index for the Run() call
	Offset int
	// Filter is a filter that is applied to the results
	Filter       string
	Short        bool
	Uninstalled  bool
	Superseded   bool
	Uninstalling bool
	Deployed     bool
	Failed       bool
	Pending      bool
}

// NewListK8sRelease - 插件List表动作
func NewListK8sRelease(setting *GlobalSetting) *ListK8sRelease {
	newList := &ListK8sRelease{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
	return newList
}

// Run - 列表
func (l *ListK8sRelease) Run() error {
	// 变量定义
	var (
		oldic *InstallConfiguration
		err   error
	)

	setting := l.GlobalSetting()
	// 初始化Helm操作配置
	InitHelmActionConfig(setting)

	// 检查K8s是否可用
	if err = setting.IsKubernetesReachable(); err != nil {
		l.ErrorExit("%s", err.Error())
		return err
	}

	if oldic, _ = setting.IsInitialized(); oldic == nil {
		l.ErrorExit("snz1dp-%s bundles not install!", utils.Version())
		return err
	}

	var outfmt output.Format = output.Format(l.OutputFormat)

	helmList := helmAction.NewList(l.setting.helmConfig)
	helmList.AllNamespaces = l.AllNamespaces
	helmList.All = l.All
	helmList.Limit = l.Limit
	helmList.Offset = l.Offset
	helmList.Filter = l.Filter
	helmList.Short = l.Short
	helmList.Uninstalled = l.Uninstalled
	helmList.Superseded = l.Superseded
	helmList.Uninstalling = l.Uninstalling
	helmList.Deployed = l.Deployed
	helmList.Failed = l.Failed
	helmList.Pending = l.Pending

	rls, err := helmList.Run()
	if err != nil {
		l.ErrorExit("%s", err.Error())
		return err
	}
	outfmt.Write(setting.OutOrStdout(), newReleaseListWriter(rls))
	return nil
}

func (r *releaseListWriter) WriteTable(out io.Writer) error {
	table := uitable.New()
	table.AddRow("NAME", "NAMESPACE", "UPDATED", "STATUS", "VERSION")
	for _, r := range r.releases {
		table.AddRow(r.Name, r.Namespace, r.Updated, r.Status, r.Version)
	}
	return output.EncodeTable(out, table)
}

func (r *releaseListWriter) WriteJSON(out io.Writer) error {
	return output.EncodeJSON(out, r.releases)
}

func (r *releaseListWriter) WriteYAML(out io.Writer) error {
	return output.EncodeYAML(out, r.releases)
}

type releaseElement struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Updated   string `json:"updated"`
	Status    string `json:"status"`
	Version   string `json:"version"`
}

type releaseListWriter struct {
	releases []releaseElement
}

func newReleaseListWriter(releases []*release.Release) *releaseListWriter {
	// Initialize the array so no results returns an empty array instead of null
	elements := make([]releaseElement, 0, len(releases))
	for _, r := range releases {
		element := releaseElement{
			Name:      r.Name,
			Namespace: r.Namespace,
			Status:    r.Info.Status.String(),
			Version:   r.Chart.Metadata.AppVersion,
		}
		t := "-"
		if tspb := r.Info.LastDeployed; !tspb.IsZero() {
			t = tspb.String()
		}
		element.Updated = t
		elements = append(elements, element)
	}
	return &releaseListWriter{elements}
}
