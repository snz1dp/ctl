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
	"fmt"

	"snz1.cn/snz1dp/snz1dpctl/kube"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

// Init 初始化动作
type Init struct {
	BaseAction
	Force  bool
	Really bool
}

// NewInit 创建
func NewInit(setting *GlobalSetting) *Init {
	return &Init{
		BaseAction: BaseAction{
			setting: setting,
		},
		Force: false,
	}
}

// Run 初始化执行
func (i *Init) Run() error {
	i.Info("execute init snz1dp action")

	var err error

	setting := i.GlobalSetting()
	setting.InitLogger("apply")

	//加载本地安装配置
	configFilePath, ic, err := setting.LoadLocalInstallConfiguration()
	if err != nil {
		i.ErrorExit("load %s error: %v", configFilePath, err)
		return err
	}

	// 初始化Helm操作配置
	InitHelmActionConfig(setting)

	// 检查K8s是否可用
	if err := setting.IsKubernetesReachable(); err != nil {
		i.ErrorExit("%s", err)
		return err
	}

	client, err := setting.KubernetesClientSet()
	if err != nil {
		i.ErrorExit("%s", err)
		return err
	}

	var icData []byte
	var oldComponents map[string]Component

	// 加载已安装配置
	var oldic *InstallConfiguration
	if oldic, _ = setting.IsInitialized(); oldic != nil {
		_, oldComponents, _ = oldic.GetBundleComponents(false)
	} else {
		if err = kube.CreateNamespace(client, ic.Snz1dp.Namespace); err != nil {
			i.ErrorExit("%s", err)
			return err
		}
	}

	// 创建名字空间与的拉取密钥
	for _, ns := range ic.GetK8sNamespaces() {

		if ic.Snz1dp.Namespace != ns {
			if err = kube.CreateNamespace(client, ns); err != nil {
				i.ErrorExit("%s", err)
			}
		}

		v := ic.Snz1dp.Registry
		k := v.GetDockerRepoURL()

		if err = kube.CreateDockerRegistrySecret(client, ns, v.GetK8sImagePullSecretName(), k, v.Username, v.Password,
			map[string]string{}); err != nil {
			i.ErrorExit("%s", err)
			return err
		}

	}

	var (
		compNames  []string
		components map[string]Component
	)

	// 核心组件
	if compNames, components, err = ic.GetBundleComponents(false); err != nil {
		i.ErrorExit("load bundle error: %s", err)
	}

	var saveconf bool = oldic != nil
	var newCompNames []string
	prmsg := bytes.NewBuffer(nil)

	storageClassNames, err := kube.GetStorageClassNames(client)
	if err != nil {
		i.ErrorExit("%s", err)
		return err
	}

	if !storageClassNames[setting.StorageClass] {
		chclass := func() string {
			for k := range storageClassNames {
				return k
			}
			return ""
		}()

		if chclass == "" {
			i.ErrorExit("not found storageclass in kubernetes!")
			return nil
		}

		if !i.Force {
			fmsg := fmt.Sprintf("StorageClass %s not existed, use %s instead, proceed? (y/N)",
				setting.StorageClass, chclass)
			if !utils.Confirm(fmsg, setting.InOrStdin(), setting.OutOrStdout()) {
				i.Println("Cancelled.")
				return nil
			}
		}
		setting.StorageClass = chclass
		ic.Kubernetes.Storageclass = chclass
		saveconf = true
	}

	// 预备安装
	for _, v := range compNames {
		comp := components[v]
		if len(oldComponents) > 0 {
			// 如果是可选安装并原来已安装过的
			if comp.BeInstall() != oldComponents[v].BeInstall() {
				actionval := "install"
				versionval := comp.GetVersion()

				if !comp.BeInstall() {
					actionval = "uninstall"
					versionval = oldComponents[v].GetVersion()
				}
				prmsg.WriteString(fmt.Sprintf("  %s %s-%s\n", actionval, v, versionval))

				// 已安装表示删除
				if oldComponents[v].BeInstall() {
					// 加入操作
					newCompNames = append(newCompNames, v)
					//可选安装并有配置则保存旧的配置
					saveconf = true
					CopyComponent(comp, oldComponents[v])
					comp.UnInstall()
					continue
				}
			} else {
				saveconf = true
				CopyComponent(comp, oldComponents[v])
				continue
			}
		} else if comp.BeInstall() {
			// 表示安装
			prmsg.WriteString(fmt.Sprintf("  install %s-%s\n", v, comp.GetVersion()))
		} else {
			continue
		}

		// 加入操作
		newCompNames = append(newCompNames, v)

		i.Info("prepare %s-%s config...", v, comp.GetVersion())

		if ch, err := comp.PreInit(i); err != nil {
			i.ErrorExit("%s\nplease fix %s", err.Error(), configFilePath)
			return err
		} else if ch && !saveconf {
			saveconf = true
		}

	}

	compNames, _ = newCompNames, compNames

	if len(compNames) == 0 {
		i.Println("snz1dp-%s profile already applied to kubernetes and not change!", utils.Version())
		return nil
	}

	if !i.Really {
		confirmmsg := fmt.Sprintf("apply snz1dp-%s profile to kubernetes:\n\n%s\nproceed? (y/N)", utils.Version(), prmsg.String())
		if !utils.Confirm(confirmmsg, setting.InOrStdin(), setting.OutOrStdout()) {
			i.Println("Cancelled.")
			return nil
		}
	}

	// 保存
	if saveconf {
		i.Info("save config to file: %s", configFilePath)
		err = setting.SaveLocalInstallConfiguration(ic, configFilePath)
		if err != nil {
			i.ErrorExit("save %s error: %s", configFilePath, err)
			return err
		}
	}

	i.Println("apply snz1dp-%s bundles to kubernetes...", ic.Snz1dp.Version)

	// 执行安装
	for _, v := range compNames {
		comp := components[v]
		if !comp.BeInstall() &&
			len(oldComponents) > 0 &&
			oldComponents[v].BeInstall() {

			i.Info("uninstall %s-%s...", v, oldComponents[v].GetVersion())
			if err := oldComponents[v].DoFini(i); err != nil {
				i.ErrorExit("%s", err)
				return err
			}
			continue
		}

		var spinner *utils.WaitSpinner = utils.NewSpinner(
			fmt.Sprintf("prepare %s-%s docker images...", v, comp.GetVersion()), setting.OutOrStdout())

		if err = comp.PushDockerImages(true); err != nil {
			spinner.Close()
			i.ErrorExit("failed: %s", err)
			return err
		}
		spinner.Close()
		i.Println("ok!")

		i.Info("install %s-%s...", v, comp.GetVersion())
		if err := comp.DoInit(i); err != nil {
			i.ErrorExit("%s", err)
			return err
		}
	}

	ic.Kubernetes = KubeConfig{}
	ic.Kubernetes.Storageclass = setting.StorageClass

	icData, err = ic.ToYaml()
	if err != nil {
		i.ErrorExit("can not convert install config to yaml format!")
		return err
	}

	// 安装完成后保存安装配置
	if oldic != nil {
		err = setting.storage.Update(icData)
	} else {
		err = setting.storage.Create(icData)
	}

	if err != nil {
		i.ErrorExit("%s", err)
		return err
	}

	i.Println("apply snz1dp-%s bundles to kubernetes success!", ic.Snz1dp.Version)

	// 显示信息
	return renderInstallConfigurationInfo(setting.OutOrStdout(), ic)
}
