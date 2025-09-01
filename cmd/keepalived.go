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

func newReStartKeepalivedCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewStopKeepalived(setting)
	a := action.NewStartKeepalived(setting)
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "restart keepalived service",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			s.Run()
			return a.Run()
		},
	}
	fs := cmd.Flags()
	fs.BoolVar(&a.GenerateConfig, "generate-config", true, "generate keepalived config and script file")
	fs.BoolVarP(&a.ForcePullImage, "force-pull", "f", false, "force pull keepalived image")

	return cmd
}

func newConfigCheckKeepAlived(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewConfigLiveDetection(setting)
	cmd := &cobra.Command{
		Use:   "check-config",
		Short: "keepalived check config",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.IP == "" || a.Port <= 0 {
				return cmd.Usage()
			}
			return a.Run()
		},
	}

	fs := cmd.Flags()

	fs.StringVar(&a.IP, "ip", "", "local ip for alive check")
	fs.Uint32VarP(&a.Port, "port", "p", 0, "local port for alive check")
	fs.Uint32Var(&a.Fall, "fall", 3, "times of check return error to KO")
	fs.Uint32Var(&a.Rise, "rise", 3, "times of check return success to OK")
	fs.Uint32Var(&a.Timeout, "timeout", 3, "check timeout seconds")
	fs.Uint32Var(&a.Interval, "interval", 3, "check interval seconds")

	return cmd
}

func newStopKeepalivedCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewStopKeepalived(setting)
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "stop keepalived service",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}
	return cmd
}

func newListKeepalivedCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewListVirtualIP(setting)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list keepalived virtual ip",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}
	return cmd
}

func newKeepalivedLogCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewLogKeepalived(setting)

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "fetch logs of keepalived",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&a.Since, "since", "", "show logs since timestamp  (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
	fs.StringVar(&a.Tail, "tail ", "all", "number of lines to show from the end of the logs")
	fs.BoolVarP(&a.Follow, "follow", "f", false, "Follow log output")
	fs.BoolVar(&a.Details, "details", false, "show extra details provided to logs")
	fs.BoolVarP(&a.Timestamps, "timestamps", "t", false, "show extra details provided to logs")
	fs.StringVar(&a.Until, "until", "", "Show logs before a timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")

	return cmd
}

func newStartKeepalivedCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewStartKeepalived(setting)
	cmd := &cobra.Command{
		Use:   "start",
		Short: "start keepalived service on local",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}

	fs := cmd.Flags()
	fs.BoolVar(&a.GenerateConfig, "generate-config", true, "generate keepalived config and script file")
	fs.BoolVarP(&a.ForcePullImage, "force-pull", "f", false, "force pull keepalived image")
	fs.BoolVar(&a.LoadImageLocal, "load-image", false, "load local image to docker")

	return cmd
}

func newAddKeepalivedVipCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewAddVirtualIP(setting)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add virtual instance to keepalived",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.InstanceName == "" || a.Interface == "" || a.VirtualIP == "" || a.VirtualRouterID <= 0 {
				return cmd.Usage()
			}
			return a.Run()
		},
	}

	fs := cmd.Flags()

	fs.StringVar(&a.InstanceName, "name", "", "instance name of virtual ip")
	fs.StringVar(&a.Interface, "interface", "", "interface name to bind virtual ip")
	fs.StringVar(&a.State, "state", "MASTER", "state of virtual ip")
	fs.StringVar(&a.AuthPass, "pass", "snz1dp", "password of virtual instance")
	fs.StringVar(&a.VirtualIP, "ip", "", "ip address of virtual instance")
	fs.Uint8Var(&a.VirtualRouterID, "router-id", 0, "unique id in subnet")
	fs.Uint32Var(&a.Priority, "priority", 100, "priority of virtual ip for localhost")
	fs.Uint32Var(&a.AdvertInterval, "interval", 3, "interval check seconds")
	fs.Uint8Var(&a.Subnet, "subnet", 24, "subnet of virtual ip")

	return cmd
}

func newRemoveKeepalivedVipCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewRemoveVirtualIP(setting)
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "remove virtual instance from keepalived",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(a.VirtualIP) == 0 && len(args) == 0 {
				return cmd.Usage()
			}
			a.VirtualIP = append(a.VirtualIP, args...)
			return a.Run()
		},
	}

	fs := cmd.Flags()
	fs.StringArrayVar(&a.VirtualIP, "ip", []string{}, "ip of virtual instance")
	return cmd
}

func newListInterfaceCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewListInterface(setting)
	cmd := &cobra.Command{
		Use:   "interface",
		Short: "list interface of host",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}
	return cmd
}

func newKeepalivedVipCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instance",
		Short: "virtual instance command tool",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newAddKeepalivedVipCmd(setting),
		newRemoveKeepalivedVipCmd(setting),
	)
	return cmd
}

func newKeepalivedCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keepalived",
		Short: "keepalived command tool",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newStartKeepalivedCmd(setting),
		newKeepalivedLogCmd(setting),
		newListKeepalivedCmd(setting),
		newStopKeepalivedCmd(setting),
		newConfigCheckKeepAlived(setting),
		newKeepalivedVipCmd(setting),
		newReStartKeepalivedCmd(setting),
		newListInterfaceCmd(setting),
	)
	return cmd
}
