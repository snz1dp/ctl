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
	"strings"

	"github.com/pkg/errors"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

func validateWebProtocol(protocol string) error {
	protocol = strings.ToLower(protocol)
	if protocol == "http" || protocol == "https" {
		return nil
	}
	return errors.Errorf("invalid web protocol: %s", protocol)
}

// PreInit Postgres安装
func (c *PostgresConfig) PreInit(a Action) (ch bool, err error) {
	if c.BeInstall() {
		if c.Admin.Username == "" {
			c.Admin.Username = "postgres"
		}
		if c.Admin.Password == "" {
			c.Admin.Password = utils.RandString(16)
			ch = true
		}
		c.Host = "postgres"
		c.Port = new(uint32)
		*c.Port = 5432
	} else {
		if c.Port == nil || *c.Port <= 0 {
			ch = true
			c.Port = new(uint32)
			*c.Port = 5432
		}
		if c.infile && c.Host == "" {
			err = errors.Errorf("invalid postgres.host value!")
			return
		}
		if c.infile && c.Admin.Username == "" {
			err = errors.Errorf("invalid postgres.admin.username value!")
			return
		}
		if c.infile && c.Admin.Password == "" {
			err = errors.Errorf("invalid postgres.admin.password value!")
			return
		}
	}
	return
}

// PreInit Redis安装
func (c *RedisConfig) PreInit(a Action) (ch bool, err error) {
	if c.BeInstall() {
		if c.Password == "" {
			c.Password = utils.RandString(16)
			ch = true
		}
		c.Host = "redis"
		c.Port = new(uint32)
		*c.Port = 6379
	} else {
		if c.Port == nil || *c.Port <= 0 {
			ch = true
			c.Port = new(uint32)
			*c.Port = 6379
		}
		if c.infile && c.Host == "" {
			err = errors.Errorf("invalid redis.host value!")
			return
		}
		if c.infile && c.Password == "" {
			err = errors.Errorf("invalid redis.password value!")
			return
		}
	}
	return
}

// PreInit Ingress安装
func (c *AppgatewayConfig) PreInit(a Action) (ch bool, err error) {
	ic := c.InstallConfiguration()
	if ic.Snz1dp.Admin.Username == "" {
		ic.Snz1dp.Admin.Username = "root"
	}
	if ic.Snz1dp.Admin.Password == "" {
		ic.Snz1dp.Admin.Password = utils.RandString(16)
		ch = true
	}
	if ic.Snz1dp.Admin.Email == "" {
		ic.Snz1dp.Admin.Email = "root@localhost.com"
	}
	if ic.Snz1dp.Ingress.Host == "" {
		ic.Snz1dp.Ingress.Host = "localhost"
	}

	if err = validateWebProtocol(ic.Snz1dp.Ingress.Protocol); err != nil {
		err = errors.Errorf("error snz1dp.ingress.protocol value: %s", ic.Snz1dp.Ingress.Protocol)
		return
	}

	ic.Snz1dp.Ingress.Protocol = strings.ToLower(ic.Snz1dp.Ingress.Protocol)

	if ic.Snz1dp.Timezone == "" {
		ic.Snz1dp.Timezone = "Asia/Shanghai"
	}

	if ic.Snz1dp.Organization == "" {
		ic.Snz1dp.Organization = "ChangSha SNZ1"
	}

	if ic.Snz1dp.Ingress.Port == nil || *ic.Snz1dp.Ingress.Port <= 0 {
		ic.Snz1dp.Ingress.Port = new(uint32)
		ch = true
		if ic.Snz1dp.Ingress.Protocol == "http" {
			*ic.Snz1dp.Ingress.Port = 80
		} else {
			*ic.Snz1dp.Ingress.Port = 443
		}
	}

	if c.BeInstall() {
		ch = true
		c.Admin.Protocol = "http"
		c.Admin.Host = "ingress"
		c.Admin.Port = new(uint32)
		*c.Admin.Port = 91

		c.Web.Protocol = "http"
		c.Web.Host = "ingress"
		c.Web.Port = new(uint32)
		*c.Web.Port = 80
	} else {
		if c.Admin.Port == nil || *c.Admin.Port <= 0 {
			ch = true
			c.Admin.Port = new(uint32)
			*c.Admin.Port = 91
			if c.Admin.Protocol == "" || c.Admin.Protocol == "http" {
				c.Admin.Protocol = "http"
			} else {
				c.Admin.Protocol = "https"
			}
		}

		if c.Web.Port == nil || *c.Web.Port <= 0 {
			ch = true
			c.Web.Port = new(uint32)
			*c.Web.Port = 80
			if c.Web.Protocol == "" || c.Web.Protocol == "http" {
				c.Web.Protocol = "http"
			} else {
				c.Web.Protocol = "https"
			}
		}

		if c.Admin.Protocol == "" {
			c.Admin.Protocol = "http"
		} else if err = validateWebProtocol(c.Admin.Protocol); err != nil {
			err = errors.Errorf("invalid appgateway.admin.protocol value: %s", c.Admin.Protocol)
			return
		}

		ch = true
		c.Admin.Protocol = strings.ToLower(c.Admin.Protocol)

		if c.Web.Protocol == "" {
			c.Web.Protocol = "http"
		} else if err = validateWebProtocol(c.Web.Protocol); err != nil {
			err = errors.Errorf("invalid appgateway.web.protocol value: %s", c.Web.Protocol)
			return
		}
		c.Web.Protocol = strings.ToLower(c.Admin.Protocol)

		if c.infile && c.Admin.Host == "" {
			err = errors.Errorf("error appgateway.admin.host value, must be not empty!")
			return
		}
		if c.infile && c.Web.Host == "" {
			err = errors.Errorf("error appgateway.web.host value, must be not empty!")
			return
		}
	}

	return
}

// PreInit Confserv安装
func (c *ConfservConfig) PreInit(a Action) (ch bool, err error) {
	if c.BeInstall() {
		ch = true
		c.Web.Protocol = "http"
		c.Web.Host = c.GetName()
		c.Web.Port = new(uint32)
		*c.Web.Port = 80
		c.Web.Webroot = "/appconfig"
	} else {
		if c.Web.Port == nil || *c.Web.Port <= 0 {
			ch = true
			c.Web.Port = new(uint32)
			if c.Web.Protocol == "" || c.Web.Protocol == "http" {
				*c.Web.Port = 80
				c.Web.Protocol = "http"
			} else {
				*c.Web.Port = 443
				c.Web.Protocol = "https"
			}
		}

		if c.Web.Protocol == "" {
			c.Web.Protocol = "http"
		} else if err = validateWebProtocol(c.Web.Protocol); err != nil {
			err = errors.Wrapf(err, "invalid confserv.web.protocol value: %s", c.Web.Protocol)
			return
		}

		ch = true
		c.Web.Protocol = strings.ToLower(c.Web.Protocol)

		if c.infile && c.Web.Host == "" {
			err = errors.Errorf("invalid confserv.web.host value, must not empty!")
			return
		}
	}
	return
}

// PreInit Xeai安装
func (c *XeaiConfig) PreInit(a Action) (ch bool, err error) {

	if c.BeInstall() {
		ch = true
		c.Web.Protocol = "http"
		c.Web.Host = c.GetName()
		c.Web.Port = new(uint32)
		*c.Web.Port = 80
		c.Web.Webroot = "/xeai"
	} else {
		if c.Web.Port == nil || *c.Web.Port <= 0 {
			c.Web.Port = new(uint32)
			ch = true
			if c.Web.Protocol == "" || c.Web.Protocol == "http" {
				c.Web.Protocol = "http"
				*c.Web.Port = 80
			} else {
				*c.Web.Port = 443
				c.Web.Protocol = "https"
			}
		}

		if c.Web.Protocol == "" {
			c.Web.Protocol = "http"
		} else if err = validateWebProtocol(c.Web.Protocol); err != nil {
			err = errors.Wrapf(err, "invalid xeai.web.protocol value: %s", c.Web.Protocol)
			return
		}

		ch = true
		c.Web.Protocol = strings.ToLower(c.Web.Protocol)
		if c.infile && c.Web.Host == "" {
			err = errors.Errorf("invalid xeai.web.host value, must not empty!")
			return
		}
	}

	return
}

// DoInit 安装
func (b *BaseVersionConfig) DoInit(a Action) (err error) {
	if _, err = InstallHelmComponent(b); err != nil {
		a.ErrorExit("%s", err)
		return
	}

	return
}

// DoFini 卸载
func (b *BaseVersionConfig) DoFini(a Action) (err error) {
	if _, err = UnInstallHelmComponent(b); err != nil {
		a.ErrorExit("%s", err)
		return
	}
	return
}
