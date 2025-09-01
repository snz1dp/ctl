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
	"strings"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/cli/output"
	"snz1.cn/snz1dp/snz1dpctl/action"
)

func newListCmd(setting *action.GlobalSetting) *cobra.Command {

	listAction := action.NewListK8sRelease(setting)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "list snz1dp installed bundles on kubernetes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listAction.Run()
		},
	}

	fs := cmd.Flags()
	fs.StringVarP(&listAction.OutputFormat, "output", "o", "table",
		fmt.Sprintf("prints the output in the specified format. Allowed values: %s", strings.Join(output.Formats(), ", ")))
	return cmd
}
