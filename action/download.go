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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"

	"github.com/docker/docker/client"
	helmGetter "helm.sh/helm/v3/pkg/getter"

	"io"
	"os"
	"path"
	"runtime"
	"strings"

	"snz1.cn/snz1dp/snz1dpctl/docker"
	"snz1.cn/snz1dp/snz1dpctl/down"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

func downloadHelmBin(i Action, out io.Writer, force bool) (string, error) {
	basebindir := i.GlobalSetting().GetBinDir()

	var (
		err     error
		fst     os.FileInfo
		spinner *utils.WaitSpinner
		pbm     string
	)

	if err = os.MkdirAll(basebindir, os.ModePerm); err != nil {
		return "", err
	}

	fcname := fmt.Sprintf("%s-%s-%s-%s.%s", BaseConfig.Helm.Name, BaseConfig.Helm.Version, runtime.GOOS, runtime.GOARCH, BaseConfig.Helm.Suffix)
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
		pbm = fmt.Sprintf("download %s-%s...", BaseConfig.Helm.Name, BaseConfig.Helm.Version)
		spinner = utils.NewSpinner(pbm, out)
		helmURL := BaseConfig.Helm.URL
		helmURL = i.GlobalSetting().ResolveDownloadURL(helmURL)
		if helmURL[len(helmURL)-1] != '/' {
			helmURL += "/"
		}
		helmURL += fcname
		i.Info("download '%s'...", helmURL)
		if mkfpath, err = down.NewBundleDownloader(out, helmURL, down.VerifyAlways).Download(basebindir, fcname); err != nil {
			spinner.Close()
			i.Println("failed: %s", err.Error())
			i.Info("download '%s' error: %v", helmURL, err)
			return "", err
		}
		i.Info("download '%s' ok, save to '%s'", helmURL, mkfpath)
		spinner.Close()
		i.Println("ok!")
	}
	return mkfpath, nil
}

func downloadNodejs(i Action, out io.Writer, force bool) (string, error) {
	basebindir := i.GlobalSetting().GetBinDir()

	var (
		err     error
		fst     os.FileInfo
		spinner *utils.WaitSpinner
		pbm     string
	)

	if err = os.MkdirAll(basebindir, os.ModePerm); err != nil {
		return "", err
	}

	fcname := fmt.Sprintf("%s-v%s-%s-%s.%s", BaseConfig.Node.Name, BaseConfig.Node.Version, runtime.GOOS, runtime.GOARCH, BaseConfig.Node.Suffix)
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
		pbm = fmt.Sprintf("download %s-%s...", BaseConfig.Node.Name, BaseConfig.Node.Version)
		spinner = utils.NewSpinner(pbm, out)
		nodejsURL := BaseConfig.Node.URL
		nodejsURL = i.GlobalSetting().ResolveDownloadURL(nodejsURL)
		if nodejsURL[len(nodejsURL)-1] != '/' {
			nodejsURL += "/"
		}
		nodejsURL += fcname
		i.Info("download '%s'...", nodejsURL)
		if mkfpath, err = down.NewBundleDownloader(out, nodejsURL, down.VerifyAlways).Download(basebindir, fcname); err != nil {
			spinner.Close()
			i.Println("failed: %s", err.Error())
			i.Info("download '%s' error: %v", nodejsURL, err)
			return "", err
		}
		i.Info("download '%s' ok, save to '%s'", nodejsURL, mkfpath)
		spinner.Close()
		i.Println("ok!")
	}
	return mkfpath, nil
}

func downloadBuildx(i Action, out io.Writer, force bool) (string, error) {
	bundleDir := i.GlobalSetting().GetBundleDir()
	os.MkdirAll(bundleDir, os.ModePerm)

	var (
		err     error
		fst     os.FileInfo
		spinner *utils.WaitSpinner
		pbm     string
	)

	if err = os.MkdirAll(bundleDir, os.ModePerm); err != nil {
		return "", err
	}

	fcname := fmt.Sprintf("buildx-v%s.%s-%s", BaseConfig.Buildx.Version, runtime.GOOS, runtime.GOARCH)
	switch runtime.GOOS {
	case "windows":
		fcname += ".exe"
	}

	mkfpath := path.Join(bundleDir, fcname)
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
		pbm = "download docker buildx..."
		spinner = utils.NewSpinner(pbm, out)

		buildxURL := BaseConfig.Buildx.Prefix
		buildxURL = i.GlobalSetting().ResolveDownloadURL(buildxURL)
		if buildxURL[len(buildxURL)-1] != '/' {
			buildxURL += "/"
		}
		buildxURL += fcname
		i.Info("download '%s'...", buildxURL)
		if mkfpath, err = down.NewBundleDownloader(out, buildxURL, down.VerifyAlways).Download(bundleDir, fcname); err != nil {
			spinner.Close()
			i.Println("failed: %s", err.Error())
			i.Info("download '%s' error: %v", buildxURL, err)
			return "", err
		}
		i.Info("download '%s' ok, save to '%s'", buildxURL, mkfpath)
		spinner.Close()
		i.Println("ok!")
	}
	return mkfpath, nil
}

func downloadBuildkit(i Action, out io.Writer, force bool) (string, string, error) {
	bundleDir := i.GlobalSetting().GetBundleDir()
	os.MkdirAll(bundleDir, os.ModePerm)

	var (
		err     error
		fst     os.FileInfo
		spinner *utils.WaitSpinner
		pbm     string
	)

	if err = os.MkdirAll(bundleDir, os.ModePerm); err != nil {
		return "", "", err
	}

	fcname := fmt.Sprintf("moby_buildkit_buildx-stable-1-%s.tar", runtime.GOARCH)
	fileSha256 := "shasha256:c09e45315c9046d566aadd1f6046a9a37d6c7c5e1dbb25442c6960851589eefd"
	switch runtime.GOARCH {
	case "amd64":
		fileSha256 = "sha256:896276ced360695d0e75a6d7f5377eca47c06c190b93c64987669e76ca6458cc"
	}

	mkfpath := path.Join(bundleDir, fcname)
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
		pbm = "download moby/buildkit:buildx-stable-1 ..."
		spinner = utils.NewSpinner(pbm, out)

		imageURL := BaseConfig.Buildx.Prefix
		imageURL = i.GlobalSetting().ResolveDownloadURL(imageURL)
		if imageURL[len(imageURL)-1] != '/' {
			imageURL += "/"
		}
		imageURL += fcname
		i.Info("download '%s'...", imageURL)
		if mkfpath, err = down.NewBundleDownloader(out, imageURL, down.VerifyAlways).Download(bundleDir, fcname); err != nil {
			spinner.Close()
			i.Println("failed: %s", err.Error())
			i.Info("download '%s' error: %v", imageURL, err)
			return "", "", err
		}
		i.Info("download '%s' ok, save to '%s'", imageURL, mkfpath)
		spinner.Close()
		i.Println("ok!")
	}
	return mkfpath, fileSha256, nil
}

func downloadKubectl(i Action, out io.Writer, force bool) (string, error) {
	basebindir := i.GlobalSetting().GetBinDir()

	var (
		err     error
		fst     os.FileInfo
		spinner *utils.WaitSpinner
		pbm     string
	)

	if err = os.MkdirAll(basebindir, os.ModePerm); err != nil {
		return "", err
	}

	fcname := fmt.Sprintf("%s-%s-%s.tgz", BaseConfig.Kubectl.Name, runtime.GOOS, runtime.GOARCH)
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
		pbm = fmt.Sprintf("download %s-%s...", BaseConfig.Kubectl.Name, BaseConfig.Kubectl.Version)
		spinner = utils.NewSpinner(pbm, out)
		kubectlURL := BaseConfig.Kubectl.URL + BaseConfig.Kubectl.Version + "/" + fcname
		kubectlURL = i.GlobalSetting().ResolveDownloadURL(kubectlURL)

		i.Info("download '%s'...", kubectlURL)
		if mkfpath, err = down.NewBundleDownloader(out, kubectlURL, down.VerifyAlways).Download(basebindir, fcname); err != nil {
			spinner.Close()
			i.Println("failed: %s", err.Error())
			i.Info("download '%s' error: %v", kubectlURL, err)
			return "", err
		}
		i.Info("download '%s' ok, save to '%s'", kubectlURL, mkfpath)
		spinner.Close()
		i.Println("ok!")
	}
	return mkfpath, nil
}

func downloadSonarScanner(i Action, out io.Writer, force bool) (string, error) {
	basebindir := i.GlobalSetting().GetBinDir()

	var (
		err     error
		fst     os.FileInfo
		spinner *utils.WaitSpinner
		pbm     string
	)

	if err = os.MkdirAll(basebindir, os.ModePerm); err != nil {
		return "", err
	}

	fcname := fmt.Sprintf("%s-%s-%s.%s", BaseConfig.Sonar.Name, BaseConfig.Sonar.Version, runtime.GOOS, BaseConfig.Sonar.Suffix)
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
		sonarURL := BaseConfig.Sonar.URL
		sonarURL = i.GlobalSetting().ResolveDownloadURL(sonarURL)
		if sonarURL[len(sonarURL)-1] != '/' {
			sonarURL += "/"
		}
		sonarURL += fcname
		pbm = fmt.Sprintf("download %s-%s...", BaseConfig.Sonar.Name, BaseConfig.Sonar.Version)
		spinner = utils.NewSpinner(pbm, out)
		i.Info("download '%s'...", sonarURL)
		if mkfpath, err = down.NewBundleDownloader(out, sonarURL, down.VerifyAlways).Download(basebindir, fcname); err != nil {
			spinner.Close()
			i.Println("failed: %s", err.Error())
			i.Info("download '%s' error: %v", sonarURL, err)
			return "", err
		}
		i.Info("download '%s' ok, save to '%s'", sonarURL, mkfpath)
		spinner.Close()
		i.Println("ok!")
	}
	return mkfpath, nil
}

func downloadSnz1dpCtl(i Action, cofile string, out io.Writer) (string, error) {
	basebindir := i.GlobalSetting().GetBinDir()

	var (
		err     error
		spinner *utils.WaitSpinner
		pbm     string
		fcname  string
		mkfpath string
		finame  string
	)

	if err = os.MkdirAll(basebindir, os.ModePerm); err != nil {
		return "", err
	}

	mkfpath = getSnz1dpCtlFile(basebindir)
	finame = getSnz1dpBinFile(basebindir)
	fcname = path.Base(mkfpath)

	pbm = fmt.Sprintf("download %s new version...", BaseConfig.Snz1dp.Ctl.Name)
	spinner = utils.NewSpinner(pbm, out)
	snz1dpctlURL := BaseConfig.Snz1dp.Ctl.URL
	snz1dpctlURL = i.GlobalSetting().ResolveDownloadURL(snz1dpctlURL)
	if snz1dpctlURL[len(snz1dpctlURL)-1] != '/' {
		snz1dpctlURL += "/"
	}
	snz1dpctlURL += fcname

	i.Info("download '%s'...", snz1dpctlURL)
	if mkfpath, err = down.NewBundleDownloader(out, snz1dpctlURL, down.VerifyAlways).Download(basebindir, fcname); err != nil {
		spinner.Close()
		i.Println("failed: %s", err.Error())
		i.Info("download '%s' error: %v", snz1dpctlURL, err)
		return "", err
	}
	i.Info("download '%s' ok, save to '%s'", snz1dpctlURL, mkfpath)
	if cofile != finame {
		if err = utils.CopyFile(mkfpath, finame); err != nil {
			return "", err
		}
		os.Chmod(finame, os.ModePerm)
	}
	spinner.Close()
	i.Println("ok!")
	return mkfpath, nil
}

func (ic *InstallConfiguration) saveDockerImages(compName, version string, flines []string, platform string, pullforce bool, notsaveImage bool, imageTarfile string) (err error) {
	var (
		fimages []string
		simages []string
		pbm     string
		spinner *utils.WaitSpinner
		dc      *client.Client
		setting *GlobalSetting = ic.GlobalSetting()
		sumData []byte
	)

	dc, err = docker.NewClient()
	if err != nil {
		return err
	}
	defer dc.Close()

	for _, fline := range flines {
		fImageNames := strings.Split(fline, " ")
		if len(fImageNames) == 0 {
			continue
		}
		imageName := fImageNames[len(fImageNames)-1]
		if imageName == "" {
			continue
		}
		fimages = append(fimages, imageName)
		if len(fImageNames) > 1 {
			simages = append(simages, fImageNames[0])
		} else {
			simages = append(simages, imageName)
		}
	}

	if len(fimages) == 0 {
		return
	}

	var savedImages []string
	for i, imgName := range fimages {
		savedImages = append(savedImages, imgName)

		if _, err = docker.ImageExisted(dc, imgName); err != nil || pullforce {
			var (
				dockerRepoUser, dockerRepoPwd string = ic.ResolveImageRepoUserAndPwd(imgName)
			)
			pbm = fmt.Sprintf("pull %s...", imgName)
			spinner = utils.NewSpinner(pbm, setting.OutOrStdout())
			err = docker.PullAndRenameImages(dc, imgName, simages[i], dockerRepoUser, dockerRepoPwd, platform)

			spinner.Close()

			if err != nil {
				setting.Println("failed: %s", err.Error())
				return
			}

			setting.Println("ok!")
		}
	}

	if notsaveImage {
		return
	}

	imageFileName := path.Base(imageTarfile)

	pbm = fmt.Sprintf("save %s-%s docker images to %s...", compName, version, imageFileName)
	spinner = utils.NewSpinner(pbm, setting.OutOrStdout())
	err = docker.SaveImageToFile(dc, savedImages, imageTarfile)
	if err != nil {
		spinner.Close()
		setting.Println("failed: %s", err.Error())
		return
	}

	sumData, err = utils.FileChecksum(imageTarfile, sha256.New())
	if err != nil {
		spinner.Close()
		setting.Println("failed: %s", err.Error())
		return
	}
	spinner.Close()
	fcksum := hex.EncodeToString(sumData)
	err = os.WriteFile(imageTarfile+".sha256", []byte(fmt.Sprintf("%s %s", fcksum, imageFileName)), 0644)

	setting.Println("ok!")
	return
}

func (ic *InstallConfiguration) getBundleImages(compNames []string, components map[string]Component) (imageNames []string, err error) {
	var (
		setting  *GlobalSetting = ic.GlobalSetting()
		imgfData []byte
	)

	if len(compNames) == 0 {
		return
	}

	bundleDir := setting.GetBundleDir()

	for _, v := range compNames {
		comp := components[v]
		if comp == nil || !comp.BeInstall() {
			continue
		}

		bundleFilename := fmt.Sprintf("%s-%s.tgz", comp.GetName(), comp.GetVersion())
		bundleFilepath := path.Join(bundleDir, bundleFilename)

		compExtractDir := path.Join(bundleDir, fmt.Sprintf("%s-%s", comp.GetRealName(), comp.GetRealVersion()))
		os.RemoveAll(compExtractDir)

		srcImageFile := fmt.Sprintf("%s-%s/IMAGES", comp.GetRealName(), comp.GetRealVersion())

		if err = ExtractBundleFile(bundleFilepath, srcImageFile, bundleDir); err != nil && strings.Contains(err.Error(), "file already exists") {
			return
		}

		imgfData, err = os.ReadFile(path.Join(compExtractDir, "IMAGES"))
		os.RemoveAll(compExtractDir)

		if err != nil {
			return
		}

		flines := strings.Split(strings.ReplaceAll(string(imgfData), "\r\n", "\n"), "\n")
		for i, fline := range flines {
			if fline == "" {
				continue
			}
			if BaseConfig.Snz1dp.Docker.URL != ic.Snz1dp.Registry.URL && strings.HasPrefix(fline, BaseConfig.Snz1dp.Docker.URL) {
				flines[i] = ic.Snz1dp.Registry.URL + fline[len(BaseConfig.Snz1dp.Docker.URL):] + " " + fline
			}
			imageNames = append(imageNames, flines[i])
		}
	}
	return
}

func (ic *InstallConfiguration) downloadBundles(
	downDisabled bool, compNames []string, components map[string]Component,
	force bool, downloadImage bool, notsaveImage bool, platform string) error {
	var (
		err          error
		fst          os.FileInfo
		spinner      *utils.WaitSpinner
		imageTarfile string
		setting      *GlobalSetting = ic.GlobalSetting()
		pbm          string
	)

	bundleDir := setting.GetBundleDir()
	os.MkdirAll(bundleDir, os.ModePerm)

	for _, v := range compNames {
		comp := components[v]
		if !downDisabled && !comp.BeInstall() {
			continue
		}

		bundleFilename := fmt.Sprintf("%s-%s.tgz", comp.GetName(), comp.GetVersion())
		bundleFilepath := path.Join(bundleDir, bundleFilename)

		imageTarfile = fmt.Sprintf("%s-%s-IMAGES.tar", comp.GetName(), comp.GetVersion())
		imageTarfile = path.Join(bundleDir, imageTarfile)

		lcFile := path.Join(bundleDir, bundleFilename)
		dodown := false

		if force {
			dodown = true
		} else {
			if fst, err = os.Stat(lcFile); err == nil && !fst.IsDir() {
				sumData, err := utils.FileChecksum(lcFile, sha256.New())
				fcksum := hex.EncodeToString(sumData)
				dodown = err != nil || down.VerifyBundle(lcFile+".sha256", fcksum) != nil
			} else {
				if fst != nil {
					os.RemoveAll(lcFile)
				}
				dodown = true
			}
		}

		var (
			bundleURL     string = comp.GetBundleURL()
			localFilepath string = comp.GetLocalFilePath()
			idxLastPath   int    = strings.LastIndex(bundleURL, "/")
			charRepo      string
			helmRegistry  *HelmRegistry
			chartDigest   string
			chartName     string = comp.GetName()
			chartVersion  string = comp.GetVersion()
		)

		if runtime.GOOS == "windows" {
			bundleURL = strings.ReplaceAll(comp.GetBundleURL(), "\\", "/")
			idxLastPath = strings.LastIndex(comp.GetBundleURL(), "/")
		}

		if localFilepath == "" {

			// 先分析一下是不是Chart仓库
			if idxLastPath > 0 {
				charRepo = bundleURL[0:idxLastPath]
				helmRegistry = ic.GetHelmRegistryByURL(charRepo)
			}

			if helmRegistry == nil {
				chartName = bundleURL[idxLastPath+1:]
				idxLastPath = strings.LastIndex(chartName, ".tgz")
				if idxLastPath > 0 {
					chartName = chartName[0:idxLastPath]
					idxLastPath = strings.LastIndex(chartName, "-")
					if idxLastPath > 0 {
						tsx := strings.LastIndex(chartName[0:idxLastPath], "-")
						if tsx > 0 && regexp.MustCompile(`\d+.\d+.\d+`).MatchString(chartName[tsx:idxLastPath]) {
							chartVersion = chartName[tsx+1:]
							chartName = chartName[0:tsx]
						} else {
							chartVersion = chartName[idxLastPath+1:]
							chartName = chartName[:idxLastPath]
						}
					} else {
						chartVersion = comp.GetVersion()
					}
				}

				comp.SetRealName(chartName)
				comp.SetRealVersion(chartVersion)

				if dodown {
					pbm = fmt.Sprintf("download %s-%s...", chartName, comp.GetVersion())
					spinner = utils.NewSpinner(pbm, setting.OutOrStdout())
					_, err = down.NewBundleDownloader(
						setting.OutOrStdout(), bundleURL, down.VerifyAlways).Download(bundleDir, bundleFilename)
					spinner.Close()
					if err != nil {
						setting.Println("failed: %s", err.Error())
						return err
					}
				}
			} else {
				InitHelmActionConfig(ic.setting)
				chartName = bundleURL[idxLastPath+1:]
				idxLastPath = strings.LastIndex(chartName, ".tgz")
				if idxLastPath > 0 {
					chartName = chartName[0:idxLastPath]
					idxLastPath = strings.LastIndex(chartName, "-")
					if idxLastPath > 0 {
						tsx := strings.LastIndex(chartName[0:idxLastPath], "-")
						if tsx > 0 && regexp.MustCompile(`\d+.\d+.\d+`).MatchString(chartName[tsx:idxLastPath]) {
							chartVersion = chartName[tsx+1:]
							chartName = chartName[0:tsx]
						} else {
							chartVersion = chartName[idxLastPath+1:]
							chartName = chartName[:idxLastPath]
						}
					} else {
						chartVersion = comp.GetVersion()
					}
				} else {
					chartName = comp.GetName()
					chartVersion = comp.GetVersion()
				}
				comp.SetRealName(chartName)
				comp.SetRealVersion(chartVersion)
				if dodown {
					pbm = fmt.Sprintf("download %s-%s...", chartName, chartVersion)
					spinner = utils.NewSpinner(pbm, setting.OutOrStdout())
					bundleURL, chartDigest, err = helmRegistry.ResolveChartURLAndDigest(chartName, chartVersion, helmGetter.All(ic.setting.helmSetting))
					if err != nil {
						spinner.Close()
						setting.Println("failed: %s", err.Error())
						return err
					}

					_, err = down.NewBundleDownloader(
						setting.OutOrStdout(), bundleURL, down.VerifyAlways,
						down.WithBasicAuth(helmRegistry.Username, helmRegistry.Password),
					).SetFileDigest(chartDigest).Download(bundleDir, bundleFilename)
					spinner.Close()
					if err != nil {
						setting.Println("failed: %s", err.Error())
						return err
					}
				}
			}
			if dodown {
				setting.Println("ok!")
			}
		} else if dodown {
			comp.SetRealName(chartName)
			comp.SetRealVersion(chartVersion)

			pbm = fmt.Sprintf("copy %s-%s...", comp.GetName(), comp.GetVersion())
			spinner = utils.NewSpinner(pbm, setting.OutOrStdout())
			err = utils.CopyFile(localFilepath, bundleFilepath)
			spinner.Close()
			if err != nil {
				setting.Println("failed: %s", err.Error())
				return err
			}
			sumData, err := utils.FileChecksum(bundleFilepath, sha256.New())
			if err != nil {
				setting.Println("failed: %s", err.Error())
				return err
			}
			fcksum := hex.EncodeToString(sumData)
			err = os.WriteFile(bundleFilepath+".sha256", []byte(fmt.Sprintf("%s %s", fcksum, bundleFilename)), 0644)
			if err != nil {
				setting.Println("failed: %s", err.Error())
				return err
			}
			setting.Println("ok!")
		} else {
			comp.SetRealName(chartName)
			comp.SetRealVersion(chartVersion)
		}

		if !downloadImage {
			continue
		}

		compExtractDir := path.Join(bundleDir, fmt.Sprintf("%s-%s", comp.GetRealName(), comp.GetRealVersion()))
		os.RemoveAll(compExtractDir)

		srcImageFile := fmt.Sprintf("%s-%s/IMAGES", comp.GetRealName(), comp.GetRealVersion())

		if err = ExtractBundleFile(bundleFilepath, srcImageFile, bundleDir); err != nil && strings.Contains(err.Error(), "file already exists") {
			continue
		}

		imgfData, err := os.ReadFile(path.Join(compExtractDir, "IMAGES"))
		os.RemoveAll(compExtractDir)

		if err != nil {
			continue
		}

		flines := strings.Split(strings.ReplaceAll(string(imgfData), "\r\n", "\n"), "\n")

		if platform == "" {
			platform = ic.Platform
		}

		for i, fline := range flines {
			if BaseConfig.Snz1dp.Docker.URL != ic.Snz1dp.Registry.URL && strings.HasPrefix(fline, BaseConfig.Snz1dp.Docker.URL) {
				flines[i] = ic.Snz1dp.Registry.URL + fline[len(BaseConfig.Snz1dp.Docker.URL):] + " " + fline
			}
		}

		err = ic.saveDockerImages(comp.GetName(), comp.GetVersion(), flines, platform, true, notsaveImage, imageTarfile)
		if err != nil {
			return err
		}

	}
	return nil
}
