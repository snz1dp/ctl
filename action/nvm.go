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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"snz1.cn/snz1dp/snz1dpctl/down"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

func IsNvmExisted(bindir string) bool {
	var kf string
	switch runtime.GOOS {
	case "windows":
		kf = path.Join(bindir, "nvm", "nvm.exe")
	default:
		kf, _ = utils.ExpandUserDir("~/.nvm/nvm.sh")
	}
	if _, err := os.Stat(kf); err == nil {
		return true
	}
	return false
}

func InstallNvm(i Action, out io.Writer, force bool) (err error) {
	winInstall := false
	switch runtime.GOOS {
	case "windows":
		winInstall = true
	default:
	}

	if winInstall {
		err = downloadNvmWindows(i, force)
	} else {
		err = downloadNvmShInstall(i, force)
	}

	return
}

func downloadNvmShInstall(i Action, force bool) error {
	basebindir := i.GlobalSetting().GetBinDir()

	var (
		err     error
		fst     os.FileInfo
		spinner *utils.WaitSpinner
		pbm     string
		ic      *InstallConfiguration
	)

	if err = os.MkdirAll(basebindir, os.ModePerm); err != nil {
		return err
	}

	if _, ic, err = i.GlobalSetting().LoadLocalInstallConfiguration(); err != nil {
		return err
	}

	fcname := fmt.Sprintf("%s-%s-install.sh", BaseConfig.Nvm.Name, BaseConfig.Nvm.Version)
	mkfpath := path.Join(basebindir, fcname)
	dodown := false
	if force {
		dodown = true
	} else {
		if fst, err = os.Stat(mkfpath); err != nil || fst.IsDir() {
			if fst != nil {
				os.RemoveAll(mkfpath)
			}
			dodown = true
		}
	}
	if dodown {
		pbm = fmt.Sprintf("download %s-%s...", BaseConfig.Nvm.Name, BaseConfig.Nvm.Version)
		spinner = utils.NewSpinner(pbm, i.GlobalSetting().OutOrStdout())
		nvmURL := fmt.Sprintf("%s/dp/tool/%s-sh/-/raw/v%s/install.sh?inline=false", ic.Snz1dp.Server.GitURL, BaseConfig.Nvm.Name, BaseConfig.Nvm.Version)
		nvmURL = i.GlobalSetting().ResolveDownloadURL(nvmURL)
		i.Info("download '%s'...", nvmURL)
		if mkfpath, err = down.NewBundleDownloader(i.GlobalSetting().OutOrStdout(), nvmURL, down.VerifyNever).Download(basebindir, fcname); err != nil {
			spinner.Close()
			i.Println("failed: %s", err.Error())
			i.Info("download '%s' error: %v", nvmURL, err)
			return err
		}
		i.Info("download '%s' ok, save to '%s'", nvmURL, mkfpath)
		spinner.Close()
		i.Println("ok!")
	}

	os.Chmod(mkfpath, os.ModePerm)

	gitURL := fmt.Sprintf("%sdp/tool/%s-sh.git", ic.Snz1dp.Server.GitURL, BaseConfig.Nvm.Name)
	i.Info("install %s from %s...", BaseConfig.Nvm.Name, gitURL)

	installCmd := exec.Command("bash", mkfpath)
	installCmd.Env = append(os.Environ(), fmt.Sprintf("NVM_SOURCE=%s", gitURL))
	installCmd.Stdout = i.GlobalSetting().OutOrStdout()
	installCmd.Stderr = i.GlobalSetting().ErrOrStderr()
	if err = installCmd.Run(); err != nil {
		return err
	}

	return nil
}

// - 下载nvm
func downloadNvmWindows(i Action, force bool) error {
	basebindir := i.GlobalSetting().GetBinDir()

	var (
		err     error
		fst     os.FileInfo
		spinner *utils.WaitSpinner
		pbm     string
	)

	if err = os.MkdirAll(basebindir, os.ModePerm); err != nil {
		return err
	}

	fcname := fmt.Sprintf("%s-%s-%s-%s.%s", BaseConfig.Nvm.Name, BaseConfig.Nvm.Windows.Version, runtime.GOOS, runtime.GOARCH, BaseConfig.Nvm.Windows.Suffix)
	mkfpath := path.Join(basebindir, fcname)
	dodown := false

	if force {
		dodown = true
	} else {
		if fst, err = os.Stat(mkfpath); err == nil && !fst.IsDir() {
			sumData, err := utils.FileChecksum(mkfpath, sha256.New())
			fcksum := hex.EncodeToString(sumData)
			i.Info("file '%s' checksum: %s", mkfpath, fcksum)
			dodown = err != nil || down.VerifyBundle(mkfpath+".sha256", fcksum) != nil
			i.Info("file '%s' checksum is %s", mkfpath, func(b bool) string {
				if b {
					return "validate"
				}
				return "invalidate"
			}(!dodown))
		} else {
			if fst != nil {
				os.RemoveAll(mkfpath)
			}
			dodown = true
		}
	}

	if dodown {
		pbm = fmt.Sprintf("download %s-%s...", BaseConfig.Nvm.Name, BaseConfig.Nvm.Windows.Version)
		spinner = utils.NewSpinner(pbm, i.GlobalSetting().OutOrStdout())
		nvmURL := BaseConfig.Snz1dp.Download.Default
		nvmURL = i.GlobalSetting().ResolveDownloadURL(nvmURL)
		if nvmURL[len(nvmURL)-1] != '/' {
			nvmURL += "/"
		}
		nvmURL += fcname
		i.Info("download '%s'...", nvmURL)
		if mkfpath, err = down.NewBundleDownloader(i.GlobalSetting().OutOrStdout(), nvmURL, down.VerifyAlways).Download(basebindir, fcname); err != nil {
			spinner.Close()
			i.Println("failed: %s", err.Error())
			i.Info("download '%s' error: %v", nvmURL, err)
			return err
		}
		i.Info("download '%s' ok, save to '%s'", nvmURL, mkfpath)
		spinner.Close()
		i.Println("ok!")
	}

	nvmPath := path.Join(basebindir, "nvm")
	nodejsPath := path.Join(basebindir, "nodejs")
	os.RemoveAll(nvmPath)

	if err = unarchiveNvm(mkfpath, basebindir); err != nil {
		return err
	}

	settings := fmt.Sprintf(
		"root: %s\r\npath: %s\r\narch: 64\r\nproxy: %s\r\n",
		nvmPath, nodejsPath, "none",
	)

	err = os.WriteFile(path.Join(nvmPath, "settings.txt"), []byte(settings), os.ModePerm)

	return err
}

func unarchiveNvm(gzfpath string, binbasedir string) error {
	var (
		err error
	)
	if err = UnarchiveBundle(gzfpath, binbasedir); err != nil {
		return err
	}
	return nil
}

func ResolveNvmPath(i Action) (nvmPath string, err error) {
	basebindir := i.GlobalSetting().GetBinDir()
	nvmExisted := false
	nvmExisted = IsNvmExisted(basebindir)
	if runtime.GOOS == "windows" {
		nvmPath = path.Join(basebindir, "nvm", "nvm.exe")
	} else {
		nvmPath = "bash"
	}
	if !nvmExisted {
		if err = InstallNvm(i, i.GlobalSetting().OutOrStdout(), false); err != nil {
			return
		}
	}
	return
}

// StartNvm -
type StartNvm struct {
	BaseAction
	Args []string
}

// StartNvm -
func NewStartNvm(setting *GlobalSetting) *StartNvm {
	return &StartNvm{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

var _nvmNodeVersion string

func (s *StartNvm) GetNodeVersion() (version string, err error) {

	if _nvmNodeVersion != "" {
		return _nvmNodeVersion, nil
	}

	setting := s.GlobalSetting()
	var (
		mkfpath string
	)

	setting.InitLogger(BaseConfig.Nvm.Name)

	mkfpath, err = ResolveNvmPath(s)
	if err != nil {
		return
	}

	nvmArgs := []string{"current"}
	if mkfpath == "bash" {
		nvmArgs = []string{
			"-c", "source ~/.bash_profile; nvm " + strings.Join(nvmArgs, " "),
		}
	} else {
		binPath := setting.GetBinDir()
		mkfpath = path.Join(binPath, "nodejs", "node.exe")
		nvmArgs = []string{"-v"}
	}

	nvmcmd := exec.CommandContext(context.Background(), mkfpath, nvmArgs...)
	var fout bytes.Buffer
	nvmcmd.Stdin = setting.InOrStdin()
	nvmcmd.Stdout = &fout
	// nvmcmd.Stderr = setting.ErrOrStderr()

	err = nvmcmd.Run()

	if err != nil {
		if mkfpath != "bash" {
			err = fmt.Errorf("nodejs not installed, please run 'nvm install <version>' to install nodejs")
			return
		}
		nvmArgs = []string{
			"-c", "source ~/.zshrc; nvm " + strings.Join(nvmArgs, " "),
		}
		nvmcmd := exec.CommandContext(context.Background(), mkfpath, nvmArgs...)
		fout.Reset()
		nvmcmd.Stdin = setting.InOrStdin()
		nvmcmd.Stdout = &fout
		// nvmcmd.Stderr = setting.ErrOrStderr()
		if err = nvmcmd.Run(); err != nil {
			err = fmt.Errorf("nodejs not installed, please run 'nvm install <version>' to install nodejs")
			return
		}
	}

	version = strings.TrimSpace(fout.String())
	s.Info("nvm current: %s", version)
	if strings.Contains(version, "system") || strings.Contains(version, "No current version") {
		version = BaseConfig.Node.Version
	} else if strings.HasPrefix(version, "v") {
		version = version[1:]
	} else {
		version = BaseConfig.Node.Version
	}
	_nvmNodeVersion = version
	return
}

// Run -
func (s *StartNvm) Run() error {
	setting := s.GlobalSetting()
	var (
		mkfpath string
		err     error
	)

	setting.InitLogger(BaseConfig.Nvm.Name)

	mkfpath, err = ResolveNvmPath(s)
	if err != nil {
		s.ErrorExit("%s", err.Error())
	}

	nvmArgs := s.Args
	if mkfpath == "bash" {
		nvmArgs = []string{
			"-c", "source ~/.bash_profile; nvm " + strings.Join(nvmArgs, " "),
		}
	}

	nvmcmd := exec.CommandContext(context.Background(), mkfpath, nvmArgs...)
	switch runtime.GOOS {
	case "windows":
		binPath := setting.GetBinDir()
		nvmPath := path.Join(binPath, "nvm")
		nodejsPath := path.Join(binPath, "nodejs")
		utils.FixBinSearchPath(nvmPath)
		utils.FixBinSearchPath(nodejsPath)
		nvmcmd.Env = append(os.Environ(),
			fmt.Sprintf("NVM_HOME=%s", nvmPath),
			fmt.Sprintf("NVM_SYMLINK=%s", nodejsPath),
		)
	default:
		nvmcmd.Env = os.Environ()
	}

	nvmcmd.Stdin = setting.InOrStdin()
	nvmcmd.Stdout = setting.OutOrStdout()
	nvmcmd.Stderr = setting.ErrOrStderr()
	if err = nvmcmd.Run(); err != nil {
		s.ErrorExit("%v", err)
	}
	return nil
}
