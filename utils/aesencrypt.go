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
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
)

// EncryptAesHex 进行AES加密并返回
func EncryptAesHex(srcdata, cipherKey, onceAesKey []byte) (hexout string, err error) {
	var (
		block  cipher.Block
		aesgcm cipher.AEAD
	)
	if block, err = aes.NewCipher(cipherKey); err != nil {
		return
	}

	if aesgcm, err = cipher.NewGCM(block); err != nil {
		return
	}
	hexout = hex.EncodeToString(aesgcm.Seal(nil, onceAesKey, srcdata, nil))
	return
}

// DecryptAesHex 进行aes解密
func DecryptAesHex(cipherHex string, cipherKey, onceAesKey []byte) (plainout []byte, err error) {
	var (
		decodedPassBytes []byte
		block            cipher.Block
		aesgcm           cipher.AEAD
	)
	if decodedPassBytes, err = hex.DecodeString(cipherHex); err != nil {
		return
	}
	block, err = aes.NewCipher(cipherKey)
	if err != nil {
		return
	}
	aesgcm, err = cipher.NewGCM(block)
	if err != nil {
		return
	}
	plainout, err = aesgcm.Open(nil, []byte(onceAesKey), decodedPassBytes, nil)
	return
}
