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
	"os"
	"path"
	"strings"

	helmGetter "helm.sh/helm/v3/pkg/getter"
	helmRepo "helm.sh/helm/v3/pkg/repo"

	"github.com/docker/docker/client"
	"snz1.cn/snz1dp/snz1dpctl/docker"
	"snz1.cn/snz1dp/snz1dpctl/down"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

// DownloadBundle -
type DownloadBundle struct {
	BaseAction
	Force        bool
	PullImage    bool
	All          bool
	Platform     string
	Bundle       []string
	NotSaveImage bool
}

// TargzBundle -
type TargzBundle struct {
	BaseAction
	Destfile  string
	Force     bool
	PullImage bool
	All       bool
	Platform  string
}

// CleanBundle -
type CleanBundle struct {
	BaseAction
	Really  bool
	Config  bool
	RunData bool
	Bundle  []string
}

type SearchBundle struct {
	BaseAction
	SearchKey string
}

type BundleList struct {
	BaseAction
}

type ShowBundleImages struct {
	BaseAction
	BundleNames []string
}

// NewDownloadBundle -
func NewDownloadBundle(setting *GlobalSetting) *DownloadBundle {
	return &DownloadBundle{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// NewTargzBundle -
func NewTargzBundle(setting *GlobalSetting) *TargzBundle {
	return &TargzBundle{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// NewCleanBundle -
func NewCleanBundle(setting *GlobalSetting) *CleanBundle {
	return &CleanBundle{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// NewSearchBundle -
func NewSearchBundle(setting *GlobalSetting) *SearchBundle {
	return &SearchBundle{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// NewBundleList -
func NewBundleList(setting *GlobalSetting) *BundleList {
	return &BundleList{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// NewShowBundleImages -
func NewShowBundleImages(setting *GlobalSetting) *ShowBundleImages {
	return &ShowBundleImages{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (d *DownloadBundle) Run() error {
	var (
		icfile             string
		ic                 *InstallConfiguration
		err                error
		imgtarfname        string
		compNames, oNames  []string
		components, oComps map[string]Component
	)

	setting := d.GlobalSetting()
	setting.InitLogger("download")

	bundleDir := setting.GetBundleDir()

	// 下载组件
	if _, err := os.Stat(bundleDir); err != nil {
		os.MkdirAll(bundleDir, os.ModePerm)
	}

	icfile, ic, err = setting.LoadLocalInstallConfiguration()
	if err != nil {
		d.ErrorExit("load %s error: %s", icfile, err)
		return err
	}

	// 核心组件
	if compNames, components, err = ic.GetBundleComponents(false); err != nil {
		d.ErrorExit("load bundle error: %s", err)
	}

	if len(d.Bundle) > 0 {
		oComps = make(map[string]Component)
		for _, v := range d.Bundle {
			if v == "keepalived" || v == "haproxy" {
				continue
			}
			if components[v] == nil {
				continue
			}
			oNames = append(oNames, v)
			oComps[v] = components[v]
		}
		compNames, components = oNames, oComps
	}

	if err := ic.downloadBundles(false, compNames, components, d.Force, d.PullImage, d.NotSaveImage, d.Platform); err != nil {
		d.ErrorExit("%v", err)
		return err
	}

	if len(d.Bundle) == 0 && d.All {
		// 下载Kubectl
		if _, err := downloadKubectl(d, setting.OutOrStdout(), d.Force); err != nil {
			d.ErrorExit("%v", err)
			return err
		}

		if d.PullImage {
			imagePlatform := d.Platform
			if imagePlatform == "" {
				imagePlatform = ic.Platform
			}

			// 下载Keepalived
			keepalivedImage := fmt.Sprintf("%s/%s:%s", BaseConfig.Snz1dp.Docker.URL, BaseConfig.Keepalived.ImageName, BaseConfig.Keepalived.ImageTag)
			imgtarfname = path.Join(bundleDir, fmt.Sprintf("%s-%s-IMAGES.tar", "keepalived", BaseConfig.Keepalived.Version))
			if err = ic.saveDockerImages("keepalived", BaseConfig.Keepalived.Version, []string{
				keepalivedImage,
			}, imagePlatform, true, d.NotSaveImage, imgtarfname); err != nil {
				d.ErrorExit("%v", err)
				return err
			}

			// 下载Haproxy
			haproxyImage := fmt.Sprintf("%s/%s:%s", BaseConfig.Snz1dp.Docker.URL, BaseConfig.Haproxy.ImageName, BaseConfig.Haproxy.ImageTag)
			imgtarfname = path.Join(bundleDir, fmt.Sprintf("%s-%s-IMAGES.tar", "haproxy", BaseConfig.Haproxy.Version))
			if err = ic.saveDockerImages("haproxy", BaseConfig.Haproxy.Version, []string{
				haproxyImage,
			}, imagePlatform, true, d.NotSaveImage, imgtarfname); err != nil {
				d.ErrorExit("%v", err)
				return err
			}
		}
	}

	return nil
}

// Run -
func (t *TargzBundle) Run() (err error) {
	setting := t.GlobalSetting()
	setting.InitLogger("tar")

	var (
		spinner *utils.WaitSpinner
		spmsg   string
	)

	if t.Destfile == "" {
		cdir, err := os.Getwd()
		if err != nil {
			t.ErrorExit("%v", err)
			return err
		}
		t.Destfile = path.Join(cdir, fmt.Sprintf("%s-snz1dp-%s.tgz", formatDate(setting.StartTime()), utils.Version()))
	}

	d := NewDownloadBundle(t.GlobalSetting())
	d.Force = t.Force
	d.All = t.All
	d.PullImage = t.PullImage
	d.Platform = t.Platform

	if err = d.Run(); err != nil {
		return
	}

	spmsg = fmt.Sprintf("tar %s to %s...", setting.GetBaseDir(), t.Destfile)
	spinner = utils.NewSpinner(spmsg, setting.OutOrStdout())

	if err = ArchiveBundle([]string{setting.GetBaseDir()}, t.Destfile); err != nil {
		spinner.Close()
		t.Error("archive file error: %+v", err)
		t.ErrorExit("failed: %s", err.Error())
		return
	}

	spinner.Close()
	t.Println("ok!")
	return
}

// Run -
func (c *CleanBundle) Run() (err error) {
	setting := c.GlobalSetting()
	if !c.Really && !utils.Confirm("delete snz1dp-"+utils.Version()+" local bundles, proceed? (y/N)", setting.InOrStdin(), setting.OutOrStdout()) {
		c.Println("Cancelled.")
		return
	}

	if len(c.Bundle) == 0 {
		var spinner *utils.WaitSpinner = utils.NewSpinner("cleanup local bundles...", setting.OutOrStdout())

		os.RemoveAll(setting.GetLogDir())
		os.RemoveAll(setting.GetBundleDir())

		if c.Config {
			var (
				icdata []byte
				icfile string = setting.GetConfigFilePath()
			)
			if icdata, err = os.ReadFile(icfile); err == nil {
				os.RemoveAll(setting.GetConfigDir())
				os.MkdirAll(setting.GetConfigDir(), os.ModePerm)
				os.WriteFile(icfile, icdata, 0644)
			}
		}

		if c.RunData {
			os.RemoveAll(path.Join(setting.GetBaseDir(), "run"))
		}

		spinner.Close()
		c.Println("ok!")
	} else {

		var (
			icfile     string
			ic         *InstallConfiguration
			components map[string]Component
		)

		icfile, ic, err = setting.LoadLocalInstallConfiguration()
		if err != nil {
			c.ErrorExit("load %s error: %s", icfile, err)
			return
		}

		// 核心组件
		if _, components, err = ic.GetBundleComponents(false); err != nil {
			c.ErrorExit("load bundle error: %s", err)
			return
		}

		for _, v := range c.Bundle {
			if components[v] == nil {
				continue
			}
			comp := components[v]
			os.RemoveAll(path.Join(setting.GetBundleDir(), fmt.Sprintf("%s-%s-IMAGES.tar", comp.GetName(), comp.GetVersion())))
			os.RemoveAll(path.Join(setting.GetBundleDir(), fmt.Sprintf("%s-%s.tgz", comp.GetName(), comp.GetVersion())))
			os.RemoveAll(path.Join(setting.GetBundleDir(), fmt.Sprintf("%s-%s.tgz.sha256", comp.GetName(), comp.GetVersion())))
			os.RemoveAll(path.Join(setting.GetConfigDir(), fmt.Sprintf("%s-kubernetes.yaml", comp.GetName())))
			os.RemoveAll(path.Join(setting.GetConfigDir(), fmt.Sprintf("%s-standalone.yaml", comp.GetName())))
			os.RemoveAll(path.Join(setting.GetBaseDir(), "run", comp.GetName(), comp.GetVersion()))
		}
	}
	return
}

// LoadBundleImage -
type LoadBundleImage struct {
	BaseAction
	Force  bool
	Bundle []string
	All    bool
}

// NewLoadBundleImage -
func NewLoadBundleImage(setting *GlobalSetting) *LoadBundleImage {
	return &LoadBundleImage{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (d *LoadBundleImage) Run() (err error) {
	setting := d.GlobalSetting()
	setting.InitLogger("load")
	bundleDir := setting.GetBundleDir()

	var (
		ic                *InstallConfiguration
		dc                *client.Client
		spinner           *utils.WaitSpinner
		imgfData          []byte
		compNames, oNames []string
		comps, oComps     map[string]Component
		configFile        string
		sumData           []byte
	)

	if configFile, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		d.ErrorExit("load %s error: %s", configFile, err)
		return
	}

	// 核心组件
	if compNames, comps, err = ic.GetBundleComponents(false); err != nil {
		d.ErrorExit("load bundle error: %s", err)
		return
	}

	if len(d.Bundle) > 0 {
		oComps = make(map[string]Component)
		for _, v := range d.Bundle {
			if comps[v] == nil {
				continue
			}
			oNames = append(oNames, v)
			oComps[v] = comps[v]
		}
		compNames, comps = oNames, oComps
	}

	dc, err = docker.NewClient()
	if err != nil {
		return err
	}

	defer dc.Close()

	for _, v := range compNames {
		s := v
		comp := comps[v]
		if !comp.BeInstall() {
			continue
		}

		bundleFilepath := path.Join(bundleDir, fmt.Sprintf("%s-%s.tgz", s, comp.GetVersion()))
		sumData, err = utils.FileChecksum(bundleFilepath, sha256.New())
		if err != nil {
			d.ErrorExit("read %s checksum error: %v", bundleFilepath, err)
			return
		}
		if err = down.VerifyBundle(bundleFilepath+".sha256", hex.EncodeToString(sumData)); err != nil {
			d.ErrorExit("checksum %s error: %v", bundleFilepath, err)
			return
		}

		srcImageFile := fmt.Sprintf("%s-%s/IMAGES", s, comp.GetVersion())
		if err = ExtractBundleFile(bundleFilepath, srcImageFile, bundleDir); err != nil && !strings.Contains(err.Error(), "file already exists") {
			d.ErrorExit("extract %s error: %v", bundleFilepath, err)
		}

		compExtractDir := path.Join(bundleDir, fmt.Sprintf("%s-%s", s, comp.GetVersion()))
		imgfData, err = os.ReadFile(path.Join(compExtractDir, "IMAGES"))
		os.RemoveAll(compExtractDir)

		if err != nil {
			continue
		}

		var loadImage = d.Force

		if !loadImage {
			flines := strings.Split(strings.ReplaceAll(string(imgfData), "\r\n", "\n"), "\n")
			for _, fline := range flines {
				fImageNames := strings.Split(fline, " ")
				if len(fImageNames) == 0 {
					continue
				}
				imageName := fImageNames[len(fImageNames)-1]
				if imageName == "" {
					continue
				}
				_, err = docker.ImageExisted(dc, imageName)
				if err != nil {
					loadImage = true
					break
				}
			}
		}

		if loadImage {
			imageTarFile := path.Join(bundleDir, fmt.Sprintf("%s-%s-IMAGES.tar", s, comp.GetVersion()))
			spinner = utils.NewSpinner(fmt.Sprintf("load docker images from %s...", imageTarFile), setting.OutOrStdout())
			sumData, err = utils.FileChecksum(imageTarFile, sha256.New())
			if err != nil {
				spinner.Close()
				d.ErrorExit("read %s checksum error: %v", imageTarFile, err)
			}
			if err = down.VerifyBundle(imageTarFile+".sha256", hex.EncodeToString(sumData)); err != nil {
				spinner.Close()
				d.ErrorExit("file %s checksum error: %v", bundleFilepath, err)
				return
			}
			_, err = docker.LoadImageFromFile(dc, imageTarFile)
			spinner.Close()
			if err != nil {
				d.ErrorExit("faild: %v", err)
				return
			}
			d.Println("ok!")
		}
		d.Println("load bundle %s-%s ok!", comp.GetName(), comp.GetVersion())
	}

	if len(d.Bundle) == 0 && d.All {
		// 下载Keepalived
		imgtarfname := path.Join(bundleDir, fmt.Sprintf("%s-%s-IMAGES.tar", "keepalived", BaseConfig.Keepalived.Version))
		spinner = utils.NewSpinner(fmt.Sprintf("load docker images from %s...", imgtarfname), setting.OutOrStdout())
		sumData, err = utils.FileChecksum(imgtarfname, sha256.New())
		if err != nil {
			spinner.Close()
			d.ErrorExit("read %s checksum error: %v", imgtarfname, err)
		}
		if err = down.VerifyBundle(imgtarfname+".sha256", hex.EncodeToString(sumData)); err != nil {
			spinner.Close()
			d.ErrorExit("file %s checksum error: %v", imgtarfname, err)
			return
		}
		spinner.Close()
		if _, err = docker.LoadImageFromFile(dc, imgtarfname); err != nil {
			d.ErrorExit("%v", err)
			return err
		}

		// 下载Haproxy
		imgtarfname = path.Join(bundleDir, fmt.Sprintf("%s-%s-IMAGES.tar", "haproxy", BaseConfig.Haproxy.Version))
		spinner = utils.NewSpinner(fmt.Sprintf("load docker images from %s...", imgtarfname), setting.OutOrStdout())
		sumData, err = utils.FileChecksum(imgtarfname, sha256.New())
		if err != nil {
			spinner.Close()
			d.ErrorExit("read %s checksum error: %v", imgtarfname, err)
		}
		if err = down.VerifyBundle(imgtarfname+".sha256", hex.EncodeToString(sumData)); err != nil {
			spinner.Close()
			d.ErrorExit("file %s checksum error: %v", imgtarfname, err)
			return
		}
		spinner.Close()
		if _, err = docker.LoadImageFromFile(dc, imgtarfname); err != nil {
			d.ErrorExit("%v", err)
			return err
		}
	}
	return
}

func (b *SearchBundle) Run() (err error) {
	setting := b.GlobalSetting()
	setting.InitLogger("search")

	icfile, ic, err := setting.LoadLocalInstallConfiguration()
	if err != nil {
		b.ErrorExit("load %s error: %s", icfile, err)
		return err
	}

	for _, registry := range ic.GetHelmRegistryUrlMap() {
		if registry.sysconfig {
			continue
		}
		chartVersions, err := registry.ListChartVersions(helmGetter.All(ic.setting.helmSetting))
		if err != nil {
			b.ErrorExit("list bundleerror: %s", err)
			return err
		}

		searchArray := strings.SplitN(b.SearchKey, "@", 2)
		var searchedVersions []*helmRepo.ChartVersion = make([]*helmRepo.ChartVersion, 0)

		for _, chartVersions := range chartVersions {
			if len(searchArray) == 1 {
				for _, chatVersion := range chartVersions {
					if strings.Contains(chatVersion.Name, searchArray[0]) || strings.Contains(chatVersion.Description, searchArray[0]) {
						searchedVersions = append(searchedVersions, chatVersion)
					}
				}
			} else {
				for _, chatVersion := range chartVersions {
					if strings.Contains(chatVersion.Name, searchArray[0]) && strings.Contains(chatVersion.Version, searchArray[1]) {
						searchedVersions = append(searchedVersions, chatVersion)
					}
				}
			}
		}

		if len(searchedVersions) == 0 {
			b.Println("No results found for %s:", len(searchedVersions), b.SearchKey)
		} else {
			// 打印搜索结果数量
			b.Println("Found %d results for %s", len(searchedVersions), b.SearchKey)
			// 打印搜索结果
			for _, chatVersion := range searchedVersions {
				componentName := chatVersion.Name + "@" + chatVersion.Version
				componentDest := chatVersion.Description
				if len(chatVersion.Description) > 60 {
					componentDest = chatVersion.Description[:57] + "..."
				} else if chatVersion.Description == "" {
					chatVersion.Description = "{No description}"
				}
				repeatSpace := " "
				if len(componentName) < 40 {
					repeatSpace = strings.Repeat(" ", 40-len(componentName))
				}
				b.Println("%s %s %s", componentName, repeatSpace, componentDest)
			}
		}

	}

	return
}

func (b *BundleList) Run() (err error) {
	setting := b.GlobalSetting()
	setting.InitLogger("list")

	icfile, ic, err := setting.LoadLocalInstallConfiguration()
	if err != nil {
		b.ErrorExit("load %s error: %s", icfile, err)
		return err
	}

	for _, registry := range ic.GetHelmRegistryUrlMap() {
		if registry.sysconfig {
			continue
		}
		chartVersions, err := registry.ListChartVersions(helmGetter.All(ic.setting.helmSetting))
		if err != nil {
			b.ErrorExit("list bundleerror: %s", err)
			return err
		}
		for _, chartVersions := range chartVersions {
			for _, chatVersion := range chartVersions {
				componentName := chatVersion.Name + "@" + chatVersion.Version
				componentDest := chatVersion.Description
				if len(chatVersion.Description) > 60 {
					componentDest = chatVersion.Description[:57] + "..."
				} else if chatVersion.Description == "" {
					chatVersion.Description = "{No description}"
				}
				repeatSpace := " "
				if len(componentName) < 40 {
					repeatSpace = strings.Repeat(" ", 40-len(componentName))
				}
				b.Println("%s %s %s", componentName, repeatSpace, componentDest)
			}
		}
	}

	return
}

func (b *ShowBundleImages) Run() (err error) {
	setting := b.GlobalSetting()
	setting.InitLogger("list")

	icfile, ic, err := setting.LoadLocalInstallConfiguration()
	if err != nil {
		b.ErrorExit("load %s error: %s", icfile, err)
		return err
	}

	var (
		compNames  []string
		components map[string]Component
		imageNames []string
	)

	// 核心组件
	if compNames, components, err = ic.GetBundleComponents(false); err != nil {
		b.ErrorExit("load bundle error: %s", err)
		return
	}

	if len(b.BundleNames) > 0 {
		oComps := make(map[string]Component)
		for _, v := range b.BundleNames {
			if components[v] == nil {
				continue
			}
			oComps[v] = components[v]
		}
		compNames, components = b.BundleNames, oComps
	}

	if imageNames, err = ic.getBundleImages(compNames, components); err != nil {
		b.ErrorExit("load images error: %s", err)
		return
	}

	for _, v := range imageNames {
		b.Println("%s", v)
	}
	return
}
