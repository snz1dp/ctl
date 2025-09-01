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
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/ghodss/yaml"
	"github.com/howeyc/gopass"
	"github.com/pkg/errors"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

// ProjectType -
type ProjectType string

// DockerBuildConfig -
type DockerBuildConfig struct {
	Image           string    `json:"image"`
	Tag             string    `json:"tag"`
	Dockerfile      string    `json:"file"`
	BaseDirectory   string    `json:"base,omitempty"`
	Depends         *[]string `json:"depends"`
	currentPlatform string

	Platform []string `json:"platform,omitempty"`

	Args    map[string]*string `json:"args"`
	Labels  map[string]string  `json:"labels"`
	SSHKeys []string           `json:"sshkey,omitempty"`
}

func (c *DockerBuildConfig) ValidatePlatform(platform string) (err error) {
	if platform == "" {
		return
	}

	for _, v := range c.Platform {
		if v == platform {
			return
		}
	}
	err = errors.Errorf("not found %s", platform)
	return
}

func (c *DockerBuildConfig) GetPlatform() (platform string) {
	if c.currentPlatform != "" {
		return c.currentPlatform
	}
	return runtime.GOOS
}

type DockerDepend struct {
	Source string
	Target string
}

func (c *DockerBuildConfig) GetDepends() (depends []DockerDepend) {
	for _, v := range *c.Depends {
		depend_images := strings.Split(v, "->")
		depend := DockerDepend{
			Source: strings.TrimSpace(depend_images[0]),
		}
		if len(depend_images) > 1 {
			depend.Target = strings.TrimSpace(depend_images[1])
		} else {
			depend.Target = strings.TrimSpace(depend_images[0])
		}
		depends = append(depends, depend)
	}
	return
}

// ProjectSonarConfig - 配置
type ProjectSonarConfig struct {
	URL      string    `json:"url,omitempty"`
	Token    string    `json:"token,omitempty"`
	Type     string    `json:"type,omitempty"`
	Commands *[]string `json:"commands,omitempty"`
}

func getMavenExecute() string {
	return getCommandExecute("mvnw")
}

func getCommandExecute(cmdName string) string {
	switch runtime.GOOS {
	case "windows":
		return cmdName
	default:
		cmdName = "./" + cmdName
	}
	return cmdName
}

// HelmPushConfig
type HelmPushConfig struct {
	RepoName string `json:"repo,omitempty"`
	RepoUser string `json:"user,omitempty"`
	RepoPwd  string `json:"-"`
}

// ProjectCommand -
type ProjectCommand struct {
	InitCommand    []string            `json:"init"`
	BuildCommand   []string            `json:"build"`
	TestCommand    []string            `json:"test"`
	RunCommand     []string            `json:"run"`
	CleanCommand   []string            `json:"clean"`
	PublishCommand []string            `json:"publish"`
	HelmPushConfig *HelmPushConfig     `json:"helmpush"`
	Sonar          *ProjectSonarConfig `json:"sonar"`
	DockerContent  string              `json:"-"`
}

// GetProjectSonarCommands -
func GetProjectSonarCommands(setting *GlobalSetting, sonarType string) (commands []string) {
	sonarType = strings.ToLower(sonarType)
	switch sonarType {
	case "maven":
		commands = []string{
			getMavenExecute(),
			"sonar:sonar",
			"-Dsonar.projectKey={{ .Name }}",
			"-Dsonar.host.url={{ .Sonar.URL }}",
			"-Dsonar.login={{ .Sonar.Token }}",
		}
	case "gradle":
		commands = []string{
			getCommandExecute("gradlew"),
			"sonarqube",
			"-Dsonar.projectKey={{ .Name }}",
			"-Dsonar.host.url={{ .Sonar.URL }}",
			"-Dsonar.login={{ .Sonar.Token }}",
		}
	default:
		commands = []string{
			path.Join(setting.GetBinDir(), fmt.Sprintf("%s-%s-%s",
				BaseConfig.Sonar.Name, BaseConfig.Sonar.Version, runtime.GOOS),
				"bin", "sonar-scanner",
			),
			"-Dsonar.projectKey={{ .Name }}",
			"-Dsonar.sources=.",
			"-Dsonar.host.url={{ .Sonar.URL }}",
			"-Dsonar.login={{ .Sonar.Token }}",
		}
	}
	return

}

// ProjectDevelopConfig -
type ProjectDevelopConfig struct {
	Configuration string           `json:"config,omitempty"`
	Jwt           *JwtSecretConfig `json:"jwt,omitempty"`
}

// ProjectBuildConfig -
type ProjectBuildConfig struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Catelog     string                `json:"catelog"`
	Version     string                `json:"version"`
	Docker      *DockerBuildConfig    `json:"docker"`
	Type        ProjectType           `json:"type"`
	Service     *StandaloneConfig     `json:"service"`
	Develop     *ProjectDevelopConfig `json:"develop,omitempty"`

	Follow     bool
	Since      string
	Tail       string
	Timestamps bool
	Details    bool
	Until      string
	Really     bool

	ForcePullImage  bool
	LoadImageLocal  bool
	SkipBuildDocker bool
	SkipBuildSource bool
	SkipPackage     bool
	SaveDockerImage bool

	action  Action
	basedir string
	ProjectCommand
}

var (
	// MavenProject -
	MavenProject ProjectType = "maven"
	// QuasarProject -
	QuasarProject ProjectType = "quasar"
	// VueProject -
	VueProject ProjectType = "vue"

	// ErrorNotSupportProjectType -
	ErrorNotSupportProjectType error = errors.Errorf("not support project type")
	// ErrorNotFoundBuildConfigFile -
	ErrorNotFoundBuildConfigFile error = errors.Errorf("not found build config file: BUILD.yaml")

	// DefaultProjectCommand -
	DefaultProjectCommand map[ProjectType]*ProjectCommand = map[ProjectType]*ProjectCommand{
		MavenProject: {
			InitCommand: []string{},
			BuildCommand: []string{
				getMavenExecute(),
				"-DskipTests",
				"package",
			},
			TestCommand: []string{
				getMavenExecute(),
				"test",
			},
			RunCommand: []string{
				getMavenExecute(),
				"-DskipTests",
				"spring-boot:run",
			},
			CleanCommand: []string{
				getMavenExecute(),
				"clean",
			},
			DockerContent: `# 引入openjdk8镜像
FROM snz1.cn/dp/openjdk8-springboot-app:2.0

# 复制打包好的jar文件到/app目录
COPY target/name-of-target.jar /app/app.jar
`,
		},
		QuasarProject: {
			InitCommand: []string{
				"npm",
				"install",
			},
			BuildCommand: []string{
				"quasar",
				"build",
			},
			TestCommand: []string{
				"quasar",
				"test",
			},
			RunCommand: []string{
				"quasar",
				"run",
			},
			CleanCommand: []string{
				"quasar",
				"clean",
			},
			DockerContent: `# 引入Vue应用标准镜像
FROM snz1.cn/dp/vueapp:2.0

ADD dist /app/html
`,
		},
		VueProject: {
			InitCommand: []string{
				"npm",
				"install",
			},
			BuildCommand: []string{
				"npm",
				"run",
				"build",
			},
			TestCommand: []string{
				"npm",
				"run",
				"test",
			},
			RunCommand: []string{
				"npm",
				"run",
				"dev",
			},
			CleanCommand: []string{
				"npm",
				"cache",
				"clean",
				"--force",
			},
			DockerContent: `# 引入Vue应用标准镜像
FROM snz1.cn/dp/vueapp:2.0

ADD dist /app/html
`,
		},
	}
)

// LoadProjectBuild -
func LoadProjectBuild(pcdata []byte) (pbc *ProjectBuildConfig, err error) {
	pbc = &ProjectBuildConfig{}
	err = yaml.Unmarshal(pcdata, pbc)
	if err != nil {
		pbc = nil
		return
	}
	return
}

// LoadProjectBuildFromDirectory -
func LoadProjectBuildFromDirectory(projectdir string) (pbc *ProjectBuildConfig, err error) {
	var (
		bdfile       string
		pbcfst       os.FileInfo
		pcdata       []byte
		backendports map[uint64]bool = make(map[uint64]bool)
	)
	bdfile = path.Join(projectdir, "BUILD.yaml")
	pbcfst, err = os.Stat(bdfile)
	if err != nil || pbcfst.IsDir() {
		bdfile = path.Join(projectdir, "BUILD.yml")
		pbcfst, err = os.Stat(bdfile)
		if err != nil || pbcfst.IsDir() {
			err = ErrorNotFoundBuildConfigFile
			return
		}
	}
	pcdata, err = os.ReadFile(bdfile)
	if err != nil {
		err = errors.Errorf("read %s error:\n%v", pbcfst.Name(), err)
		return
	}

	pbc, err = LoadProjectBuild(pcdata)
	if err != nil {
		err = errors.Errorf("%s format error:\n%v", pbcfst.Name(), err)
		return
	}
	pbc.basedir = projectdir

	if pbc.Name == "" {
		pbc.Name = filepath.Base(projectdir)
	}

	if pbc.Catelog == "" {
		pbc.Catelog = "application"
	}

	if pbc.Description == "" {
		pbc.Description = pbc.Name + ", snz1dp cloude native application"
	}

	if len(pbc.Name) > 32 {
		err = errors.Errorf("name '%s' length too long", pbc.Name)
		return
	}

	if pbc.Service != nil {
		for _, v := range pbc.Service.Ports {
			var (
				vst         int = strings.Index(v, ":")
				backendport string
				portval     uint64
				vtp         int
			)
			if vst < 0 {
				backendport = v
			} else {
				backendport = v[vst+1:]
			}

			vtp = strings.Index(backendport, "/")
			if vtp > 0 {
				backendport = backendport[:vtp]
			}

			portval, err = strconv.ParseUint(backendport, 10, 32)
			if err != nil {
				err = errors.Errorf("error service ports: %s", v)
				return
			}
			backendports[portval] = true
		}

		for _, v := range pbc.Service.Ingress {
			if v.BackendPort == 0 {
				v.BackendPort = 80
			}
		}

		if pbc.Service.HealthCheck != nil {
			_, err = time.ParseDuration(pbc.Service.HealthCheck.Interval)
			if err != nil {
				err = errors.Errorf("error healtch check interval: %s", pbc.Service.HealthCheck.Interval)
				return
			}

			_, err = time.ParseDuration(pbc.Service.HealthCheck.Timeout)
			if err != nil {
				err = errors.Errorf("error healtch check timeout: %s", pbc.Service.HealthCheck.Timeout)
				return
			}

			_, err = time.ParseDuration(pbc.Service.HealthCheck.StartPeriod)
			if err != nil {
				err = errors.Errorf("error healtch check start period: %s", pbc.Service.HealthCheck.StartPeriod)
				return
			}
		}

	}

	if pbc.Docker == nil {
		err = errors.Errorf("not found docker config")
	}

	pbc.GetVersion()

	return
}

func parseCommand(cmdargs []string) (cmdpath string, outargs []string) {
	if len(cmdargs) == 0 {
		return
	}

	if len(cmdargs) > 1 {
		cmdpath = cmdargs[0]
		outargs = cmdargs[1:]
	} else {
		cmdpath = cmdargs[0]
	}

	return
}

// RenderStrings 模板
func (p *ProjectBuildConfig) RenderStrings(ss ...string) (ret []string, err error) {

	var (
		tpl *template.Template
	)

	for _, v := range ss {
		if tpl, err = template.New("render").Parse(v); err != nil {
			return
		}
		var tbuf *bytes.Buffer = bytes.NewBuffer(nil)
		if err = tpl.Execute(tbuf, p); err != nil {
			return
		}
		ret = append(ret, tbuf.String())
	}
	return
}

// Clean -
func (p *ProjectBuildConfig) Clean() (err error) {
	var (
		buildcmd *exec.Cmd
		cmdpath  string
		setting  *GlobalSetting = p.action.GlobalSetting()
		cmdargs  []string
	)

	if len(p.CleanCommand) == 0 {
		var pcmd *ProjectCommand = DefaultProjectCommand[p.Type]
		if pcmd == nil || len(pcmd.CleanCommand) == 0 {
			return
		}
		cmdargs = pcmd.CleanCommand
	} else {
		cmdargs = p.CleanCommand
	}

	if cmdargs, err = p.RenderStrings(cmdargs...); err != nil {
		p.action.ErrorExit("error clean command in BUILD.yaml: %s", err)
	}

	p.action.Println("%s", strings.Join(cmdargs, " "))

	cmdpath, cmdargs = parseCommand(cmdargs)

	switch runtime.GOOS {
	case "windows":
		if strings.Contains(cmdpath, "/") {
			cmdpath = strings.ReplaceAll(cmdpath, "/", "\\")
		}
	default:
		if strings.Contains(cmdpath, "\\") {
			cmdpath = strings.ReplaceAll(cmdpath, "\\", "/")
		}
	}

	if cmdpath == "npm" {
		cmdPaths, err := resolveNpmPaths(p.action)
		if err != nil {
			p.action.ErrorExit("error run command in BUILD.yaml: %s", err)
		}
		cmdpath = cmdPaths[0]
		cmdargs = append(cmdPaths[1:], cmdargs...)
	}

	buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
	buildcmd.Stdin = setting.InOrStdin()
	buildcmd.Stdout = setting.OutOrStdout()
	buildcmd.Stderr = setting.ErrOrStderr()

	if err = buildcmd.Run(); err != nil {
		p.action.ErrorExit("%s", err)
	}

	return
}

// Publish -
func (p *ProjectBuildConfig) Publish() (err error) {

	// 先执行打包
	if !p.SkipPackage {
		p.Package(true)
	}

	// 推送镜像
	var (
		setting *GlobalSetting = p.action.GlobalSetting()
		ic      *InstallConfiguration
	)

	// 执行命令
	var (
		buildcmd     *exec.Cmd
		cmdpath      string
		cmdargs      []string
		repoName     string
		repoUser     string
		repoLoginPwd string
	)

	// 加载本地配置
	_, ic, err = setting.LoadLocalInstallConfiguration()
	if err != nil {
		return
	}

	if len(p.PublishCommand) == 0 {
		if p.HelmPushConfig != nil {
			repoName = p.HelmPushConfig.RepoName
			repoUser = p.HelmPushConfig.RepoUser
			repoLoginPwd = p.HelmPushConfig.RepoPwd

			if repoName == "" || repoName == ic.Snz1dp.HelmRepo.Name {
				repoName = ic.Snz1dp.HelmRepo.Name

				if repoUser == "" {
					repoUser = ic.Snz1dp.HelmRepo.Username
				}
				if repoLoginPwd == "" {
					repoLoginPwd = ic.Snz1dp.HelmRepo.Password
				}
			}

			if repoUser == "" {
				for {
					fmt.Fprintf(setting.OutOrStdout(), "helm repo %s user: ", repoName)
					_, err = fmt.Fscanf(setting.InOrStdin(), "%s", &repoUser)
					repoUser = strings.TrimSpace(repoUser)
					if err != nil || repoUser == "" {
						continue
					}
					break
				}
			}

			if repoLoginPwd == "" {
				for {
					var repoPwd []byte
					repoPwd, err = gopass.GetPasswdPrompt(
						fmt.Sprintf("helm user %s password: ", repoUser),
						true, setting.InOrStdin().(gopass.FdReader),
						setting.OutOrStdout(),
					)
					if err != nil || len(repoPwd) == 0 {
						continue
					}
					repoLoginPwd = string(repoPwd)
					break
				}
			}

			var (
				helmRegistry *HelmRegistry
			)

			if helmRegistry = ic.GetHelmRegistryByName(repoName); helmRegistry == nil {
				if repoName == ic.Snz1dp.HelmRepo.Name {
					helmRegistry = &HelmRegistry{
						Name:     repoName,
						URL:      ic.Snz1dp.HelmRepo.URL,
						Username: repoUser,
						Password: repoLoginPwd,
					}
					helmRegistry.sysconfig = false
				} else {
					p.action.ErrorExit("helm repo %s not existed, please add it before publish", repoName)
				}
			} else {
				var oregistry *HelmRegistry = new(HelmRegistry)
				*oregistry = *helmRegistry
				helmRegistry, oregistry = oregistry, helmRegistry
			}

			var spinner *utils.WaitSpinner = utils.NewSpinner(
				fmt.Sprintf("push %s to %s(%s)...",
					fmt.Sprintf("out/%s-%s.tgz", p.Name, p.GetVersion()),
					repoName, helmRegistry.URL,
				), setting.OutOrStdout())

			var helmBinPath string
			if helmBinPath, err = resolveHelmBinPath(p.action, ic); err != nil {
				p.action.ErrorExit("%s", err)
			}

			if !helmRegistry.sysconfig {
				helmRepoAdd := exec.CommandContext(
					context.Background(),
					helmBinPath,
					"repo", "add",
					helmRegistry.Name,
					helmRegistry.URL,
					"--username",
					repoUser,
					"--password",
					repoLoginPwd,
				)
				var testBuffer bytes.Buffer
				helmRepoAdd.Stdout = &testBuffer
				if err = helmRepoAdd.Run(); err != nil {
					p.action.ErrorExit("helm repo add error: %s", err)
				}
			}

			// 写入nexus-push插件所需的密钥
			var repoPushConfigPath = path.Join(utils.GetUserHomeDir(), ".config", "helm")
			if err = os.MkdirAll(repoPushConfigPath, os.ModePerm); err != nil {
				return
			}

			var repoPushAuthFile = path.Join(repoPushConfigPath, fmt.Sprintf("auth.%s", helmRegistry.Name))
			os.WriteFile(repoPushAuthFile, []byte(fmt.Sprintf("%s:%s", repoUser, repoLoginPwd)), 0644)

			cmdargs = []string{
				helmBinPath,
				BaseConfig.Snz1dp.Helm.PushPlugin,
				repoName,
				fmt.Sprintf("out/%s-%s.tgz", p.Name, p.GetVersion()),
			}

			var helmStdoutBuffer bytes.Buffer
			cmdpath, cmdargs = parseCommand(cmdargs)
			buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
			buildcmd.Stdin = setting.InOrStdin()
			buildcmd.Stdout = &helmStdoutBuffer

			if err = buildcmd.Run(); err != nil {
				spinner.Close()
				p.action.Println("failed.")
				p.action.ErrorExit("\n%v", err)
			}
			spinner.Close()

			var helmStdout = helmStdoutBuffer.String()
			if !strings.Contains(helmStdout, "200 OK") && !strings.Contains(helmStdout, "HTTP/2 200") {
				p.action.Println("failed.")
				p.action.ErrorExit("\n%s", helmStdout)
			}
			p.action.Println("ok!")
			return
		} else {
			var pcmd *ProjectCommand = DefaultProjectCommand[p.Type]
			if pcmd == nil || len(pcmd.PublishCommand) == 0 {
				return
			}
			cmdargs = pcmd.PublishCommand

			if cmdargs, err = p.RenderStrings(cmdargs...); err != nil {
				p.action.ErrorExit("error publish command in BUILD.yaml: %s", err)
				return
			}

			cmdpath, cmdargs = parseCommand(cmdargs)
			buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
			buildcmd.Stdin = setting.InOrStdin()
			buildcmd.Stdout = setting.OutOrStdout()
			buildcmd.Stderr = setting.ErrOrStderr()

			if err = buildcmd.Run(); err != nil {
				p.action.ErrorExit("%v", err)
			}
			return
		}
	} else {
		cmdargs = p.PublishCommand

		if cmdargs, err = p.RenderStrings(cmdargs...); err != nil {
			p.action.ErrorExit("error publish command in BUILD.yaml: %s", err)
			return
		}

		cmdpath, cmdargs = parseCommand(cmdargs)
		buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
		buildcmd.Stdin = setting.InOrStdin()
		buildcmd.Stdout = setting.OutOrStdout()
		buildcmd.Stderr = setting.ErrOrStderr()

		if err = buildcmd.Run(); err != nil {
			p.action.ErrorExit("%v", err)
		}
		return
	}
}

// Build -
func (p *ProjectBuildConfig) Init() (err error) {
	var (
		buildcmd *exec.Cmd
		cmdpath  string
		setting  *GlobalSetting = p.action.GlobalSetting()
		cmdargs  []string
	)

	if len(p.InitCommand) == 0 {
		var pcmd *ProjectCommand = DefaultProjectCommand[p.Type]
		if pcmd == nil || len(pcmd.BuildCommand) == 0 {
			return
		}
		cmdargs = pcmd.InitCommand
	} else {
		cmdargs = p.InitCommand
	}

	if len(cmdargs) == 0 {
		return
	}

	if cmdargs, err = p.RenderStrings(cmdargs...); err != nil {
		p.action.ErrorExit("error init command in BUILD.yaml: %s", err)
	}

	p.action.Println("%s", strings.Join(cmdargs, " "))

	cmdpath, cmdargs = parseCommand(cmdargs)
	if cmdpath == "npm" {
		cmdPaths, err := resolveNpmPaths(p.action)
		if err != nil {
			p.action.ErrorExit("error run command in BUILD.yaml: %s", err)
		}
		cmdpath = cmdPaths[0]
		cmdargs = append(cmdPaths[1:], cmdargs...)
	}

	buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
	buildcmd.Stdin = setting.InOrStdin()
	buildcmd.Stdout = setting.OutOrStdout()
	buildcmd.Stderr = setting.ErrOrStderr()

	if err = buildcmd.Run(); err != nil {
		p.action.ErrorExit("%v", err)
	}

	return
}

// Build -
func (p *ProjectBuildConfig) Build() (err error) {

	var (
		buildcmd *exec.Cmd
		cmdpath  string
		setting  *GlobalSetting = p.action.GlobalSetting()
		cmdargs  []string
	)

	if len(p.BuildCommand) == 0 {
		var pcmd *ProjectCommand = DefaultProjectCommand[p.Type]
		if pcmd == nil || len(pcmd.BuildCommand) == 0 {
			return
		}
		cmdargs = pcmd.BuildCommand
	} else {
		cmdargs = p.BuildCommand
	}

	if cmdargs, err = p.RenderStrings(cmdargs...); err != nil {
		p.action.ErrorExit("error build command in BUILD.yaml: %s", err)
	}

	p.action.Println("%s", strings.Join(cmdargs, " "))

	cmdpath, cmdargs = parseCommand(cmdargs)
	if cmdpath == "npm" {
		cmdPaths, err := resolveNpmPaths(p.action)
		if err != nil {
			p.action.ErrorExit("error run command in BUILD.yaml: %s", err)
		}
		cmdpath = cmdPaths[0]
		cmdargs = append(cmdPaths[1:], cmdargs...)
	}

	buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
	buildcmd.Stdin = setting.InOrStdin()
	buildcmd.Stdout = setting.OutOrStdout()
	buildcmd.Stderr = setting.ErrOrStderr()

	if err = buildcmd.Run(); err != nil {
		p.action.ErrorExit("%v", err)
	}

	return
}

// Test -
func (p *ProjectBuildConfig) Test() (err error) {
	var (
		testcmd *exec.Cmd
		cmdpath string
		setting *GlobalSetting = p.action.GlobalSetting()
		cmdargs []string
	)

	if len(p.TestCommand) == 0 {
		var pcmd *ProjectCommand = DefaultProjectCommand[p.Type]
		if pcmd == nil || len(pcmd.TestCommand) == 0 {
			return
		}
		cmdargs = pcmd.TestCommand
	} else {
		cmdargs = p.TestCommand
	}

	if cmdargs, err = p.RenderStrings(cmdargs...); err != nil {
		p.action.ErrorExit("error test command in BUILD.yaml: %s", err)
	}

	p.action.Println("%s", strings.Join(cmdargs, " "))

	cmdpath, cmdargs = parseCommand(cmdargs)
	testcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
	testcmd.Stdin = setting.InOrStdin()
	testcmd.Stdout = setting.OutOrStdout()
	testcmd.Stderr = setting.ErrOrStderr()

	if err = testcmd.Run(); err != nil {
		p.action.ErrorExit("%v", err)
	}
	return
}

// Run -
func (p *ProjectBuildConfig) Run() (err error) {
	var (
		runcmd  *exec.Cmd
		cmdpath string
		setting *GlobalSetting = p.action.GlobalSetting()
		cmdargs []string
	)

	if len(p.RunCommand) == 0 {
		var pcmd *ProjectCommand = DefaultProjectCommand[p.Type]
		if pcmd == nil || len(pcmd.RunCommand) == 0 {
			return
		}
		cmdargs = pcmd.RunCommand
	} else {
		cmdargs = p.RunCommand
	}

	if cmdargs, err = p.RenderStrings(cmdargs...); err != nil {
		p.action.ErrorExit("error run command in BUILD.yaml: %s", err)
	}

	p.action.Println("%s", strings.Join(cmdargs, " "))

	cmdpath, cmdargs = parseCommand(cmdargs)
	if cmdpath == "npm" {
		cmdPaths, err := resolveNpmPaths(p.action)
		if err != nil {
			p.action.ErrorExit("error run command in BUILD.yaml: %s", err)
		}
		cmdpath = cmdPaths[0]
		cmdargs = append(cmdPaths[1:], cmdargs...)
	}
	runcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
	runcmd.Stdin = setting.InOrStdin()
	runcmd.Stdout = setting.OutOrStdout()
	runcmd.Stderr = setting.ErrOrStderr()

	if err = runcmd.Run(); err != nil {
		p.action.ErrorExit("%v", err)
	}
	return
}

// GetVersion -
func (p *ProjectBuildConfig) GetVersion() (version string) {
	var (
		vfpath string = path.Join(p.basedir, "VERSION")
		vfst   os.FileInfo
		err    error
		vfdata []byte
	)
	version = p.Version
	if version != "" {
		return
	}

	vfst, err = os.Stat(vfpath)
	if err != nil || vfst.IsDir() {
		return
	}

	vfdata, err = os.ReadFile(vfpath)
	if err != nil {
		return
	}

	version = string(vfdata)
	version = strings.Trim(version, "\r\n")

	p.Version = version

	return
}

// BuildDocker -
func (p *ProjectBuildConfig) dockerBuildX(platform []string) (err error) {

	var (
		setting          *GlobalSetting = p.action.GlobalSetting()
		cmdpath          string
		cmdargs          []string
		platforms        string
		imageName        string
		tagName          string
		dockerfile       string
		baseDirectory    string
		buildcmd         *exec.Cmd
		buildxFilePath   string
		buildxPluginPath string
		configFilePath   string
		build_cmd        string
		ic               *InstallConfiguration
	)

	if configFilePath, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		p.action.ErrorExit("load %s error: %s", configFilePath, err)
		return
	}

	build_cmd = "docker"
	cmdargs = []string{
		build_cmd,
		"-v",
	}

	cmdpath, cmdargs = parseCommand(cmdargs)
	buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
	if err = buildcmd.Run(); err != nil {
		build_cmd = "podman"
	}

	imageName = p.Docker.Image
	if BaseConfig.Snz1dp.Docker.URL != ic.Snz1dp.Registry.URL && strings.HasPrefix(imageName, BaseConfig.Snz1dp.Docker.URL) {
		imageName = ic.Snz1dp.Registry.URL + imageName[len(BaseConfig.Snz1dp.Docker.URL):]
	}

	tagName = p.Docker.Tag
	if tagName == "" {
		tagName = p.GetVersion()
	}
	if tagName != "" {
		imageName = imageName + ":" + tagName
	} else {
		imageName = imageName + ":latest"
	}

	dockerfile = p.Docker.Dockerfile
	dockerfile = path.Join(p.basedir, dockerfile)
	baseDirectory = path.Join(p.basedir, p.Docker.BaseDirectory)

	if build_cmd == "docker" && p.Docker.Dockerfile != "" {
		if buildxFilePath, err = downloadBuildx(p.action, setting.OutOrStdout(), false); err != nil {
			p.action.ErrorExit("%v", err)
		}

		buildxPluginPath = path.Join(utils.UserHomePath, ".docker", "cli-plugins")

		if err = os.MkdirAll(buildxPluginPath, os.ModePerm); err != nil {
			p.action.ErrorExit("mkdir %s error %v", buildxPluginPath, err)
		}

		buildxBinFile := path.Join(buildxPluginPath, "docker-buildx")
		switch runtime.GOOS {
		case "windows":
			buildxBinFile += ".exe"
		}

		var lst os.FileInfo
		if lst, err = os.Stat(buildxBinFile); err != nil || (lst != nil && lst.IsDir()) {
			if lst != nil && lst.IsDir() {
				os.RemoveAll(buildxBinFile)
			}
			utils.CopyFile(buildxFilePath, buildxBinFile)
			switch runtime.GOOS {
			case "windows":
				break
			default:
				os.Chmod(buildxBinFile, 0777)
			}
		}

		cmdpath, cmdargs = parseCommand([]string{
			build_cmd,
			"image",
			"inspect",
			"moby/buildkit:buildx-stable-1",
		})

		buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
		buildcmd.Env = append(os.Environ(), "DOCKER_CLI_EXPERIMENTAL=enabled")

		if err = buildcmd.Run(); err != nil {
			var (
				buildkitImageFilePath, buildkitImageSha256 string
			)
			if buildkitImageFilePath, buildkitImageSha256, err = downloadBuildkit(p.action, setting.OutOrStdout(), false); err != nil {
				p.action.ErrorExit("%v", err)
			}

			cmdpath, cmdargs = parseCommand([]string{
				build_cmd,
				"load",
				"-i",
				buildkitImageFilePath,
			})

			buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
			buildcmd.Env = append(os.Environ(), "DOCKER_CLI_EXPERIMENTAL=enabled")

			if err = buildcmd.Run(); err != nil {
				p.action.ErrorExit("load moby/buildkit:buildx-stable-1 image error: %v", err)
			}

			cmdpath, cmdargs = parseCommand([]string{
				build_cmd,
				"tag",
				buildkitImageSha256,
				"moby/buildkit:buildx-stable-1",
			})

			buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
			buildcmd.Env = append(os.Environ(), "DOCKER_CLI_EXPERIMENTAL=enabled")

			if err = buildcmd.Run(); err != nil {
				p.action.ErrorExit("tag moby/buildkit:buildx-stable-1 image error : %v", err)
			}
		}

		cmdpath, cmdargs = parseCommand([]string{
			build_cmd,
			"buildx",
			"inspect",
			"snz1dp",
		})

		buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
		buildcmd.Env = append(os.Environ(), "DOCKER_CLI_EXPERIMENTAL=enabled")

		if err = buildcmd.Run(); err != nil {
			cmdpath, cmdargs = parseCommand([]string{
				build_cmd,
				"buildx",
				"create",
				"--use",
				"--name",
				"snz1dp",
				"--driver",
				"docker-container",
				"--platform",
				"linux/amd64,linux/amd64/v2,linux/amd64/v3,linux/arm64,linux/arm/v7,linux/arm/v6",
			})
			for _, v := range p.Docker.SSHKeys {
				cmdargs = append(cmdargs, "--ssh", v)
			}
			buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
			buildcmd.Env = append(os.Environ(), "DOCKER_CLI_EXPERIMENTAL=enabled")

			if err = buildcmd.Run(); err != nil {
				p.action.ErrorExit(" docker buildx create instance error: %v", err)
			}
		} else {
			cmdpath, cmdargs = parseCommand([]string{
				build_cmd,
				"buildx",
				"use",
				"snz1dp",
			})
			buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
			buildcmd.Env = append(os.Environ(), "DOCKER_CLI_EXPERIMENTAL=enabled")

			if err = buildcmd.Run(); err != nil {
				p.action.ErrorExit("docker buildx use instance error: %v", err)
			}
		}

		platforms = strings.Join(platform, ",")

		cmdargs = append([]string{
			build_cmd,
			"buildx",
			"build",
			"--push",
			"--platform",
		}, platforms, "-t", imageName)

		if len(p.Docker.Args) > 0 {
			for k, v := range p.Docker.Args {
				if v == nil || *v == "" {
					continue
				}
				cmdargs = append(cmdargs, "--build-arg", k+"="+*v)
			}
		}

		if len(p.Docker.Labels) > 0 {
			for k, v := range p.Docker.Labels {
				cmdargs = append(cmdargs, "--label", k+"="+v)
			}
		}

		cmdpath, cmdargs = parseCommand(append(cmdargs, "-f", dockerfile, baseDirectory))

		buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
		buildcmd.Env = append(os.Environ(), "DOCKER_CLI_EXPERIMENTAL=enabled")

		buildcmd.Stdin = setting.InOrStdin()
		buildcmd.Stdout = setting.OutOrStdout()
		buildcmd.Stderr = setting.ErrOrStderr()

		p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))
		if err = buildcmd.Run(); err != nil {
			p.action.ErrorExit("%v", err)
		}
	} else if p.Docker.Dockerfile != "" {
		// 处理其他情况, podman不能同时构建多平台，只能通过单独构建每个平台来实现
		var imageAndTags = []string{}
		for platformIndex := range platform {
			targetPlatform := platform[platformIndex]
			arch := strings.Split(targetPlatform, "/")[1]
			imageAndTag := imageName + "-" + arch
			cmdargs = []string{
				build_cmd,
				"build",
				"--platform",
				targetPlatform,
				"-t",
				imageAndTag,
			}

			if len(p.Docker.Args) > 0 {
				for k, v := range p.Docker.Args {
					if v == nil || *v == "" {
						continue
					}
					cmdargs = append(cmdargs, "--build-arg", k+"="+*v)
				}
			}

			cmdpath, cmdargs = parseCommand(append(cmdargs, "-f", dockerfile, baseDirectory))
			buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
			buildcmd.Stdin = setting.InOrStdin()
			buildcmd.Stdout = setting.OutOrStdout()
			buildcmd.Stderr = setting.ErrOrStderr()

			p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))
			if err = buildcmd.Run(); err != nil {
				p.action.ErrorExit("%v", err)
				return
			}
			imageAndTags = append(imageAndTags, imageAndTag)
		}
		cmdpath, cmdargs = parseCommand([]string{
			build_cmd,
			"manifest",
			"create",
			imageName,
		})
		buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
		buildcmd.Stdin = setting.InOrStdin()
		buildcmd.Stdout = setting.OutOrStdout()
		buildcmd.Stderr = setting.ErrOrStderr()

		p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))

		if err = buildcmd.Run(); err != nil {
			p.action.ErrorExit("%v", err)
			return
		}
		for _, imageAndTag := range imageAndTags {
			cmdpath, cmdargs = parseCommand([]string{
				build_cmd,
				"manifest",
				"add",
				imageName,
				imageAndTag,
			})
			buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
			buildcmd.Stdin = setting.InOrStdin()
			buildcmd.Stdout = setting.OutOrStdout()
			buildcmd.Stderr = setting.ErrOrStderr()

			p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))
			if err = buildcmd.Run(); err != nil {
				p.action.ErrorExit("%v", err)
				return
			}
		}
		cmdpath, cmdargs = parseCommand([]string{
			build_cmd,
			"manifest",
			"push",
			imageName,
		})

		buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
		buildcmd.Stdin = setting.InOrStdin()
		buildcmd.Stdout = setting.OutOrStdout()
		buildcmd.Stderr = setting.ErrOrStderr()

		p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))
		if err = buildcmd.Run(); err != nil {
			p.action.ErrorExit("%v", err)
			return
		}
	}

	// 转推依赖镜像
	if p.Docker.Depends != nil && len(*p.Docker.Depends) > 0 {
		for _, v := range p.Docker.GetDepends() {
			sourceImage := v.Source
			targetImage := v.Target

			if BaseConfig.Snz1dp.Docker.URL != ic.Snz1dp.Registry.URL && strings.HasPrefix(targetImage, BaseConfig.Snz1dp.Docker.URL) {
				targetImage = ic.Snz1dp.Registry.URL + targetImage[len(BaseConfig.Snz1dp.Docker.URL):]
			}

			if sourceImage != targetImage {
				// 创建临时的Dockerfile文件
				var dockerfd *os.File
				if dockerfd, err = os.CreateTemp(p.basedir, ".temp-dockerfile-"); err != nil {
					p.action.ErrorExit("%v", err)
				}
				dockerfd.WriteString(fmt.Sprintf("FROM %s", sourceImage))
				dockerfd.Sync()
				tmpDockerFileName := dockerfd.Name()

				if build_cmd == "docker" {
					cmdargs = append([]string{
						build_cmd,
						"buildx",
						"build",
						"--push",
						"--platform",
					}, platforms, "-t", targetImage)

					cmdpath, cmdargs = parseCommand(append(cmdargs, "-f", tmpDockerFileName, p.basedir))
					buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
					buildcmd.Env = append(os.Environ(), "DOCKER_CLI_EXPERIMENTAL=enabled")
					buildcmd.Stdin = setting.InOrStdin()
					buildcmd.Stdout = setting.OutOrStdout()
					buildcmd.Stderr = setting.ErrOrStderr()

					p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))
					if err = buildcmd.Run(); err != nil {
						dockerfd.Close()
						os.Remove(tmpDockerFileName)
						p.action.ErrorExit("%v", err)
					} else {
						dockerfd.Close()
						os.Remove(tmpDockerFileName)
					}
				} else {
					// 处理其他情况, podman不能同时构建多平台，只能通过单独构建每个平台来实现
					targetImageAndTags := []string{}
					for platformIndex := range platform {
						targetPlatform := platform[platformIndex]
						arch := strings.Split(targetPlatform, "/")[1]
						targetImageAndTag := targetImage + "-" + arch
						cmdpath, cmdargs = parseCommand([]string{
							build_cmd,
							"build",
							"--platform",
							targetPlatform,
							"-t",
							targetImageAndTag,
						})
						buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
						buildcmd.Stdin = setting.InOrStdin()
						buildcmd.Stdout = setting.OutOrStdout()
						buildcmd.Stderr = setting.ErrOrStderr()
						p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))
						if err = buildcmd.Run(); err != nil {
							dockerfd.Close()
							os.Remove(tmpDockerFileName)
							p.action.ErrorExit("%v", err)
							return
						}
						targetImageAndTags = append(targetImageAndTags, targetImageAndTag)
					}
					dockerfd.Close()
					os.Remove(tmpDockerFileName)

					cmdpath, cmdargs = parseCommand([]string{
						build_cmd,
						"manifest",
						"create",
						"-a",
						targetImage,
					})
					buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
					buildcmd.Stdin = setting.InOrStdin()
					buildcmd.Stdout = setting.OutOrStdout()
					buildcmd.Stderr = setting.ErrOrStderr()

					p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))
					if err = buildcmd.Run(); err != nil {
						p.action.ErrorExit("%v", err)
						return
					}

					for _, imageAndTag := range targetImageAndTags {
						cmdpath, cmdargs = parseCommand([]string{
							build_cmd,
							"manifest",
							"add",
							targetImage,
							imageAndTag,
						})
						buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
						buildcmd.Stdin = setting.InOrStdin()
						buildcmd.Stdout = setting.OutOrStdout()
						buildcmd.Stderr = setting.ErrOrStderr()
						p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))
						if err = buildcmd.Run(); err != nil {
							p.action.ErrorExit("%v", err)
							return
						}
					}

					// 推送镜像
					cmdpath, cmdargs = parseCommand([]string{
						build_cmd,
						"manifest",
						"push",
						targetImage,
					})

					buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
					buildcmd.Stdin = setting.InOrStdin()
					buildcmd.Stdout = setting.OutOrStdout()
					buildcmd.Stderr = setting.ErrOrStderr()

					p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))
					if err = buildcmd.Run(); err != nil {
						p.action.ErrorExit("%v", err)
						return
					}
				}
			}
		}
	}

	return
}

// BuildDocker -
func (p *ProjectBuildConfig) buildDocker(push bool) (err error) {
	var (
		setting        *GlobalSetting = p.action.GlobalSetting()
		configFilePath string
		cmdpath        string
		cmdargs        []string
		imageTag       string
		srcImage       string
		tagName        string
		dockerfile     string
		baseDirectory  string
		build_cmd      string
		buildcmd       *exec.Cmd
		ic             *InstallConfiguration
	)
	if configFilePath, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		p.action.ErrorExit("load %s error: %s", configFilePath, err)
		return
	}

	if p.Docker.Dockerfile != "" {
		imageTag = p.Docker.Image
		srcImage = p.Docker.Image
		if BaseConfig.Snz1dp.Docker.URL != ic.Snz1dp.Registry.URL && strings.HasPrefix(imageTag, BaseConfig.Snz1dp.Docker.URL) {
			imageTag = ic.Snz1dp.Registry.URL + imageTag[len(BaseConfig.Snz1dp.Docker.URL):]
		}

		tagName = p.Docker.Tag
		if tagName == "" {
			tagName = p.GetVersion()
		}
		if tagName != "" {
			imageTag = imageTag + ":" + tagName
			srcImage = srcImage + ":" + tagName
		} else {
			imageTag = imageTag + ":latest"
			srcImage = srcImage + ":latest"
		}

		dockerfile = p.Docker.Dockerfile
		dockerfile = path.Join(p.basedir, dockerfile)
		baseDirectory = path.Join(p.basedir, p.Docker.BaseDirectory)

		if p.Docker.currentPlatform == "" {
			p.Docker.currentPlatform = ic.Platform
		}

		build_cmd = "docker"
		cmdargs = []string{
			build_cmd,
			"-v",
		}

		cmdpath, cmdargs = parseCommand(cmdargs)
		buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
		if err = buildcmd.Run(); err != nil {
			build_cmd = "podman"
		}

		cmdargs = append([]string{
			build_cmd,
			"build",
		}, "-t", imageTag)
		if imageTag != srcImage {
			cmdargs = append(cmdargs, "-t", srcImage)
		}

		for _, v := range p.Docker.SSHKeys {
			cmdargs = append(cmdargs, "--ssh", v)
		}

		cmdargs = append(cmdargs, "-f", dockerfile)
		if p.Docker.currentPlatform != "" {
			cmdargs = append(cmdargs, "--platform", p.Docker.currentPlatform)
		}

		if len(p.Docker.Args) > 0 {
			for k, v := range p.Docker.Args {
				if v == nil || *v == "" {
					continue
				}
				cmdargs = append(cmdargs, "--build-arg", k+"="+*v)
			}
		}

		if len(p.Docker.Labels) > 0 {
			for k, v := range p.Docker.Labels {
				cmdargs = append(cmdargs, "--label", k+"="+v)
			}
		}

		cmdargs = append(cmdargs, baseDirectory)
		cmdpath, cmdargs = parseCommand(cmdargs)

		buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
		buildcmd.Stdin = setting.InOrStdin()
		buildcmd.Stdout = setting.OutOrStdout()
		buildcmd.Stderr = setting.ErrOrStderr()

		p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))
		if err = buildcmd.Run(); err != nil {
			p.action.ErrorExit("%v", err)
		}

		if push {
			cmdpath, cmdargs = parseCommand(append([]string{
				build_cmd,
				"push",
			}, imageTag))

			buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
			buildcmd.Stdin = setting.InOrStdin()
			buildcmd.Stdout = setting.OutOrStdout()
			buildcmd.Stderr = setting.ErrOrStderr()

			p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))
			if err = buildcmd.Run(); err != nil {
				p.action.ErrorExit("%v", err)
			}
		}
	}

	if p.Docker.Depends != nil && len(*p.Docker.Depends) > 0 {
		for _, v := range p.Docker.GetDepends() {
			sourceImage := v.Source
			targetImage := v.Target

			if BaseConfig.Snz1dp.Docker.URL != ic.Snz1dp.Registry.URL && strings.HasPrefix(targetImage, BaseConfig.Snz1dp.Docker.URL) {
				targetImage = ic.Snz1dp.Registry.URL + targetImage[len(BaseConfig.Snz1dp.Docker.URL):]
			}

			if sourceImage != targetImage {
				cmdpath, cmdargs = parseCommand(append([]string{
					build_cmd,
					"pull",
				}, sourceImage))
				buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
				buildcmd.Stdin = setting.InOrStdin()
				buildcmd.Stdout = setting.OutOrStdout()
				buildcmd.Stderr = setting.ErrOrStderr()

				p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))
				if err = buildcmd.Run(); err != nil {
					p.action.ErrorExit("%v", err)
				}

				cmdpath, cmdargs = parseCommand(append([]string{
					build_cmd,
					"tag",
				}, sourceImage, targetImage))
				buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
				buildcmd.Stdin = setting.InOrStdin()
				buildcmd.Stdout = setting.OutOrStdout()
				buildcmd.Stderr = setting.ErrOrStderr()

				p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))
				if err = buildcmd.Run(); err != nil {
					p.action.ErrorExit("%v", err)
				}

				if push {
					cmdpath, cmdargs = parseCommand(append([]string{
						build_cmd,
						"push",
					}, targetImage))
					buildcmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
					buildcmd.Stdin = setting.InOrStdin()
					buildcmd.Stdout = setting.OutOrStdout()
					buildcmd.Stderr = setting.ErrOrStderr()

					p.action.Println("%s %s", cmdpath, strings.Join(cmdargs, " "))
					if err = buildcmd.Run(); err != nil {
						p.action.ErrorExit("%v", err)
					}
				}
			}
		}
	}

	return
}

// BuildDocker -
func (p *ProjectBuildConfig) BuildDocker(mulplat bool) (err error) {
	if p.Docker == nil {
		p.action.Println("no docker in BUILD.yaml")
		return
	}

	if !p.SkipBuildSource {
		p.Build()
	}

	if mulplat {
		if len(p.Docker.Platform) > 0 {
			err = p.dockerBuildX(p.Docker.Platform)
		} else {
			err = p.buildDocker(true)
		}
	} else {
		err = p.buildDocker(false)
	}

	return
}

const projectRunDir = ".run"

// ApplyDeveleopConfig -
func (p *ProjectBuildConfig) ApplyDeveleopConfig() (err error) {
	if p.Service == nil {
		p.action.ErrorExit("no service defined in BUILD.yaml")
		return
	}

	var (
		setting        *GlobalSetting = p.action.GlobalSetting()
		configFilePath string
		oldic, ic      *InstallConfiguration
	)

	if p.Develop == nil || p.Develop.Configuration == "" {
		p.action.ErrorExit("no develop.config defined in BUILD.yaml")
		return
	}

	// 安装配置文件
	if configFilePath, oldic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		p.action.ErrorExit("load %s error: %s", err)
		return
	}

	if ic, err = LoadInstallConfigurationFromBytes([]byte(p.Develop.Configuration)); err != nil {
		p.action.ErrorExit("%s-%s develop config error: %s", p.Name, p.GetVersion(), err)
		return
	}

	// 设置老的配置
	ic.inline = oldic.inline

	if err = setting.InitInstallConfiguration(ic); err != nil {
		p.action.ErrorExit("%s-%s develop config error: %s", p.Name, p.GetVersion(), err)
		return
	}

	if ic.Snz1dp.Registry == nil {
		ic.Snz1dp.Registry = new(DockerRegistry)
	}
	if ic.Snz1dp.Registry.URL == "" {
		ic.Snz1dp.Registry.URL = oldic.Snz1dp.Registry.URL
		ic.Snz1dp.Registry.Secure = new(bool)
		*ic.Snz1dp.Registry.Secure = *oldic.Snz1dp.Registry.Secure
	}
	if ic.Snz1dp.Registry.Username == "" || ic.Snz1dp.Registry.Password == "" {
		ic.Snz1dp.Registry.Username = oldic.Snz1dp.Registry.Username
		ic.Snz1dp.Registry.Password = oldic.Snz1dp.Registry.Password
	}
	if ic.Snz1dp.Registry.Secure == nil {
		ic.Snz1dp.Registry.Secure = new(bool)
		*ic.Snz1dp.Registry.Secure = *oldic.Snz1dp.Registry.Secure
	}

	if ic.Snz1dp.HelmRepo == nil {
		ic.Snz1dp.HelmRepo = new(HelmRegistry)
	}
	if ic.Snz1dp.HelmRepo.Name == "" {
		ic.Snz1dp.HelmRepo.Name = oldic.Snz1dp.HelmRepo.Name
	}
	if ic.Snz1dp.HelmRepo.URL == "" {
		ic.Snz1dp.HelmRepo.URL = oldic.Snz1dp.HelmRepo.URL
	}
	if ic.Snz1dp.HelmRepo.Username == "" {
		ic.Snz1dp.HelmRepo.Username = oldic.Snz1dp.HelmRepo.Username
	}

	if ic.Snz1dp.HelmRepo.Password == "" {
		ic.Snz1dp.HelmRepo.Password = oldic.Snz1dp.HelmRepo.Password
	}

	if ic.Snz1dp.MavenRepo == nil {
		ic.Snz1dp.MavenRepo = new(MavenRegistry)
	}
	if ic.Snz1dp.MavenRepo.ID == "" {
		ic.Snz1dp.MavenRepo.ID = oldic.Snz1dp.MavenRepo.ID
	}
	if ic.Snz1dp.MavenRepo.Mirrors == nil {
		ic.Snz1dp.MavenRepo.Mirrors = oldic.Snz1dp.MavenRepo.Mirrors
	}
	if ic.Snz1dp.MavenRepo.URL == "" {
		ic.Snz1dp.MavenRepo.URL = oldic.Snz1dp.MavenRepo.URL
	}
	if ic.Snz1dp.MavenRepo.Username == "" {
		ic.Snz1dp.MavenRepo.Username = oldic.Snz1dp.MavenRepo.Username
	}

	if ic.Snz1dp.MavenRepo.Password == "" {
		ic.Snz1dp.MavenRepo.Password = oldic.Snz1dp.MavenRepo.Password
	}

	if ic.Snz1dp.NpmRepo == nil {
		ic.Snz1dp.NpmRepo = new(NpmRegistry)
	}
	if ic.Snz1dp.NpmRepo.ID == "" {
		ic.Snz1dp.NpmRepo.ID = oldic.Snz1dp.NpmRepo.ID
	}
	if ic.Snz1dp.NpmRepo.URL == "" {
		ic.Snz1dp.NpmRepo.URL = oldic.Snz1dp.NpmRepo.URL
	}
	if ic.Snz1dp.NpmRepo.Username == "" {
		ic.Snz1dp.NpmRepo.Username = oldic.Snz1dp.NpmRepo.Username
	}

	if ic.Snz1dp.NpmRepo.Password == "" {
		ic.Snz1dp.NpmRepo.Password = oldic.Snz1dp.NpmRepo.Password
	}
	if ic.Snz1dp.NpmRepo.Private == nil {
		ic.Snz1dp.NpmRepo.Private = new(bool)
		*ic.Snz1dp.NpmRepo.Private = *oldic.Snz1dp.NpmRepo.Private
	}

	if ic.Snz1dp.SassSite == nil {
		ic.Snz1dp.SassSite = new(SassBinarySite)
	}
	if ic.Snz1dp.SassSite.ID == "" {
		ic.Snz1dp.SassSite.ID = oldic.Snz1dp.SassSite.ID
	}
	if ic.Snz1dp.SassSite.URL == "" {
		ic.Snz1dp.SassSite.URL = oldic.Snz1dp.SassSite.URL
	}
	if ic.Snz1dp.SassSite.Username == "" {
		ic.Snz1dp.SassSite.Username = oldic.Snz1dp.SassSite.Username
	}
	if ic.Snz1dp.SassSite.Password == "" {
		ic.Snz1dp.SassSite.Password = oldic.Snz1dp.SassSite.Password
	}
	if ic.Snz1dp.SassSite.Private == nil {
		ic.Snz1dp.SassSite.Private = new(bool)
		*ic.Snz1dp.SassSite.Private = *oldic.Snz1dp.SassSite.Private
	}

	// 如果老的Jwt已有配置则设置
	if ic.Appgateway.GetJwtConfig() == nil && oldic.Appgateway.infile && oldic.Appgateway.GetJwtConfig() != nil {
		ic.Appgateway.SetJwtConfig(oldic.Appgateway.GetJwtConfig())
	}

	if _, _, err = ic.GetBundleComponents(false); err != nil {
		p.action.ErrorExit("load bundle error: %s", err.Error())
		return
	}

	if err = setting.SaveLocalInstallConfiguration(ic, configFilePath); err != nil {
		p.action.ErrorExit("save %s-%s develop config to %s error: %s", p.Name, p.GetVersion(), err)
		return
	}

	p.action.Println("save %s success!", configFilePath)

	return
}

// StartDevelopProfile -
func (p *ProjectBuildConfig) StartDevelopProfile() (err error) {

	if p.Service == nil {
		p.action.ErrorExit("no service defined in BUILD.yaml")
		return
	}

	var (
		setting        *GlobalSetting = p.action.GlobalSetting()
		configFilePath string
		oldic, ic      *InstallConfiguration
	)

	if p.Develop == nil || p.Develop.Configuration == "" {
		p.action.ErrorExit("no develop.config defined in BUILD.yaml")
		return
	}

	// 安装配置文件
	if configFilePath, oldic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		p.action.ErrorExit("load %s error: %s", err)
		return
	}

	if ic, err = LoadInstallConfigurationFromBytes([]byte(p.Develop.Configuration)); err != nil {
		p.action.ErrorExit("%s@%s develop config error: %s", p.Name, p.GetVersion(), err)
		return
	}

	// 设置内置配置
	ic.inline = oldic.inline
	ic.Snz1dp = oldic.Snz1dp

	if err = setting.InitInstallConfiguration(ic); err != nil {
		p.action.ErrorExit("%s@%s develop config error: %s", p.Name, p.GetVersion(), err)
		return
	}

	// 如果老的Jwt已有配置则设置
	if ic.Appgateway.GetJwtConfig() == nil && oldic.Appgateway.infile && oldic.Appgateway.GetJwtConfig() != nil {
		ic.Appgateway.SetJwtConfig(oldic.Appgateway.GetJwtConfig())
	}

	if _, _, err = ic.GetBundleComponents(false); err != nil {
		p.action.ErrorExit("load bundle error: %s", err.Error())
		return
	}

	if err = setting.SaveLocalInstallConfiguration(ic, configFilePath); err != nil {
		p.action.ErrorExit("save %s@%s develop config to %s error: %s", p.Name, p.GetVersion(), err)
		return
	}

	p.action.Println("save %s success!", configFilePath)

	startprofile := NewStartStandaloneService(setting)
	startprofile.ServiceName = DefaultProfileName
	return startprofile.Run()
}

// GetJwtConfig 获取JWT配置
func (p *ProjectBuildConfig) GetJwtConfig() (out *JwtSecretConfig) {
	if p.Develop != nil && p.Develop.Jwt != nil {
		out = p.Develop.Jwt
	} else {
		out = new(JwtSecretConfig)
		out.inline = true
	}
	return
}

// StartStandalone 独立运行
func (p *ProjectBuildConfig) StartStandalone() (err error) {

	if p.Service == nil {
		p.action.ErrorExit("no service defined in BUILD.yaml")
		return
	}

	if !p.SkipBuildDocker {
		p.BuildDocker(false)
	}

	var (
		setting        *GlobalSetting = p.action.GlobalSetting()
		configFilePath string
		ic, oldic      *InstallConfiguration
		inline         *InstallConfiguration
	)

	if configFilePath, oldic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		p.action.ErrorExit("load %s error: %s", configFilePath, err)
		return
	}

	inline = oldic.inline

	if p.Develop != nil && p.Develop.Configuration != "" {
		if ic, err = LoadInstallConfigurationFromBytes([]byte(p.Develop.Configuration)); err != nil {
			p.action.ErrorExit("%s@%s develop config error: %s", p.Name, p.GetVersion(), err)
			return
		}

		if ic.Snz1dp.Registry == nil {
			ic.Snz1dp.Registry = new(DockerRegistry)
		}
		if ic.Snz1dp.Registry.URL == "" {
			ic.Snz1dp.Registry.URL = oldic.Snz1dp.Registry.URL
			ic.Snz1dp.Registry.Secure = new(bool)
			*ic.Snz1dp.Registry.Secure = *oldic.Snz1dp.Registry.Secure
		}
		if ic.Snz1dp.Registry.Username == "" || ic.Snz1dp.Registry.Password == "" {
			ic.Snz1dp.Registry.Username = oldic.Snz1dp.Registry.Username
			ic.Snz1dp.Registry.Password = oldic.Snz1dp.Registry.Password
		}
		if ic.Snz1dp.Registry.Secure == nil {
			ic.Snz1dp.Registry.Secure = new(bool)
			*ic.Snz1dp.Registry.Secure = *oldic.Snz1dp.Registry.Secure
		}

		if ic.Snz1dp.HelmRepo == nil {
			ic.Snz1dp.HelmRepo = new(HelmRegistry)
		}
		if ic.Snz1dp.HelmRepo.Name == "" {
			ic.Snz1dp.HelmRepo.Name = oldic.Snz1dp.HelmRepo.Name
		}
		if ic.Snz1dp.HelmRepo.URL == "" {
			ic.Snz1dp.HelmRepo.URL = oldic.Snz1dp.HelmRepo.URL
		}
		if ic.Snz1dp.HelmRepo.Username == "" {
			ic.Snz1dp.HelmRepo.Username = oldic.Snz1dp.HelmRepo.Username
		}

		if ic.Snz1dp.HelmRepo.Password == "" {
			ic.Snz1dp.HelmRepo.Password = oldic.Snz1dp.HelmRepo.Password
		}

		if ic.Snz1dp.MavenRepo == nil {
			ic.Snz1dp.MavenRepo = new(MavenRegistry)
		}
		if ic.Snz1dp.MavenRepo.ID == "" {
			ic.Snz1dp.MavenRepo.ID = oldic.Snz1dp.MavenRepo.ID
		}
		if ic.Snz1dp.MavenRepo.Mirrors == nil {
			ic.Snz1dp.MavenRepo.Mirrors = oldic.Snz1dp.MavenRepo.Mirrors
		}
		if ic.Snz1dp.MavenRepo.URL == "" {
			ic.Snz1dp.MavenRepo.URL = oldic.Snz1dp.MavenRepo.URL
		}
		if ic.Snz1dp.MavenRepo.Username == "" {
			ic.Snz1dp.MavenRepo.Username = oldic.Snz1dp.MavenRepo.Username
		}

		if ic.Snz1dp.MavenRepo.Password == "" {
			ic.Snz1dp.MavenRepo.Password = oldic.Snz1dp.MavenRepo.Password
		}

		if ic.Snz1dp.NpmRepo == nil {
			ic.Snz1dp.NpmRepo = new(NpmRegistry)
		}
		if ic.Snz1dp.NpmRepo.ID == "" {
			ic.Snz1dp.NpmRepo.ID = oldic.Snz1dp.NpmRepo.ID
		}
		if ic.Snz1dp.NpmRepo.URL == "" {
			ic.Snz1dp.NpmRepo.URL = oldic.Snz1dp.NpmRepo.URL
		}
		if ic.Snz1dp.NpmRepo.Username == "" {
			ic.Snz1dp.NpmRepo.Username = oldic.Snz1dp.NpmRepo.Username
		}

		if ic.Snz1dp.NpmRepo.Password == "" {
			ic.Snz1dp.NpmRepo.Password = oldic.Snz1dp.NpmRepo.Password
		}
		if ic.Snz1dp.NpmRepo.Private == nil {
			ic.Snz1dp.NpmRepo.Private = new(bool)
			*ic.Snz1dp.NpmRepo.Private = *oldic.Snz1dp.NpmRepo.Private
		}

		if ic.Snz1dp.SassSite == nil {
			ic.Snz1dp.SassSite = new(SassBinarySite)
		}
		if ic.Snz1dp.SassSite.ID == "" {
			ic.Snz1dp.SassSite.ID = oldic.Snz1dp.SassSite.ID
		}
		if ic.Snz1dp.SassSite.URL == "" {
			ic.Snz1dp.SassSite.URL = oldic.Snz1dp.SassSite.URL
		}
		if ic.Snz1dp.SassSite.Username == "" {
			ic.Snz1dp.SassSite.Username = oldic.Snz1dp.SassSite.Username
		}
		if ic.Snz1dp.SassSite.Password == "" {
			ic.Snz1dp.SassSite.Password = oldic.Snz1dp.SassSite.Password
		}
		if ic.Snz1dp.SassSite.Private == nil {
			ic.Snz1dp.SassSite.Private = new(bool)
			*ic.Snz1dp.SassSite.Private = *oldic.Snz1dp.SassSite.Private
		}

		ic.inline = inline
		if err = setting.InitInstallConfiguration(ic); err != nil {
			p.action.ErrorExit("%s@%s develop config error: %s", p.Name, p.GetVersion(), err)
			return
		}

		// 如果老的Jwt已有配置则设置
		if ic.Appgateway.GetJwtConfig() == nil && oldic.Appgateway.infile && oldic.Appgateway.GetJwtConfig() != nil {
			ic.Appgateway.SetJwtConfig(oldic.Appgateway.GetJwtConfig())
		}

		if _, _, err = ic.GetBundleComponents(false); err != nil {
			p.action.ErrorExit("load bundle error: %s", err.Error())
			return
		}

		if err = setting.SaveLocalInstallConfiguration(ic, configFilePath); err != nil {
			p.action.ErrorExit("save %s@%s develop config to %s error: %s", p.Name, p.GetVersion(), configFilePath, err)
			return
		}

		startprofile := NewStartStandaloneService(setting)
		startprofile.ForcePullImage = p.ForcePullImage
		startprofile.LoadImageLocal = p.LoadImageLocal
		startprofile.ServiceName = DefaultProfileName

		startprofile.Run()

		if configFilePath, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
			p.action.ErrorExit("load %s error: %s", configFilePath, err)
			return
		}

	} else {
		ic = oldic
	}

	var (
		envVariables []string
		volumeBinds  []string
		runFiles     map[string]string
		healthcheck  *container.HealthConfig
		runCmd       []string
		initCmd      []string
	)

	volumeBinds, runFiles, err = MergeVolumeFiles(ic, p.Service.Volumes, p.Service.RunFiles)

	if err != nil {
		p.action.ErrorExit("%s@%s volumes error: %s", p.Name, p.GetVersion(), err)
		return
	}

	ic.Snz1dp.ExternalIP = utils.GetExternalIpv4()

	// 设置jwt
	ic.Snz1dp.Jwt = p.GetJwtConfig()

	envVariables = p.Service.Envs
	if envVariables, err = ic.RenderEnvVariables(envVariables); err != nil {
		p.action.ErrorExit("%s@%s env error: %s\nenv: %v", p.Name, p.GetVersion(), err, envVariables)
		return
	}
	p.Service.Envs = envVariables

	if runCmd, err = ic.RenderStrings(p.Service.Cmd); err != nil {
		p.action.ErrorExit("%s@%s run command error: %s\ncommand: %v", p.Name, p.GetVersion(), err, p.Service.Cmd)
		return
	}

	initCmd = p.Service.InitCmd

	if len(initCmd) == 0 && p.Service.InitJob != nil && len(p.Service.InitJob.Command) > 0 {
		initCmd = p.Service.InitJob.Command
	}

	if initCmd, err = ic.RenderStrings(initCmd); err != nil {
		p.action.ErrorExit("%s@%s init command error: %s\ncommand: %v", p.Name, p.GetVersion(), err, initCmd)
		return
	}

	healthcheck, err = p.Service.getHealthcheck()
	if err != nil {
		p.action.ErrorExit("%s@%s healtch check error: %s", p.Name, p.GetVersion(), err)
		return
	}

	runDir := path.Join(p.basedir, projectRunDir)
	os.MkdirAll(runDir, os.ModePerm)

	dockerImageTag := p.Docker.Tag

	if dockerImageTag == "" {
		dockerImageTag = p.GetVersion()
	}

	sls := &StandaloneService{
		BaseDir:        runDir,
		ServiceName:    p.Name,
		Version:        p.GetVersion(),
		ImageName:      p.Docker.Image,
		ImageTag:       dockerImageTag,
		ForcePullImage: false,
		PortBinds:      p.Service.Ports,
		VolumeBinds:    volumeBinds,
		VolumeOwners:   p.Service.Owners,
		EnvVariables:   envVariables,
		Cmd:            runCmd,
		InitCmd:        initCmd,
		RunFiles:       runFiles,
		Healthcheck:    healthcheck,
		Ingress:        p.Service.Ingress,
		action:         p.action,
		ic:             ic,
	}

	if err = sls.Start(); err != nil {
		p.action.ErrorExit("start %s@%s error: %s", p.Name, p.GetVersion(), err)
		return
	}

	if err = sls.WaitHealthy(); err != nil {
		p.action.ErrorExit("wait %s@%s health error: %s", p.Name, p.GetVersion(), err)
		return
	}

	if err = sls.Init(); err != nil {
		p.action.ErrorExit("init %s@%s error: %v", p.Name, p.GetVersion(), err)
		return
	}

	if ic.Appgateway.infile {
		if err = sls.ApplyIngress(); err != nil {
			p.action.Println("apply %s@%s ingress error: %v", p.Name, p.GetVersion(), err)
		}
	}

	return

}

// StopStandalone 独立运行
func (p *ProjectBuildConfig) StopStandalone(all bool) (err error) {

	if p.Service == nil {
		p.action.ErrorExit("no service defined in BUILD.yaml")
		return
	}

	var (
		healthcheck *container.HealthConfig
	)

	healthcheck, err = p.Service.getHealthcheck()
	if err != nil {
		p.action.ErrorExit("%s-%s health check error: %s", p.Name, p.GetVersion(), err)
		return err
	}

	runDir := path.Join(p.basedir, projectRunDir)
	os.MkdirAll(runDir, os.ModePerm)

	dockerImageTag := p.Docker.Image
	if dockerImageTag == "" {
		dockerImageTag = p.GetVersion()
	}

	sls := &StandaloneService{
		BaseDir:        runDir,
		ServiceName:    p.Name,
		Version:        p.GetVersion(),
		ImageName:      p.Docker.Image,
		ImageTag:       dockerImageTag,
		ForcePullImage: false,
		PortBinds:      p.Service.Ports,
		VolumeBinds:    p.Service.Volumes,
		VolumeOwners:   p.Service.Owners,
		EnvVariables:   p.Service.Envs,
		Cmd:            p.Service.Cmd,
		InitCmd:        p.Service.InitCmd,
		RunFiles:       p.Service.RunFiles,
		Healthcheck:    healthcheck,
		Ingress:        p.Service.Ingress,
		action:         p.action,
	}

	if err = sls.Stop(); err != nil {
		p.action.ErrorExit("stop %s@%s error: %s", p.Name, p.Docker.Image, err)
		return
	}

	if all {
		stopprofile := NewStopStandaloneService(p.action.GlobalSetting())
		stopprofile.ServiceName = AllBundleName
		stopprofile.Really = p.Really
		err = stopprofile.Run()
	}

	return
}

// LogStandalone 独立运行
func (p *ProjectBuildConfig) LogStandalone() (err error) {

	if p.Service == nil {
		p.action.ErrorExit("no service defined in BUILD.yaml")
		return
	}

	runDir := path.Join(p.basedir, projectRunDir)

	dockerImageTag := p.Docker.Image
	if dockerImageTag == "" {
		dockerImageTag = p.GetVersion()
	}

	sls := &StandaloneService{
		BaseDir:        runDir,
		ServiceName:    p.Name,
		Version:        p.GetVersion(),
		ImageName:      p.Docker.Image,
		ImageTag:       dockerImageTag,
		ForcePullImage: false,
		PortBinds:      p.Service.Ports,
		VolumeBinds:    p.Service.Volumes,
		EnvVariables:   p.Service.Envs,
		VolumeOwners:   p.Service.Owners,
		Cmd:            p.Service.Cmd,
		InitCmd:        p.Service.InitCmd,
		RunFiles:       p.Service.RunFiles,
		Ingress:        p.Service.Ingress,
		action:         p.action,
		Follow:         p.Follow,
		Since:          p.Since,
		Tail:           p.Tail,
		Timestamps:     p.Timestamps,
		Details:        p.Details,
		Until:          p.Until,
	}

	return sls.Log()
}

// CleanStandalone 独立运行
func (p *ProjectBuildConfig) CleanStandalone(all bool) (err error) {

	if p.Service == nil {
		p.action.ErrorExit("no service defined in BUILD.yaml")
		return
	}

	runDir := path.Join(p.basedir, projectRunDir)

	dockerImageTag := p.Docker.Image
	if dockerImageTag == "" {
		dockerImageTag = p.GetVersion()
	}

	sls := &StandaloneService{
		BaseDir:        runDir,
		ServiceName:    p.Name,
		Version:        p.GetVersion(),
		ImageName:      p.Docker.Image,
		ImageTag:       dockerImageTag,
		ForcePullImage: false,
		PortBinds:      p.Service.Ports,
		VolumeBinds:    p.Service.Volumes,
		EnvVariables:   p.Service.Envs,
		Cmd:            p.Service.Cmd,
		InitCmd:        p.Service.InitCmd,
		RunFiles:       p.Service.RunFiles,
		Ingress:        p.Service.Ingress,
		action:         p.action,
	}

	if err = sls.Clean(); err != nil {
		p.action.Println("clean %s@%s error: %s", p.Name, p.Docker.Image, err)
	}

	if all {
		cleanprofile := NewCleanStandaloneService(p.action.GlobalSetting())
		cleanprofile.ServiceName = AllBundleName
		cleanprofile.Really = p.Really
		err = cleanprofile.Run()
	}

	return

}

// SonarScanner -
func (p *ProjectBuildConfig) SonarScanner() (err error) {
	var (
		sonarCmd *exec.Cmd
		cmdpath  string
		setting  *GlobalSetting = p.action.GlobalSetting()
		cmdargs  []string
	)

	if p.Sonar == nil || p.Sonar.URL == "" {
		p.action.ErrorExit("no sonar or sonar.url in BUILD.yaml: %s", err)
		return
	}

	if p.Sonar.Commands == nil || len(*p.Sonar.Commands) == 0 {
		cmdargs = GetProjectSonarCommands(setting, string(p.Type))
	} else {
		cmdargs = *p.Sonar.Commands
	}

	if cmdargs, err = p.RenderStrings(cmdargs...); err != nil {
		p.action.ErrorExit("error sonar command in BUILD.yaml: %s", err)
	}

	p.action.Println("%s", strings.Join(cmdargs, " "))

	cmdpath, cmdargs = parseCommand(cmdargs)
	sonarCmd = exec.CommandContext(context.Background(), cmdpath, cmdargs...)
	sonarCmd.Stdin = setting.InOrStdin()
	sonarCmd.Stdout = setting.OutOrStdout()
	sonarCmd.Stderr = setting.ErrOrStderr()

	if err = sonarCmd.Run(); err != nil {
		p.action.ErrorExit("%v", err)
	}

	return
}

// Package -
func (p *ProjectBuildConfig) Package(mulplat bool) (err error) {

	if p.Service == nil {
		p.action.ErrorExit("no service defined in BUILD.yaml")
		return
	}

	if !p.SkipBuildDocker {
		p.BuildDocker(mulplat)
	}

	var (
		curdir, bundleMd5File string
		pbc                   *ProjectBuildConfig = p

		outdir       string = path.Join(p.basedir, "out")
		imageTarfile string = path.Join(outdir, fmt.Sprintf("%s-%s-IMAGES.tar", p.Name, p.Version))

		chart             *HelmChart
		chartdir          string
		packdir           string
		fi                os.FileInfo
		rundata, sumData  []byte
		bundledir         string
		imgfname, packtgz string
		imgfData          []byte

		setting *GlobalSetting = p.action.GlobalSetting()
		ic      *InstallConfiguration

		spinner     *utils.WaitSpinner
		readmefpath string
		readmedata  []byte
		readmeinfo  string
		imageTag    string
		tagName     string
	)

	// 加载本地配置
	_, ic, err = setting.LoadLocalInstallConfiguration()
	if err != nil {
		return
	}

	imageTag = p.Docker.Image
	tagName = p.Docker.Tag
	if tagName == "" {
		tagName = p.GetVersion()
	}
	if tagName == "" {
		tagName = "latest"
	}

	if p.SaveDockerImage {
		imageTag := fmt.Sprintf("%s:%s", imageTag, tagName)
		if BaseConfig.Snz1dp.Docker.URL != ic.Snz1dp.Registry.URL && strings.HasPrefix(imageTag, BaseConfig.Snz1dp.Docker.URL) {
			imageTag = imageTag + " " + ic.Snz1dp.Registry.URL + imageTag[len(BaseConfig.Snz1dp.Docker.URL):]
		}
		if err = ic.saveDockerImages(p.Name, p.Version, []string{imageTag},
			p.Docker.currentPlatform, mulplat, false, imageTarfile,
		); err != nil {
			return
		}
	}

	curdir = p.basedir
	bundledir = path.Join(curdir, ".bundle")
	os.RemoveAll(bundledir)

	// 打包目录
	packdir = path.Join(bundledir, fmt.Sprintf("%s-%s", p.Name, p.GetVersion()))
	defer os.RemoveAll(bundledir)

	readmefpath = path.Join(curdir, "README.md")
	readmedata, err = os.ReadFile(readmefpath)
	if err != nil {
		readmeinfo = "无说明文件"
	} else {
		readmeinfo = string(readmedata)
	}

	chartdir = path.Join(curdir, "chart")
	if fi, err = os.Stat(chartdir); err == nil {
		if !fi.IsDir() {
			p.action.ErrorExit("%s already exists and is not a directory", chartdir)
		}
		if err = utils.CopyDir(chartdir, packdir); err != nil {
			p.action.ErrorExit("%v", err)
		}

		chart = &HelmChart{
			Version:         pbc.GetVersion(),
			Name:            pbc.Name,
			Description:     pbc.Description,
			Type:            pbc.Catelog,
			ImageRepository: imageTag,
			ImageTag:        tagName,
			Service:         pbc.Service,
		}

	} else {

		chart = &HelmChart{
			Readme:          readmeinfo,
			Version:         pbc.GetVersion(),
			Name:            pbc.Name,
			Description:     pbc.Description,
			Type:            pbc.Catelog,
			ImageRepository: imageTag,
			ImageTag:        tagName,
			Service:         pbc.Service,
		}

		if _, err = CreateChart(packdir, chart); err != nil {
			os.RemoveAll(bundledir)
			p.action.ErrorExit("create %s@%s char error: %s", p.Name, p.GetVersion(), err)
			return
		}

	}

	imgfname = path.Join(packdir, "IMAGES")

	if fi, err = os.Stat(imgfname); err != nil || fi.IsDir() {
		if fi != nil && fi.IsDir() {
			os.RemoveAll(imgfname)
		}
		targetImage := fmt.Sprintf("%s:%s", chart.ImageRepository, chart.ImageTag)
		var imgDataBuff bytes.Buffer

		imgDataBuff.WriteString(targetImage)
		if chart.Service.InitJob != nil && chart.Service.InitJob.DockerImage != "" && targetImage != chart.Service.InitJob.DockerImage {
			imgDataBuff.WriteString("\n")
			imgDataBuff.WriteString(chart.Service.InitJob.DockerImage)
		}

		if p.Docker != nil && p.Docker.Depends != nil && len(*p.Docker.Depends) > 0 {
			for _, v := range p.Docker.GetDepends() {
				imgDataBuff.WriteString("\n")
				targetImage := v.Target
				if BaseConfig.Snz1dp.Docker.URL != ic.Snz1dp.Registry.URL && strings.HasPrefix(targetImage, BaseConfig.Snz1dp.Docker.URL) {
					targetImage = ic.Snz1dp.Registry.URL + targetImage[len(BaseConfig.Snz1dp.Docker.URL):]
				}
				imgDataBuff.WriteString(targetImage)
			}
		}

		imgfData = imgDataBuff.Bytes()
		if err = os.WriteFile(path.Join(packdir, "IMAGES"), imgfData, 0644); err != nil {
			os.RemoveAll(bundledir)
			p.action.ErrorExit("write %s@%s image list error: %s", p.Name, p.GetVersion(), err)
			return
		}
	} else if _, err = os.ReadFile(imgfname); err != nil {
		os.RemoveAll(bundledir)
		p.action.ErrorExit("read %s@%s image list error: %s", p.Name, p.GetVersion(), err)
		return
	}

	pbc.Service.Docker = &StandaloneDocker{
		Image: imageTag,
		Tag:   tagName,
	}

	if len(pbc.Docker.Platform) == 0 {
		pbc.Service.Platform = []string{"linux/" + runtime.GOARCH}
	} else {
		pbc.Service.Platform = pbc.Docker.Platform
	}

	if rundata, err = yaml.Marshal(pbc.Service); err != nil {
		os.RemoveAll(bundledir)
		p.action.ErrorExit("write %s@%s run file error: %s", p.Name, p.GetVersion(), err)
	}

	if err = os.WriteFile(path.Join(packdir, "RUN.yaml"), rundata, 0644); err != nil {
		os.RemoveAll(bundledir)
		p.action.ErrorExit("%s", err)
	}

	os.MkdirAll(outdir, os.ModePerm)

	packtgz = path.Join(outdir, fmt.Sprintf("%s-%s.tgz", p.Name, chart.Version))
	os.Remove(packtgz)

	spinner = utils.NewSpinner(fmt.Sprintf("save %s...", packtgz), setting.OutOrStdout())

	if err = ArchiveBundle([]string{packdir}, packtgz); err != nil {
		spinner.Close()
		os.RemoveAll(bundledir)
		p.action.ErrorExit("failed: %v", err)
		return
	}

	if sumData, err = utils.FileChecksum(packtgz, sha256.New()); err != nil {
		spinner.Close()
		os.RemoveAll(bundledir)
		p.action.ErrorExit("sha256sum %s error: %s", packtgz, err)
		return
	}

	bundleMd5File = packtgz + ".sha256"

	if err = os.WriteFile(bundleMd5File, []byte(fmt.Sprintf("%s %s", hex.EncodeToString(sumData), filepath.Base(packtgz))), 0644); err != nil {
		spinner.Close()
		os.RemoveAll(bundledir)
		p.action.ErrorExit("save %s error: %s", bundleMd5File, err)
		return
	}

	spinner.Close()
	p.action.Println("%s", "ok!")

	return
}

// CreateBuildFile -
type CreateBuildFile struct {
	BaseAction
	Force   bool
	Name    string
	Version string
	Type    string
}

// NewCreateBuildFile 创建build.yaml
func NewCreateBuildFile(setting *GlobalSetting) *CreateBuildFile {
	return &CreateBuildFile{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

const defaultBuildYaml = `
# 工程名称
name: <PROJECT_NAME>
<PROJECT_VERSION><PROJECT_TYPE>

# 镜像编译参数
docker:
  # 镜像名称
  image: path/of/project/imagename
  # 镜像编译文件(Dockerfile)
  file: Dockerfile
<BUILD_COMMAND><TEST_COMMAND><RUN_COMMAND><CLEAN_COMMAND><PUBLISH_COMMAND>
# 服务定义
service:
  # 端口定义
  ports:
  # - 908x:80

  # 环境变量
  envs:
  # - CONFIG_PROFILE=prod
  # - JWT_TOKEN={{ .Snz1dp.Jwt.Token }}
  # - JWT_PRIVKEY={{ .Snz1dp.Jwt.PrivateKey }}
  # - PG_DATABASE=demo
  # - PG_HOST="{{ .Postgres.Host }}"
  # - PG_PORT="{{ .Postgres.Port }}"
  # - PG_USER="{{ .Postgres.Admin.Username }}"
  # - PG_PASSWORD="{{ .Postgres.Admin.Password }}"
  # - CACHE_TYPE=redis
  # - REDIS_DB=6
  # - REDIS_PASSWORD="{{ .Redis.Password }}"
  # - REDIS_SERVER="{{ .Redis.Host }}:{{ .Redis.Port }}"
  # - INITIAL_USERNAME="{{ .Snz1dp.Admin.Username }}"
  # - INITIAL_PASSWORD="{{ .Snz1dp.Admin.Password }}"
  # - CONFIG_TYPE=cluster
  # - CONFIG_URL="{{ .Confserv.Web.Protocol }}://{{ .Confserv.Web.Host }}:{{ .Confserv.Web.Port }}{{ .Confserv.Web.Webroot }}"
  # - XEAI_URL="{{ .Xeai.Web.Protocol }}://{{ .Xeai.Web.Host }}:{{ .Xeai.Web.Port }}{{ .Xeai.Web.Webroot }}"

  # 心跳检查
  healthcheck:
    url: /path/health
    interval: 10s
    timeout: 10s
    period: 30s
    retries: 30

  # 路由定义
  ingress:
  - backend-port: 80
    strip-path: false
    # sso:
    # - /path/sso/api
    # anonymous:
    # - /path/anonymous/api
    # jwt:
    # - /path/jwt/api
`

const defaultDockerContent = `# 引入源镜像
#FROM path/of/your/source/imagename:tag

#ADD add src file to docker
`

// Run -
func (c *CreateBuildFile) Run() (err error) {
	var (
		setting       *GlobalSetting = c.GlobalSetting()
		curdir        string
		buildOutfile  string
		dockerOutfile string
		buildContext  string
	)
	curdir, err = os.Getwd()
	if err != nil {
		c.ErrorExit("%s", err)
	}

	buildOutfile = path.Join(curdir, "BUILD.yaml")
	dockerOutfile = path.Join(curdir, "Dockerfile")

	if _, err = os.Stat(buildOutfile); err == nil {
		if !c.Force && !utils.Confirm(fmt.Sprintf("file %s existed, proceed? (y/N)", buildOutfile), setting.InOrStdin(), setting.OutOrStdout()) {
			c.Println("Cancelled.")
			return
		}
		os.RemoveAll(buildOutfile)
	}

	buildContext = strings.ReplaceAll(defaultBuildYaml, "<PROJECT_NAME>", c.Name)
	buildContext = strings.ReplaceAll(buildContext, "<PROJECT_VERSION>", "version: "+c.Version)

	var pcmd *ProjectCommand = DefaultProjectCommand[ProjectType(c.Type)]

	if pcmd == nil {
		buildContext = strings.ReplaceAll(buildContext, "<PROJECT_TYPE>", "")

		buildContext = strings.ReplaceAll(buildContext, "<BUILD_COMMAND>", "\n\nbuild:\n"+"# - <command here>")
		buildContext = strings.ReplaceAll(buildContext, "<TEST_COMMAND>", "\n\ntest:\n"+"# - <command here>")
		buildContext = strings.ReplaceAll(buildContext, "<RUN_COMMAND>", "\n\nrun:\n"+"# - <command here>")
		buildContext = strings.ReplaceAll(buildContext, "<CLEAN_COMMAND>", "\n\nclean:\n"+"# - <command here>")
		buildContext = strings.ReplaceAll(buildContext, "<PUBLISH_COMMAND>", "\n\npublish:\n"+"# - <command here>")
	} else {
		buildContext = strings.ReplaceAll(buildContext, "<PROJECT_TYPE>", "\ntype: "+c.Type)

		buildContext = strings.ReplaceAll(buildContext, "<BUILD_COMMAND>", "")
		buildContext = strings.ReplaceAll(buildContext, "<TEST_COMMAND>", "")
		buildContext = strings.ReplaceAll(buildContext, "<RUN_COMMAND>", "")
		buildContext = strings.ReplaceAll(buildContext, "<CLEAN_COMMAND>", "")
		buildContext = strings.ReplaceAll(buildContext, "<PUBLISH_COMMAND>", "")

	}

	if err = os.WriteFile(buildOutfile, []byte(buildContext), 0644); err != nil {
		c.ErrorExit("write %s error: %s", buildOutfile, err)
		return
	}

	if _, err = os.Stat(dockerOutfile); err == nil {
		if !c.Force && !utils.Confirm(fmt.Sprintf("file %s existed, proceed? (y/N)", dockerOutfile), setting.InOrStdin(), setting.OutOrStdout()) {
			c.Println("Cancelled.")
			return
		}
		os.RemoveAll(dockerOutfile)
	}

	if pcmd == nil {
		err = os.WriteFile(dockerOutfile, []byte(defaultDockerContent), 0644)
	} else {
		err = os.WriteFile(dockerOutfile, []byte(pcmd.DockerContent), 0644)
	}

	if err != nil {
		c.ErrorExit("write %s error: %s", dockerOutfile, err)
		return
	}

	c.Println("save %s ok!", buildOutfile)
	c.Println("save %s ok!", dockerOutfile)
	c.Println("Please modify these two files manually according to the actual situation!")

	return
}
