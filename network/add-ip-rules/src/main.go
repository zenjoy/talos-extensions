package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/vishvananda/netlink"
	"gopkg.in/yaml.v2"
)

var configFile string

type IPNet struct {
	net.IPNet
}

type Route struct {
	To     IPNet  `yaml:"to"`
	Via    net.IP `yaml:"via"`
	Metric *int   `yaml:"metric,omitempty"`
	Table  int    `yaml:"table"`
}

type RoutingPolicy struct {
	From     *IPNet `yaml:"from,omitempty"`
	To       *IPNet `yaml:"to,omitempty"`
	Priority *int   `yaml:"priority,omitempty"`
	Table    int    `yaml:"table"`
}

type NetplanConfig struct {
	Network struct {
		Ethernets map[string]struct {
			Routes        *[]Route         `yaml:"routes,omitempty"`
			RoutingPolicy *[]RoutingPolicy `yaml:"routing-policy,omitempty"`
		} `yaml:"ethernets"`
	} `yaml:"network"`
}

func (n *IPNet) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return err
	}
	n.IPNet = *ipnet
	return nil
}

func defaultInt(i *int, def int) int {
	if i != nil {
		return *i
	}
	return def
}

func main() {
	flag.StringVar(&configFile, "config", "", "the yaml config file")
	flag.Parse()

	log.Printf("starting add-ip-rules service")
	defer log.Printf("stopping add-ip-rules service")

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// throw error an exit if config is not set
	if configFile == "" {
		log.Fatalf("error: config file not set")
	}

	// Read the YAML configuration file
	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("error reading config file: %v", err)
	}

	// Unmarshal the YAML data into a struct
	var config NetplanConfig
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("error unmarshaling YAML data: %v", err)
	}

	// add routes and policies for each interface
	for name, eth := range config.Network.Ethernets {
		link, err := netlink.LinkByName(name)
		if err != nil {
			log.Fatalf("error getting link for interface %s: %v", name, err)
		}

		dev, ok := link.(*netlink.Device)
		if !ok {
			log.Fatalf("error converting link to device: %v", link)
		}

		if eth.RoutingPolicy != nil {
			for _, policy := range *eth.RoutingPolicy {
				nsrule := netlink.NewRule()
				nsrule.Table = policy.Table

				if policy.From != nil {
					log.Printf("adding routing policy on table %d from %s", policy.Table, policy.From.String())
					nsrule.Src = &policy.From.IPNet
				}
				if policy.To != nil {
					log.Printf("adding routing policy on table %d to %s", policy.Table, policy.To.String())
					nsrule.Dst = &policy.To.IPNet
				}
				if policy.Priority != nil {
					nsrule.Priority = *policy.Priority
				}

				err := netlink.RuleAdd(nsrule)
				if err != nil {
					log.Fatalf("error adding routing policy: %v", err)
				}
			}
		}

		if eth.Routes != nil {
			for _, route := range *eth.Routes {
				log.Printf("adding route for %s via %s", route.To.String(), route.Via.String())

				priority := defaultInt(route.Metric, 1024)

				err := netlink.RouteAdd(&netlink.Route{
					Dst:       &route.To.IPNet,
					Gw:        route.Via,
					Table:     route.Table,
					Priority:  priority,
					LinkIndex: dev.Index,
				})
				if err != nil {
					log.Fatalf("error adding route: %v", err)
				}
			}
		}
	}

	// Print the parsed configuration
	fmt.Printf("done!\n")
}
