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
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"snz1.cn/snz1dp/snz1dpctl/action"
	"snz1.cn/snz1dp/snz1dpctl/utils"
)

var (
	websiteURL   = "https://alidocs.dingtalk.com/i/nodes/1DKw2zgV2vkxoOzgf1rDyzwwVB5r9YAn?utm_scene=team_space"
	globalBanner = `  ____            _ ____  ____
 / ___| _ __  ___/ |  _ \|  _ \
 \___ \| '_ \|_  / | | | | |_) |
  ___) | | | |/ /| | |_| |  __/
 |____/|_| |_/___|_|____/|_|
                           1.0
  Snz1DP command line utility

`
)

func newDocCmd(setting *action.GlobalSetting) *cobra.Command {
	var cmd = &cobra.Command{
		Use:          "doc",
		Short:        "open snz1dp doc website",
		Long:         globalBanner,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			writer := setting.OutOrStdout()
			fmt.Fprintf(writer, "snz1dp doc website:\n%s\n", websiteURL)
			err := utils.OpenBrowser(websiteURL)
			if err != nil {
				fmt.Fprintf(writer, "Failed to open browser; open %s in your browser.\n", websiteURL)
			}
			return err
		},
	}
	return cmd
}

func newRootCmd(setting *action.GlobalSetting, args []string) *cobra.Command {
	// 初始化基本内容
	action.InitBase()

	websiteURL = action.BaseConfig.Website
	globalBanner = action.BaseConfig.Banner

	var rootCmd = &cobra.Command{
		Use:          action.BaseConfig.Snz1dp.Ctl.Name,
		Short:        "snz1dp controller",
		Long:         globalBanner,
		SilenceUsage: true,
	}

	// 全局参数
	flags := rootCmd.PersistentFlags()
	setting.AddFlags(rootCmd.PersistentFlags())

	flags.ParseErrorsWhitelist.UnknownFlags = true
	flags.Parse(args)

	// 子命令
	rootCmd.AddCommand(
		// K8s命令
		// newK8sCmd(setting),

		//文档命令
		newDocCmd(setting),
		//版本
		newVersionCmd(setting),
		//打包
		newBundleCmd(setting),
		//配置
		newProfileCmd(setting),
		//控制台: TODO: 准备修改为： snz1dpctl k8s forward <bundle> | [...]
		// newDashboardCmd(setting),
		//HaProxy代理
		// newHaProxyCmd(setting),
		//更新
		newUpgradeCliCmd(setting),
		//kubectl
		newStartKubectlCmd(setting),
		// helm
		newStartHelmCmd(setting),
		//Standalone
		newStandaloneCmd(setting),
		//project
		newProjectCmd(setting),
		//create
		newCreateCmd(setting),
		// ingress
		newIngressCmd(setting),
		// node
		newStartNodeCmd(setting),
		// nrm
		newStartNrmCmd(setting),
		// npm
		newStartNpmCmd(setting),
		// runner
		newRunnerCmd(setting),
		// nvm
		newStartNvmCmd(setting),
	)

	// Linux下添加keepalived命令
	switch runtime.GOOS {
	case "linux":
		rootCmd.AddCommand(
			//keepalived
			newKeepalivedCmd(setting),
		)
	}

	return rootCmd
}

// Execute - 启动
func Execute(in io.ReadCloser, out io.Writer, err io.Writer, args []string) {
	cwd, _ := os.Getwd()
	startTime := time.Now()
	setting := action.NewGlobalSetting(cwd, in, out, err, args, startTime)
	rootCmd := newRootCmd(setting, args)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
