package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mackerelio/mackerel-agent/config"
	"gopkg.in/urfave/cli.v1"
)

type PackageInfo struct {
	Version string `json:"version"`
}

func main() {
	app := cli.NewApp()
	app.Name = "mkr-meta-pkg"
	app.Version = "0.0.1"
	app.Commands = []cli.Command{
		{
			Name:   "collect",
			Usage:  "collect and store package data",
			Action: actionCollect,
		},
		{
			Name:      "dump",
			Usage:     "dump package metadata",
			ArgsUsage: "[hostId]",
			Action:    actionDump,
		},
	}

	app.Run(os.Args)
}

func loadMackerelConfig() (*config.Config, error) {
	conf, err := config.LoadConfig(config.DefaultConfig.Conffile)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func actionCollect(c *cli.Context) error {
	conf, err := loadMackerelConfig()
	if err != nil {
		return err
	}
	apiKey := conf.Apikey
	hostId, err := conf.LoadHostID()
	if err != nil {
		return err
	}

	cmdOutput, err := exec.Command("rpm", "-qa", "--queryformat", "%{NAME}\t%{VERSION}-%{RELEASE}\n").Output()
	if err != nil {
		return err
	}

	packages := map[string]PackageInfo{}

	for _, line := range strings.Split(string(cmdOutput), "\n") {
		if line == "" {
			break
		}
		pkginfo := strings.Split(line, "\t")
		packages[pkginfo[0]] = PackageInfo{pkginfo[1]}
	}
	jsonData, err := json.Marshal(packages)
	if err != nil {
		return err
	}

	url, _ := url.Parse(conf.Apibase)
	url.Path = fmt.Sprintf("/api/v0/hosts/%s/metadata/packages", hostId)

	req, err := http.NewRequest("PUT", url.String(), bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Api-Key", apiKey)
	req.Header.Set("User-Agent", "mkr-meta-pkg")

	client := &http.Client{}
	client.Timeout = 30 * time.Second
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("API Request Error: %d", resp.StatusCode)
	}

	fmt.Println("Update Success!!")
	return nil
}

func actionDump(c *cli.Context) error {
	hostId := c.Args().Get(0)

	conf, err := loadMackerelConfig()
	if err != nil {
		return err
	}
	apiKey := conf.Apikey

	url, _ := url.Parse(conf.Apibase)
	url.Path = fmt.Sprintf("/api/v0/hosts/%s/metadata/packages", hostId)

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Add("X-Api-Key", apiKey)
	req.Header.Set("User-Agent", "mkr-meta-pkg")

	client := &http.Client{}
	client.Timeout = 30 * time.Second
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("API Request Error: %d", resp.StatusCode)
	}

	jsonData, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(jsonData))
	return nil
}
