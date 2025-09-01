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
	flag "github.com/spf13/pflag"
	"snz1.cn/snz1dp/snz1dpctl/action"
)

func newShowProfile(setting *action.GlobalSetting) *cobra.Command {
	ac := action.NewShowProfile(setting)
	cmd := &cobra.Command{
		Use:   "show",
		Short: "show local snz1dp config",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ac.Run()
		},
	}
	return cmd
}

func setupProfileFlags(fs *flag.FlagSet, a *action.SetProfile) {
	fs.StringVar(&a.GlobalSetting().StorageClass, "storage-class", action.DefaultStorageClass, "specifies the Kubernetes persistent volumn class")

	fs.StringVar(&a.Username, "username", "", "username of snz1dp repo")
	fs.StringVar(&a.Password, "password", "", "password of snz1dp repo")

	fs.StringVar(&a.DockerRepo, "docker-repo", "", "url of snz1dp docker registry")
	fs.StringVar(&a.DockerUsername, "docker-username", "", "username of snz1dp docker registry")
	fs.StringVar(&a.DockerPassword, "docker-password", "", "password of snz1dp docker registry")

	fs.StringVar(&a.HelmRepo, "helm-url", "", "url of snz1dp helm repo")
	fs.StringVar(&a.HelmRepoName, "helm-repo", "", "name of snz1dp helm repo")
	fs.StringVar(&a.HelmUsername, "helm-username", "", "username of snz1dp helm repo")
	fs.StringVar(&a.HelmPassword, "helm-password", "", "password of snz1dp helm repo")

	fs.StringVar(&a.MavenRepoID, "maven-repo-id", "", "id of snz1dp maven repo")
	fs.StringVar(&a.MavenRepoURL, "maven-repo-url", "", "url of snz1dp maven repo")
	fs.StringVar(&a.MavenUsername, "maven-username", "", "username of snz1dp maven repo")
	fs.StringVar(&a.MavenPassword, "maven-passsword", "", "password of snz1dp maven repo")

	fs.StringVar(&a.NpmRepoID, "npm-repo-id", "", "id of snz1dp npm repo")
	fs.StringVar(&a.NpmRepoURL, "npm-repo-url", "", "url of snz1dp npm repo")
	fs.StringVar(&a.NpmUsername, "npm-username", "", "username of snz1dp npm repo")
	fs.StringVar(&a.NpmPassword, "npm-password", "", "password of snz1dp npm repo")
	fs.StringVar(&a.SassBinarySite, "sass-binary-site", "", "sass-binary-site of snz1dp npm repo")
	fs.StringVar(&a.SassSiteID, "sass-site-id", "", "sass-binary-site of npm repo id")
}

func newSetProfile(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewSetProfile(setting)
	cmd := &cobra.Command{
		Use:     "setup",
		Short:   "setup snz1dp install profile type",
		Long:    globalBanner,
		Example: action.BaseConfig.Snz1dp.Ctl.Name + " profile setup [minimal | production | normal | ...]",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Usage()
			}
			switch action.ProfileType(args[0]) {
			case action.ProductionProfile:
				s.ConfigType = action.ProductionProfile
			case action.NormalProfile:
				s.ConfigType = action.NormalProfile
			case action.MinimalProfile:
				s.ConfigType = action.MinimalProfile
			default:
				s.ConfigType = action.ProfileType(args[0])
			}
			return s.Run()
		},
	}
	setupProfileFlags(cmd.Flags(), s)
	return cmd
}

func newLoginRegistry(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewLoginRegistry(setting)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "login snz1dp repo",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				s.RepoType = args[0]
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()

	fs.StringVar(&s.RepoURL, "repo-url", "", "url of snz1dp repo")
	fs.StringVar(&s.Username, "username", "", "username of snz1dp repo")
	fs.StringVar(&s.Password, "password", "", "password of snz1dp repo")

	fs.StringVar(&s.DockerRepo, "docker-repo", "", "url of snz1dp docker registry")
	fs.StringVar(&s.DockerUsername, "docker-username", "", "username of snz1dp docker registry")
	fs.StringVar(&s.DockerPassword, "docker-password", "", "password of snz1dp docker registry")

	fs.StringVar(&s.HelmRepo, "helm-url", "", "url of snz1dp helm repo")
	fs.StringVar(&s.HelmRepoName, "helm-repo", "", "name of snz1dp helm repo")
	fs.StringVar(&s.HelmUsername, "helm-username", "", "username of snz1dp helm repo")
	fs.StringVar(&s.HelmPassword, "helm-password", "", "password of snz1dp helm repo")

	fs.StringVar(&s.MavenRepoID, "maven-repo-id", "", "id of snz1dp maven repo")
	fs.StringVar(&s.MavenRepoURL, "maven-repo-url", "", "url of snz1dp maven repo")
	fs.StringVar(&s.MavenUsername, "maven-username", "", "username of snz1dp maven repo")
	fs.StringVar(&s.MavenPassword, "maven-password", "", "password of snz1dp maven repo")

	fs.StringVar(&s.NpmRepoID, "npm-repo-id", "", "id of snz1dp npm repo")
	fs.StringVar(&s.NpmRepoURL, "npm-repo-url", "", "url of snz1dp npm repo")
	fs.StringVar(&s.NpmUsername, "npm-username", "", "username of snz1dp npm repo")
	fs.StringVar(&s.NpmPassword, "npm-password", "", "password of snz1dp npm repo")
	fs.StringVar(&s.SassBinarySite, "sass-binary-site", "", "sass-binary-site of snz1dp npm repo")
	fs.StringVar(&s.SassSiteID, "sass-site-id", "", "sass-binary-site of npm repo id")

	fs.BoolVar(&s.PromptInput, "prompt-input", false, "prompt input user and password")

	return cmd
}

func newInstallBundle(setting *action.GlobalSetting) *cobra.Command {
	i := action.NewInstallBundle(setting)
	cmd := &cobra.Command{
		Use:     "add",
		Short:   "add bundle to snz1dp install profile",
		Long:    globalBanner,
		Example: action.BaseConfig.Snz1dp.Ctl.Name + " profile install [bundle name | bundle file ...]",
		RunE: func(cmd *cobra.Command, args []string) error {
			if i.From == "" && len(args) == 0 || i.From != "" && len(args) > 0 || len(args) > 1 {
				return cmd.Usage()
			} else if i.From == "" {
				i.From = args[0]
			}
			return i.Run()
		},
	}
	fs := cmd.Flags()
	fs.StringVarP(&i.Namespace, "namespace", "n", action.DefaultAppNS, "namespace scope for install bundle")
	fs.StringVarP(&i.Name, "name", "", "", "name of app bundle")
	fs.StringArrayVarP(&i.Envs, "env", "e", []string{}, "set environment variables")
	fs.StringArrayVarP(&i.Bind, "bind", "b", []string{}, "bind host port to docker port (host:container)")
	fs.StringArrayVarP(&i.HostAlias, "host", "H", []string{}, "add a custom host-to-IP mapping (host:ip)")
	fs.StringArrayVarP(&i.Volume, "volume", "v", []string{}, "bind mount a volume")
	fs.StringArrayVarP(&i.Command, "command", "c", []string{}, "set command for container")
	fs.StringArrayVarP(&i.RunFiles, "file", "f", []string{}, "set file for container")
	fs.StringArrayVarP(&i.HealthCommand, "health-command", "", []string{}, "set healthcheck command for container")
	fs.StringVarP(&i.HealthURL, "health-url", "", "", "set healthcheck url for container")
	fs.StringVarP(&i.Runtime, "runtime", "", "", "set runtime for container")
	fs.StringVarP(&i.DockerImage, "image", "", "", "set image for container")
	fs.StringVarP(&i.GPU, "gpus", "", "", "set gpu for container")
	fs.BoolVarP(&i.PortDisabled, "no-listen", "", false, "disable listen host port")
	fs.BoolVarP(&i.Overlay, "overlay", "y", false, "force overlay config of bundle")

	return cmd
}

func newExportProfileCmd(setting *action.GlobalSetting) *cobra.Command {
	i := action.NewExportProfile(setting)
	cmd := &cobra.Command{
		Use:   "export",
		Short: "export snz1dp install profile",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return cmd.Usage()
			} else if i.OutputFile == "" && len(args) == 1 {
				i.OutputFile = args[0]
			}
			return i.Run()
		},
	}
	fs := cmd.Flags()
	fs.StringVarP(&i.OutputFile, "output-file", "o", "", "config output file name")
	fs.StringVarP(&i.OutputPassword, "output-password", "p", "", "password for output")
	fs.BoolVarP(&i.PlainPassword, "plain-password", "t", false, "show plain password")
	fs.BoolVarP(&i.DetailConfig, "detail-config", "d", false, "detail config for bundle")
	fs.BoolVarP(&i.LoginConfig, "login-config", "l", false, "export login config")

	return cmd
}

func newShowJwtInfoCmd(setting *action.GlobalSetting) *cobra.Command {
	i := action.NewJwtExport(setting)
	cmd := &cobra.Command{
		Use:   "jwt",
		Short: "export jwt of ingress install profile",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return i.Run()
		},
	}
	// fs := cmd.Flags()

	return cmd
}

func newRemoveBundle(setting *action.GlobalSetting) *cobra.Command {
	r := action.NewRemoveBundle(setting)
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "remove bundle from snz1dp install profile",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(r.Name) == 0 && len(args) == 0 {
				return cmd.Usage()
			}
			r.Name = append(r.Name, args...)
			return r.Run()
		},
	}
	fs := cmd.Flags()
	fs.StringArrayVarP(&r.Name, "name", "n", []string{}, "name of bundle for snz1dp install profile")
	return cmd
}

func newListBundle(setting *action.GlobalSetting) *cobra.Command {
	l := action.NewListBundle(setting)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list bundles of snzdp install profile",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return l.Run()
		},
	}
	return cmd
}

func newProfileCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "snz1dp install profile commands",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newShowProfile(setting),
		newSetProfile(setting),
		newInstallBundle(setting),
		newRemoveBundle(setting),
		newListBundle(setting),
		newLoginRegistry(setting),
		newExportProfileCmd(setting),
		newShowJwtInfoCmd(setting),
	)
	return cmd
}
