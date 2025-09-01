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

package docker

import (
	"github.com/docker/docker/api/types"
	"github.com/wonderivan/logger"
	"golang.org/x/net/context"
	"testing"
)

func TestNewClient(t *testing.T) {
	dockerClient, err := NewClient()
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	defer dockerClient.Close()
	sv, err := dockerClient.ServerVersion(context.Background())
	if err != nil {
		t.Errorf("%v", err)
	}
	logger.Info("%+v", sv)

	info, err := dockerClient.Info(context.Background())
	if err != nil {
		t.Errorf("%v", err)
	}
	logger.Info("%+v", info)

	slt, err := dockerClient.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		t.Errorf("%v", err)
	}
	logger.Info("%+v", slt)

	lst, err := dockerClient.ConfigList(context.Background(), types.ConfigListOptions{})
	if err != nil {
		t.Errorf("%v", err)
	}
	logger.Info("%+v", lst)
}
