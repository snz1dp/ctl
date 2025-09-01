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
	"github.com/pkg/errors"
)

// IngressLoginConfig 添加服务
type IngressLoginConfig struct {
	BaseIngressAction
	DefaultURL string
	LogoutURL  string
	LoginURL   string
}

// IngressLoginConfig 添加服务
func NewIngressLoginConfig(setting *GlobalSetting) *IngressLoginConfig {
	return &IngressLoginConfig{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// Run 设置
func (s *IngressLoginConfig) Run() (err error) {
	// 添加自定义登录接口
	var (
		xc *XeaiClient
	)

	if xc, err = s.CreateXeai(nil); err != nil {
		err = errors.Errorf("create ingress xeai client error: %s", err)
		return
	}

	if err = xc.SetDefaultLogin(s.DefaultURL, s.LoginURL, s.LogoutURL); err != nil {
		err = errors.Errorf("setup default login config error: %s", err)
		return
	}

	err = nil
	s.Println("ingress login config success.")

	return
}
