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

package cmd

import (
	"github.com/spf13/cobra"
	"snz1.cn/snz1dp/snz1dpctl/action"
)

func newPullK8sImageCmd(setting *action.GlobalSetting) *cobra.Command {
	action := action.NewPullK8sImage(setting)
	cmd := &cobra.Command{
		Use:   "pullimage",
		Short: "pull kubernetes docker images to local computer",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return action.Run()
		},
	}
	fs := cmd.Flags()
	fs.StringVarP(&action.Version, "version", "v", "", "kubernetes version")

	fs.BoolVar(&action.SaveLocalFile, "save-file", false, "save k8s images to local file")
	fs.BoolVarP(&action.Force, "force", "f", false, "force to pull k8s image")
	return cmd
}

func newLoadK8sImageCmd(setting *action.GlobalSetting) *cobra.Command {
	action := action.NewLoadK8sImage(setting)
	cmd := &cobra.Command{
		Use:   "loadimage",
		Short: "load kubernetes docker images from local file",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return action.Run()
		},
	}
	fs := cmd.Flags()
	fs.StringVarP(&action.Version, "version", "v", "", "kubernetes version")
	fs.StringVarP(&action.Filename, "file", "f", "", "image tar file")
	return cmd
}

func newGetK8sVersionCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewGetK8sVersion(setting)
	cmd := &cobra.Command{
		Use:   "version",
		Short: "show kubernetes version info",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}
	fs := cmd.Flags()
	setting.AddK8sFlags(fs)
	return cmd
}

func newProfileApplyCmd(setting *action.GlobalSetting) *cobra.Command {
	init := action.NewInit(setting)
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "applly snz1dp install profile to kubernetes",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return init.Run()
		},
	}
	fs := cmd.Flags()
	setting.AddK8sFlags(fs)

	fs.StringVar(&setting.StorageClass, "storage-class", action.DefaultStorageClass, "specifies the Kubernetes persistent volumn class")
	fs.BoolVar(&init.Really, "really", false, "really to execute command.")
	fs.BoolVarP(&init.Force, "force", "f", false, "force set proceed to yes.")
	return cmd
}

func newFetchProfile(setting *action.GlobalSetting) *cobra.Command {
	ac := action.NewFetchProfile(setting)
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "fetch snz1dp snz1dp install profile to local",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ac.Run()
		},
	}
	fs := cmd.Flags()
	setting.AddK8sFlags(fs)
	return cmd
}

func newK8sCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s",
		Short: "snz1dp bundle kubernetes commands",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		// 在K8s中安装部署
		newProfileApplyCmd(setting),
		// 拉取配置
		newFetchProfile(setting),
		// 删除安装
		newFiniCmd(setting),
		//列出已安装
		newListCmd(setting),
		// K8s操作
		newPullK8sImageCmd(setting),
		newLoadK8sImageCmd(setting),
		newGetK8sVersionCmd(setting),
	)
	return cmd
}
