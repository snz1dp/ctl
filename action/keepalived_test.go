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
	"net"
	"testing"

	"github.com/ghodss/yaml"
)

func TestShowKeepAlivedConfig(t *testing.T) {

	cfg := keepalivedConfig{
		VirtualInstances: []virtualInstance{
			{},
		},
	}

	bcData, err := yaml.Marshal(cfg)

	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(bcData))

}

func TestShowInterface(t *testing.T) {
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%#v", ifaces)
}

func TestShowStandalone(t *testing.T) {
	a := StandaloneConfig{
		RunFiles: map[string]string{
			"test": "12345567fsafdjsalkfjdsklafjdsalkfjdlsakjfdlsakfdsa",
		},
	}
	m, err := yaml.Marshal(a)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(m))
}
