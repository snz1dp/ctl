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
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/pkg/errors"
	"snz1.cn/snz1dp/snz1dpctl/docker"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

// RunnerAdd -
type RunnerAdd struct {
	BaseAction
	RunnerId      string
	RunnerSecret  string
	ServerURL     string
	RunnerImage   string
	WorkspacePath string
	ExtrasHosts   []string
	Envs          []string
}

type RunnerList struct {
	BaseAction
}

// NewRunnerList -
func NewRunnerList(setting *GlobalSetting) *RunnerList {
	return &RunnerList{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// NewRunnerAdd -
func NewRunnerAdd(setting *GlobalSetting) *RunnerAdd {
	return &RunnerAdd{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// RunnerStart -
type RunnerStart struct {
	BaseAction
	RunnerId       string
	RunnerSecret   string
	ServerURL      string
	RunnerImage    string
	WorkspacePath  string
	ExtrasHosts    []string
	Envs           []string
	ForcePullImage bool
}

type RunnerStop struct {
	BaseAction
	RunnerId string
	Really   bool
}

type RunnerRemove struct {
	BaseAction
	RunnerId string
	Really   bool
}

// RunnerReStart -
type RunnerReStart struct {
	BaseAction
	RunnerId string
	Really   bool
}

// RunnerLogs -
type RunnerLogs struct {
	BaseAction
	RunnerId   string
	Follow     bool
	Since      string
	Tail       string
	Timestamps bool
	Details    bool
	Until      string
}

// NewRunnerLogs -
func NewRunnerLogs(setting *GlobalSetting) *RunnerLogs {
	return &RunnerLogs{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// NewRunnerStart -
func NewRunnerStart(setting *GlobalSetting) *RunnerStart {
	return &RunnerStart{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// NewRunnerStop -
func NewRunnerStop(setting *GlobalSetting) *RunnerStop {
	return &RunnerStop{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// NewRunnerRemove -
func NewRunnerRemove(setting *GlobalSetting) *RunnerRemove {
	return &RunnerRemove{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *PipelineRunner) Stop(action Action) (err error) {

	setting := action.GlobalSetting()

	var (
		dc            *client.Client
		ct            types.Container
		ccid          string
		spinner       *utils.WaitSpinner
		containerName string
	)

	dc, err = docker.NewClient()
	if err != nil {
		return
	}
	defer dc.Close()

	containerName = fmt.Sprintf("runner-%s", s.ID)
	ct, err = docker.ContainerExisted(dc, containerName)
	if err != nil {
		action.Println("runner-%s not running", s.ID)
		err = nil
		return
	}

	ccid = ct.ID

	if ct.State != "exited" && ct.State != "created" {
		spinner = utils.NewSpinner(fmt.Sprintf("stop runner-%s...", s.ID), setting.OutOrStdout())
		dc.ContainerStop(context.Background(), ccid, nil)
		spinner.Close()
		action.Println("ok!")
	}

	dc.ContainerRemove(context.Background(), ccid, types.ContainerRemoveOptions{
		Force: true,
	})
	return
}

// Run -
func (s *RunnerStop) Run() (err error) {
	setting := s.GlobalSetting()
	var (
		ic *InstallConfiguration
	)

	// 安装配置文件
	if _, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		s.ErrorExit("load config error: %s", err)
		return
	}

	if s.RunnerId == "" {
		if len(ic.Runner) == 0 {
			s.ErrorExit("there is no runner!")
			return
		}
		if !s.Really && !utils.Confirm("will stop all runners, proceed? (y/N)", setting.InOrStdin(), setting.OutOrStdout()) {
			s.Println("Cancelled.")
			return nil
		}

		for _, v := range ic.Runner {
			if err = v.Stop(s); err != nil {
				s.ErrorExit("start runner-%d error: %s", v.ID, err)
				return
			}
		}
	} else {
		runner := ic.GetRunner(s.RunnerId)
		if runner == nil {
			s.ErrorExit("runner-%s not found", s.RunnerId)
		}
		if !s.Really && !utils.Confirm("will stop runner-"+s.RunnerId+", proceed? (y/N)", setting.InOrStdin(), setting.OutOrStdout()) {
			s.Println("Cancelled.")
			return nil
		}
		runner.Stop(s)
	}
	return
}

// Run -
func (s *RunnerRemove) Run() (err error) {
	setting := s.GlobalSetting()
	var (
		ic             *InstallConfiguration
		configFilePath string
	)

	// 安装配置文件
	if configFilePath, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		s.ErrorExit("load config error: %s", err)
		return
	}

	runner := ic.GetRunner(s.RunnerId)
	if runner == nil {
		s.ErrorExit("runner-%s not found", s.RunnerId)
	}
	if !s.Really && !utils.Confirm("will stop and remove runner-"+s.RunnerId+", proceed? (y/N)", setting.InOrStdin(), setting.OutOrStdout()) {
		s.Println("Cancelled.")
		return nil
	}

	if err = runner.Stop(s); err != nil {
		s.ErrorExit("stop runner-%d error: %s", runner.ID, err)
		return
	}

	ic.RemoveRunner(s.RunnerId)
	if err = setting.SaveLocalInstallConfiguration(ic, configFilePath); err != nil {
		s.ErrorExit("save profile to %s error: %s", configFilePath, err.Error())
		return err
	}

	return
}

// Run -
func (s *RunnerAdd) Run() (err error) {
	setting := s.GlobalSetting()
	var (
		ic             *InstallConfiguration
		configFilePath string
	)

	// 安装配置文件
	if configFilePath, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		s.ErrorExit("load config error: %s", err)
		return
	}

	var runner *PipelineRunner = nil
	for _, v := range ic.Runner {
		if v.ID == s.RunnerId {
			runner = v
			break
		}
	}

	optionAction := "update"
	if runner == nil {
		runner = new(PipelineRunner)
		ic.Runner = append(ic.Runner, runner)
		optionAction = "add"
	}

	runner.ID = s.RunnerId
	runner.Secret = s.RunnerSecret
	runner.ServerURL = s.ServerURL
	runner.DockerImage = s.RunnerImage
	runner.WorkDir = s.WorkspacePath
	runner.ExtrasHosts = s.ExtrasHosts
	runner.Envs = s.Envs

	if err = setting.SaveLocalInstallConfiguration(ic, configFilePath); err != nil {
		s.ErrorExit("save profile to %s error: %s", configFilePath, err.Error())
		return err
	}

	s.Println("%s runner-%s success!", optionAction, s.RunnerId)
	return
}

func (ic *InstallConfiguration) GetRunner(id string) *PipelineRunner {
	for _, v := range ic.Runner {
		if v.ID == id {
			return v
		}
	}
	return nil
}

func (ic *InstallConfiguration) RemoveRunner(id string) (removed bool) {
	var ridx int = -1
	for i, v := range ic.Runner {
		if v.ID == id {
			ridx = i
			removed = true
			break
		}
	}
	if removed {
		ic.Runner = append(ic.Runner[:ridx], ic.Runner[ridx+1:]...)
	}
	return
}

func (s *PipelineRunner) Start(action Action, forcePullImage bool) (err error) {
	setting := action.GlobalSetting()
	var (
		dc               *client.Client
		ct               types.Container
		ccid             string
		imageName        string
		img              *types.ImageSummary
		spinner          *utils.WaitSpinner
		config           *container.Config
		hostConfig       *container.HostConfig
		networkingConfig *network.NetworkingConfig
		cc               container.ContainerCreateCreatedBody
		volumeBinds      []string
		envVariables     []string
		containerName    string
		networkID        string
	)

	dc, err = docker.NewClient()
	if err != nil {
		return
	}
	defer dc.Close()

	volumeBinds = append(volumeBinds, "/var/run/docker.sock:/var/run/docker.sock")
	if s.WorkDir != "" {
		volumeBinds = append(volumeBinds, fmt.Sprintf("%s:/workspace", s.WorkDir))
	}

	envVariables = append(envVariables, fmt.Sprintf("CONFIG_PROFILE=%s", "prod"))
	envVariables = append(envVariables, fmt.Sprintf("RUNNER_ID=%s", s.ID))
	envVariables = append(envVariables, fmt.Sprintf("RUNNER_SECRET=%s", s.Secret))
	envVariables = append(envVariables, fmt.Sprintf("SERVER_URL=%s", s.ServerURL))

	containerName = fmt.Sprintf("runner-%s", s.ID)

	ct, err = docker.ContainerExisted(dc, containerName)
	if err == nil {
		if ct.State != "exited" && ct.State != "created" {
			action.Println("runner-%s is %s, id=%s", s.ID, ct.State, ct.ID)
			return
		}
		ccid = ct.ID
	}

	imageName = s.DockerImage
	img, err = docker.ImageExisted(dc, imageName)
	if forcePullImage || img == nil {
		var repoUsername, repoPassword string
		repoUsername, repoPassword = setting.installConfig.ResolveImageRepoUserAndPwd(imageName)
		spinner = utils.NewSpinner(fmt.Sprintf("pull %s image...", imageName), setting.OutOrStdout())
		err = docker.PullAndRenameImages(dc, imageName, "", repoUsername, repoPassword, "")
		spinner.Close()
		if err != nil {
			action.Println("failed: %v", err.Error())
			return err
		}
		action.Println("ok!")
	}

	if ccid == "" {
		config = &container.Config{
			Hostname:     "runner" + s.ID,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Image:        imageName,
			Env:          envVariables,
			// Cmd:          s.Cmd,
			Healthcheck: &container.HealthConfig{
				Test: []string{
					"CMD",
					"curl",
					"-k",
					"http://localhost/health",
				},
				Interval:    time.Second * 10,
				Timeout:     time.Second * 6,
				StartPeriod: time.Second * 10,
			},
		}

		hostConfig = &container.HostConfig{
			Binds: volumeBinds,
			RestartPolicy: container.RestartPolicy{
				Name: "always",
			},
			Privileged: true,
			ExtraHosts: s.ExtrasHosts,
		}

		networkingConfig = &network.NetworkingConfig{
			EndpointsConfig: make(map[string]*network.EndpointSettings),
		}

		networkID, err = docker.NetworkExisted(dc, "snz1dp")
		if err != nil {
			networkID, err = docker.CreateNetwork(dc, "snz1dp")
			if err != nil {
				return err
			}
		}

		networkingConfig.EndpointsConfig["snz1dp"] = &network.EndpointSettings{
			NetworkID: networkID,
		}

		cc, err = dc.ContainerCreate(context.Background(), config, hostConfig, networkingConfig, containerName)
		if err != nil {
			return
		}

		ccid = cc.ID
	}

	err = dc.ContainerStart(context.Background(), ccid, types.ContainerStartOptions{})
	if err != nil {
		return
	}

	action.Println("start runner-%s success, id=%s", s.ID, ccid)
	if len(cc.Warnings) > 0 {
		action.Println("warnning: %v", cc.Warnings)
	}
	return
}

// Run -
func (s *RunnerStart) Run() (err error) {
	setting := s.GlobalSetting()

	var (
		ic             *InstallConfiguration
		configFilePath string
	)

	// 安装配置文件
	if configFilePath, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		s.ErrorExit("load config error: %s", err)
		return
	}

	if s.RunnerId == "" {
		for _, v := range ic.Runner {
			if err = v.Start(s, s.ForcePullImage); err != nil {
				s.ErrorExit("start runner-%d error: %s", v.ID, err)
				return
			}
		}
	} else {
		saveic := false
		runner := ic.GetRunner(s.RunnerId)
		if runner == nil {
			if s.RunnerSecret == "" {
				s.ErrorExit("runner secret is required")
				return
			}

			saveic = true
			runner = new(PipelineRunner)
			runner.ID = s.RunnerId
			runner.Secret = s.RunnerSecret

			if s.ServerURL == "" {
				runner.ServerURL = ic.Snz1dp.Server.GetApiPrefix()
			} else {
				runner.ServerURL = s.ServerURL
			}
			if s.RunnerImage == "" {
				runner.DockerImage = ic.Snz1dp.RunnerConfig.DockerImage
			} else {
				runner.DockerImage = s.RunnerImage
			}

			runner.WorkDir = s.WorkspacePath
			runner.ExtrasHosts = s.ExtrasHosts
			runner.Envs = s.Envs

			ic.Runner = append(ic.Runner, runner)
		} else {
			if s.RunnerSecret != "" && runner.Secret != s.RunnerSecret {
				saveic = true
				runner.Secret = s.RunnerSecret
			}

			if runner.Secret == "" {
				s.ErrorExit("runner secret is required")
				return
			}

			if s.RunnerImage != "" && runner.DockerImage != s.RunnerImage {
				saveic = true
				runner.DockerImage = s.RunnerImage
			}
			if s.ServerURL != "" && runner.ServerURL != s.ServerURL {
				saveic = true
				runner.ServerURL = s.ServerURL
			}
			if s.WorkspacePath != "" && runner.WorkDir != s.WorkspacePath {
				saveic = true
				runner.WorkDir = s.WorkspacePath
			}
			if len(s.ExtrasHosts) > 0 {
				saveic = true
				runner.ExtrasHosts = s.ExtrasHosts
			}
			if len(s.Envs) > 0 {
				saveic = true
				runner.Envs = s.Envs
			}
		}
		if saveic {
			// 保存配置文件
			if err = setting.SaveLocalInstallConfiguration(ic, configFilePath); err != nil {
				s.ErrorExit("save profile to %s error: %s", configFilePath, err.Error())
				return
			}
			// 安装配置文件
			if _, _, err = setting.LoadLocalInstallConfiguration(); err != nil {
				s.ErrorExit("load config error: %s", err)
				return
			}
		}
		err = runner.Start(s, s.ForcePullImage)
	}

	return
}

// Run -
func (s *RunnerLogs) Run() (err error) {
	setting := s.GlobalSetting()
	var (
		dc            *client.Client
		oc            io.ReadCloser
		ct            types.Container
		containerName string
		c             types.ContainerJSON
	)

	dc, err = docker.NewClient()
	if err != nil {
		s.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	containerName = fmt.Sprintf("runner-%s", s.RunnerId)

	ct, err = docker.ContainerExisted(dc, containerName)
	if err != nil {
		s.ErrorExit("%v", err)
		return
	}

	c, err = dc.ContainerInspect(context.Background(), ct.ID)
	if err != nil {
		s.ErrorExit("%v", err)
		return
	}

	oc, err = dc.ContainerLogs(context.Background(), ct.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     s.Follow,
		Since:      s.Since,
		Tail:       s.Tail,
		Timestamps: s.Timestamps,
		Details:    s.Details,
		Until:      s.Until,
	})

	if err != nil {
		s.ErrorExit("%v", err)
		return
	}

	defer oc.Close()

	if c.Config.Tty {
		_, err = io.Copy(setting.OutOrStdout(), oc)
	} else {
		_, err = stdcopy.StdCopy(setting.OutOrStdout(), setting.ErrOrStderr(), oc)
	}
	return
}

type RunnerExecCmd struct {
	BaseAction
	RunnerId    string
	Tty         bool
	Interactive bool
	Detach      bool
	Cmd         []string
	Env         []string
}

func NewRunnerExecCmd(setting *GlobalSetting) *RunnerExecCmd {
	return &RunnerExecCmd{
		BaseAction: BaseAction{
			setting: setting,
		},
	}
}

// Run -
func (s *RunnerExecCmd) Run() (err error) {
	setting := s.GlobalSetting()

	var (
		dc            *client.Client
		ct            types.Container
		containerName string
		response      types.IDResponse
		execID        string
		execResp      types.HijackedResponse
	)

	dc, err = docker.NewClient()
	if err != nil {
		s.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	containerName = fmt.Sprintf("runner-%s", s.RunnerId)
	ct, err = docker.ContainerExisted(dc, containerName)
	if err != nil {
		return
	}

	response, err = dc.ContainerExecCreate(context.Background(), ct.ID, types.ExecConfig{
		Privileged:   false,
		AttachStdin:  s.Interactive,
		AttachStdout: s.Tty,
		AttachStderr: s.Tty,
		Cmd:          s.Cmd,
		Tty:          s.Tty,
		Detach:       s.Detach,
		Env:          s.Env,
	})

	if err != nil {
		return
	}

	if s.Detach {
		return
	}
	execID = response.ID
	if execID == "" {
		err = errors.Errorf("exec ID empty")
		return
	}

	execResp, err = dc.ContainerExecAttach(context.Background(), execID, types.ExecStartCheck{
		Detach: s.Detach,
		Tty:    false,
	})

	if err != nil {
		return
	}

	defer execResp.Close()

	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)
		errCh <- func() error {
			streamer := hijackedIOStreamer{
				inputStream:  io.ReadCloser(setting.InOrStdin()),
				outputStream: setting.OutOrStdout(),
				errorStream:  setting.ErrOrStderr(),
				resp:         execResp,
				tty:          s.Tty,
				detachKeys:   "",
			}

			return streamer.stream(context.Background())
		}()
	}()

	if err = <-errCh; err != nil {
		err = errors.Errorf("Error hijack: %s", err)
		return
	}

	resp, err := dc.ContainerExecInspect(context.Background(), execID)
	if err != nil {
		err = errors.Errorf("%s", err)
		return
	}
	status := resp.ExitCode
	if status != 0 {
		err = errors.Errorf("status is %d", status)
	} else {
		err = nil
	}
	return
}

func (s *RunnerList) Run() (err error) {
	setting := s.GlobalSetting()
	var (
		ic            *InstallConfiguration
		dc            *client.Client
		ct            types.Container
		containerName string
	)

	// 安装配置文件
	if _, ic, err = setting.LoadLocalInstallConfiguration(); err != nil {
		s.ErrorExit("load config error: %s", err)
		return
	}

	if len(ic.Runner) == 0 {
		s.ErrorExit("there is no runner!")
		return
	}

	dc, err = docker.NewClient()
	if err != nil {
		s.ErrorExit("%v", err)
		return
	}
	defer dc.Close()

	for _, v := range ic.Runner {
		state := "not running"
		containerName = fmt.Sprintf("runner-%s", v.ID)
		ct, err = docker.ContainerExisted(dc, containerName)
		if err != nil {
			err = nil
		} else {
			state = ct.State
		}
		s.Println("runner-%s is %s, server=%s", v.ID, state, v.ServerURL)
	}

	return
}
