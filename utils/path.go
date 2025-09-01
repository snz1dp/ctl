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
	homedir "github.com/mitchellh/go-homedir"
	"os"
	"path"
)

const (
	isDev = true
)

const (
	// InstallConfigurationFileName - 配置文件名称
	InstallConfigurationFileName = "global.yaml"
)

var (
	// ExecutableDirectory 执行文件目录
	ExecutableDirectory string
	// UserHomePath - 用户主目录
	UserHomePath string
	// KubeConfigFile 缺省kubeconfig文件路径
	KubeConfigFile string
)

// GetExecutableFile -
func GetExecutableFile() string {
	filename, err := os.Executable()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return filename
}

// GetExecutableDirectory -
func GetExecutableDirectory() string {
	cwdPath := path.Join(path.Dir(GetExecutableFile()), "")
	return cwdPath
}

// GetUserHomeDir -
func GetUserHomeDir() string {
	home, _ := homedir.Dir()
	return home
}

// ExpandUserDir 获取正式的用户目录
func ExpandUserDir(upath string) (string, error) {
	return homedir.Expand(upath)
}

// GetKubeConfigFile -
func GetKubeConfigFile() string {
	kubeconfigFile := path.Join(GetUserHomeDir(), ".kube", "config")
	return kubeconfigFile
}

// Init 全局初始化
func init() {
	KubeConfigFile = GetKubeConfigFile()
	ExecutableDirectory = GetExecutableDirectory()
	UserHomePath = GetUserHomeDir()
}

// IsDev -
func IsDev() bool {
	return isDev
}
