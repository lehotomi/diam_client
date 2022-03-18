package main

/*
socat tcp4-listen:3868,reuseaddr,fork -
*/
import (
	"client/common"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
	c "github.com/lehotomi/diam/conn"
	d "github.com/lehotomi/diam/diam"
	l "github.com/lehotomi/diam/mlog"
	t "github.com/lehotomi/diam/templ"	
)

const (
	EVENT = iota
	SESSION
)

const (
	const_timeout time.Duration = 2 * time.Second
)

var c_send_ch chan d.Message = make(chan d.Message, 100)
var c_rcv_ch chan d.Message = make(chan d.Message, 100)
var c_mgmt_ch chan c.Event = make(chan c.Event)
var end_chanel chan int = make(chan int)

var g_pars map[string]string
var g_session_type int = EVENT
var g_goproc_chan chan d.Message = make(chan d.Message)

func init() {	
	args := os.Args[1:]
	if len(args) != 1 {
		fmt.Println(`  Should have one parameter!
  usage: ./executable config_file`)
		os.Exit(1)
	}

	cfg_file := args[0]

	cfg_byte, err := os.ReadFile(cfg_file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	common.ReadConfig(cfg_byte)

	l.Init(os.Stdout, os.Stdout, os.Stderr, os.Stderr /*log.Ldate|*/ /*log.Lshortfile|*/, log.Ltime) //ioutil.Discard

	c_dic_dict, ok := common.GetConfig("conf_dir", "dictionary")
	if !ok {
		fmt.Println("cannot find dictionary config: conf_dir/dictionary")
		os.Exit(1)
	}

	c_templ_dict, ok := common.GetConfig("conf_dir", "templates")
	if !ok {
		fmt.Println("cannot find template config: conf_dir/templates")
		os.Exit(1)
	}

	d.Init(c_dic_dict)
	t.Init(c_templ_dict)

}

var c_end_func func(string) = func(ssid string) {
	//fmt.Println("end")
	end_chanel <- 0
}

var source rand.Source = rand.NewSource(time.Now().UnixNano())
var rr *rand.Rand = rand.New(source)

var c_diam c.DiamConn

func main() {
	c_template_type, ok := common.GetConfig("template", "type")

	if !ok {
		fmt.Println("cannot find template name config: template/type")
		os.Exit(1)
	}

	if c_template_type != "event" {
		fmt.Println("template/type should be \"event\"")
		os.Exit(1)
	}
	var c_template string

	if c_template_type == "event" {
		g_session_type = EVENT
		c_template, ok = common.GetConfig("template", "name")
		if !ok {
			fmt.Println("cannot find template name config: template/name")
			os.Exit(1)
		}

		if !t.IsTemplateExists(c_template) {
			fmt.Printf("template %s does not exist", c_template)
			os.Exit(1)
		}
	}

	c_diam = c.NewDiamConn(c_send_ch, c_rcv_ch, c_mgmt_ch, map[string]interface{}{
		"name": "D1",
		"tcp_conf": map[string]string{
			"peer": common.GetConfigVal("diam", "peer"), //"localhost:3869",
		},
		"diam_conf": map[string]string{
			"origin_host":  common.GetConfigVal("diam", "origin_host"),
			"origin_realm": common.GetConfigVal("diam", "origin_realm"),
			"host_ip":      "172.16.26.2",
		},
	})

	g_pars = common.GetConfigMap("props")

	go c_diam.Start()

	timer := time.NewTimer(const_timeout)
	select {
	case c_event := <-c_mgmt_ch:

		inc_bytes := c_event.Data.([]byte)
		c_cea := d.Decode(inc_bytes)
		//fmt.Println(c_cea)
		res_code_avp := c_cea.FindAVP(0, 268)
		var res_code int = -1
		if res_code_avp != nil {
			res_code = res_code_avp.GetIntValue()
		} else {
			fmt.Println("CEA does not contain result code")
			os.Exit(2)
		}

		if res_code != 2001 {
			fmt.Printf("Got result code %d in CEA", res_code)
			os.Exit(3)
		}

		l.Trace.Printf("CEA res code:%d", res_code)
		//
	case <-timer.C:
		fmt.Println("Connect timeout to:", common.GetConfigVal("diam", "peer"))
		os.Exit(1)
	}

	go recv_handler()
	createNextSession(c_template)
	<-end_chanel
}

func recv_handler() {
	for {
		select {
		case inc_msg := <-c_rcv_ch:
			fmt.Println("<- got result:", inc_msg) //TODO pretty print
			g_goproc_chan <- inc_msg
		}
	}
}

func print_stat(s string, b d.AVP) {
	fmt.Printf("%s\n % x l:%d\n", s, b.Encode(), len(b.Encode()))
}

func createNextSession(template string /*, f_end func (sess string)*/) {
	c_sent_sid := c_diam.Gen_Session_Id()

	var pars_plus map[string]string = make(map[string]string)

	for k, v := range g_pars {
		pars_plus[k] = v
	}

	pars_plus["session_id"] = c_sent_sid
	pars_plus["origin_host"] = common.GetConfigVal("diam", "origin_host")
	pars_plus["origin_realm"] = common.GetConfigVal("diam", "origin_realm")
	pars_plus["destination_realm"] = common.GetConfigVal("diam", "destination_realm")

	l.Trace.Println("parameters for session:", pars_plus)
	fmt.Printf("-> sending template %s with properties: %s\n", template, pars_plus)
	if g_session_type == EVENT {
		go common.EventSession(template, c_sent_sid, pars_plus, c_send_ch, g_goproc_chan, c_end_func)
	}

}
