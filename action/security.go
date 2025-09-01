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
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

const (
	normalAesKey   = "GbM'[]20dks+_!2020@3,.*&&^@#1952"
	registryAeskey = "PO'wM18dks+_!1010}9,.*&&^@#301K3"
	onceAesKey     = "000000000000"
)

func (ic *InstallConfiguration) encryptNormalPasssword(password string) (encodedPassword string, err error) {
	encodedPassword, err = utils.EncryptAesHex([]byte(password), []byte(ic.Encryption.packpass), []byte(onceAesKey))
	return
}

func (ic *InstallConfiguration) decryptNormalPassword(encodedPassword string) (password string, err error) {
	var (
		srcBytes []byte
	)
	if srcBytes, err = utils.DecryptAesHex(encodedPassword, []byte(ic.Encryption.packpass), []byte(onceAesKey)); err != nil {
		return
	}
	password = string(srcBytes)
	return
}

func encryptRegistryPasssword(password string) (encodedPassword string, err error) {
	encodedPassword, err = utils.EncryptAesHex([]byte(password), []byte(registryAeskey), []byte(onceAesKey))
	return
}

func decryptRegistryPassword(encodedPassword string) (password string, err error) {
	var (
		srcBytes []byte
	)
	if srcBytes, err = utils.DecryptAesHex(encodedPassword, []byte(registryAeskey), []byte(onceAesKey)); err != nil {
		return
	}
	password = string(srcBytes)
	return
}
