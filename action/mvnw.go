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
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	rice "github.com/GeertJohan/go.rice"
	"snz1.cn/snz1dp/snz1dpctl/down"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

type MavenServerItem struct {
	ID       string `xml:"id"`
	URL      string `xml:"url,omitempty"`
	Username string `xml:"username,omitempty"`
	Password string `xml:"password,omitempty"`
}

type MavenServers struct {
	Items []*MavenServerItem `xml:"server"`
}

type MavenMirrorItem struct {
	ID       string `xml:"id"`
	Name     string `xml:"name,omitempty"`
	URL      string `xml:"url,omitempty"`
	MirrorOf string `xml:"mirrorOf,omitempty"`
}

type MavenMirrors struct {
	Items []*MavenMirrorItem `xml:"mirror"`
}

type MavenSettings struct {
	XMLName xml.Name      `xml:"settings"`
	Servers *MavenServers `xml:"servers,omitempty"`
	Mirrors *MavenMirrors `xml:"mirrors,omitempty"`
}

// CreateMavenWrapper -
type CreateMavenWrapper struct {
	BaseAction
	Force bool
}

// NewCreateMavenWrapper -
func NewCreateMavenWrapper(setting *GlobalSetting) *CreateMavenWrapper {
	return &CreateMavenWrapper{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (c *CreateMavenWrapper) Run() (err error) {
	var (
		setting       *GlobalSetting = c.GlobalSetting()
		spinner       *utils.WaitSpinner
		furl          *url.URL
		mvnwFileName  string
		mvnwFilepath  string
		mvnwURL       string
		curdir        string
		tmpFilePath   string
		mvnwFiles     []string
		bundleDir     string = setting.GetBundleDir()
		force, dodown bool   = c.Force, false
		fst           os.FileInfo
	)

	_, _, err = setting.LoadLocalInstallConfiguration()
	if err != nil {
		c.ErrorExit("%s", err)
	}

	curdir, err = os.Getwd()
	if err != nil {
		c.ErrorExit("%s", err)
		return
	}

	mvnwURL = c.GlobalSetting().ResolveDownloadURL(BaseConfig.MavenWrap.URL)
	if furl, err = url.Parse(mvnwURL); err != nil {
		c.ErrorExit("mvnw url %s error: %s", mvnwURL, err)
		return
	}

	mvnwFileName = filepath.Base(furl.Path)
	mvnwFilepath = path.Join(bundleDir, mvnwFileName)

	if force {
		dodown = true
	} else {
		if fst, err = os.Stat(mvnwFilepath); err == nil && !fst.IsDir() {
			sumData, err := utils.FileChecksum(mvnwFilepath, sha256.New())
			fcksum := hex.EncodeToString(sumData)
			dodown = err != nil || down.VerifyBundle(mvnwFilepath+".sha256", fcksum) != nil
		} else {
			if fst != nil {
				os.RemoveAll(mvnwFilepath)
			}
			dodown = true
		}
	}

	if dodown {

		spinner = utils.NewSpinner(fmt.Sprintf("download %s...", furl.String()), setting.OutOrStdout())
		mvnwFilepath, err = down.NewBundleDownloader(setting.OutOrStdout(), furl.String(), down.VerifyAlways).Download(
			setting.GetBundleDir(), mvnwFileName)

		spinner.Close()
		if err != nil {
			c.ErrorExit("failed: %s", err.Error())
			return
		}
		c.Println("ok!")
	}

	tmpFilePath = strings.TrimSuffix(mvnwFilepath, ".tgz")
	os.RemoveAll(tmpFilePath)
	defer os.RemoveAll(tmpFilePath)

	if err = UnarchiveBundle(mvnwFilepath, setting.GetBundleDir()); err != nil {
		c.ErrorExit("maven wrapper bundle eror: %s", err.Error())
		return
	}

	if mvnwFiles, err = utils.FindDirFiles(tmpFilePath, utils.FindFileOption{Subdir: true}); err != nil {
		c.ErrorExit("maven wrapper bundle eror: %s", err.Error())
		return
	}

	for _, v := range mvnwFiles {
		if _, err = os.Stat(v); err != nil {
			continue
		}

		var (
			tmpfile string = strings.TrimPrefix(v, tmpFilePath+"/")
			tofile  string = path.Join(curdir, tmpfile)
		)

		c.Println("%s --> %s", tmpfile, tofile)

		os.MkdirAll(filepath.Dir(tofile), os.ModePerm)
		err = utils.CopyFile(v, tofile)

		if err != nil {
			c.ErrorExit("%s", err)
			return
		}

	}

	wrapperConfigFile := path.Join(curdir, ".mvn", "wrapper", "maven-wrapper.properties")
	distributionUrl := setting.ResolveDownloadURL(BaseConfig.MavenWrap.DistributionURL)
	wrapperUrl := setting.ResolveDownloadURL(BaseConfig.MavenWrap.WrapperURL)

	tempConfig := fmt.Sprintf("distributionUrl=%s\nwrapperUrl=%s", distributionUrl, wrapperUrl)
	if err = os.WriteFile(wrapperConfigFile, []byte(tempConfig), 0664); err != nil {
		c.ErrorExit("%s", err)
	}

	return
}

// CreateJavaBackend -
type CreateJavaBackend struct {
	BaseAction
	Group   string
	Name    string
	Package string
	Version string
	Force   bool
}

// NewCreateJavaBackend -
func NewCreateJavaBackend(setting *GlobalSetting) *CreateJavaBackend {
	return &CreateJavaBackend{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (c *CreateJavaBackend) Run() (err error) {

	var (
		setting          *GlobalSetting = c.GlobalSetting()
		ic               *InstallConfiguration
		spinner          *utils.WaitSpinner
		furl             *url.URL
		jarFileName      string
		jarFilepath      string
		bundleDir        string = setting.GetBundleDir()
		force, dodown    bool   = c.Force, false
		fst              os.FileInfo
		failedRemove     bool   = true
		archetypeURL     string = BaseConfig.Archetype.URL
		mavenProjectInfo MavenProjectInfo
	)

	if _, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		c.ErrorExit("%s", err)
		return
	}

	confBox, err := rice.FindBox("../asset/java")
	if err != nil {
		panic(err)
	}

	archetypeURL = setting.ResolveDownloadURL(BaseConfig.Archetype.URL)

	if furl, err = url.Parse(archetypeURL); err != nil {
		c.ErrorExit("java archetype url %s error: %s", archetypeURL, err)
		return
	}

	// 下载组件
	if _, err := os.Stat(bundleDir); err != nil {
		os.MkdirAll(bundleDir, os.ModePerm)
	}

	jarFileName = filepath.Base(furl.Path)

	jarFilepath = path.Join(bundleDir, jarFileName)

	if force {
		dodown = true
	} else {
		if fst, err = os.Stat(jarFilepath); err == nil && !fst.IsDir() {
			sumData, err := utils.FileChecksum(jarFilepath, sha256.New())
			fcksum := hex.EncodeToString(sumData)
			dodown = err != nil || down.VerifyBundle(jarFilepath+".sha256", fcksum) != nil
		} else {
			if fst != nil {
				os.RemoveAll(jarFilepath)
			}
			dodown = true
		}
	}

	if dodown {
		spinner = utils.NewSpinner(fmt.Sprintf("download %s...", furl.String()), setting.OutOrStdout())
		jarFilepath, err = down.NewBundleDownloader(setting.OutOrStdout(), furl.String(), down.VerifyAlways).Download(
			setting.GetBundleDir(), jarFileName)

		spinner.Close()
		if err != nil {
			c.ErrorExit("failed: %s", err.Error())
			return
		}
		c.Println("ok!")
	}

	var (
		curdir string
		prjdir string
	)

	curdir, err = os.Getwd()
	if err != nil {
		c.ErrorExit("%s", err)
		return
	}

	prjdir = path.Join(curdir, c.Name)

	if _, err = os.Stat(prjdir); err == nil {
		if !utils.Confirm("project directory "+prjdir+" existed, Proceed? (y/N)", setting.InOrStdin(), setting.OutOrStdout()) {
			c.Println("Cancelled.")
			return
		}
		failedRemove = false
	}

	if err = os.MkdirAll(prjdir, os.ModePerm); err != nil {
		c.ErrorExit("create directory %s error: %s", prjdir, err)
		return
	}

	if err = os.Chdir(prjdir); err != nil {
		c.ErrorExit("cd %s error: %s", prjdir, err)
		return
	}

	crd := NewCreateMavenWrapper(c.setting)
	crd.Force = c.Force

	if err = crd.Run(); err != nil {
		c.ErrorExit("create maven wrapper error: %s", err)
		return
	}

	var (
		m2dir    string
		commands []string
		execCmd  *exec.Cmd
		cmdpath  string
		tmpdir   string
		tmpFiles []string
		mvncmd   string
	)

	m2dir, _ = utils.ExpandUserDir("~/.m2")
	os.MkdirAll(m2dir, os.ModePerm)
	if err = os.WriteFile(path.Join(m2dir, "archetype-catalog.xml"), ArchetypeCatalog, 0644); err != nil {
		os.Chdir(curdir)
		os.RemoveAll(prjdir)
		c.ErrorExit("config maven error: %s", err)
		return
	}

	if runtime.GOOS == "windows" {
		mvncmd = path.Join(prjdir, "mvnw.cmd")
	} else {
		mvncmd = path.Join(prjdir, "mvnw")
	}

	// 先下载依赖
	commands = []string{
		mvncmd,
		"install:install-file",
		"-DgroupId=" + BaseConfig.Archetype.Group,
		"-DartifactId=" + BaseConfig.Archetype.Artifact,
		"-Dversion=" + BaseConfig.Archetype.Version,
		"-Dpackaging=jar",
		"-Dfile=" + jarFilepath,
		"-DrepositoryId=" + ic.Snz1dp.MavenRepo.ID,
		"-Durl=" + ic.Snz1dp.MavenRepo.URL,
	}

	cmdpath, commands = parseCommand(commands)
	execCmd = exec.CommandContext(context.Background(), cmdpath, commands...)
	execCmd.Stdin = setting.InOrStdin()
	execCmd.Stdout = setting.OutOrStdout()
	execCmd.Stderr = setting.ErrOrStderr()

	if err = execCmd.Run(); err != nil {
		os.Chdir(curdir)
		if failedRemove {
			os.RemoveAll(prjdir)
		}
		c.ErrorExit("%s", err)
		return
	}

	// 创建工程
	commands = []string{}
	commands = append(commands, mvncmd)
	commands = append(commands, "archetype:generate")
	commands = append(commands, "-DarchetypeGroupId="+BaseConfig.Archetype.Group)
	commands = append(commands, "-DarchetypeArtifactId="+BaseConfig.Archetype.Artifact)
	commands = append(commands, "-DarchetypeVersion="+BaseConfig.Archetype.Version)
	commands = append(commands, "-DarchetypeCatalog=local")
	if c.Group != "" {
		commands = append(commands, fmt.Sprintf("-DgroupId=%s", c.Group))
	}
	if c.Name != "" {
		commands = append(commands, fmt.Sprintf("-DartifactId=%s", c.Name))
	}
	if c.Package != "" {
		commands = append(commands, fmt.Sprintf("-Dpackage=%s", c.Package))
	}
	if c.Version != "" {
		commands = append(commands, fmt.Sprintf("-Dversion=%s", c.Version))
	}
	if c.Group != "" {
		commands = append(commands, "-B")
	}

	cmdpath, commands = parseCommand(commands)
	execCmd = exec.CommandContext(context.Background(), cmdpath, commands...)
	execCmd.Stdin = setting.InOrStdin()
	execCmd.Stdout = setting.OutOrStdout()
	execCmd.Stderr = setting.ErrOrStderr()

	if err = execCmd.Run(); err != nil {
		os.Chdir(curdir)
		if failedRemove {
			os.RemoveAll(prjdir)
		}
		c.ErrorExit("%s", err)
		return
	}

	tmpdir = path.Join(prjdir, c.Name)

	if tmpFiles, err = utils.FindDirFiles(tmpdir, utils.FindFileOption{Subdir: true}); err != nil {
		os.Chdir(curdir)
		if failedRemove {
			os.RemoveAll(prjdir)
		}
		c.ErrorExit("read file error: %s", err.Error())
		return
	}

	for _, v := range tmpFiles {
		if _, err = os.Stat(v); err != nil {
			continue
		}

		var (
			tmpfile string = strings.TrimPrefix(v, tmpdir+"/")
			tofile  string = path.Join(prjdir, tmpfile)
		)

		os.MkdirAll(filepath.Dir(tofile), os.ModePerm)
		err = utils.CopyFile(v, tofile)

		if err != nil {
			os.Chdir(curdir)
			if failedRemove {
				os.RemoveAll(prjdir)
			}
			c.ErrorExit("copy %s", v, err)
			return
		}

	}

	var (
		tpl    *template.Template
		bDatas []byte
	)

	mavenProjectInfo.Group = c.Group
	mavenProjectInfo.Package = c.Package
	mavenProjectInfo.Name = c.Name
	mavenProjectInfo.Version = c.Version
	mavenProjectInfo.MavenID = ic.Snz1dp.MavenRepo.ID
	mavenProjectInfo.MavenURL = ic.Snz1dp.MavenRepo.URL
	mavenProjectInfo.ServerURL = ic.Snz1dp.Server.URL
	mavenProjectInfo.DownloadURL = ic.Snz1dp.Server.DownloadPrefix

	if mavenProjectInfo.ServerURL == BaseConfig.Snz1dp.Server.URL {
		mavenProjectInfo.ServerURL = ""
	}

	for _, templateFile := range []string{
		"BUILD.yaml", "pom.xml", "VERSION", "Makefile", "Dockerfile",
		"Jenkinsfile", ".workflow/build.sh", ".workflow/publish.sh", ".workflow/deploy.sh",
	} {
		bDatas, _ = confBox.Bytes(templateFile)
		if tpl, err = template.New("render").Parse(string(bDatas)); err != nil {
			os.Chdir(curdir)
			if failedRemove {
				os.RemoveAll(prjdir)
			}
			c.ErrorExit("load %s template error: %s", templateFile, err)
			return
		}

		var tbuf bytes.Buffer
		if err = tpl.Execute(&tbuf, &mavenProjectInfo); err != nil {
			os.Chdir(curdir)
			if failedRemove {
				os.RemoveAll(prjdir)
			}
			c.ErrorExit("render %s template error: %s", templateFile, err)
			return
		}

		os.WriteFile(path.Join(prjdir, templateFile),
			[]byte(strings.ReplaceAll(strings.ReplaceAll(tbuf.String(), "[[{", "{{"), "}]]", "}}")),
			os.ModePerm)
	}

	os.RemoveAll(tmpdir)

	c.Println("success create %s.", c.Name)

	return
}

type MavenProjectInfo struct {
	Name        string
	Version     string
	Group       string
	Package     string
	DockerHub   string
	MavenID     string
	MavenURL    string
	ServerURL   string
	DownloadURL string
}
