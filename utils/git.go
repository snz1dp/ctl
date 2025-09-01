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
	"bytes"
	"context"
	"os/exec"
)

var gitVersion string

// - 是否安装了git
func IsGitInstalled() bool {
	if gitVersion == "" {
		var gitStdout bytes.Buffer
		gitcmd := exec.CommandContext(context.Background(), "git", "version")
		gitcmd.Stdout = &gitStdout
		if err := gitcmd.Run(); err != nil {
			gitVersion = "git not install"
			return false
		} else {
			gitVersion = gitStdout.String()
			return true
		}
	} else if gitVersion == "git not install" {
		return false
	} else {
		return true
	}
}

// - 获取git版本
func GetGitVersion() string {
	if gitVersion == "" {
		IsGitInstalled()
	}
	return gitVersion
}
