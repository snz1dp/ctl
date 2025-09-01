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
	"github.com/GeertJohan/go.rice"
	"io"
	"text/template"
)

func renderInstallConfigurationInfo(out io.Writer, ic *InstallConfiguration) error {
	templateBox, err := rice.FindBox("template")
	if err != nil {
		return err
	}
	tpldata, err := templateBox.String("info.tpl")
	if err != nil {
		return err
	}

	tpl, err := template.New("info").Parse(tpldata)
	if err != nil {
		return err
	}
	return tpl.Execute(out, ic)
}

func renderInstallBundlesVersion(out io.Writer, ic *InstallConfiguration) error {
	templateBox, err := rice.FindBox("template")
	if err != nil {
		return err
	}
	tpldata, err := templateBox.String("version.tpl")
	if err != nil {
		return err
	}

	tpl, err := template.New("version").Parse(tpldata)
	if err != nil {
		return err
	}
	return tpl.Execute(out, ic)
}
