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
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"

	"github.com/pkg/errors"
)

// StartKubectl -
type StartKubectl struct {
	BaseAction
	Args []string
}

// NewStartKubectl -
func NewStartKubectl(setting *GlobalSetting) *StartKubectl {
	return &StartKubectl{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *StartKubectl) Run() error {
	setting := s.GlobalSetting()
	var (
		mkfpath string
		gzfpath string
		err     error
	)

	setting.InitLogger(BaseConfig.Kubectl.Name)

	mkfpath, err = kubectlFileExisted(setting.GetBinDir())
	if err != nil {
		if gzfpath, err = downloadKubectl(s, setting.OutOrStdout(), false); err != nil {
			s.ErrorExit("%s", err.Error())
			return err
		}
		if err = unarchiveKubectl(gzfpath, setting.GetBinDir()); err != nil {
			s.ErrorExit("%s", err.Error())
			return err
		}
	}

	kubectlArgs := s.Args

	kubectl := exec.CommandContext(context.Background(), mkfpath, kubectlArgs...)
	kubectl.Stdin = setting.InOrStdin()
	kubectl.Stdout = setting.OutOrStdout()
	kubectl.Stderr = setting.ErrOrStderr()
	if err = kubectl.Run(); err != nil {
		s.Info("%v", err)
	}
	return nil
}

func getKubectlFile(bindir string) string {
	fcname := fmt.Sprintf("kubectl-%s-%s", runtime.GOOS, runtime.GOARCH)
	switch runtime.GOOS {
	case "windows":
		fcname = fcname + ".exe"
	}
	return path.Join(bindir, fcname)
}

func kubectlFileExisted(bindir string) (kf string, err error) {
	kf = getKubectlFile(bindir)
	var lst os.FileInfo
	if lst, err = os.Stat(kf); err == nil && !lst.IsDir() {
		return
	}
	if lst != nil && lst.IsDir() {
		os.RemoveAll(kf)
	}
	err = errors.Errorf("not existed")
	return
}

func unarchiveKubectl(gzfpath string, binbasedir string) error {
	var (
		err error
	)
	if err = UnarchiveBundle(gzfpath, binbasedir); err != nil {
		return err
	}
	return nil
}
