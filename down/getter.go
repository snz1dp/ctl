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
	"github.com/pkg/errors"
	"hash"
	"io"
	"time"
)

// options are generic parameters to be provided to the getter during instantiation.
//
// Getters may or may not ignore these parameters as they are passed in.
type options struct {
	url                   string
	certFile              string
	keyFile               string
	caFile                string
	insecureSkipVerifyTLS bool
	username              string
	password              string
	userAgent             string
	timeout               time.Duration
}

// Option allows specifying various settings configurable by the user for overriding the defaults
// used when performing Get operations with the Getter.
type Option func(*options)

// Progress -
type Progress func(dc int64)

// WithURL informs the getter the server name that will be used when fetching objects. Used in conjunction with
// WithTLSClientConfig to set the TLSClientConfig's server name.
func WithURL(url string) Option {
	return func(opts *options) {
		opts.url = url
	}
}

// WithBasicAuth sets the request's Authorization header to use the provided credentials
func WithBasicAuth(username, password string) Option {
	return func(opts *options) {
		opts.username = username
		opts.password = password
	}
}

// WithUserAgent sets the request's User-Agent header to use the provided agent name.
func WithUserAgent(userAgent string) Option {
	return func(opts *options) {
		opts.userAgent = userAgent
	}
}

// WithInsecureSkipVerifyTLS determines if a TLS Certificate will be checked
func WithInsecureSkipVerifyTLS(insecureSkipVerifyTLS bool) Option {
	return func(opts *options) {
		opts.insecureSkipVerifyTLS = insecureSkipVerifyTLS
	}
}

// WithTLSClientConfig sets the client auth with the provided credentials.
func WithTLSClientConfig(certFile, keyFile, caFile string) Option {
	return func(opts *options) {
		opts.certFile = certFile
		opts.keyFile = keyFile
		opts.caFile = caFile
	}
}

// WithRequestTimeout setup timeout for request
func WithRequestTimeout(timeout time.Duration) Option {
	return func(opts *options) {
		opts.timeout = timeout
	}
}

// Getter is an interface to support GET to the specified URL.
type Getter interface {
	// Get file content by url string
	Get(url string, out io.Writer, pgr Progress, hc hash.Hash, options ...Option) (int64, error)
}

// Constructor is the function for every getter which creates a specific instance
// according to the configuration
type Constructor func(options ...Option) (Getter, error)

// Provider represents any getter and the schemes that it supports.
//
// For example, an HTTP provider may provide one getter that handles both
// 'http' and 'https' schemes.
type Provider struct {
	Schemes []string
	New     Constructor
}

// Provides returns true if the given scheme is supported by this Provider.
func (p Provider) Provides(scheme string) bool {
	for _, i := range p.Schemes {
		if i == scheme {
			return true
		}
	}
	return false
}

// Providers is a collection of Provider objects.
type Providers []Provider

// ByScheme returns a Provider that handles the given scheme.
//
// If no provider handles this scheme, this will return an error.
func (p Providers) ByScheme(scheme string) (Getter, error) {
	for _, pp := range p {
		if pp.Provides(scheme) {
			return pp.New()
		}
	}
	return nil, errors.Errorf("scheme %q not supported", scheme)
}

var httpProvider = Provider{
	Schemes: []string{"http", "https"},
	New:     NewHTTPGetter,
}

// AllProviders -
func AllProviders() Providers {
	result := Providers{httpProvider}
	return result
}
