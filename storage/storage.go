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

package storage

import (
	"context"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"snz1.cn/snz1dp/snz1dpctl/utils"
	"strconv"
	"time"
)

const (
	snz1dpInstallConfig = "snz1dp-install-config"
)

var (
	// ErrNotFound indicates that a release already exists.
	ErrNotFound = errors.New("not found")
	// ErrExists indicates that a release already exists.
	ErrExists = errors.New("already exists")
	// ErrInvalidKey indicates that a release key could not be parsed.
	ErrInvalidKey = errors.New("invalid key")
)

func newSecretsObject(key string, src []byte, lbs labels) (*v1.Secret, error) {
	const owner = "snz1dp"

	// encode the release
	s, err := utils.Encode(src)
	if err != nil {
		return nil, err
	}

	if lbs == nil {
		lbs.init()
	}

	// apply labels
	lbs.set("name", snz1dpInstallConfig)
	lbs.set("owner", owner)

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   key,
			Labels: lbs.toMap(),
		},
		Type: "snz1dp/release.v1",
		Data: map[string][]byte{"config": []byte(s)},
	}, nil
}

// DefaultStorageDriver - 缺省驱动
type DefaultStorageDriver struct {
	impl corev1.SecretInterface
	Log  func(string, ...interface{})
}

// Get 获取配置内容
func (d *DefaultStorageDriver) Get() ([]byte, error) {
	// fetch the secret holding the release named by key
	var key = snz1dpInstallConfig
	obj, err := d.impl.Get(context.Background(), key, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, errors.Wrapf(err, "get: failed to get %q", key)
	}
	// found the secret, decode the base64 data string
	r, err := utils.Decode(string(obj.Data["config"]))
	return r, errors.Wrapf(err, "get: failed to decode data %q", key)
}

// Create 创建安装存储
func (d *DefaultStorageDriver) Create(src []byte) error {
	var lbs labels

	lbs.init()
	lbs.set("modifiedAt", strconv.Itoa(int(time.Now().Unix())))

	var key = snz1dpInstallConfig

	// create a new secret to hold the release
	obj, err := newSecretsObject(key, src, lbs)
	if err != nil {
		return errors.Wrapf(err, "failed to encode config")
	}
	// push the secret object out into the kubiverse
	if _, err := d.impl.Create(context.Background(), obj, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return ErrExists
		}

		return errors.Wrap(err, "failed to create config")
	}
	return nil
}

// Update 更新
func (d *DefaultStorageDriver) Update(src []byte) error {
	var lbs labels

	lbs.init()
	lbs.set("modifiedAt", strconv.Itoa(int(time.Now().Unix())))

	var key = snz1dpInstallConfig
	// create a new secret object to hold the release
	obj, err := newSecretsObject(key, src, lbs)
	if err != nil {
		return errors.Wrapf(err, "failed to encode config")
	}
	// push the secret object out into the kubiverse
	_, err = d.impl.Update(context.Background(), obj, metav1.UpdateOptions{})
	return errors.Wrap(err, "failed to update config")
}

// Delete 删除配置
func (d *DefaultStorageDriver) Delete() (config []byte, err error) {
	if config, err = d.Get(); err != nil {
		return nil, err
	}
	// delete the config
	var key = snz1dpInstallConfig
	err = d.impl.Delete(context.Background(), key, metav1.DeleteOptions{})
	return config, err
}

// StorageDriver - 驱动
type storageDriver interface {
	// 获取
	Get() ([]byte, error)
	// 创建
	Create(src []byte) error
	// 更新
	Update([]byte) error
	// 删除
	Delete() ([]byte, error)
}

// ConfigStorage - 配置存储
type ConfigStorage struct {
	storageDriver
	Log func(string, ...interface{})
}

// Get - 获取
func (s *ConfigStorage) Get() ([]byte, error) {
	return s.storageDriver.Get()
}

// Create - 创建
func (s *ConfigStorage) Create(config []byte) error {
	return s.storageDriver.Create(config)
}

// Update - 更新
func (s *ConfigStorage) Update(config []byte) error {
	return s.storageDriver.Update(config)
}

// Delete - 删除
func (s *ConfigStorage) Delete() ([]byte, error) {
	return s.storageDriver.Delete()
}

// Init - 初始化
func Init(impl corev1.SecretInterface, log func(string, ...interface{})) *ConfigStorage {
	return &ConfigStorage{
		storageDriver: &DefaultStorageDriver{
			impl,
			log,
		},
		Log: log,
	}
}
