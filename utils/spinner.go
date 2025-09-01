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

package utils

import (
	"fmt"
	"io"
	"os"
	"time"
)

func spinnerRoute(out io.Writer, delay time.Duration, mp string, ech chan int) {
	for {
		for _, r := range `-\|/` {
			fmt.Fprintf(out, "\r%s%c", mp, r)
			time.Sleep(delay)
		}
		select {
		case <-ech:
			return
		default:
		}
	}
}

// WaitSpinner -
type WaitSpinner struct {
	endCh  chan int
	out    io.Writer
	stdout bool
	mp     string
}

// Close 关闭
func (w *WaitSpinner) Close() {
	if !w.stdout {
		return
	}
	defer func() {
		if w.endCh != nil {
			close(w.endCh)
		}
		w.endCh = nil
		w.out = nil
	}()

	if w.endCh != nil {
		w.endCh <- 1
		fmt.Fprintf(w.out, "\r%s", w.mp)
	}

}

// NewSpinner 创建新的spinner
func NewSpinner(mp string, out io.Writer) *WaitSpinner {
	spinner := &WaitSpinner{
		mp: mp,
	}
	if out != os.Stdout {
		return spinner
	}
	spinner.stdout = true
	spinner.endCh = make(chan int)
	spinner.out = out
	go spinnerRoute(spinner.out, 100*time.Millisecond, mp, spinner.endCh)
	return spinner
}
