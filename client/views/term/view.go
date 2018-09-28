// interactive terminal interface for local clients
package term

import (
	"cngrok/client/mvc"
	"cngrok/log"
	"cngrok/proto"
	"cngrok/util"
	termbox "github.com/nsf/termbox-go"
	"time"
)

type TermView struct {
	ctl      mvc.Controller
	updates  chan interface{}
	flush    chan int
	shutdown chan int
	redraw   *util.Broadcast
	subviews []mvc.View
	log.Logger
	*area
}

func NewTermView(ctl mvc.Controller) *TermView {
	// initialize terminal display
	termbox.Init()

	w, _ := termbox.Size()

	v := &TermView{
		ctl:      ctl,
		updates:  ctl.Updates().Reg(),
		redraw:   util.NewBroadcast(),
		flush:    make(chan int),
		shutdown: make(chan int),
		Logger:   log.NewPrefixLogger("view", "term"),
		area:     NewArea(0, 0, w, 10),
	}

	ctl.Go(v.run)
	ctl.Go(v.input)

	return v
}

func connStatusRepr(status mvc.ConnStatus) (string, termbox.Attribute) {
	switch status {
	case mvc.ConnConnecting:
		return "正在连接", termbox.ColorCyan
	case mvc.ConnReconnecting:
		return "正在重连", termbox.ColorRed
	case mvc.ConnOnline:
		return "在线", termbox.ColorGreen
	}
	return "未知", termbox.ColorWhite
}

func (v *TermView) draw() {
	state := v.ctl.State()

	v.Clear()

	// quit instructions
	quitMsg := "(Ctrl+C to quit)"
	v.Printf(v.w-len(quitMsg), 0, quitMsg)

	// new version message
	updateStatus := state.GetUpdateStatus()
	var updateMsg string
	switch updateStatus {
	case mvc.UpdateNone:
		updateMsg = ""
	case mvc.UpdateInstalling:
		updateMsg = "ngrok正在更新"
	case mvc.UpdateReady:
		updateMsg = "ngrok已经更新，请重启新版本ngrok"
	case mvc.UpdateAvailable:
		updateMsg = "新版本在 http://ngrok.chengang.win"
	default:
		pct := float64(updateStatus) / 100.0
		const barLength = 25
		full := int(barLength * pct)
		bar := make([]byte, barLength+2)
		bar[0] = '['
		bar[barLength+1] = ']'
		for i := 0; i < 25; i++ {
			if i <= full {
				bar[i+1] = '#'
			} else {
				bar[i+1] = ' '
			}
		}
		updateMsg = "正在下载更新: " + string(bar)
	}

	if updateMsg != "" {
		v.APrintf(termbox.ColorYellow, 30, 0, updateMsg)
	}

	v.APrintf(termbox.ColorBlue|termbox.AttrBold, 0, 0, "www.cngrok.com")
	statusStr, statusColor := connStatusRepr(state.GetConnStatus())
	v.APrintf(statusColor, 0, 2, "%-30s%s", "隧道状态", statusStr)

	v.Printf(0, 3, "%-15s%s/%s", "版本信息", state.GetClientVersion(), state.GetServerVersion())
	var i int = 4
	for _, t := range state.GetTunnels() {
		v.Printf(0, i, "%-15s%s -> %s", "转发详情", t.PublicUrl, t.LocalAddr)
		i++
	}
	v.Printf(0, i+0, "%-15s%s", "状态界面", v.ctl.GetWebInspectAddr())

	connMeter, connTimer := state.GetConnectionMetrics()
	v.Printf(0, i+1, "%-15s%d", "连接次数", connMeter.Count())

	msec := float64(time.Millisecond)
	v.Printf(0, i+2, "%-15s%.2fms", "平均时间", connTimer.Mean()/msec)

	termbox.Flush()
}

func (v *TermView) run() {
	defer close(v.shutdown)
	defer termbox.Close()

	redraw := v.redraw.Reg()
	defer v.redraw.UnReg(redraw)

	v.draw()
	for {
		v.Debug("等待更新")
		select {
		case <-v.flush:
			termbox.Flush()

		case <-v.updates:
			v.draw()

		case <-redraw:
			v.draw()

		case <-v.shutdown:
			return
		}
	}
}

func (v *TermView) Shutdown() {
	v.shutdown <- 1
	<-v.shutdown
}

func (v *TermView) Flush() {
	v.flush <- 1
}

func (v *TermView) NewHttpView(p *proto.Http) *HttpView {
	return newTermHttpView(v.ctl, v, p, 0, 12)
}

func (v *TermView) input() {
	for {
		ev := termbox.PollEvent()
		switch ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyCtrlC:
				v.Info("有退出命令")
				v.ctl.Shutdown("")
			}

		case termbox.EventResize:
			v.Info("重新调整大小")
			v.redraw.In() <- 1

		case termbox.EventError:
			panic(ev.Err)
		}
	}
}
