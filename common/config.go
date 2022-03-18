package common

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

var env map[string]string = make(map[string]string)

func readEnv() {
	for _, e := range os.Environ() {
		if i := strings.Index(e, "="); i >= 0 {
			env[e[:i]] = e[i+1:]
		}
	}
}

type Config map[string]map[string]string

var c_conf Config

func GetConfigVal(section string, par string) string {

	val, exists := GetConfig(section, par)
	if exists {
		return val
	}
	return "not_set"
}

func GetConfigMap(section string) map[string]string {
	//fmt.Println("map:",section)
	sec, exists := c_conf[section]
	if exists {
		return sec
	}

	return nil
}

func GetConfig(section string, par string) (string, bool) {

	/*val,exists := env[section+"_"+par]
	if exists {
		return val, true
	}*/

	var secs map[string]string
	secs, exists := c_conf[section]
	if ! exists {
		return "", false
	}

	val, exists := secs[par]
	if exists {
		return val, true
	}

	return "", false

}

func getEnvSection(sec string) map[string]string {
	var ret map[string]string = make(map[string]string)

	for env_key, env_val := range env {

		if strings.HasPrefix(env_key, sec+"_") {
			//fmt.Println("ENV:",env_key)
			ind := strings.Index(env_key, "_")
			//fmt.Println("_"+env_key[ind+1:]+"_")
			ret[env_key[ind+1:]] = env_val
		}
	}

	return ret
}

func ReadConfig(in []byte) {
	readEnv()

	if err := yaml.Unmarshal(in, &c_conf); err != nil {
		fmt.Println(err)
		os.Exit(3)
	}

	//fmt.Println("naaaaaaa")

	for key, _ := range c_conf {
		//fmt.Println("FFF:",key,entry)
		sec_env := getEnvSection(key)
		for sec_key, sec_val := range sec_env {
			c_conf[key][sec_key] = sec_val
		}
	}

	/*
		fmt.Println(c_conf["tcp"]["peer"])*/
	/*
			    switch i := entry.(type) {
		        case string:
		            fmt.Printf("i is a string: %+v\n", i)
		        case map[string]interface{}:
		            fmt.Printf("i is a map.")
		            for k,v := range i {
		                fmt.Printf("%s=%v\n",k,v)
		            }
		        default:
		            fmt.Printf("Type i=%s", i)
		        }
		    }
	*/
}
