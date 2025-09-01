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
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/hbagdi/go-kong/kong"
	"github.com/pkg/errors"
	"snz1.cn/snz1dp/snz1dpctl/down"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

const (
	// JWTAlgorithm 默认JWT算法名称
	JWTAlgorithm = "RS256"
)

// GetIngressConsumer 获取应用
func GetIngressConsumer(kc *kong.Client, nameOrID string) (consumer *kong.Consumer, err error) {
	consumer, err = kc.Consumers.Get(context.Background(), kong.String(nameOrID))
	return
}

// DeleteIngressConsumer -
func DeleteIngressConsumer(kc *kong.Client, nameOrID string) (err error) {
	err = kc.Consumers.Delete(context.Background(), kong.String(nameOrID))
	return
}

// DeleteIngressConsumerJWTAuth -
func DeleteIngressConsumerJWTAuth(kc *kong.Client, consumer *kong.Consumer, jwtAuth ...string) (err error) {
	for _, k := range jwtAuth {
		if err = kc.JWTAuths.Delete(context.Background(), consumer.ID, kong.String(k)); err != nil {
			return
		}
	}
	return
}

// GetIngressConsumerCustomID 获取CustomID
func GetIngressConsumerCustomID(consumer *kong.Consumer) (cid string) {
	if consumer.CustomID == nil {
		return
	}
	cid = *consumer.CustomID
	return
}

// GetIngressConsumerTagList 获取路径
func GetIngressConsumerTagList(consumer *kong.Consumer) (tags string) {
	if consumer.Tags == nil || len(consumer.Tags) == 0 {
		tags = "<None>"
		return
	}
	rtags := []string{}
	for _, t := range consumer.Tags {
		rtags = append(rtags, *t)
	}
	tags = strings.Join(rtags, ",")
	return
}

// ConsumerWrapper -
type ConsumerWrapper struct {
	kong.Consumer
	JWTAuths []*kong.JWTAuth  `json:"jwtauth,omitempty"`
	ACLGroup []*kong.ACLGroup `json:"aclgroup,omitempty"`
}

// ConsumerVisit -
type ConsumerVisit func(*kong.Consumer, []*kong.JWTAuth, []*kong.ACLGroup)

// GetIngressConsumerJWTAuths -
func GetIngressConsumerJWTAuths(kc *kong.Client, consumer *kong.Consumer) (jwtAuths []*kong.JWTAuth, err error) {

	var (
		opt *kong.ListOpt = &kong.ListOpt{
			Size: 1000,
		}
		auths []*kong.JWTAuth
	)

	for {
		if auths, opt, err = kc.JWTAuths.ListForConsumer(context.Background(), consumer.ID, opt); err != nil {
			return
		}
		jwtAuths = append(jwtAuths, auths...)
		if opt == nil {
			break
		}
	}

	return
}

// GetIngressConsumerACLGroups -
func GetIngressConsumerACLGroups(kc *kong.Client, consumer *kong.Consumer) (aclGroups []*kong.ACLGroup, err error) {
	var (
		opt *kong.ListOpt = &kong.ListOpt{
			Size: 1000,
		}
		acls []*kong.ACLGroup
	)

	for {
		if acls, opt, err = kc.ACLs.ListForConsumer(context.Background(), consumer.ID, opt); err != nil {
			return
		}
		aclGroups = append(aclGroups, acls...)
		if opt == nil {
			break
		}
	}

	return
}

// ListIngressConsumers -
func ListIngressConsumers(kc *kong.Client, opt *kong.ListOpt, visit ConsumerVisit) (err error) {
	var (
		consumerList []*kong.Consumer
		jwtAuthList  []*kong.JWTAuth
		aclGroup     []*kong.ACLGroup
	)
	if opt == nil {
		opt = new(kong.ListOpt)
		opt.Size = 100
	}
	for {
		consumerList, opt, err = kc.Consumers.List(context.Background(), opt)
		if len(consumerList) == 0 {
			break
		}
		for _, consumer := range consumerList {
			if jwtAuthList, err = GetIngressConsumerJWTAuths(kc, consumer); err != nil {
				return
			}
			if aclGroup, err = GetIngressConsumerACLGroups(kc, consumer); err != nil {
				return
			}
			visit(consumer, jwtAuthList, aclGroup)
		}
		if opt == nil {
			break
		}
	}
	return
}

// ApplyIngressConsumer 创建或更新
func ApplyIngressConsumer(kc *kong.Client, orgConsumer *kong.Consumer, existedUpdate bool) (consumer *kong.Consumer, err error) {
	var (
		idOrUsername *string = orgConsumer.ID
	)

	if idOrUsername == nil {
		idOrUsername = orgConsumer.Username
	}

	if consumer, err = kc.Consumers.Get(context.Background(), idOrUsername); err != nil && !kong.IsNotFoundErr(err) {
		return
	}

	if consumer == nil {
		consumer, err = kc.Consumers.Create(context.Background(), orgConsumer)
	} else if existedUpdate {
		if orgConsumer.CustomID != nil {
			consumer.CustomID = orgConsumer.CustomID
		}
		if orgConsumer.Username != nil {
			consumer.Username = orgConsumer.Username
		}
		if len(orgConsumer.Tags) > 0 {
			consumer.Tags = orgConsumer.Tags
		}
		consumer, err = kc.Consumers.Update(context.Background(), consumer)
	} else {
		err = errors.Errorf("consumer %s existed: id=%s, constomid=%s, username=%s",
			*idOrUsername, *consumer.ID, GetIngressConsumerCustomID(consumer), *consumer.Username)
	}

	return
}

// ApplyIngressConsumerGroups -
func ApplyIngressConsumerGroups(kc *kong.Client, consumer *kong.Consumer, groups []string) (acls []*kong.ACLGroup, err error) {
	var (
		opt *kong.ListOpt = &kong.ListOpt{
			Size: 100,
		}
		aclGroups  []*kong.ACLGroup
		aclGroup   *kong.ACLGroup
		groupMap   map[string]bool = make(map[string]bool)
		existedMap map[string]bool = make(map[string]bool)
	)

	for _, g := range groups {
		groupMap[g] = true
	}

	for {
		if aclGroups, opt, err = kc.ACLs.ListForConsumer(context.Background(), consumer.ID, opt); err != nil {
			return
		}
		for _, g := range aclGroups {
			if groupMap[*g.Group] {
				existedMap[*g.Group] = true
				acls = append(acls, g)
			} else if err = kc.ACLs.Delete(context.Background(), consumer.ID, g.ID); err != nil {
				return
			}
		}
		if opt == nil {
			break
		}
	}

	for k := range groupMap {
		if existedMap[k] {
			continue
		}
		aclGroup = &kong.ACLGroup{Group: kong.String(k)}
		if aclGroup, err = kc.ACLs.Create(context.Background(), consumer.ID, aclGroup); err != nil {
			return
		}
		acls = append(acls, aclGroup)
	}

	return
}

// ApplyIngressAppConsumerJWTAuth 设置JWT认证
func ApplyIngressAppConsumerJWTAuth(kc *kong.Client, consumer *kong.Consumer, keyID, rsaPublicKey string, existedUpdate bool) (jwtPl *kong.JWTAuth, err error) {
	var (
		jwtAuths                    []*kong.JWTAuth
		keyIDExisted, pubKeyExisted bool
	)

	if jwtAuths, _, err = kc.JWTAuths.ListForConsumer(context.Background(), consumer.ID, &kong.ListOpt{Size: 1000}); err != nil {
		return
	}

	for _, pl := range jwtAuths {
		if *pl.Key == keyID && *pl.RSAPublicKey == rsaPublicKey {
			keyIDExisted = true
			pubKeyExisted = true
			jwtPl = pl
			break
		} else if *pl.Key == keyID {
			keyIDExisted = true
			jwtPl = pl
			break
		} else if *pl.RSAPublicKey == rsaPublicKey {
			pubKeyExisted = true
			jwtPl = pl
			break
		}
	}

	if jwtPl == nil {
		jwtPl = &kong.JWTAuth{
			Consumer:     consumer,
			Algorithm:    kong.String(JWTAlgorithm),
			RSAPublicKey: kong.String(rsaPublicKey),
		}
		if keyID != "" {
			jwtPl.Key = kong.String(keyID)
		}
		jwtPl, err = kc.JWTAuths.Create(context.Background(), consumer.ID, jwtPl)
	} else if existedUpdate {
		jwtPl.RSAPublicKey = kong.String(rsaPublicKey)
		jwtPl.Algorithm = kong.String(JWTAlgorithm)
		if keyID != "" {
			jwtPl.Key = kong.String(keyID)
		}
		jwtPl, err = kc.JWTAuths.Update(context.Background(), consumer.ID, jwtPl)
	} else if keyIDExisted && pubKeyExisted {
		err = errors.Errorf("rsa public key and jwt key existed, keyid=%s", *jwtPl.Key)
	} else if keyIDExisted {
		err = errors.Errorf("other rsa public key bind jwt key, keyid=%s", *jwtPl.Key)
	} else if pubKeyExisted {
		err = errors.Errorf("rsa public key and jwt key existed, keyid=%s", *jwtPl.Key)
	}
	return
}

// CreateRSAKeyPair 创建密钥对
type CreateRSAKeyPair struct {
	BaseAction
	Name   string
	Public string
	Bits   int
	Force  bool
}

// NewCreateRSAKeyPair -
func NewCreateRSAKeyPair(setting *GlobalSetting) *CreateRSAKeyPair {
	return &CreateRSAKeyPair{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run 创建密钥对实现
func (c *CreateRSAKeyPair) Run() (err error) {
	var (
		setting *GlobalSetting = c.GlobalSetting()
		rsaKey  *rsa.PrivateKey
		privPem string
		pubPem  string
	)
	if rsaKey, err = utils.GenerateRSAKeyPair(c.Bits); err != nil {
		c.ErrorExit("create rsa keypair error: %s", err)
		return
	}

	if privPem, err = utils.RSAPrivateKeyToPem(rsaKey); err != nil {
		c.ErrorExit("convert rsa private key to pem error: %s", err)
		return
	}

	if pubPem, err = utils.RSAPublicKeyToPem(&rsaKey.PublicKey); err != nil {
		c.ErrorExit("convert rsa public key to pem error: %s", err)
		return
	}

	if c.Name == "" {
		c.Printf("%s", privPem)
	} else {
		if _, err = os.Stat(c.Name); err == nil {
			if !c.Force && !utils.Confirm(fmt.Sprintf("file %s existed, proceed? (y/N)", c.Name), setting.InOrStdin(), setting.OutOrStdout()) {
				c.Println("Cancelled.")
				return nil
			}
			os.RemoveAll(c.Name)
		}

		if err = os.WriteFile(c.Name, []byte(privPem), 0644); err != nil {
			err = errors.Errorf("write %s error: %s", c.Name, err)
		} else {
			c.Println("rsa private key save to %s!", c.Name)
		}
	}

	if c.Public == "" {
		return
	}

	if _, err = os.Stat(c.Public); err == nil {
		if !c.Force && !utils.Confirm(fmt.Sprintf("file %s existed, proceed? (y/N)", c.Public), setting.InOrStdin(), setting.OutOrStdout()) {
			c.Println("Cancelled.")
			return nil
		}
		os.RemoveAll(c.Public)
	}

	if err = os.WriteFile(c.Public, []byte(pubPem), 0644); err != nil {
		err = errors.Errorf("write %s error: %s", c.Public, err)
	} else {
		c.Println("rsa public key save to %s!", c.Public)
	}

	return
}

// CreateJwtConsumer JWT调用者
type CreateJwtConsumer struct {
	BaseIngressAction
	ConsumerID   string
	KeyID        string
	Override     bool
	RSAPublicKey string
	Tags         []string
	Group        []string
}

// IngressListAppConsumer -
type IngressListAppConsumer struct {
	BaseIngressAction
	ConsumerID   []string
	OutputFormat string
	Tags         []string
	MatchAllTags bool
}

// IngressDeleteConsumer -
type IngressDeleteConsumer struct {
	BaseIngressAction
	ConsumerID []string
	KeyID      []string
	Force      bool
}

// ToKongConsumer 转kong.Consumer
func (c *CreateJwtConsumer) ToKongConsumer() (consumer *kong.Consumer) {
	consumer = &kong.Consumer{
		Username: kong.String(c.ConsumerID),
	}
	for _, t := range c.Tags {
		consumer.Tags = append(consumer.Tags, kong.String(t))
	}
	return
}

// NewCreateJwtConsumer 创建
func NewCreateJwtConsumer(setting *GlobalSetting) *CreateJwtConsumer {
	return &CreateJwtConsumer{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// NewIngressListAppConsumer 创建
func NewIngressListAppConsumer(setting *GlobalSetting) *IngressListAppConsumer {
	return &IngressListAppConsumer{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// NewIngressDeleteConsumer -
func NewIngressDeleteConsumer(setting *GlobalSetting) *IngressDeleteConsumer {
	return &IngressDeleteConsumer{
		BaseIngressAction: BaseIngressAction{
			BaseAction: BaseAction{
				setting: setting,
			},
		},
	}
}

// GetNameMap 获取名称
func (i *IngressListAppConsumer) GetNameMap() (ret map[string]bool) {
	ret = make(map[string]bool)
	for _, name := range i.ConsumerID {
		ret[name] = true
	}
	return
}

// Run 执行创建JWT调用者
func (c *CreateJwtConsumer) Run() (err error) {
	var (
		setting                     *GlobalSetting = c.GlobalSetting()
		currentDir                  string
		rsaPublicKey, rsaPrivateKey string
		icdata                      []byte
		furl                        *url.URL
		spinner                     *utils.WaitSpinner
		rsaKey                      *rsa.PublicKey
		privKey                     *rsa.PrivateKey
		jwtPl                       *kong.JWTAuth
		consumer                    *kong.Consumer
		kc                          *kong.Client
	)

	if currentDir, err = os.Getwd(); err != nil {
		err = errors.Errorf("get cwd error: %s", err)
		return
	}

	if kc, err = c.CreateKong(nil); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}

	if c.RSAPublicKey != "" {
		// 尝试解码内容
		if strings.Index(c.RSAPublicKey, "-----BEGIN") == 0 {
			rsaPublicKey = strings.ReplaceAll(c.RSAPublicKey, "\\n", "\n")
		} else {
			if furl, err = url.Parse(c.RSAPublicKey); err != nil {
				err = errors.Errorf("error url %s: %s", c.RSAPublicKey, err)
				return
			}

			if furl.Scheme == "" || furl.Scheme == "file" {
				var fromFilePath string = furl.Path
				if !filepath.IsAbs(fromFilePath) {
					fromFilePath = filepath.Join(currentDir, fromFilePath)
					if fromFilePath, err = filepath.Abs(fromFilePath); err != nil {
						err = errors.Errorf("error file path: %s", err)
						return
					}
				}

				if icdata, err = os.ReadFile(fromFilePath); err != nil {
					err = errors.Errorf("read %s error: %s", fromFilePath, err)
					return
				}

			} else {
				spinner = utils.NewSpinner(fmt.Sprintf("download %s...", furl.String()), setting.OutOrStdout())
				var g down.Getter
				if g, err = down.AllProviders().ByScheme(furl.Scheme); err != nil {
					spinner.Close()
					c.ErrorExit("failed: %s", err)
					return
				}

				var fout *bytes.Buffer = bytes.NewBuffer(nil)

				if _, err = g.Get(c.RSAPublicKey, fout, nil, nil); err != nil {
					spinner.Close()
					c.ErrorExit("failed: %s", err)
					return
				}
				spinner.Close()

				icdata = fout.Bytes()
				c.Println("ok!")

			}
			rsaPublicKey = string(icdata)
		}

		if rsaKey, privKey, err = utils.DecodePublicKeyFromPEM(rsaPublicKey); err != nil {
			err = errors.Errorf("decode pem error: %s\n%s", err, rsaPublicKey)
			return
		}

		if rsaPublicKey, err = utils.RSAPublicKeyToPem(rsaKey); err != nil {
			err = errors.Errorf("encode rsa public to pem error: %s\n%s", err, rsaPublicKey)
			return
		}

		if privKey != nil {
			if rsaPrivateKey, err = utils.RSAPrivateKeyToPem(privKey); err != nil {
				err = errors.Errorf("ecode rsa private key to pem error: %s", err)
				return
			}
			rsaPrivateKey = strings.ReplaceAll(rsaPrivateKey, "\n", "\\n")
		}

	}

	if consumer, err = ApplyIngressConsumer(kc, c.ToKongConsumer(), true); err != nil {
		err = errors.Errorf("create consumer error: %s", err)
		return
	}

	if _, err = ApplyIngressConsumerGroups(kc, consumer, c.Group); err != nil {
		err = errors.Errorf("create consumer group error: %s", err)
		return
	}

	if rsaPublicKey != "" {
		if jwtPl, err = ApplyIngressAppConsumerJWTAuth(kc, consumer, c.KeyID, rsaPublicKey, c.Override); err != nil {
			err = errors.Errorf("add consumer %s jwtauth error: %s", c.ConsumerID, err)
			return
		}

		c.Println("create %s jwtauth success, keyid: %s", c.ConsumerID, *jwtPl.Key)
		c.Println("#-----------------------------------------------")
		c.Println("# copy follow lines to application.properties")
		c.Println("#-----------------------------------------------")
		if rsaPrivateKey != "" {
			c.Println("app.jwt.private_key=%s", rsaPrivateKey)
		} else {
			c.Println("app.jwt.private_key=<pem context of rsa private key>")
		}
		c.Println("app.jwt.token=%s", *jwtPl.Key)
		c.Println("app.jwt.live_time=1800")
		c.Println("#-----------------------------------------------")
		c.Println("")
		c.Println("#-----------------------------------------------")
		c.Println("# copy follow lines to BUILD.yaml(service.env)")
		c.Println("#-----------------------------------------------")
		c.Println("  - JWT_TOKEN=%s", *jwtPl.Key)
		if rsaPrivateKey != "" {
			c.Println("  - JWT_PRIVKEY=%s", rsaPrivateKey)
		} else {
			c.Println("  - JWT_PRIVKEY=<pem context of rsa private key>")
		}
		c.Println("#-----------------------------------------------")
		c.Println("")
		c.Println("#-----------------------------------------------")
		c.Println("# copy follow line to bundle node of global.yaml")
		c.Println("#-----------------------------------------------")
		c.Println("  jwt:")
		c.Println("    token: %s", *jwtPl.Key)
		if rsaPrivateKey != "" {
			c.Println("    rsakey: %s", rsaPrivateKey)
		} else {
			c.Println("    rsakey=<pem context of rsa private key>")
		}
		c.Println("#-----------------------------------------------")
	} else {
		s := NewIngressListAppConsumer(setting)
		s.ConsumerID = []string{*consumer.Username}
		err = s.Run()
	}

	return
}

// Run -
func (i IngressListAppConsumer) Run() (err error) {
	var (
		kc    *kong.Client
		opt   *kong.ListOpt = &kong.ListOpt{Size: 100}
		names map[string]bool
	)
	if kc, err = i.CreateKong(nil); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}

	for _, tag := range i.Tags {
		opt.Tags = append(opt.Tags, kong.String(tag))
	}

	names = i.GetNameMap()
	opt.MatchAllTags = i.MatchAllTags

	switch i.OutputFormat {
	case "table", "":
		i.Println("%-38s %-30s %-38s %-48s %-40s", "ID", "AppID", "JWT Key IDs", "ACL Groups", "Tags")

		err = ListIngressConsumers(kc, opt, func(consumer *kong.Consumer, jwtAuths []*kong.JWTAuth, aclGroups []*kong.ACLGroup) {
			var (
				keyVal   string
				keyIDs   []string
				groupVal string
				groups   []string
			)
			if len(names) > 0 && !(names[*consumer.Username] || names[*consumer.ID]) {
				return
			}
			for _, p := range jwtAuths {
				keyIDs = append(keyIDs, *p.Key)
			}
			keyVal = strings.Join(keyIDs, ",")
			if keyVal == "" {
				keyVal = "<None>"
			}
			for _, g := range aclGroups {
				groups = append(groups, *g.Group)
			}
			groupVal = strings.Join(groups, ",")
			if groupVal == "" {
				groupVal = "<None>"
			}
			i.Println("%-38s %-30s %-38s %-48s %-40s", *consumer.ID, *consumer.Username,
				keyVal, groupVal, GetIngressConsumerTagList(consumer))
		})

	case "yaml", "json":
		var (
			objList   []*ConsumerWrapper
			routeData []byte
		)
		_ = ListIngressConsumers(kc, opt, func(consumer *kong.Consumer, jwtAuths []*kong.JWTAuth, aclGroups []*kong.ACLGroup) {
			if len(names) > 0 && !(names[*consumer.Username] || names[*consumer.ID]) {
				return
			}
			objList = append(objList, &ConsumerWrapper{Consumer: *consumer, JWTAuths: jwtAuths, ACLGroup: aclGroups})
		})
		if i.OutputFormat == "yaml" {
			routeData, err = yaml.Marshal(objList)
		} else {
			routeData, err = json.Marshal(objList)
		}
		if err == nil {
			i.Println("%s", string(routeData))
		}
	default:
		err = errors.Errorf("not found output format: %s", i.OutputFormat)
	}
	return
}

// Run -
func (i *IngressDeleteConsumer) Run() (err error) {
	var (
		setting  *GlobalSetting = i.GlobalSetting()
		kc       *kong.Client
		consumer *kong.Consumer
		cmsg     string
	)
	if kc, err = i.CreateKong(nil); err != nil {
		err = errors.Errorf("create ingress admin client error: %s", err)
		return
	}

	for _, nameOrID := range i.ConsumerID {

		if consumer, err = GetIngressConsumer(kc, nameOrID); err != nil {
			if kong.IsNotFoundErr(err) {
				continue
			}
			return err
		}

		if len(i.KeyID) > 0 {
			cmsg = fmt.Sprintf("delete consumer jwt auth:\n  id=%s,\n  appid=%s\n",
				*consumer.ID, *consumer.Username)

			for _, keyid := range i.KeyID {
				cmsg += fmt.Sprintf("  keyid=%s,\n", keyid)
			}

			cmsg += "proceed? (y/N)"
			if !i.Force && !utils.Confirm(cmsg, setting.InOrStdin(), setting.OutOrStdout()) {
				i.Println("Cancelled.")
				return
			}

			for _, keyid := range i.KeyID {
				if err = DeleteIngressConsumerJWTAuth(kc, consumer, keyid); err != nil {
					i.ErrorExit("delete consumer %s jwt auth(keyid=%s) error: %s", *consumer.Username, keyid, err)
					return
				}
			}
			return
		}

		cmsg = fmt.Sprintf("delete consumer:\n  id=%s,\n  appid=%s\nproceed? (y/N)", *consumer.ID, *consumer.Username)
		if !i.Force && !utils.Confirm(cmsg, setting.InOrStdin(), setting.OutOrStdout()) {
			i.Println("Cancelled.")
			return
		}

		if err = DeleteIngressConsumer(kc, *consumer.ID); err != nil {
			if kong.IsNotFoundErr(err) {
				continue
			}
			return err
		}

	}

	err = nil

	return
}
