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
	"path"
	"strings"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"snz1.cn/snz1dp/snz1dpctl/docker"
	"snz1.cn/snz1dp/snz1dpctl/down"
	"snz1.cn/snz1dp/snz1dpctl/kube"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

// PullK8sImage 拉取版本
type PullK8sImage struct {
	BaseAction
	Version       string
	SaveLocalFile bool
	Force         bool
}

// GetK8sVersion -
type GetK8sVersion struct {
	BaseAction
}

// NewPullK8sImage -
func NewPullK8sImage(setting *GlobalSetting) *PullK8sImage {
	return &PullK8sImage{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// NewGetK8sVersion -
func NewGetK8sVersion(setting *GlobalSetting) *GetK8sVersion {
	return &GetK8sVersion{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// ResolveK8sImageMap -
func ResolveK8sImageMap(version string) (map[string]string, error) {

	imageFileURL := fmt.Sprintf("%s%s/%s",
		BaseConfig.Snz1dp.K8s.Image.Prefix,
		version, BaseConfig.Snz1dp.K8s.Image.Filename)

	u, err := url.Parse(imageFileURL)
	if err != nil {
		return nil, err
	}
	g, err := down.AllProviders().ByScheme(u.Scheme)
	if err != nil {
		return nil, err
	}
	body := bytes.NewBuffer(nil)
	_, err = g.Get(u.String(), body, nil, nil)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]string)
	flines := strings.Split(body.String(), "\n")
	for _, v := range flines {
		imgepares := strings.Split(v, "=")
		if len(imgepares) < 2 {
			continue
		}
		ret[imgepares[0]] = imgepares[1]
	}
	return ret, nil
}

// Run 执行
func (p *PullK8sImage) Run() error {
	setting := p.GlobalSetting()

	icfile, ic, err := setting.LoadLocalInstallConfiguration()

	if err != nil {
		p.ErrorExit("load %s error: %v", icfile, err)
		return err
	}

	if p.Version == "" {
		emsg := "please use --version or -v to specify the version!"
		p.ErrorExit(emsg)
		return errors.Errorf(emsg)
	}

	if p.Version[0] == 'v' || p.Version[0] == 'V' {
		p.Version = p.Version[1:]
	}

	spinner := utils.NewSpinner(fmt.Sprintf("fetch kubernetes-%s images list...", p.Version), setting.OutOrStdout())
	imageMap, err := ResolveK8sImageMap(p.Version)
	if err != nil {
		spinner.Close()
		p.ErrorExit("failed: %s", err.Error())
		return err
	}

	spinner.Close()

	dclient, err := docker.NewClient()
	if err != nil {
		p.ErrorExit("%s", err.Error())
		return err
	}

	defer dclient.Close()
	p.Println("ok!")

	var imageNames []string
	for k, v := range imageMap {
		pullit := p.Force
		if !pullit {
			_, err := docker.ImageExisted(dclient, k)
			if err != nil {
				pullit = true
			}
		}
		if pullit {
			var (
				repoUsername, repoPassword = ic.ResolveImageRepoUserAndPwd(v)
			)
			spinner := utils.NewSpinner(fmt.Sprintf("pull %s image...", k), setting.OutOrStdout())
			err := docker.PullAndRenameImages(dclient, v, k, repoUsername, repoPassword, "")
			spinner.Close()
			if err != nil {
				p.ErrorExit("failed: %v", err.Error())
				return err
			}
			p.Println("ok!")
		}
		imageNames = append(imageNames, k)
	}

	if p.SaveLocalFile {
		spinner := utils.NewSpinner(fmt.Sprintf("save images to %s-%s-IMAGES.tar...", "k8s", p.Version), p.GlobalSetting().OutOrStdout())
		err = docker.SaveImageToFile(dclient, imageNames, path.Join(setting.GetBundleDir(), fmt.Sprintf("%s-%s-IMAGES.tar", "k8s", p.Version)))
		spinner.Close()
		if err != nil {
			p.ErrorExit("failed: %v", err.Error())
			return err
		}
		p.Println("ok!")
	}

	return nil
}

// LoadK8sImage -
type LoadK8sImage struct {
	BaseAction
	Version  string
	Filename string
}

// NewLoadK8sImage -
func NewLoadK8sImage(setting *GlobalSetting) *LoadK8sImage {
	return &LoadK8sImage{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *LoadK8sImage) Run() (err error) {
	setting := s.GlobalSetting()
	var (
		dclient    *client.Client
		spinner    *utils.WaitSpinner
		k8starfile string
		filename   string
	)

	if s.Version == "" && s.Filename == "" {
		s.ErrorExit("error version or filename!")
		return nil
	}

	if s.Version == "" {
		k8starfile = s.Filename
		filename = path.Base(k8starfile)
	} else {
		filename = fmt.Sprintf("%s-%s-IMAGES.tar", "k8s", s.Version)
		k8starfile = path.Join(setting.GetBundleDir(), filename)
	}

	dclient, err = docker.NewClient()
	if err != nil {
		s.ErrorExit("%s", err.Error())
		return
	}

	defer dclient.Close()

	spinner = utils.NewSpinner(fmt.Sprintf("load %s...", filename), setting.OutOrStdout())
	_, err = docker.LoadImageFromFile(dclient, k8starfile)
	spinner.Close()
	if err != nil {
		s.ErrorExit("failed: %s", err.Error())
		return
	}
	s.Println("ok!")
	return
}

// Run -
func (g *GetK8sVersion) Run() error {
	setting := g.GlobalSetting()

	// 初始化Helm操作配置
	InitHelmActionConfig(setting)

	client, err := setting.KubernetesClientSet()

	if err != nil {
		g.ErrorExit("\nkubernetes config error   : %v\n", err)
		return err
	}

	sv, err := kube.ServerVersion(client)
	if err != nil {
		g.ErrorExit("\nkubernetes connect error  : %s\n", err)
		return err
	}

	g.Println("kubernetes server version : %s", sv)
	var oldic *InstallConfiguration
	if oldic, err = setting.IsInitialized(); err != nil {
		g.Println("snz1d-%s not install!", utils.Version())
		return nil
	}

	// 显示版本
	return renderInstallBundlesVersion(setting.OutOrStdout(), oldic)
}
