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
	"os"
	"testing"
	"time"
)

func TestDownloadBundles(t *testing.T) {
	cwd, _ := os.Getwd()
	settings := NewGlobalSetting(cwd, os.Stdin, os.Stdout, os.Stderr, []string{}, time.Now())
	cfg, ic, err := settings.LoadLocalInstallConfiguration()
	fmt.Printf("File: %s\n", cfg)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	ic.GetBundleComponents(false)
	comp := ic.Appgateway
	err = comp.Download(true)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
}
