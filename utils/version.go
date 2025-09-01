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
	"github.com/GeertJohan/go.rice"
	"strings"
)

const (
	ctlVersion = "1.0.0-alpha"
)

// Version 版本
func Version() string {
	vbox := rice.MustFindBox("../asset/version")
	return vbox.MustString("VERSION")
}

// UserAgent -
func UserAgent() string {
	return fmt.Sprintf("Snz1dpCtl/%s", Version())
}

// GitCommitID -
func GitCommitID() string {
	//git rev-parse --short HEAD
	vbox := rice.MustFindBox("../asset/version")
	return vbox.MustString("COMMITID")
}

var (
	// VersionBig 版本大
	VersionBig = 1
	// VersionSmall 版本小
	VersionSmall = -1
	// VersionEqual 版本相等
	VersionEqual = 0
)

// CompareStrVer 比较版本
func CompareStrVer(verA, verB string) int {
	verStrArrA := spliteStrByNet(verA)
	verStrArrB := spliteStrByNet(verB)
	lenStrA := len(verStrArrA)
	lenStrB := len(verStrArrB)
	if lenStrA != lenStrB {
		panic("")
	}
	return compareArrStrVers(verStrArrA, verStrArrB)
}

func compareArrStrVers(verA, verB []string) int {
	for index := range verA {
		littleResult := compareLittleVer(verA[index], verB[index])
		if littleResult != VersionEqual {
			return littleResult
		}
	}
	return VersionEqual
}

func compareLittleVer(verA, verB string) int {
	bytesA := []byte(verA)
	bytesB := []byte(verB)
	lenA := len(bytesA)
	lenB := len(bytesB)
	if lenA > lenB {
		return VersionBig
	}
	if lenA < lenB {
		return VersionSmall
	}
	return compareByBytes(bytesA, bytesB)
}

func compareByBytes(verA, verB []byte) int {
	for index := range verA {
		if verA[index] > verB[index] {
			return VersionBig
		}
		if verA[index] < verB[index] {
			return VersionSmall
		}

	}
	return VersionEqual
}

func spliteStrByNet(strV string) []string {
	return strings.Split(strV, ".")
}
