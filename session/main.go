package main

/*
socat tcp4-listen:3868,reuseaddr,fork -
*/
import (
	"client/common"
	eh "client/handler/event"
	gy "client/handler/gy"

	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
	"strings"
	c "github.com/lehotomi/diam/conn"
	d "github.com/lehotomi/diam/diam"
	l "github.com/lehotomi/diam/mlog"
	t "github.com/lehotomi/diam/templ"	

)

const (
	EVENT = iota
	SESSION_GY
	NOT_KNOWN
)

var	template_types map[string]int = map[string]int{
		"event" : 0,
		"gy":0,
	}

var conf_sess common.Config
var conf_peer common.Config


const (
	const_timeout time.Duration = 2 * time.Second
)

var c_send_ch chan d.Message = make(chan d.Message, 100)
var c_rcv_ch chan d.Message = make(chan d.Message, 100)
var c_mgmt_ch chan c.Event = make(chan c.Event)
var end_chanel chan int = make(chan int)

var g_pars map[string]string
var g_sess_pars map[string]string
var g_session_type int = NOT_KNOWN
var g_goproc_chan chan d.Message = make(chan d.Message)

func init() {	
	args := os.Args[1:]
	if len(args) != 2 {
		fmt.Println(`  Should have one parameter!
  usage: ./executable peer_file param_file`)
		os.Exit(1)
	}
	peer_file := args[0] 
	cfg_file := args[1]

	peer_byte, err := os.ReadFile(peer_file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cfg_byte, err := os.ReadFile(cfg_file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	
	conf_peer = common.ReadConfig(peer_byte)
	conf_sess = common.ReadConfig(cfg_byte)

	l.Init(os.Stdout, os.Stdout, os.Stderr, os.Stderr /*log.Ldate|*/, log.Llongfile | log.Ltime) //ioutil.Discard

	c_dic_dict, ok := conf_sess.GetConfig("conf_dir", "dictionary")
	if !ok {
		fmt.Println("cannot find dictionary config: conf_dir/dictionary")
		os.Exit(1)
	}

	c_templ_dict, ok := conf_sess.GetConfig("conf_dir", "templates")
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
	c_template_type, ok := conf_sess.GetConfig("template", "type")

	if !ok {
		fmt.Println("cannot find template name config: template/type")
		os.Exit(1)
	}

	_,ok = template_types[c_template_type]
	if ! ok {
		var type_list []string
		for key, _ := range template_types {
			type_list = append(type_list,key)

		}
		fmt.Println("template/type should be in: " + strings.Join(type_list,","))
		os.Exit(1)
	}

	var c_template []string = make([]string,1)

	if c_template_type == "event" {
		g_session_type = EVENT
		c_template[0], ok = conf_sess.GetConfig("template", "name")
		if !ok {
			fmt.Println("cannot find template name config: template/name")
			os.Exit(1)
		}

		if !t.IsTemplateExists(c_template[0]) {
			fmt.Printf("template %s does not exist", c_template[0])
			os.Exit(1)
		}
	}

	if c_template_type == "gy" {
		g_session_type = SESSION_GY
		var c_temps []string = make([]string,3)

		c_temps[0], ok = conf_sess.GetConfig("template", "name_init")
		if !ok {
			fmt.Println("cannot find template name config: template/name_init")
			os.Exit(1)
		}

		c_temps[1], ok = conf_sess.GetConfig("template", "name_upd")
		if !ok {
			fmt.Println("cannot find template name config: template/name_upd")
			os.Exit(1)
		}

		c_temps[2], ok = conf_sess.GetConfig("template", "name_term")
		if !ok {
			fmt.Println("cannot find template name config: template/name_term")
			os.Exit(1)
		}

		if ! t.IsTemplateExists(c_temps[0]) {
			fmt.Printf("template %s does not exist", c_temps[0])
			os.Exit(1)
		}
		
		if ! t.IsTemplateExists(c_temps[1]) {
			fmt.Printf("template %s does not exist", c_temps[1])
			os.Exit(1)
		}
		
		if ! t.IsTemplateExists(c_temps[2]) {
			fmt.Printf("template %s does not exist", c_temps[2])
			os.Exit(1)
		}
		//copy(c_template,c_temps)
		c_template = c_temps		
	}
	
	g_sess_pars = conf_sess.GetConfigMap("session_parameters")

	c_diam = c.NewDiamConn(c_send_ch, c_rcv_ch, c_mgmt_ch, map[string]interface{}{
		"name": "D1",
		"tcp_conf": map[string]string{
			"peer": conf_peer.GetConfigVal("diam", "peer"), //"localhost:3869",
		},
		"diam_conf": map[string]string{
			"origin_host":  conf_peer.GetConfigVal("diam", "origin_host"),
			"origin_realm": conf_peer.GetConfigVal("diam", "origin_realm"),
			"host_ip":      "172.16.26.2",
		},
	})

	g_pars = conf_sess.GetConfigMap("props")

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
			common.Log.Println("CEA does not contain result code")
			os.Exit(2)
		}

		if res_code != 2001 {
			common.Log.Printf("Got result code %d in CEA", res_code)
			os.Exit(3)
		}

		//common.Log.Printf("CEA res code:%d", res_code)
		//
	case <-timer.C:
		fmt.Println("Connect timeout to:", conf_peer.GetConfigVal("diam", "peer"))
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
//			fmt.Println("<- got result:", inc_msg) //TODO pretty print
			//fmt.Println(inc_msg.ToString())
			g_goproc_chan <- inc_msg
		}
	}
}

func print_stat(s string, b d.AVP) {
	fmt.Printf("%s\n % x l:%d\n", s, b.Encode(), len(b.Encode()))
}

func createNextSession(template []string /*, f_end func (sess string)*/) {
	c_sent_sid := c_diam.Gen_Session_Id()

	var pars_plus map[string]string = make(map[string]string)

	for k, v := range g_pars {
		pars_plus[k] = v
	}

	pars_plus["session_id"] = c_sent_sid
	pars_plus["origin_host"] = conf_peer.GetConfigVal("diam", "origin_host")
	pars_plus["origin_realm"] = conf_peer.GetConfigVal("diam", "origin_realm")
	pars_plus["destination_realm"] = conf_peer.GetConfigVal("diam", "destination_realm")

	//common.Log.Println("parameters for session:", pars_plus)
	fmt.Printf("-> sending template %s with properties: %s\n", template, pars_plus)
	if g_session_type == EVENT {
		go eh.EventHandler(template[0], g_sess_pars, c_sent_sid, pars_plus, c_send_ch, g_goproc_chan, c_end_func)
	}

	if g_session_type == SESSION_GY {
		go gy.GySession(template, g_sess_pars, c_sent_sid, pars_plus, c_send_ch, g_goproc_chan, c_end_func)
	}

}
