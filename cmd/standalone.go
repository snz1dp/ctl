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

func setupLogFlags(fs *flag.FlagSet, a *action.LogStandaloneService) {
	fs.StringVar(&a.Since, "since", "", "show logs since timestamp  (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
	fs.StringVar(&a.Tail, "tail ", "all", "number of lines to show from the end of the logs")
	fs.BoolVarP(&a.Follow, "follow", "f", false, "Follow log output")
	fs.BoolVar(&a.Details, "details", false, "show extra details provided to logs")
	fs.BoolVarP(&a.Timestamps, "timestamps", "", false, "set timestamps for log entries")
	fs.StringVar(&a.Until, "until", "", "Show logs before a timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
}

func newListStandaloneServiceCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewListStandaloneService(setting)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list standalone service of profile",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}
	return cmd
}

func newStartStandaloneCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewStartStandaloneService(setting)
	b := action.NewDownloadBundle(setting)
	b.NotSaveImage = true

	cmd := &cobra.Command{
		Use:   "start",
		Short: "start snz1dp bundle standalone service",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.ServiceName == "" && len(args) == 0 {
				return cmd.Usage()
			} else if len(args) > 0 {
				for _, sname := range args {
					a.ServiceName = sname
					if b.PullImage || b.Force {
						b.Bundle = []string{a.ServiceName}
						if err := b.Run(); err != nil {
							return err
						}
					}
					if err := a.Run(); err != nil {
						return err
					}
				}
				if b.PullImage || b.Force {
					b.Bundle = []string{a.ServiceName}
					if err := b.Run(); err != nil {
						return err
					}
				}
				return nil
			}
			return a.Run()
		},
	}

	flag := cmd.Flags()
	flag.StringVarP(&a.ServiceName, "name", "n", "", "bundle name of snz1dp installe profile")
	flag.StringArrayVarP(&a.ExtrasHosts, "host", "H", []string{}, "add a custom host-to-IP mapping (host:ip)")

	flag.StringArrayVarP(&a.EnvVariables, "env", "e", []string{}, "set environment variables")

	flag.BoolVarP(&a.ForcePullImage, "pull-image", "p", false, "force pull docker image of bundle")
	flag.BoolVarP(&b.Force, "force-download", "f", false, "force to download snz1dp bundle")

	flag.StringVarP(&a.GPU, "gpus", "", "", "gpu device for docker")
	flag.StringVarP(&a.Runtime, "runtime", "", "", "runtime for docker")

	return cmd
}

func newRestartStandaloneCmd(setting *action.GlobalSetting) *cobra.Command {

	a := action.NewStopStandaloneService(setting)
	c := action.NewStartStandaloneService(setting)
	b := action.NewDownloadBundle(setting)
	b.NotSaveImage = true

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "restart snz1dp bundle standalone service",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.ServiceName == "" && len(args) == 0 {
				return cmd.Usage()
			} else if len(args) > 0 {
				for _, sname := range args {
					a.ServiceName = sname
					if b.PullImage || b.Force {
						b.Bundle = []string{a.ServiceName}
						if err := b.Run(); err != nil {
							return err
						}
					}
					if err := a.Run(); err != nil {
						return err
					}
					c.ServiceName = a.ServiceName
					if err := c.Run(); err != nil {
						return err
					}
				}
				return nil
			}
			if b.PullImage || b.Force {
				b.Bundle = []string{a.ServiceName}
				if err := b.Run(); err != nil {
					return err
				}
			}
			c.ServiceName = a.ServiceName
			a.Run()
			return c.Run()
		},
	}

	flag := cmd.Flags()
	flag.StringVarP(&a.ServiceName, "name", "n", "", "bundle name of snz1dp installe profile")
	flag.BoolVarP(&a.Really, "really", "y", false, "really execute command")

	flag.StringArrayVarP(&c.ExtrasHosts, "host", "H", []string{}, "add a custom host-to-IP mapping (host:ip)")
	flag.StringArrayVarP(&c.EnvVariables, "env", "e", []string{}, "set environment variables")

	flag.BoolVarP(&b.PullImage, "pull-image", "p", false, "force pull docker image of bundle")
	flag.BoolVarP(&b.Force, "force-download", "f", false, "force to download snz1dp bundle")
	flag.StringVarP(&c.GPU, "gpus", "", "", "gpu device for docker")
	flag.StringVarP(&c.Runtime, "runtime", "", "", "runtime for docker")

	return cmd
}

func newStopStandaloneCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewStopStandaloneService(setting)
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "stop snz1dp bundle standalone service",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.ServiceName == "" && len(args) == 0 {
				return cmd.Usage()
			} else if len(args) > 0 {
				for _, sname := range args {
					a.ServiceName = sname
					if err := a.Run(); err != nil {
						return err
					}
				}
				return nil
			}
			return a.Run()
		},
	}

	flag := cmd.Flags()
	flag.StringVarP(&a.ServiceName, "name", "n", "", "bundle name of snz1dp installe profile")
	flag.BoolVarP(&a.Really, "really", "y", false, "really execute command")

	return cmd
}

func newLogStandaloneCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewLogStandaloneService(setting)

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "show logs of snz1dp bundle standalone service",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.ServiceName == "" && len(args) == 0 {
				return cmd.Usage()
			} else if len(args) > 0 {
				a.ServiceName = args[0]
			}
			return a.Run()
		},
	}
	flag := cmd.Flags()
	flag.StringVarP(&a.ServiceName, "name", "n", "", "bundle name of snz1dp installe profile")
	setupLogFlags(flag, a)
	return cmd
}

func newExecStandaloneCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewExecStandaloneService(setting)

	cmd := &cobra.Command{
		Use:   "exec",
		Short: "execute a command in a running standalone service",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.ServiceName == "" && len(args) == 0 {
				return cmd.Usage()
			} else if a.ServiceName == "" {
				a.ServiceName = args[0]
				args = args[1:]
			}
			a.Cmd = args
			return a.Run()
		},
	}
	flag := cmd.Flags()
	flag.StringVarP(&a.ServiceName, "name", "n", "", "bundle name of snz1dp installe profile")
	flag.BoolVarP(&a.Tty, "tty", "t", false, "allocate a pseudo-TTY")
	flag.BoolVarP(&a.Detach, "detach", "d", false, "detached mode: run command in the background")
	flag.BoolVarP(&a.Interactive, "interactive", "i", false, "keep STDIN open even if not attached")
	flag.StringArrayVarP(&a.Env, "env", "e", []string{}, "set environment variables")

	return cmd
}

func newCleanStandaloneCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewCleanStandaloneService(setting)

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "clean run files of snz1dp bundle standalone service",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.ServiceName == "" && len(args) == 0 {
				return cmd.Usage()
			} else if len(args) > 0 {
				for _, sname := range args {
					a.ServiceName = sname
					if err := a.Run(); err != nil {
						return err
					}
				}
				return nil
			}
			return a.Run()
		},
	}
	flag := cmd.Flags()
	flag.StringVarP(&a.ServiceName, "name", "n", "", "bundle name of snz1dp installe profile")
	flag.BoolVarP(&a.Really, "really", "y", false, "really execute command")
	return cmd
}

func newStandaloneCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "standalone",
		Aliases: []string{"alone"},
		Short:   "snz1dp bundle standalone command",
		Long:    globalBanner,
	}
	cmd.AddCommand(
		newStartStandaloneCmd(setting),
		newStopStandaloneCmd(setting),
		newRestartStandaloneCmd(setting),
		newLogStandaloneCmd(setting),
		newCleanStandaloneCmd(setting),
		newListStandaloneServiceCmd(setting),
		newExecStandaloneCmd(setting),
	)
	return cmd
}
