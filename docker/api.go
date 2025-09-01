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

package docker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

// NewClient -
func NewClient() (*client.Client, error) {
	return client.NewEnvClient()
}

func buildAuthConfig(u string, p string) (auth string) {
	if u == "" || p == "" {
		return
	}
	var (
		authconfig types.AuthConfig
		authsrc    []byte
	)
	authconfig.Username = u
	authconfig.Password = p
	authsrc, _ = json.Marshal(authconfig)
	auth = base64.StdEncoding.EncodeToString(authsrc)
	return
}

// LoginRegistry -
func LoginRegistry(c *client.Client, repourl, u, p string) (registry.AuthenticateOKBody, error) {
	return c.RegistryLogin(context.Background(), types.AuthConfig{
		Username:      u,
		Password:      p,
		ServerAddress: repourl,
	})
}

// PullAndRenameImages -
func PullAndRenameImages(c *client.Client, f string, t string, u string, p string, platform string) error {

	resp, err := c.ImagePull(context.Background(), f, types.ImagePullOptions{
		RegistryAuth: buildAuthConfig(u, p),
		Platform:     platform,
	})

	if err != nil {
		return err
	}
	defer resp.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp)

	if t != "" && t != f {
		err = c.ImageTag(context.Background(), f, t)
		if err != nil {
			return err
		}
	}
	return nil
}

// PushImage -
func PushImage(c *client.Client, f string, u string, p string, platform string) error {
	resp, err := c.ImagePush(context.Background(), f, types.ImagePushOptions{
		RegistryAuth: buildAuthConfig(u, p),
		Platform:     platform,
	})
	if err != nil {
		return err
	}
	defer resp.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp)
	return nil
}

// TagImage -
func TagImage(c *client.Client, f string, t string) error {
	err := c.ImageTag(context.Background(), f, t)
	if err != nil {
		return err
	}
	return err
}

// ImageExisted -
func ImageExisted(c *client.Client, f string) (*types.ImageSummary, error) {
	lst, err := c.ImageList(context.Background(), types.ImageListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("reference", f),
		),
	})

	if err != nil {
		return nil, err
	}

	if len(lst) == 0 {
		return nil, errors.Errorf("not found %s", f)
	}

	return &lst[0], nil
}

// ContainerExisted -
func ContainerExisted(dc *client.Client, cn string) (ret types.Container, err error) {
	var (
		clst []types.Container
	)
	clst, err = dc.ContainerList(context.TODO(), types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", cn)),
	})
	if err != nil {
		return
	}

	for _, lst := range clst {
		for _, v := range lst.Names {
			if v[1:] == cn {
				ret = lst
				return
			}
		}
	}

	err = errors.Errorf("container %s not existed!", cn)

	return
}

// LoadImageFromFile -
func LoadImageFromFile(dc *client.Client, tarfile string) (ret string, err error) {
	var (
		fin   *os.File
		resp  types.ImageLoadResponse
		rdata []byte
	)
	fin, err = os.OpenFile(tarfile, os.O_RDONLY, 0644)
	if err != nil {
		return
	}

	defer fin.Close()
	resp, err = dc.ImageLoad(context.Background(), fin, false)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if rdata, err = ioutil.ReadAll(resp.Body); err != nil {
		return
	}
	ret = string(rdata)
	return
}

// SaveImageToFile -
func SaveImageToFile(dc *client.Client, images []string, tarfile string) (err error) {
	var (
		imageIDs   []string
		imgSummary *types.ImageSummary
		respBody   io.ReadCloser
		fout       *os.File
	)

	for _, img := range images {
		imgSummary, err = ImageExisted(dc, img)
		if err != nil {
			return
		}
		imageIDs = append(imageIDs, imgSummary.ID)
		imageIDs = append(imageIDs, img)
	}

	if len(imageIDs) == 0 {
		err = errors.Errorf("not found any images!")
		return
	}

	respBody, err = dc.ImageSave(context.Background(), imageIDs)
	if err != nil {
		return
	}

	defer respBody.Close()
	fout, err = os.OpenFile(tarfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer fout.Close()
	_, err = io.Copy(fout, respBody)

	return
}

// CreateNetwork -
func CreateNetwork(dc *client.Client, name string) (id string, err error) {
	nc := types.NetworkCreate{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Driver: "default",
		},
		CheckDuplicate: true,
		Internal:       false,
		EnableIPv6:     false,
		Attachable:     false,
		Ingress:        false,
		Scope:          "",
		ConfigOnly:     false,
	}

	var resp types.NetworkCreateResponse

	resp, err = dc.NetworkCreate(context.Background(), name, nc)
	if err != nil {
		return
	}
	id = resp.ID
	return
}

// NetworkExisted -
func NetworkExisted(dc *client.Client, name string) (id string, err error) {
	lst, err := dc.NetworkList(context.Background(), types.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", name)),
	})
	if err != nil {
		return
	}
	if len(lst) == 0 {
		err = errors.Errorf("network %s not existed!", name)
		return
	}
	id = lst[0].ID
	return
}

// IsDockerInToolBox -
func IsDockerInToolBox(dc *client.Client) (isTbx bool, err error) {
	var (
		sv types.Version
		sd int
	)
	sv, err = dc.ServerVersion(context.Background())
	if err != nil {
		return
	}
	sd = strings.Index(sv.KernelVersion, "boot2docker")
	isTbx = sd >= 0
	return
}

func IsDockerRunning() (ret bool) {
	dc, err := NewClient()
	if err != nil {
		ret = false
	} else {
		defer dc.Close()
		_, err := dc.ServerVersion(context.Background())
		if err != nil {
			ret = false
		} else {
			ret = true
		}
	}
	return
}
