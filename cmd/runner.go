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

func newRunnerListCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewRunnerList(setting)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "list runner client",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.Run()
		},
	}
	return cmd
}

func newRunnerAddCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewRunnerAdd(setting)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "add runner client",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && s.RunnerId == "" {
				s.RunnerId = args[0]
			}
			if s.RunnerId == "" || s.RunnerSecret == "" {
				return cmd.Usage()
			} else {
				return s.Run()
			}
		},
	}

	fs := cmd.Flags()

	fs.StringVar(&s.RunnerImage, "docker-image", "", "docker image of runner")
	fs.StringVar(&s.ServerURL, "server-url", "", "url of runner server")
	fs.StringVar(&s.RunnerId, "id", "", "id of runner client")
	fs.StringVar(&s.RunnerSecret, "secret", "", "secret of runner client")
	fs.StringVar(&s.WorkspacePath, "work-dir", "", "path of workspace")
	fs.StringArrayVarP(&s.ExtrasHosts, "host", "H", []string{}, "add a custom host-to-IP mapping (host:ip)")
	fs.StringArrayVarP(&s.Envs, "env", "e", []string{}, "set environment variables")

	return cmd
}

func newRunnerStartCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewRunnerStart(setting)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "start runner client",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && s.RunnerId == "" {
				s.RunnerId = args[0]
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()

	fs.StringVar(&s.RunnerImage, "docker-image", "", "docker image of runner")
	fs.StringVar(&s.ServerURL, "server-url", "", "url of runner server")
	fs.StringVar(&s.RunnerId, "id", "", "id of runner client")
	fs.StringVar(&s.RunnerSecret, "secret", "", "secret of runner client")
	fs.StringVar(&s.WorkspacePath, "work-dir", "", "path of workspace")
	fs.BoolVarP(&s.ForcePullImage, "force-pull-image", "f", false, "force pull docker image")
	fs.StringArrayVarP(&s.ExtrasHosts, "host", "H", []string{}, "add a custom host-to-IP mapping (host:ip)")

	return cmd
}

func newRunnerStopCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewRunnerStop(setting)

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "stop runner client",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && s.RunnerId == "" {
				s.RunnerId = args[0]
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()

	fs.StringVar(&s.RunnerId, "runner-id", "", "id of runner client")
	fs.BoolVarP(&s.Really, "really", "y", false, "really execute command")
	return cmd
}

func newRunnerRemoveCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewRunnerStop(setting)

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "remove runner client",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if s.RunnerId == "" && len(args) == 0 {
				return cmd.Usage()
			}
			if len(args) > 0 && s.RunnerId == "" {
				s.RunnerId = args[0]
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()

	fs.StringVar(&s.RunnerId, "runner-id", "", "id of runner client")
	fs.BoolVarP(&s.Really, "really", "y", false, "really execute command")
	return cmd
}

func newRunnerExecCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewRunnerExecCmd(setting)

	cmd := &cobra.Command{
		Use:   "exec",
		Short: "execute a command in runner client",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if s.RunnerId == "" && len(args) == 0 {
				return cmd.Usage()
			} else if s.RunnerId == "" {
				s.RunnerId = args[0]
				args = args[1:]
			}
			s.Cmd = args
			return s.Run()
		},
	}

	fs := cmd.Flags()

	fs.StringVar(&s.RunnerId, "runner-id", "", "id of runner client")
	fs.BoolVarP(&s.Tty, "tty", "t", false, "allocate a pseudo-TTY")
	fs.BoolVarP(&s.Detach, "detach", "d", false, "detached mode: run command in the background")
	fs.BoolVarP(&s.Interactive, "interactive", "i", false, "keep STDIN open even if not attached")
	fs.StringArrayVarP(&s.Env, "env", "e", []string{}, "set environment variables")

	return cmd
}

func newRunnerRestartCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewRunnerStop(setting)
	start := action.NewRunnerStart(setting)

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "restart runner client",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && s.RunnerId == "" {
				s.RunnerId = args[0]
			}
			start.RunnerId = s.RunnerId
			if err := s.Run(); err != nil {
				return err
			} else {
				return start.Run()
			}
		},
	}

	fs := cmd.Flags()

	fs.StringVar(&s.RunnerId, "runner-id", "", "id of runner client")
	fs.BoolVarP(&start.ForcePullImage, "force-pull-image", "f", false, "force pull docker image")
	fs.BoolVarP(&s.Really, "really", "y", false, "really execute command")
	return cmd
}

func newRunnerLogsCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewRunnerLogs(setting)

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "show logs of runner client",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.RunnerId == "" && len(args) == 0 {
				return cmd.Usage()
			} else if len(args) > 0 {
				a.RunnerId = args[0]
			}
			return a.Run()
		},
	}
	fs := cmd.Flags()
	fs.StringVar(&a.RunnerId, "runner-id", "", "id of runner client")
	setupRunnerLogFlags(fs, a)
	return cmd
}

func newRunnerCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runner",
		Short: "runner command of snz1dp",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newRunnerListCmd(setting),
		newRunnerAddCmd(setting),
		newRunnerStartCmd(setting),
		newRunnerStopCmd(setting),
		newRunnerRemoveCmd(setting),
		newRunnerLogsCmd(setting),
		newRunnerRestartCmd(setting),
		newRunnerExecCmd(setting),
	)
	return cmd
}

func setupRunnerLogFlags(fs *flag.FlagSet, a *action.RunnerLogs) {
	fs.StringVar(&a.Since, "since", "", "show logs since timestamp  (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
	fs.StringVar(&a.Tail, "tail ", "all", "number of lines to show from the end of the logs")
	fs.BoolVarP(&a.Follow, "follow", "f", false, "Follow log output")
	fs.BoolVar(&a.Details, "details", false, "show extra details provided to logs")
	fs.BoolVarP(&a.Timestamps, "timestamps", "", false, "show extra details provided to logs")
	fs.StringVar(&a.Until, "until", "", "Show logs before a timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
}
