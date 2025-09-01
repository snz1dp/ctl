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
	"os"

	"github.com/pkg/errors"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

// Fini 撤销动作
type Fini struct {
	BaseAction
	Force  bool
	Really bool
}

// NewFini 创建
func NewFini(setting *GlobalSetting) *Fini {
	return &Fini{
		BaseAction: BaseAction{
			setting: setting,
		},
		Force:  false,
		Really: false,
	}
}

// Run 执行
func (f *Fini) Run() error {
	f.Info("execute fini snz1dp action")
	var err error

	// 全局配置
	setting := f.GlobalSetting()
	setting.InitLogger("remove")

	if !f.Really && !utils.Confirm("will uninstall snz1dp-"+utils.Version()+" profile, proceed? (y/N)", setting.InOrStdin(), setting.OutOrStdout()) {
		f.Println("Cancelled.")
		return nil
	}

	// 初始化Helm操作配置
	InitHelmActionConfig(setting)

	// 检查K8s是否可用
	if err := setting.IsKubernetesReachable(); err != nil {
		f.ErrorExit("%s", err)
		return err
	}

	// 变量定义
	var (
		ic         *InstallConfiguration
		saveold    bool = false
		compNames  []string
		components map[string]Component
	)

	// 安装配置文件
	configFilePath, lic, _ := setting.LoadLocalInstallConfiguration()

	// 加载已安装信息
	if ic, _ = setting.IsInitialized(); ic != nil {
		f.Println("uninstall snz1dp-%s profile...", ic.Snz1dp.Version)
		saveold = true
		ic.Kubernetes.Apiserver = setting.KubeAPIServer()
		ic.Kubernetes.Config = setting.KubeConfig()
		ic.Kubernetes.Context = setting.KubeContext()
		ic.Kubernetes.Token = setting.KubeToken()
	} else {
		if !f.Force {
			if !utils.Confirm("the snz1dp-"+utils.Version()+" profile not found, Proceed? (y/N)", setting.InOrStdin(), setting.OutOrStdout()) {
				f.Println("Cancelled.")
				return nil
			}
			f.Force = true
		}

		if fst, err := os.Stat(configFilePath); err != nil {
			f.ErrorExit("not found file: %s", configFilePath)
			return err
		} else if fst.IsDir() {
			err := errors.Errorf("not found file: %s", configFilePath)
			f.ErrorExit(err.Error())
			return err
		}

		//加载本地安装配置
		ic = lic
		f.Println("force uninstall snz1dp-%s bundles...", ic.Snz1dp.Version)
		return nil
	}

	// 核心组件
	compNames, components, _ = ic.GetBundleComponents(false)

	namesLen := len(compNames)
	for i := 0; i < namesLen; i++ {
		v := compNames[namesLen-(i+1)]
		if err := components[v].DoFini(f); err != nil {
			f.Error("uninstall %s error: ", v, err)
			f.Println("uninstall %s error: %s", v, err)
		}
	}

	// 删除配置
	setting.storage.Delete()

	if !saveold {
		return nil
	}

	err = setting.SaveLocalInstallConfiguration(ic, configFilePath)
	if err != nil {
		f.ErrorExit("save %s error: %s", configFilePath, err)
		return err
	}

	return nil
}
