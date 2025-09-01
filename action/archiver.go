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
	archiver "github.com/mholt/archiver/v3"
)

// UnarchiveBundle -
func UnarchiveBundle(bcFile string, destDir string) error {
	return archiver.Unarchive(bcFile, destDir)
}

// ArchiveBundle -
func ArchiveBundle(bcdir []string, destFile string) error {
	return archiver.Archive(bcdir, destFile)
}

// ExtractBundleFile -
func ExtractBundleFile(bcFile string, srcName string, destFile string) error {
	return archiver.Extract(bcFile, srcName, destFile)
}
