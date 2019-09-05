package main

import (
	"bytes"
	"fmt"
	"github.com/Unknwon/goconfig"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)


const ShellToUse = "bash"

type confstr struct {
	interval time.Duration
	close_wait_max int
	service_command string
	watch_command string
}

var conf_vale confstr

func init() {
	cfg, err := goconfig.LoadConfigFile("conf")
	if err != nil {
		log.Println("读取配置文件失败[config.ini]")
		panic(err)
	}
	conf_vale.Getconf(cfg,err)
}

func (this *confstr) Getconf(c *goconfig.ConfigFile,err error) {
	this.service_command, err  = c.GetValue("main","service_command")
	if err != nil {
		log.Panic(err)
	}
	this.watch_command = "netstat -n | awk '/^tcp/ {++S[$NF]} END {for(a in S) print a, S[a]}'"

	close_num ,err:= c.GetValue("main","close_wait_max")
	if err != nil {
		log.Panic(err)
	}
	this.close_wait_max,err = strconv.Atoi(close_num)
	if err != nil {
		log.Panic(err)
	}
	interval_num, err := c.GetValue("main","interval")
	if err != nil {
		log.Panic(err)
	}

	this.interval, err = time.ParseDuration(interval_num)
	if err != nil {
		log.Panic(err)
	}
}

func Shellout(command string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(ShellToUse, "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err, stdout.String(), stderr.String()
}



func get_close_wait_num(c string) (num int){
	err, sout ,_ := Shellout(c)
	if err != nil {
		log.Println(err)
	}
	a := strings.Split(sout,"\n")
	for _,i := range a {
		res := strings.HasPrefix(i,"CLOSE_WAIT")
		if res {
				snum := strings.Split(i," ")
				num,err = strconv.Atoi(fmt.Sprint(snum[1]))
				break
		}
	}
	return num
}

func restart_service(c string)  {
	err, sout ,serr := Shellout(c)
	if err != nil {
		log.Println(err)
	}
	log.Println("restart service ",sout)
	log.Println("standerror :",serr)
}

func main()  {
	ticker := time.NewTicker(conf_vale.interval)
	var wg  sync.WaitGroup

	wg.Add(1)
	log.Println("service wather start")
	go func() {
		defer wg.Done()
		for {
			select {
			case <- ticker.C:
				num := get_close_wait_num(conf_vale.watch_command)
				if num > conf_vale.close_wait_max {
					restart_service(conf_vale.service_command)
				}
			}
		}
	}()
	wg.Wait()
}
