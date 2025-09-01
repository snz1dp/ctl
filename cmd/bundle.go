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

func newDownloadBundleCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewDownloadBundle(setting)
	cmd := &cobra.Command{
		Use:   "download",
		Short: "downoad snz1dp bundle files to local machine.",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Bundle = append(a.Bundle, args...)
			return a.Run()
		},
	}
	fs := cmd.Flags()

	fs.BoolVarP(&a.Force, "force", "f", false, "force to download snz1dp bundle")
	fs.BoolVarP(&a.PullImage, "docker-image", "d", false, "download docker image")
	fs.BoolVarP(&a.All, "additional", "a", false, "bundle of additional docker image")
	fs.StringArrayVarP(&a.Bundle, "name", "n", []string{}, "bundle name")
	fs.StringVarP(&a.Platform, "platform", "p", "", "platform name: linux/amd64 or linux/arm64")

	return cmd
}

func newLoadBundleImageCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewLoadBundleImage(setting)
	cmd := &cobra.Command{
		Use:   "load",
		Short: "load snz1dp bundle docker image files",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Bundle = append(a.Bundle, args...)
			return a.Run()
		},
	}
	fs := cmd.Flags()
	fs.BoolVarP(&a.Force, "force", "f", false, "force to load snz1dp bundle docker image")
	fs.BoolVarP(&a.All, "additional", "a", false, "bundle of additional docker image")
	fs.StringArrayVarP(&a.Bundle, "name", "n", []string{}, "bundle name")
	return cmd
}

func newTarballBundleCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewTargzBundle(setting)
	cmd := &cobra.Command{
		Use:   "package",
		Short: "package and gzip all bundle files of snz1dp install profile",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}
	fs := cmd.Flags()
	fs.StringVarP(&a.Destfile, "outfile", "o", "", "output targz file(*.tgz)")
	fs.BoolVarP(&a.Force, "force", "f", false, "force to download snz1dp bundle")
	fs.BoolVarP(&a.PullImage, "docker-image", "d", false, "download docker image")
	fs.BoolVarP(&a.All, "additional", "a", false, "bundle of additional docker image")
	fs.StringVarP(&a.Platform, "platform", "p", "", "platform name: linux/amd64 or linux/arm64")
	return cmd
}

func newCleanBundleCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewCleanBundle(setting)
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "clean bundle files and logs, or config, rundata.",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Bundle = append(a.Bundle, args...)
			return a.Run()
		},
	}
	fs := cmd.Flags()
	fs.BoolVarP(&a.Really, "really", "y", false, "really to execute command.")
	fs.BoolVar(&a.Config, "config", false, "delete all bundle config file.")
	fs.BoolVar(&a.RunData, "rundata", false, "delete all bundle rundata directory.")
	fs.StringArrayVarP(&a.Bundle, "name", "n", []string{}, "bundle name")
	return cmd
}

func newBundleCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "snz1dp bundle tool commands",
		Long:  globalBanner,
	}

	cmd.AddCommand(
		newDownloadBundleCmd(setting),
		newTarballBundleCmd(setting),
		newCleanBundleCmd(setting),
		newLoadBundleImageCmd(setting),
		newBundleSearchCmd(setting),
		newBundleListCmd(setting),
		newBundleImageCmd(setting),
	)

	return cmd
}

func newBundleListCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewBundleList(setting)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list all snz1dp bundle files",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}
	return cmd
}

func newBundleSearchCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewSearchBundle(setting)
	cmd := &cobra.Command{
		Use:   "search",
		Short: "search snz1dp bundle files",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && a.SearchKey == "" {
				return cmd.Usage()
			} else if len(args) == 1 && a.SearchKey == "" {
				a.SearchKey = args[0]
			} else if len(args) > 1 && a.SearchKey != "" {
				return cmd.Usage()
			}
			return a.Run()
		},
	}
	fs := cmd.Flags()
	fs.StringVarP(&a.SearchKey, "name", "n", "", "bundle name")
	return cmd
}

func newBundleImageCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewShowBundleImages(setting)
	cmd := &cobra.Command{
		Use:   "images",
		Short: "show snz1dp bundle docker images",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			a.BundleNames = append(a.BundleNames, args...)
			return a.Run()
		},
	}
	fs := cmd.Flags()
	fs.StringArrayVarP(&a.BundleNames, "name", "n", []string{}, "bundle name")
	return cmd
}
