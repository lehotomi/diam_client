package event

import (
	//"fmt"
	"time"
	d "github.com/lehotomi/diam/diam"
	//l "github.com/lehotomi/diam/mlog"
	t "github.com/lehotomi/diam/templ"
	com "client/common"
)



func EventHandler(template_name string, sess_par map[string]string, sess_id string, pars map[string]string, ch_send chan<- d.Message,
	ch_recv <-chan d.Message, f_end func(sess string)) {

	//defer f_end(sess_id)
	//chechSessPar
	//sess_par
	var m_print_decoded_mess bool = false
	if sess_par["print_decoded_message"] == "true" {
		m_print_decoded_mess = true
	}
	//sess par end

	//fmt.Printf("%T %s",sess_par["print_decoded_message"], sess_par["print_decoded_message"])
	c_sms, _ := t.FillTemplate(template_name, pars)
	startTime := time.Now()
	if m_print_decoded_mess { 
		com.Log.Println("\nREQ:\n---\n"+c_sms.ToString())
	}
	ch_send <- c_sms

	timeout := time.NewTimer(3 * time.Second)
	com.Log.Printf("-> %s", pars["msisdn_a"])

	select {
	case inc_m := <-ch_recv:
		timeout.Stop()

		diff := time.Now().Sub(startTime)
		if m_print_decoded_mess { 
			com.Log.Println("\nRES:\n---\n"+inc_m.ToString())
		}
		c_orighost_avp := inc_m.FindAVP(0, 264)
		var c_orighost string

		if c_orighost_avp != nil {
			c_orighost = c_orighost_avp.GetStringValue()
		} else {
			com.Log.Printf("init answer dows not contain OrigHost AVP:%s", sess_id)
			f_end(sess_id)
			return
		}

		res_code_avp := inc_m.FindAVP(0, 268)
		var res_code int = -1
		if res_code_avp != nil {
			res_code = res_code_avp.GetIntValue()
		} else {
			com.Log.Printf("result does not contain the result code avp:%s", sess_id)
			com.Log.Printf("<- %s %s %s", pars["msisdn_a"], c_orighost, diff)
			f_end(sess_id)
			return
		}

		com.Log.Printf("<- %s %s(%d) %s %s", pars["msisdn_a"], com.Result_code_to_string(res_code), res_code, c_orighost, diff)
		timeout.Stop()
	case <-timeout.C:
		com.Log.Println("session timeout:", sess_id)
	}
	f_end(sess_id)
}
