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

func newStartNodeCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewStartNodejs(setting)
	cmd := &cobra.Command{
		Use:                "node",
		Short:              "nodejs command tool",
		Long:               globalBanner,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Args = args
			return a.Run()
		},
	}
	return cmd
}

func newStartNrmCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewStartNrm(setting)
	cmd := &cobra.Command{
		Use:                "nrm",
		Short:              "nrm command tool",
		Long:               globalBanner,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Args = args
			return a.Run()
		},
	}
	return cmd
}

func newStartNpmCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewStartNpm(setting)
	cmd := &cobra.Command{
		Use:                "npm",
		Short:              "npm command tool",
		Long:               globalBanner,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Args = args
			return a.Run()
		},
	}
	return cmd
}
