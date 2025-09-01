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

func newCreateHelmChartCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewCreateHelmChart(setting)

	cmd := &cobra.Command{
		Use:   "chart",
		Short: "create helm chart for snz1dp cloud native project",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.Run()
		},
	}
	return cmd
}

func newCreateMavenWrapperCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewCreateMavenWrapper(setting)

	cmd := &cobra.Command{
		Use:   "mvnw",
		Short: "create maven wrapper for snz1dp cloud native project",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.Run()
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&s.Force, "force", "f", false, "force download depends files.")

	return cmd
}

func newCreateJavaBackend(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewCreateJavaBackend(setting)

	cmd := &cobra.Command{
		Use:   "java",
		Short: "use template create java maven project",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if s.Name == "" && len(args) == 0 || len(args) > 1 {
				return cmd.Usage()
			} else if s.Name == "" && len(args) == 1 {
				s.Name = args[0]
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()
	fs.StringVarP(&s.Group, "group", "g", "", "java group name(maven groupId).")
	fs.StringVarP(&s.Name, "name", "n", "", "java artifact (maven artifactId).")
	fs.StringVarP(&s.Package, "package", "p", "", "java package (package).")
	fs.StringVarP(&s.Version, "version", "v", "1.0.0", "maven project version (package).")
	fs.BoolVarP(&s.Force, "force", "f", false, "force download depends files.")

	return cmd
}

func newCreateVueProject(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewCreateVueProject(setting)

	cmd := &cobra.Command{
		Use:   "vue",
		Short: "use template create vue project",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if s.Name == "" && len(args) == 0 || len(args) > 1 {
				return cmd.Usage()
			} else if s.Name == "" && len(args) == 1 {
				s.Name = args[0]
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()
	fs.StringVarP(&s.Name, "name", "n", "", "name of vue project.")
	fs.StringVarP(&s.Path, "path", "p", "", "web path of vue project.")
	fs.StringVarP(&s.Api, "api", "a", "", "api url of backend for dev.")
	fs.StringVarP(&s.Version, "version", "v", "1.0.0", "vue project version.")
	fs.BoolVarP(&s.Force, "force", "f", false, "force download depends files.")

	return cmd
}

func newCreateK8sDeployYaml(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewCreateK8sDeployYaml(setting)
	cmd := &cobra.Command{
		Use:   "deploy.yaml",
		Short: "use template create deploy.yaml for kubernetes",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.Run()
		},
	}
	fs := cmd.Flags()
	fs.BoolVarP(&s.Force, "force", "f", false, "force create file.")
	return cmd
}

func newCreateBuildFileCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewCreateBuildFile(setting)
	cmd := &cobra.Command{
		Use:   "build",
		Short: "use template create build file for snz1dp project",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return cmd.Usage()
			} else if s.Name == "" && len(args) == 0 {
				return cmd.Usage()
			} else if len(args) == 1 {
				s.Name = args[0]
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&s.Force, "force", "f", false, "force create file.")
	fs.StringVarP(&s.Name, "name", "n", "", "project name.")
	fs.StringVarP(&s.Version, "version", "v", "1.0.0", "project version.")
	fs.StringVarP(&s.Type, "type", "t", "", "project type(maven, quasar, vue).")
	return cmd
}

func newCreateRSAKeyPairCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewCreateRSAKeyPair(setting)
	cmd := &cobra.Command{
		Use:   "rsa",
		Short: "create rsa keypair",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return cmd.Usage()
			} else if len(args) == 1 {
				s.Name = args[0]
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()
	fs.StringVarP(&s.Name, "output", "o", "", "pem file name of rsa private key.")
	fs.StringVarP(&s.Public, "public", "p", "", "pem file name of rsa public key.")
	fs.IntVarP(&s.Bits, "bits", "b", 1024, "rsa key bits.")
	fs.BoolVarP(&s.Force, "force", "y", false, "force overidde existed file.")
	return cmd
}

func newCreateCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create command of snz1dp cloud native project",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newCreateHelmChartCmd(setting),
		newCreateMavenWrapperCmd(setting),
		newCreateJavaBackend(setting),
		newCreateVueProject(setting),
		newCreateK8sDeployYaml(setting),
		newCreateBuildFileCmd(setting),
		newCreateRSAKeyPairCmd(setting),
	)
	return cmd
}
