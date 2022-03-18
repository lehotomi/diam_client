package common

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	d "github.com/lehotomi/diam/diam"
	l "github.com/lehotomi/diam/mlog"
	t "github.com/lehotomi/diam/templ"
	//"math/rand"
	"os"
)

const time_between_updates_milli = 1350
const time_between_init_and_first_update_milli = 10
const g_number_of_updates = 8

const (
	CLOSED = iota
	INIT
	FAILED
	QUOTA_REQUESTED
	QUOTA_GRANTED
	REDIRECTED
	LIMIT_REACHED
	FORCED_AUTH
)

const (
	WAITING_FOR_ANSWER = iota
	GOT_ANSWER
)

var rg_state_to_string map[int]string = map[int]string{
	CLOSED:          "CLOSED",
	INIT:            "INIT",
	FAILED:          "FAILED",
	QUOTA_REQUESTED: "QUOTA_REQUESTED",
	QUOTA_GRANTED:   "QUOTA_GRANTED",
	REDIRECTED:      "REDIRECTED",
	LIMIT_REACHED:   "LIMIT_REACHED",
}

const (
	THRESHOLD               = 1
	QHT                     = 1
	FINAL                   = 2
	QUOTA_EXHAUSTED         = 3
	VALIDITY_TIME           = 4
	OTHER_QUOTA_TYPE        = 5
	RATING_CONDITION_CHANGE = 6
	FORCED_REAUTHORISATION  = 7
)

type mscc struct {
	rating_group     uint32
	granted_octet    int
	state            int
	acc_usage        int
	number_of_upd    int
	start_time       time.Time
	last_result_code int
}

func (m mscc) String() string {
	var ret string

	ret += fmt.Sprint("rg:", m.rating_group)
	ret += fmt.Sprint(" last_result_code:", m.last_result_code)
	ret += fmt.Sprint(" state:", rg_state_to_string[m.state])
	ret += fmt.Sprint(" granted_ocet:", m.granted_octet)
	ret = "[" + ret + "]"

	return ret
}

//var r1 Rand =
//var source rand.Source = rand.NewSource(time.Now().UnixNano())
//var rr *rand.Rand = rand.New(source)

const (
	INIT_SENT = iota
	FIRST_UPDATE_SENDING
	FIRST_UPDATE_SENT
	UPDATE_SENDING
	UPDATE_SENT
	TERM_SENDING
	TERM_SENT
)

func Datasession(sess_id string, base_pars map[string]string, ch_send chan<- d.Message, ch_recv <-chan d.Message, f_end func(sess string),

//f_timer func (when time.Duration),
) {
	//defer f_end(sess_id)
	m_nof_updates := g_number_of_updates
	m_state := WAITING_FOR_ANSWER
	pars := make(map[string]string)
	for k, v := range base_pars {
		pars[k] = v
	}

	var req_cat []uint32

	for _, part := range strings.Split(pars["req_category"], ",") {
		c_cat, err := strconv.Atoi(part)
		if err != nil {
			l.Error.Printf("Cannot convert category %s to integer", part)
			os.Exit(-1)
		}
		req_cat = append(req_cat, uint32(c_cat))
	}

	var cats map[uint32]*mscc = make(map[uint32]*mscc)

	m_used_units, err := strconv.ParseUint(pars["total_used_octets"], 10, 64)

	if err != nil {
		l.Warn.Printf("Cannot convert %s to integer", pars["total_used_octets"])
		m_used_units = 1000
	}
	for _, v := range req_cat {
		cats[v] = &mscc{
			rating_group:  v,
			granted_octet: -1,
			state:         INIT,
			acc_usage:     0,
			number_of_upd: 0,
			//start:time.Now(),
		}
	}

	//pars["dest_host"] = "STAYWITHME"
	pars["user_name"] = pars["msisdn_a"] + "@" + pars["apn"]
	c_data, _ := t.FillTemplate("data_init", pars)
	//c_data_upd, _ := t.FillTemplate("data_upd",pars)

	//c_data_term, _ := t.FillTemplate("data_term",pars)

	startTime := time.Now()
	c_sess_state := INIT_SENT
	ite := 0
	l.Trace.Printf("sending init:%s", sess_id)
	l.Info.Printf("-> I %d %s", ite, pars["msisdn_a"])
	ch_send <- c_data

	const_timeout := 2 * time.Second
	///timeout := time.NewTimer(const_timeout)
	//sched := time.NewTimer(100*time.Second)
	ticker := time.NewTicker(const_timeout)
	defer ticker.Stop()
	//c_req_num := 1

	//c_term_send := false
	//time.Sleep(rand.Int)
	for {
		select {
		case inc_m := <-ch_recv:

			if inc_m.IsAnswer() && (inc_m.GetCmdCode() == d.CC_CREDIT_CONTROL) { // CCA

				///timeout.Stop()
				m_state = GOT_ANSWER
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

				//res_code_avp := inc_m.FindAVP(0,268)
				res_code_avp := inc_m.FindAVP(0, 268)
				var res_code int = -1
				if res_code_avp != nil {
					res_code = res_code_avp.GetIntValue()
				} else {
					l.Warn.Printf("result does not contain the result code avp:%s", sess_id)
					l.Info.Printf("<- I %d %s %s %s", ite, pars["msisdn_a"], c_orighost, diff)
					f_end(sess_id)
					return
				}

				//l.Info.Println("data on_message:",pars["msisdn_a"],sess_id,ite, diff, res_code, inc_m.GetCmdCode())
				if ite == 0 {
					l.Info.Printf("<- I %d %s %s(%d) %s %s", ite, pars["msisdn_a"], result_code_to_string(res_code), res_code, c_orighost, diff)
				}

				if res_code != 2001 {
					l.Warn.Printf("Got result code %d for message:%s %s", res_code, sess_id, pars["msisdn_a"])
					f_end(sess_id)
					return
				}
				//

				if c_sess_state == INIT_SENT { //init answer
					//TODO chech result code
					//check orig host in incoming message

					pars["destination_host"] = c_orighost
					l.Trace.Printf("sending first update:%s", sess_id)
					c_sess_state = UPDATE_SENDING
				}

				if c_sess_state == UPDATE_SENT { // update answer
					l.Trace.Printf("sending updates:%s", sess_id)
					//l.Info.Println("bef cats:")
					//for k,v := range cats {
					//   l.Info.Println(k,v)
					//}

					c_mscc_s := inc_m.FindAVPs(0, d.AVP_CODE_Multiple_Services_Credit_Control)
					var info []string
					for _, v := range c_mscc_s {
						//l.Info.Println("mscc",v)
						rg_avp := v.FindAVP(0, 432)
						if rg_avp == nil {
							l.Warn.Printf("mscc result does not contain rating group:%s", sess_id)
							continue
						}
						rg_int := rg_avp.GetIntValue()

						rc_avp := v.FindAVP(0, 268)
						if rc_avp == nil {
							l.Warn.Printf("mscc result does not contain result code avp, sessid %s rg:%d", sess_id, rg_int)
							continue
						}

						c_category, ok := cats[uint32(rg_avp.GetIntValue())]
						if !ok {
							l.Warn.Printf("mscc result contains category which was not requested sessid %s rg:%d", sess_id, rg_int)
						}
						rc_int := rc_avp.GetIntValue()

						c_category.last_result_code = rc_int

						//l.Info.Println("rg",rg_int)
						//l.Info.Println("\trc",rc_int)

						if rc_int == d.CREDIT_LIMIT_REACHED {
							c_category.state = LIMIT_REACHED
						} else if rc_int >= 3000 {
							l.Warn.Printf("mscc result code >= 3000, rc:%d sessid %s rg:%d %s", rc_int, sess_id, rg_int, pars["msisdn_a"])
							c_category.state = FAILED
							continue
						}

						granted_avp := v.FindAVP(0, 431)
						redir_avp := v.FindAVP(0, 434)

						granted_int := -1
						if granted_avp != nil {
							if !granted_avp.IsGrouped() {
								l.Warn.Printf("mscc result granted_unit avp is not grouped sessid %s rg:%d", sess_id, rg_avp.GetIntValue())
								continue
							}
							gavps := granted_avp.GetGroupAVPs()
							for _, v := range gavps {
								//l.Info.Println("\tgranted unit avp",v.GetAVPCode())
								if v.IsTheSameAVP(0, 421) {
									//l.Info.Println("\ttotal octets avp:",v.GetIntValue())
									c_category.granted_octet = v.GetIntValue()
									c_category.state = QUOTA_GRANTED
									granted_int = v.GetIntValue()
								}
							}
						} else {

						}

						var redir_url string = ""

						if redir_avp != nil {
							//l.Info.Println("\tredir avp:")
							c_category.state = REDIRECTED
							if redir_avp.IsGrouped() {
								gavps := redir_avp.GetGroupAVPs()
								for _, v := range gavps {
									//l.Info.Println("\tgranted unit avp",v.GetAVPCode())
									if v.IsTheSameAVP(0, 435) {
										//l.Info.Println("\ttotal octets avp:",v.GetIntValue())
										redir_url = v.GetStringValue()
									}
								}
							}

						}
						if c_category.state != REDIRECTED {
							info = append(info, fmt.Sprintf("[%d,%s(%d),%d]", rg_int, result_code_to_string(rc_int), rc_int, granted_int))
						} else {
							info = append(info, fmt.Sprintf("[%d,%s(%d),redir:%s]", rg_int, result_code_to_string(rc_int), rc_int, redir_url))
						}
						//l.Info.Println("\trg",granted_avp.GetIntValue())
					} //range cat
					l.Info.Printf("<- U %d %s %s %s %s", ite, pars["msisdn_a"], c_orighost, strings.Join(info, " "), diff)

					//l.Info.Println("aft cats:",cats)
					//l.Info.Println("aft cats:")
					//for k,v := range cats {
					//   l.Info.Println(k,v)
					//}
					//         c_sess_state = UPDATE_SENDING
					if ite > m_nof_updates {
						//l.Info.Printf("sending term:%s",sess_id)
						c_sess_state = TERM_SENDING
					} else {
						//l.Info.Printf("sending %d update",ite)
						c_sess_state = UPDATE_SENDING
					}

				}

				if c_sess_state == TERM_SENT {
					l.Info.Printf("<- T %d %s %s(%d) %s %s", ite, pars["msisdn_a"], result_code_to_string(res_code), res_code, c_orighost, diff)
					l.Trace.Printf("session ended:%s", sess_id)
					f_end(sess_id)
					return
				}

				if ite == 0 {
					ticker.Reset(time_between_init_and_first_update_milli * time.Millisecond)
				} else {
					ticker.Reset(time_between_updates_milli * time.Millisecond)
				}
				ite = ite + 1
				//sched = time.NewTimer(500*time.Millisecond)
			} else {
				//         l.Warn.Println("Received something other than CCA:",inc_m.GetCmdFlags())
				//         l.Warn.Println("Received something other than CCA:",inc_m.GetCmdCode(), inc_m.IsRequest(),inc_m.IsAnswer())
				if inc_m.IsRequest() && (inc_m.GetCmdCode() == d.CC_RE_AUTH) { // CCA
					rea_avp := []d.AVP{
						d.AVP_UTF8String(d.AVP_CODE_Origin_Host, pars["origin_host"], d.MAND, 0),
						d.AVP_UTF8String(d.AVP_CODE_Origin_Realm, pars["origin_realm"], d.MAND, 0),
						d.AVP_UTF8String(d.AVP_CODE_Session_Id, sess_id, d.MAND, 0),
						d.AVP_Enumerated(d.AVP_CODE_Result_Code, d.LIMITED_SUCCESS, d.MAND, 0),
						d.AVP_UTF8String(d.AVP_CODE_User_Name, pars["user_name"], d.MAND, 0),
					}
					rea := d.GenMess(d.CC_RE_AUTH, false, true, d.APPID_CC, inc_m.Get_hop_by_hop(), inc_m.Get_end_to_end(), rea_avp)
					ch_send <- rea

					for _, v := range cats {
						if v.state == INIT || v.state == LIMIT_REACHED || v.state == REDIRECTED {
							v.state = FORCED_AUTH
						}

					}
					ticker.Reset(time_between_init_and_first_update_milli * time.Millisecond)
				}

				//l.Warn.Println(inc_m)

			}
		//l.Warn.Println(inc_m)
		/*case  <-timeout.C:
		  fmt.Println("session timeout:",sess_id)
		  f_end(sess_id)
		  return*/
		//----------------------------------------------------------------------------------------------------
		case <-ticker.C:
			//case  <-sched.C:
			//fmt.Println("ticker:",sess_id)
			if m_state == WAITING_FOR_ANSWER {
				l.Warn.Println("session timeout:", sess_id, pars["msisdn_a"])
				f_end(sess_id)
				return
			}
			//timeout.Reset(const_timeout)
			//req_num := c_data_term.FindAVP(0,d.AVP_CODE_CC_Request_Number)
			//req_num.SetIntValue(c_req_num)

			//ticker.Stop()
			//c_term_send = true
			startTime = time.Now()
			if c_sess_state == UPDATE_SENDING {
				pars["request_number"] = fmt.Sprint(ite)
				n_data_term, _ := t.FillTemplate("data_upd", pars)
				var c_mscc []d.AVP
				var info []string
				for _, v := range cats {
					if v.state == INIT || v.state == LIMIT_REACHED || v.state == REDIRECTED {
						c_mscc = append(c_mscc,
							d.AVP_Group(d.AVP_CODE_Multiple_Services_Credit_Control, []d.AVP{
								d.AVP_Group(d.AVP_CODE_Requested_Service_Unit, []d.AVP{}, true, 0),
								d.AVP_Unsigned32(d.AVP_CODE_Rating_Group, v.rating_group, true, 0),
							}, true, 0),
						)

						v.start_time = time.Now()
						v.state = QUOTA_REQUESTED
						info = append(info, fmt.Sprintf("[%d,%s]", v.rating_group, "req_quota"))

					}

					if v.state == QUOTA_GRANTED {
						c_used := m_used_units
						if c_used > uint64(v.granted_octet) {
							c_used = uint64(v.granted_octet)
						}
						c_mscc = append(c_mscc,
							d.AVP_Group(d.AVP_CODE_Multiple_Services_Credit_Control, []d.AVP{
								d.AVP_Group(d.AVP_CODE_Requested_Service_Unit, []d.AVP{}, true, 0),
								d.AVP_Unsigned32(d.AVP_CODE_Rating_Group, v.rating_group, true, 0),
								d.AVP_Group(d.AVP_CODE_Used_Service_Unit, []d.AVP{
									d.AVP_Unsigned64(d.AVP_CODE_CC_Total_Octets, c_used, true, 0),
									d.AVP_Unsigned32(d.AVP_CODE_Reporting_Reason, 0, true, d.VENDOR_VODAFONE),
								}, true, 0),
							}, true, 0),
						)

						v.start_time = time.Now()
						v.state = QUOTA_REQUESTED
						info = append(info, fmt.Sprintf("[%d,used:%d,%s]", v.rating_group, c_used, "req_quota"))
					}

					if v.state == FORCED_AUTH {

						c_mscc = append(c_mscc,
							d.AVP_Group(d.AVP_CODE_Multiple_Services_Credit_Control, []d.AVP{
								d.AVP_Group(d.AVP_CODE_Requested_Service_Unit, []d.AVP{}, true, 0),
								d.AVP_Unsigned32(d.AVP_CODE_Rating_Group, v.rating_group, true, 0),
								d.AVP_Unsigned32(d.AVP_CODE_Reporting_Reason, 7, true, d.VENDOR_VODAFONE),
							}, true, 0),
						)

						v.start_time = time.Now()
						v.state = QUOTA_REQUESTED
						info = append(info, fmt.Sprintf("[%d,%s]", v.rating_group, "forced"))
					}

				}

				n_data_term.AddAVPs_Tail(c_mscc)
				l.Info.Printf("-> U %d %s %s", ite, pars["msisdn_a"], strings.Join(info, " "))
				ch_send <- n_data_term
				c_sess_state = UPDATE_SENT
			}

			if c_sess_state == TERM_SENDING {
				pars["request_number"] = fmt.Sprint(ite)
				var c_mscc []d.AVP
				var info []string
				for _, v := range cats {

					if v.state == QUOTA_GRANTED {
						c_mscc = append(c_mscc,
							d.AVP_Group(d.AVP_CODE_Multiple_Services_Credit_Control, []d.AVP{

								d.AVP_Unsigned32(d.AVP_CODE_Rating_Group, v.rating_group, true, 0),
								d.AVP_Group(d.AVP_CODE_Used_Service_Unit, []d.AVP{
									d.AVP_Unsigned64(d.AVP_CODE_CC_Total_Octets, 0, true, 0),
									d.AVP_Unsigned32(d.AVP_CODE_Reporting_Reason, 2, true, d.VENDOR_VODAFONE),
								}, true, 0),
							}, true, 0),
						)

						v.start_time = time.Now()
						v.state = CLOSED
						info = append(info, fmt.Sprintf("[%d,used:0,final]", v.rating_group))
					}
				}

				n_data_term, _ := t.FillTemplate("data_term", pars)
				n_data_term.AddAVPs_Tail(c_mscc)
				l.Info.Printf("-> T %d %s %s", ite, pars["msisdn_a"], strings.Join(info, " "))
				ch_send <- n_data_term
				c_sess_state = TERM_SENT
			}
			m_state = WAITING_FOR_ANSWER
			ticker.Reset(const_timeout)
			//timeout.Reset(const_timeout)

		} //select
	} //for
}

func create_MSCC(category int, rep_reason int, used_unit int, req_unit bool) d.AVP {
	var gavps []d.AVP

	gavps = append(gavps,
		d.AVP_Unsigned32(d.AVP_CODE_Rating_Group, uint32(category), true, 0),
		d.AVP_Unsigned32(d.AVP_CODE_Reporting_Reason, uint32(rep_reason), true, d.VENDOR_VODAFONE),
	)

	if req_unit {
		gavps = append(gavps, d.AVP_Group(d.AVP_CODE_Requested_Service_Unit, []d.AVP{}, true, 0))
	}

	return d.AVP_Group(d.AVP_CODE_Multiple_Services_Credit_Control, gavps, true, 0)

}

func result_code_to_string(rc int) string {

	switch rc {
	case d.SUCCESS:
		return "SUCCESS"
	case d.END_USER_SERVICE_DENIED:
		return "END_USER_SERVICE_DENIED"
	case d.CREDIT_CONTROL_NOT_APPLICABLE:
		return "CREDIT_CONTROL_NOT_APPLICABLE"
	case d.CREDIT_LIMIT_REACHED:
		return "CREDIT_LIMIT_REACHED"
	case d.USER_UNKNOWN:
		return "USER_UNKNOWN"
	case d.UNKNOWN_SESSION_ID:
		return "UNKNOWN_SESSION_ID"
	case d.RATING_FAILED:
		return "RATING_FAILED"
	case d.AUTHORIZATION_REJECTED:
		return "AUTHORIZATION_REJECTED"
	case d.AUTHENTICATION_REJECTED:
		return "AUTHENTICATION_REJECTED"
	case d.UNABLE_TO_DELIVER:
		return "UNABLE_TO_DELIVER"
	default:
		return "UNKNOWN"
	}

}
