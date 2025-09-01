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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"

	"github.com/pkg/errors"
)

// GenerateRSAKeyPair 生成密钥对
func GenerateRSAKeyPair(bits int) (rsaKey *rsa.PrivateKey, err error) {
	rsaKey, err = rsa.GenerateKey(rand.Reader, bits)
	return
}

// RSAPrivateKeyToPem 转成PEM格式
func RSAPrivateKeyToPem(rsaKey *rsa.PrivateKey) (pemContent string, err error) {

	var (
		derStream []byte
		pemBlock  *pem.Block
		outBuffer *bytes.Buffer
	)

	derStream = x509.MarshalPKCS1PrivateKey(rsaKey)

	pemBlock = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}

	outBuffer = bytes.NewBuffer(nil)

	if err = pem.Encode(outBuffer, pemBlock); err != nil {
		return
	}

	pemContent = outBuffer.String()
	return
}

// DecodePrivateKeyFromPEM -
func DecodePrivateKeyFromPEM(pemContent string) (rsaKey *rsa.PrivateKey, err error) {
	var (
		pemBlock *pem.Block
	)
	if pemBlock, _ = pem.Decode([]byte(pemContent)); pemBlock == nil {
		err = errors.Errorf("pem format error")
		return
	}

	if strings.Contains(strings.ToLower(pemBlock.Type), "private") {
		rsaKey, err = x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
		if err != nil {
			var pkit interface{}
			if pkit, err = x509.ParsePKCS8PrivateKey(pemBlock.Bytes); err != nil {
				return
			}
			rsaKey = pkit.(*rsa.PrivateKey)
		}
	} else {
		err = errors.Errorf("not private key")
	}
	return
}

// RSAPublicKeyToPem 转成PEM格式
func RSAPublicKeyToPem(rsaKey *rsa.PublicKey) (pemContent string, err error) {

	var (
		derStream []byte
		pemBlock  *pem.Block
		outBuffer *bytes.Buffer
	)

	if derStream, err = x509.MarshalPKIXPublicKey(rsaKey); err != nil {
		return
	}

	pemBlock = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derStream,
	}

	outBuffer = bytes.NewBuffer(nil)

	if err = pem.Encode(outBuffer, pemBlock); err != nil {
		return
	}

	pemContent = outBuffer.String()
	return

}

// DecodePublicKeyFromPEM 加载公钥匙
func DecodePublicKeyFromPEM(pemContent string) (rsaKey *rsa.PublicKey, privKey *rsa.PrivateKey, err error) {
	var (
		pemBlock *pem.Block
	)
	if pemBlock, _ = pem.Decode([]byte(pemContent)); pemBlock == nil {
		err = errors.Errorf("pem format error")
		return
	}

	if strings.Contains(strings.ToLower(pemBlock.Type), "private") {
		if privKey, err = x509.ParsePKCS1PrivateKey(pemBlock.Bytes); err != nil {
			return
		}
		rsaKey = &privKey.PublicKey
	} else {
		if rsaKey, err = x509.ParsePKCS1PublicKey(pemBlock.Bytes); err != nil {
			var p interface{}
			if p, err = x509.ParsePKIXPublicKey(pemBlock.Bytes); err != nil {
				return
			}
			rsaKey = p.(*rsa.PublicKey)
		}
	}
	return
}
