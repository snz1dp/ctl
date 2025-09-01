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
	"github.com/ghodss/yaml"
	"path"
	"testing"
)

const testPostgresYaml = `
version: "9.6"
install: true
host: postgres
port: 5432
admin:
  username: postgres
  password: ""
`

func TestLoadInstallConfiguration(t *testing.T) {

	var postgresConfig = &PostgresConfig{}
	err := yaml.Unmarshal([]byte(testPostgresYaml), postgresConfig)
	if err != nil {
		t.Error(err)
		return
	}

	testDGCFilePath := path.Join("/Users/neeker/Documents/snz1dp/installer", "config", "global.yaml")
	fmt.Println(testDGCFilePath)
	installConfiguration, err := LoadInstallConfigurationFromFile(testDGCFilePath)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(installConfiguration)

	yamlBytes, err := yaml.Marshal(installConfiguration)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(string(yamlBytes))

}

func TestShowHelathcheck(t *testing.T) {

	a := HealthCheckConfig{
		Test: []string{"CMD"},
	}

	b, err := yaml.Marshal(a)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(b))

}
