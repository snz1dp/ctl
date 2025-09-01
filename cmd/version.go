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
	"fmt"

	"github.com/spf13/cobra"
	"snz1.cn/snz1dp/snz1dpctl/action"
)

func newVersionCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewShowVersion(setting)
	cmd := &cobra.Command{
		Use:   "version",
		Short: "show snz1dp version",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(setting.OutOrStdout(), globalBanner)
			return a.Run()
		},
	}
	fs := cmd.Flags()
	setting.AddK8sFlags(fs)
	return cmd
}
