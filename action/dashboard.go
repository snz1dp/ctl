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
	"runtime"

	"github.com/pkg/errors"
	"istio.io/istio/istioctl/pkg/kubernetes"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

// OpenDashboard -
type OpenDashboard struct {
	BaseAction
	HTTPS bool
	Port  int
	Open  bool
}

// NewOpenDashboard -
func NewOpenDashboard(setting *GlobalSetting) *OpenDashboard {
	return &OpenDashboard{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (o *OpenDashboard) Run() (err error) {
	setting := o.GlobalSetting()

	// 初始化Helm操作配置
	InitHelmActionConfig(setting)

	// 检查K8s是否可用
	if err = setting.IsKubernetesReachable(); err != nil {
		o.ErrorExit("%s", err.Error())
		return
	}

	var oldic *InstallConfiguration
	if oldic, err = setting.IsInitialized(); err != nil {
		o.Println("snz1d-%s bundles not install!", utils.Version())
		return nil
	}

	restConfig, err := setting.RESTClientGetter().ToRESTConfig()
	if err != nil {
		o.ErrorExit("%s", err.Error())
		return
	}

	client, err := newKubeExecClient(restConfig)

	if err != nil {
		o.ErrorExit("%s", err.Error())
		return
	}

	localIP := utils.GetExternalIpv4()

	pl, err := client.PodsForSelector(oldic.Snz1dp.Namespace, "app.kubernetes.io/name=ingress")
	if err != nil {
		return fmt.Errorf("not able to locate appgateway pod: %v", err)
	}

	if len(pl.Items) < 1 {
		return errors.New("no appgateway pods found")
	}

	var portProto string = "http"
	var remotePort int = 80

	if o.HTTPS {
		portProto = "https"
		if o.Port == 0 {
			o.Port = 443
		}
		remotePort = 443
	} else if o.Port == 0 {
		o.Port = 80
	}

	podName := pl.Items[0].Name

	fw, err := client.BuildPortForwarder(podName, oldic.Snz1dp.Namespace, "0.0.0.0", int(o.Port), int(remotePort))
	if err != nil {
		o.ErrorExit("could not build port forwarder for snz1dp dashboard: %v", err)
		return err
	}

	hostsFileName := "/etc/hosts"

	switch runtime.GOOS {
	case "windows":
		hostsFileName = `c:\Windows\System32\drivers\etc\hosts`
	}

	if err = kubernetes.RunPortForwarder(fw, func(fw *kubernetes.PortForward) error {
		o.Println("snz1dp dashboard port forwarder listen on %s:%d", localIP, o.Port)
		if oldic.Snz1dp.Ingress.Host != "localhost" && oldic.Snz1dp.Ingress.Host == "127.0.0.1" {
			o.Println("please edit file %s and add follow line: \n%s   %s\n",
				hostsFileName, localIP, oldic.Snz1dp.Ingress.Host)
		}
		dashURL := fmt.Sprintf("%s://localhost/", portProto)
		o.Println("dashboard url: %s", dashURL)
		if o.Open {
			utils.OpenBrowser(dashURL)
		}
		return nil
	}); err != nil {
		o.ErrorExit("%v", err)
		return nil
	}

	return
}
