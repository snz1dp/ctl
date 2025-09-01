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

func newDashboardCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewOpenDashboard(setting)
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "forward snz1dp dashboard to local port",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}
	fs := cmd.Flags()
	fs.BoolVar(&a.HTTPS, "https", false, "listen https port for snz1dp dashboard.")
	fs.BoolVar(&a.Open, "open", true, "open snz1dp dashboard url in browser.")
	fs.IntVar(&a.Port, "port", 0, "local port for snz1dp dashboard.")
	return cmd
}
