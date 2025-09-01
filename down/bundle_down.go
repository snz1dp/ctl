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

package down

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

// VerificationStrategy describes a strategy for determining whether to verify a chart.
type VerificationStrategy int

const (
	// VerifyNever will skip all verification of a chart.
	VerifyNever VerificationStrategy = iota
	// VerifyIfPossible will attempt a verification, it will not error if verification
	// data is missing. But it will not stop processing if verification fails.
	VerifyIfPossible
	// VerifyAlways will always attempt a verification, and will fail if the
	// verification fails.
	VerifyAlways
	// VerifyLater will fetch verification data, but not do any verification.
	// This is to accommodate the case where another step of the process will
	// perform verification.
	VerifyLater
)

// BundleDownloader -
type BundleDownloader struct {
	out       io.Writer
	providers Providers
	options   []Option
	verify    VerificationStrategy
	digest    string
	href      string
}

// NewBundleDownloader -
func NewBundleDownloader(out io.Writer, href string,
	verify VerificationStrategy, options ...Option) *BundleDownloader {
	ret := &BundleDownloader{
		out:       out,
		href:      href,
		verify:    verify,
		options:   options,
		providers: AllProviders(),
	}
	return ret
}

// SetFileDigest
func (c *BundleDownloader) SetFileDigest(d string) *BundleDownloader {
	c.digest = d
	return c
}

// Download -
func (c *BundleDownloader) Download(dest string, name string) (string, error) {
	ref := c.href
	u, err := url.Parse(ref)
	if err != nil {
		return "", errors.Errorf("invalid bundle URL format: %s", ref)
	}

	g, err := c.providers.ByScheme(u.Scheme)
	if err != nil {
		return "", err
	}

	if name == "" {
		name = filepath.Base(u.Path)
	}

	destfile := filepath.Join(dest, name)

	tempfile, fout, err := utils.CreateFileTempWriter(destfile, 0644)
	if err != nil {
		return destfile, err
	}

	defer func() {
		os.Remove(tempfile)
	}()

	sha256sum := sha256.New()

	_, err = g.Get(u.String(), fout, nil, sha256sum, c.options...)
	if err != nil {
		fout.Close()
		return destfile, err
	}

	if err = fout.Close(); err != nil {
		return destfile, err
	}

	if err := utils.RenameWithFallback(tempfile, destfile); err != nil {
		return destfile, err
	}

	sha256subData := sha256sum.Sum(nil)

	if c.verify > VerifyNever {
		var digestBody *bytes.Buffer
		if c.digest == "" {
			digestBody = bytes.NewBuffer(nil)
			_, err := g.Get(u.String()+".sha256", digestBody, nil, nil)
			if err != nil {
				if c.verify == VerifyAlways {
					return destfile, errors.Errorf("failed to fetch %q", u.String()+".sha256")
				}
				fmt.Fprintf(c.out, "WARNING: Verification not found for %s: %s\n", ref, err)
				return destfile, nil
			}
		} else {
			digestBody = bytes.NewBuffer([]byte(c.digest))
		}

		provfile := destfile + ".sha256"
		if err := utils.AtomicWriteFile(provfile, digestBody, 0644); err != nil {
			return destfile, err
		}

		if c.verify != VerifyLater {
			err = VerifyBundle(provfile, hex.EncodeToString(sha256subData))
			if err != nil {
				// Fail always in this case, since it means the verification step
				// failed.
				return destfile, err
			}
		}
	}
	return destfile, nil
}

// VerifyBundle -
func VerifyBundle(path, keyring string) error {
	shabytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	vals := strings.Split(string(shabytes), " ")
	if len(vals) <= 0 {
		return errors.Errorf("bundle error checksum file")
	}

	if vals[0] != keyring {
		return errors.Errorf("bundle sha256 checksum error")
	}

	return nil
}
