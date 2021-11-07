/* data_load.go - load tcp keepalive config from json file */
/*
modification history
--------------------
2021/9/7, by Yu Hui, create
*/
/*
DESCRIPTION
*/

package mod_tcp_keepalive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
)

/*
{
  "Version": "x",
  "Config": {
    "Product1": [{
      "VipConf": ["1.1.1.1", "1.1.1.2"],
      "KeepAliveParam": {
		"Disable": false,
        "KeepIdle" : 70,
        "KeepIntvl" : 15,
		"KeepCnt": 9
      }
    }]
  }
}
*/
// ProductRuleConf match the original tcp_keepalive.data
type ProductRuleConf struct {
	Version string
	Config  map[string]ProductRulesFile
}

type ProductRulesFile []ProductRuleFile
type ProductRuleFile struct {
	VipConf        []string
	KeepAliveParam KeepAliveParam
}

/*
{
  "Version": "x",
  "Config": {
    "Product1": {
	  "1.1.1.1": {
		"Disable": false,
        "KeepIdle" : 70,
        "KeepIntvl" : 15,
		"KeepCnt": 9
	  },
	  "1.1.1.2": {
		"Disable": false,
        "KeepIdle" : 70,
        "KeepIntvl" : 15,
		"KeepCnt": 9
	  }
    }
  }
}
*/
// ProductRuleData contains data convert from ProductRuleConf
type ProductRuleData struct {
	Version string
	Config  ProductRules
}

type KeepAliveParam struct {
	Disable   bool
	KeepIdle  int
	KeepIntvl int
	KeepCnt   int
}
type KeepAliveRules map[string]KeepAliveParam
type ProductRules map[string]KeepAliveRules

func ConvertConf(c ProductRuleConf) (ProductRuleData, error) {
	data := ProductRuleData{}
	data.Version = c.Version
	data.Config = ProductRules{}

	for product, rules := range c.Config {
		data.Config[product] = KeepAliveRules{}
		for _, rule := range rules {
			for _, val := range rule.VipConf {
				ip, err := formatIP(val)
				if err != nil {
					return data, err
				}
				if _, ok := data.Config[product][ip]; ok {
					return data, fmt.Errorf("duplicated ip[%s] in product[%s]", val, product)
				}
				data.Config[product][ip] = rule.KeepAliveParam
			}
		}
	}

	return data, nil
}

func RulesCheck(conf KeepAliveRules) error {
	for ip, val := range conf {
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("invalid ip: %s", ip)
		}

		if val.KeepIdle < 0 || val.KeepIntvl < 0 || val.KeepCnt < 0 {
			return fmt.Errorf("invalid keepalive param: %+v", val)
		}
	}

	return nil
}

func ProductRulesCheck(conf ProductRules) error {
	for product, rules := range conf {
		if product == "" {
			return fmt.Errorf("no product name")
		}
		if rules == nil {
			return fmt.Errorf("no rules for product: %s", product)
		}

		err := RulesCheck(rules)
		if err != nil {
			return fmt.Errorf("ProductRules:%s, %s", product, err.Error())
		}
	}

	return nil
}

func ProductRuleDataCheck(conf ProductRuleData) error {
	var err error

	// check Version
	if conf.Version == "" {
		return errors.New("no Version")
	}

	// check Config
	if conf.Config == nil {
		return errors.New("no Config")
	}

	err = ProductRulesCheck(conf.Config)
	if err != nil {
		return fmt.Errorf("Config: %s", err.Error())
	}

	return nil
}

func formatIP(s string) (string, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return "", fmt.Errorf("formatIP: net.ParseIP() error, ip: %s", s)
	}

	ret := ip.String()
	if ret == "<nil>" {
		return "", fmt.Errorf("formatIP: ip.String() error, ip: %s", s)
	}

	return ret, nil
}

func KeepAliveDataLoad(filename string) (ProductRuleData, error) {
	var err error
	var conf ProductRuleConf
	var data ProductRuleData

	// open the file
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return data, err
	}

	// decode the file
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&conf)
	if err != nil {
		return data, err
	}

	// convert to ProductRuleData
	data, err = ConvertConf(conf)
	if err != nil {
		return data, err
	}

	// check data
	err = ProductRuleDataCheck(data)
	if err != nil {
		return data, err
	}

	return data, nil
}
