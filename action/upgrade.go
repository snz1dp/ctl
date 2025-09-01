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
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"time"

	"github.com/go-ping/ping"
	"snz1.cn/snz1dp/snz1dpctl/down"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

var (
	remoteCommitID string
)

// UpgradeCtl -
type UpgradeCtl struct {
	BaseAction
	Args []string
}

// NewUpgradeCtl -
func NewUpgradeCtl(setting *GlobalSetting) *UpgradeCtl {
	return &UpgradeCtl{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

func resolveRemoteVersion(settings *GlobalSetting) (commit string, err error) {
	var (
		u *url.URL
		g down.Getter
		c int64
	)

	ctlURL := BaseConfig.Snz1dp.Ctl.URL
	if ctlURL[len(ctlURL)-1] != '/' {
		ctlURL += "/"
	}
	ctlURL += "COMMITID"
	ctlURL = settings.ResolveDownloadURL(ctlURL)
	if u, err = url.Parse(ctlURL); err != nil {
		return "", err
	}

	if g, err = down.AllProviders().ByScheme(u.Scheme); err != nil {
		return "", err
	}

	body := bytes.NewBuffer(nil)
	if c, err = g.Get(u.String(), body, nil, nil, down.WithRequestTimeout(time.Second*3)); err != nil || c > 100 {
		return "", err
	}
	return body.String(), nil
}

func getSnz1dpCtlFile(binbasedir string) string {
	fcname := fmt.Sprintf("%s-%s-%s", BaseConfig.Snz1dp.Ctl.Name, runtime.GOOS, runtime.GOARCH)
	switch runtime.GOOS {
	case "windows":
		fcname = fcname + ".exe"
	}
	return path.Join(binbasedir, fcname)
}

func getSnz1dpBinFile(binbasedir string) string {
	fcname := BaseConfig.Snz1dp.Ctl.Name
	switch runtime.GOOS {
	case "windows":
		fcname = fcname + ".exe"
	}
	return path.Join(binbasedir, fcname)
}

// Run -
func (u *UpgradeCtl) Run() (err error) {
	setting := u.GlobalSetting()
	setting.InitLogger("upgrade")

	var (
		ctlfile  string
		currfile string
		cofile   string
		upg      *exec.Cmd
		isold    bool
	)

	// 安装配置文件
	if _, _, err = setting.LoadLocalInstallConfiguration(); err != nil {
		u.ErrorExit("load config error: %s", err)
		return
	}

	remoteCommitID, err = resolveRemoteVersion(u.GlobalSetting())
	if err != nil {
		u.ErrorExit("%v", err)
		return
	}

	if remoteCommitID == utils.GitCommitID() {
		u.ErrorExit("current %s version is latest!", BaseConfig.Snz1dp.Ctl.Name)
		return
	}

	currfile = utils.GetExecutableFile()

	isold = len(u.Args) > 1 && u.Args[0] == "old"

	if !isold {
		cofile = currfile
		currfile += ".old"
		switch runtime.GOOS {
		case "windows":
			currfile += ".exe"
		}
		err = utils.CopyFile(cofile, currfile)
		if err != nil {
			u.ErrorExit("%v", err)
			return
		}
		u.Println("%s upgrading...", BaseConfig.Snz1dp.Ctl.Name)
		upg = exec.Command(currfile, "upgrade", "old", cofile)
		upg.Stdin = setting.InOrStdin()
		upg.Stdout = setting.OutOrStdout()
		upg.Stderr = setting.ErrOrStderr()
		upg.Start()
		return
	}

	cofile = u.Args[1]

	err = os.Remove(cofile)
	if err != nil {
		time.Sleep(time.Second)
	}

	ctlfile, err = downloadSnz1dpCtl(u, cofile, setting.OutOrStdout())
	if err != nil {
		if cofile != ctlfile {
			utils.CopyFile(currfile, cofile)
		}
		return
	}

	os.Chmod(ctlfile, os.ModePerm)

	if cofile != ctlfile {
		err = utils.CopyFile(ctlfile, cofile)
		if err != nil {
			u.ErrorExit("%v", err)
			utils.CopyFile(currfile, cofile)
			return
		}
	}

	return
}

func checkNewVersion(settings *GlobalSetting) {
	var (
		err    error
		pinger *ping.Pinger
		pstat  *ping.Statistics
	)

	if pinger, err = ping.NewPinger(BaseConfig.Snz1dp.Ctl.Pinger); err != nil {
		return
	}
	pinger.Timeout = time.Second * 3 // 3秒超时
	pinger.Count = 2
	if err = pinger.Run(); err != nil {
		return
	}

	if pstat = pinger.Statistics(); pstat.PacketsRecv < 2 {
		return
	}

	remoteCommitID, err = resolveRemoteVersion(settings)
	if err != nil {
		return
	}

	if remoteCommitID != utils.GitCommitID() {
		fmt.Println(BaseConfig.NewHint)
	}

}
