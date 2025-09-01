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

func newFiniCmd(setting *action.GlobalSetting) *cobra.Command {
	fini := action.NewFini(setting)
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "remove snz1dp profile and uninstall bundles from kubernetes",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fini.Run()
		},
	}
	fs := cmd.Flags()
	setting.AddK8sFlags(fs)
	fs.BoolVar(&fini.Force, "force", false, "force uninstall if snz1dp not existed.")
	fs.BoolVar(&fini.Really, "really", false, "really to execute command.")
	return cmd
}
