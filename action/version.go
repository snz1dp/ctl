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
	"context"
	"fmt"
	"strings"

	"snz1.cn/snz1dp/snz1dpctl/docker"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

// ShowVersion -
type ShowVersion struct {
	BaseAction
}

// NewShowVersion -
func NewShowVersion(setting *GlobalSetting) *ShowVersion {
	return &ShowVersion{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *ShowVersion) Run() (err error) {
	setting := s.GlobalSetting()

	// 安装配置文件
	if _, _, err = setting.LoadLocalInstallConfiguration(); err != nil {
		s.ErrorExit("load config error: %s", err)
		return
	}

	checkNewVersion(s.GlobalSetting())

	// 初始化Helm操作配置
	InitHelmActionConfig(setting)

	s.Println("%s version  : %s", BaseConfig.Snz1dp.Ctl.Name, utils.Version())
	s.Println("source gitcommitid : %s", utils.GitCommitID())

	s.Println("")
	s.Println("%s", utils.GetGitVersion())

	dc, err := docker.NewClient()
	if err != nil {
		s.Println("docker cli version    : not install")
		s.Println("docker server version : not install")
	} else {
		defer dc.Close()
		s.Println("docker cli version        : %v", dc.ClientVersion())
		sv, err := dc.ServerVersion(context.Background())
		if err != nil {
			s.Println("docker server version     : not running")
		} else {
			tbox := fmt.Sprintf("%s %s %s-%s", sv.Platform.Name, sv.Version, sv.Os, sv.Arch)
			if strings.Contains(sv.KernelVersion, "boot2docker") {
				tbox = fmt.Sprintf("%s (%s)", tbox, "Legacy desktop")
			}
			s.Println("docker server version     : %s\n%v", tbox, sv)
		}
	}

	// 显示版本
	return nil
}
