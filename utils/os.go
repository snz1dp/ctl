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

package utils

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

func FixBinSearchPath(binpath string) (err error) {
	pathVal := os.Getenv("PATH")
	if strings.Index(pathVal, binpath) < 0 {
		if runtime.GOOS == "windows" {
			err = os.Setenv("PATH", fmt.Sprintf("%s;%s", binpath, pathVal))
		} else {
			err = os.Setenv("PATH", fmt.Sprintf("%s:%s", binpath, pathVal))
		}
	}
	return
}
