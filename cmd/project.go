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

package cmd

import (
	"github.com/spf13/cobra"
	"snz1.cn/snz1dp/snz1dpctl/action"
)

func newProjectInitCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewProjectInit(setting)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "init snz1dp cloud native project",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.Run()
		},
	}
	return cmd
}

func newProjectBuildCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewProjectBuild(setting)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "build snz1dp cloud native project",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.Run()
		},
	}
	return cmd
}

func newProjectPublishCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewProjectPublish(setting)
	var (
		labels    ListOpts = NewListOpts(ValidateEnv)
		buildArgs ListOpts = NewListOpts(ValidateEnv)
	)

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "publish snz1dp cloud native project",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			s.BuildArgs = buildArgs.GetAll()
			s.Labels = labels.GetAll()
			return s.Run()
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&s.SkipBuildDocker, "skip-build-docker", false, "skip build docker")
	flags.BoolVar(&s.SkipBuildSource, "skip-build-source", false, "skip build source")
	flags.BoolVar(&s.SkipPackage, "skip-package", false, "skip package")
	flags.Var(&buildArgs, "build-arg", "Set build-time variables")
	flags.Var(&labels, "label", "Set metadata for an image")
	flags.StringVar(&s.HelmRepo, "helm-repo", "", "Helm repo name of publish")
	flags.StringVar(&s.HelmUserName, "helm-user", "", "Helm repo username of publish")
	flags.StringVar(&s.HelmUserPwd, "helm-pwd", "", "Helm repo password of publish")

	return cmd
}

func newProjectRunCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewProjectRun(setting)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "run snz1dp cloud native project",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.Run()
		},
	}
	return cmd
}

func newProjectTestCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewProjectTest(setting)

	cmd := &cobra.Command{
		Use:   "test",
		Short: "test snz1dp cloud native project",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.Run()
		},
	}
	return cmd
}

func newProjectPackCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewProjectPackage(setting)

	cmd := &cobra.Command{
		Use:   "package",
		Short: "package snz1dp cloud native project",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.Run()
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&s.SkipBuildDocker, "skip-build-docker", false, "skip build docker")
	flags.BoolVar(&s.SkipBuildSource, "skip-build-source", false, "skip build source code")
	flags.BoolVar(&s.PushDockerImage, "push-docker-image", false, "push docker image to repository")
	flags.BoolVar(&s.SaveDockerImage, "save-docker-image", false, "save docker image file to directory")
	flags.StringVarP(&s.DockerImagePlatform, "docker-image-platform", "p", "", "platform of docker image file")

	return cmd
}

func newProjectCleanmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewProjectClean(setting)

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "clean snz1dp cloud native project",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.Run()
		},
	}
	return cmd
}

func newProjectSonar(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewMakeSonar(setting)

	cmd := &cobra.Command{
		Use:   "sonar",
		Short: "sonar scan snz1dp cloud native project",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.Run()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&s.SonarURL, "url", "", "", "sonar host url")
	flags.StringVarP(&s.Token, "token", "p", "", "access token for sonar")

	return cmd
}

func newProjectInstallCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewMakeInstall(setting)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "install to snz1dp install profile",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.Run()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&s.Namespace, "namespace", "n", action.DefaultAppNS, "namespace scope for install bundle")
	flags.BoolVarP(&s.Overlay, "overlay", "y", false, "force overlay config of bundle")

	return cmd
}

func newProjectDockerCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewProjectDocker(setting)

	var (
		labels    ListOpts = NewListOpts(ValidateEnv)
		buildArgs ListOpts = NewListOpts(ValidateEnv)
		sshKeys   ListOpts = NewListOpts(ValidateEnv)
	)

	cmd := &cobra.Command{
		Use:   "docker",
		Short: "docker build snz1dp cloud native project",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			s.BuildArgs = ConvertKVStringsToMapWithNil(buildArgs.GetAll())
			s.Labels = ConvertKVStringsToMap(labels.GetAll())
			s.SSHKey = sshKeys.GetAll()
			return s.Run()
		},
	}

	flags := cmd.Flags()
	flags.Var(&buildArgs, "build-arg", "Set build-time variables")
	flags.Var(&sshKeys, "ssh", "SSH agent socket or keys to expose to the build")
	flags.Var(&labels, "label", "Set metadata for an image")

	flags.StringVarP(&s.Platform, "platform", "p", "", "build platform of docker image")
	flags.BoolVar(&s.SkipBuildSource, "skip-build-source", false, "skip build source code")
	flags.BoolVar(&s.Push, "push", false, "push docker image to repository")
	flags.StringVarP(&s.Dockerfile, "dockerfile", "f", "", "name of the Dockerfile")
	flags.StringVarP(&s.ImageTag, "tag", "t", "", "tag of docker image")

	return cmd
}

func newProjectStandaloneCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewProjectStandalone(setting)

	var (
		labels    ListOpts = NewListOpts(ValidateEnv)
		buildArgs ListOpts = NewListOpts(ValidateEnv)
	)

	cmd := &cobra.Command{
		Use:     "standalone",
		Aliases: []string{"alone"},
		Short:   "control standalone service of snz1dp cloud native project",
		Long:    globalBanner,
		Args:    cobra.MaximumNArgs(1),
		Example: action.BaseConfig.Snz1dp.Ctl.Name + " make standalone [start (default) | stop | stop-all | clean | clean-all | develop | apply]",
		RunE: func(cmd *cobra.Command, args []string) error {
			var aname string
			if len(args) == 0 {
				aname = "start"
			} else if len(args) == 1 {
				aname = args[0]
			}
			s.Command = aname
			s.BuildArgs = ConvertKVStringsToMapWithNil(buildArgs.GetAll())
			s.Labels = ConvertKVStringsToMap(labels.GetAll())
			return s.Run()
		},
	}

	flags := cmd.Flags()
	flags.Var(&buildArgs, "build-arg", "Set build-time variables")
	flags.Var(&labels, "label", "Set metadata for an image")

	flags.StringVar(&s.Since, "since", "", "show logs since timestamp  (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
	flags.StringVar(&s.Tail, "tail ", "all", "number of lines to show from the end of the logs")
	flags.BoolVarP(&s.Follow, "follow", "f", false, "Follow log output")
	flags.BoolVar(&s.Details, "details", false, "show extra details provided to logs")
	flags.BoolVarP(&s.Timestamps, "timestamps", "t", false, "show extra details provided to logs")
	flags.StringVar(&s.Until, "until", "", "Show logs before a timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
	flags.BoolVarP(&s.Really, "really", "y", false, "really to execute command.")

	flags.BoolVarP(&s.ForcePullImage, "force-pull", "p", false, "force pull bundle image")
	flags.BoolVar(&s.LoadImageLocal, "load-image", false, "load local image to docker")
	flags.BoolVar(&s.SkipBuildDocker, "skip-build-docker", false, "skip build docker")

	return cmd
}

func newShowBundleInfoCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewGetBundleInfo(setting)
	cmd := &cobra.Command{
		Use:   "info",
		Short: "show make project information",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}
	fs := cmd.Flags()
	fs.BoolVarP(&a.ShowName, "name", "n", true, "show project name.")
	fs.BoolVar(&a.ShowVersion, "version", false, "show project version.")
	return cmd
}

func newProjectCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "make",
		Short: "make command of snz1dp cloud native project",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newProjectInitCmd(setting),
		newProjectBuildCmd(setting),
		newProjectRunCmd(setting),
		newProjectTestCmd(setting),
		newProjectPackCmd(setting),
		newProjectDockerCmd(setting),
		newProjectPublishCmd(setting),
		newProjectStandaloneCmd(setting),
		newProjectInstallCmd(setting),
		newProjectCleanmd(setting),
		newProjectSonar(setting),
		newShowBundleInfoCmd(setting),
	)
	return cmd
}
