package common

import (
	"fmt"
	"time"
	d "github.com/lehotomi/diam/diam"
	l "github.com/lehotomi/diam/mlog"
	t "github.com/lehotomi/diam/templ"
)

func Smssession(sess_id string, pars map[string]string, ch_send chan<- d.Message, ch_recv <-chan d.Message, f_end func(sess string),

//f_timer func (when time.Duration),
) {
	defer f_end(sess_id)
	c_sms, _ := t.FillTemplate("sms_momt", pars)
	startTime := time.Now()

	ch_send <- c_sms

	timeout := time.NewTimer(3 * time.Second)
	l.Info.Printf("-> I %s", pars["msisdn_a"])

	select {
	case inc_m := <-ch_recv:
		timeout.Stop()

		diff := time.Now().Sub(startTime)

		c_orighost_avp := inc_m.FindAVP(0, 264)
		var c_orighost string

		if c_orighost_avp != nil {
			c_orighost = c_orighost_avp.GetStringValue()
		} else {
			l.Warn.Printf("init answer dows not contain OrigHost AVP:%s", sess_id)
			f_end(sess_id)
			return
		}

		res_code_avp := inc_m.FindAVP(0, 268)
		var res_code int = -1
		if res_code_avp != nil {
			res_code = res_code_avp.GetIntValue()
		} else {
			l.Warn.Printf("result does not contain the result code avp:%s", sess_id)
			l.Info.Printf("<- I %s %s %s", pars["msisdn_a"], c_orighost, diff)
			f_end(sess_id)
			return
		}

		//res_code_avp := inc_m.FindAVP(0,268)
		//res_code_avp := inc_m.FindAVP(0,268)

		//var res_code int = -1;

		l.Info.Printf("<- I %s %s(%d) %s %s", pars["msisdn_a"], result_code_to_string(res_code), res_code, c_orighost, diff)
		timeout.Stop()
		//fmt.Println(" sms on_message:",pars["msisdn_a"],sess_id,diff, res_code)

	case <-timeout.C:
		fmt.Println("session timeout:", sess_id)
	}
}
