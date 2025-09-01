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

package kube

import (
	"fmt"
	"github.com/wonderivan/logger"
	helmAction "helm.sh/helm/v3/pkg/action"
	helmCli "helm.sh/helm/v3/pkg/cli"
	"testing"
)

func debug(f string, v ...interface{}) {
	logger.Debug(f, v)
}

func TestGetDefaultStorageClass(t *testing.T) {

	helmSetting := helmCli.New()
	helmActionConfig := new(helmAction.Configuration)
	helmDriver := ""

	if err := helmActionConfig.Init(helmSetting.RESTClientGetter(), "default", helmDriver, debug); err != nil {
		t.Error(err)
	}

	client, err := helmActionConfig.KubernetesClientSet()
	if err != nil {
		t.Error(err)
	}

	classNames, err := GetStorageClassNames(client)
	if err != nil {
		t.Error(err)
	}
	t.Log(classNames)
	fmt.Printf("%#v", classNames)
}
