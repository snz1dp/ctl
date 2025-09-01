/*
Copyright The Helm Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package down

import (
	"crypto/tls"
	"hash"
	"io"
	"net/http"
	"snz1.cn/snz1dp/snz1dpctl/tlsutil"
	"snz1.cn/snz1dp/snz1dpctl/urlutil"
	"snz1.cn/snz1dp/snz1dpctl/utils"

	"github.com/pkg/errors"
)

// HTTPGetter is the default HTTP(/S) backend handler
type HTTPGetter struct {
	opts options
}

//Get performs a Get from repo.Getter and returns the body.
func (g *HTTPGetter) Get(href string, out io.Writer, pgr Progress, sm hash.Hash, options ...Option) (int64, error) {
	for _, opt := range options {
		opt(&g.opts)
	}
	return g.get(href, out, pgr, sm)
}

func (g *HTTPGetter) get(href string, out io.Writer, pgr Progress, sm hash.Hash) (int64, error) {
	//buf := bytes.NewBuffer(nil)

	// Set a helm specific user agent so that a repo server and metrics can
	// separate helm calls from other tools interacting with repos.
	req, err := http.NewRequest("GET", href, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("User-Agent", utils.UserAgent())
	if g.opts.userAgent != "" {
		req.Header.Set("User-Agent", g.opts.userAgent)
	}

	if g.opts.username != "" && g.opts.password != "" {
		req.SetBasicAuth(g.opts.username, g.opts.password)
	}

	client, err := g.httpClient()
	if err != nil {
		return 0, err
	}

	if g.opts.timeout != client.Timeout {
		client.Timeout = g.opts.timeout
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != 200 {
		return 0, errors.Errorf("failed to fetch %s : %s", href, resp.Status)
	}

	defer resp.Body.Close()

	if sm == nil {
		return io.Copy(out, resp.Body)
	}

	readbuf := make([]byte, 4096)
	var (
		rc int64 = 0
		wc int   = 0
	)
	for {
		n, err := io.ReadFull(resp.Body, readbuf)
		if n > 0 {
			wc, err = out.Write(readbuf[:n])
			if err != nil {
				return rc, err
			}
			rc += int64(wc)
			_, err = sm.Write(readbuf[:n])
			if err != nil {
				return rc, err
			}
			if pgr != nil {
				pgr(rc)
			}
		}
		if err != nil {
			if err == io.ErrUnexpectedEOF || err == io.EOF {
				return rc, nil
			}
			return rc, err
		}
	}
}

// NewHTTPGetter constructs a valid http/https client as a Getter
func NewHTTPGetter(options ...Option) (Getter, error) {
	var client HTTPGetter

	for _, opt := range options {
		opt(&client.opts)
	}

	return &client, nil
}

func (g *HTTPGetter) httpClient() (*http.Client, error) {
	transport := &http.Transport{
		DisableCompression: true,
		Proxy:              http.ProxyFromEnvironment,
	}
	if (g.opts.certFile != "" && g.opts.keyFile != "") || g.opts.caFile != "" {
		tlsConf, err := tlsutil.NewClientTLS(g.opts.certFile, g.opts.keyFile, g.opts.caFile)
		if err != nil {
			return nil, errors.Wrap(err, "can't create TLS config for client")
		}
		tlsConf.BuildNameToCertificate()

		sni, err := urlutil.ExtractHostname(g.opts.url)
		if err != nil {
			return nil, err
		}
		tlsConf.ServerName = sni

		transport.TLSClientConfig = tlsConf
	}

	if g.opts.insecureSkipVerifyTLS {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}

	}

	client := &http.Client{
		Transport: transport,
	}

	return client, nil
}
