package main

import (
	"bytes"
	"flag"
	"path/filepath"
	"fmt"
	"github.com/Unknwon/goconfig"
	"log"
	"net"
	"os"
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
	log bool
	log_fle string
	service_port string
}

var conf_vale confstr
var logFile *os.File
var Logtofile *log.Logger


func init() {
	s := flag.String("conf","./watcher.conf","-c /etc/watcher.conf")
	flag.Parse()

	if *s == "" {
		flag.Usage()
		panic("process exsit!")
	}
	c, err := filepath.Abs(*s)
	if err != nil {
		panic(err)
	}

	cfg, err := goconfig.LoadConfigFile(c)
	if err != nil {
		log.Println("读取配置文件失败[config.ini]")
		panic(err)
	}
	conf_vale.Getconf(cfg,err)
	if conf_vale.log {
		logFile, err = os.OpenFile(conf_vale.log_fle, os.O_WRONLY | os.O_CREATE | os.O_APPEND, 0644)
		if err != nil {
			log.Fatal(err)
		}
		Logtofile = log.New(logFile, "[watcher] ", log.Ldate|log.Ltime|log.LstdFlags)
	}else {
		Logtofile = log.New(os.Stdout, "[watcher] ", log.Ldate|log.Ltime|log.LstdFlags)
	}
}

func (this *confstr) Getconf(c *goconfig.ConfigFile,err error) {
	this.service_command, err  = c.GetValue("main","service_command")
	if err != nil {
		Logtofile.Panic(err)
	}
	this.watch_command = "netstat -n | awk '/^tcp/ {++S[$NF]} END {for(a in S) print a, S[a]}'"

	close_num ,err:= c.GetValue("main","close_wait_max")
	if err != nil {
		Logtofile.Panic(err)
	}
	this.close_wait_max,err = strconv.Atoi(close_num)
	if err != nil {
		Logtofile.Panic(err)
	}
	interval_num, err := c.GetValue("main","interval")
	if err != nil {
		Logtofile.Panic(err)
	}

	this.interval, err = time.ParseDuration(interval_num)
	if err != nil {
		Logtofile.Panic(err)
	}

	logstr, err := c.GetValue("main","log")
	if err != nil {
		Logtofile.Panic(err)
	}
	this.log = ParseBool(logstr)

	this.log_fle, err = c.GetValue("main","log_file")

	this.service_port,err = c.GetValue("main","service_port")
	if err != nil {
		Logtofile.Panic(err)
	}
}

func ParseBool(str string) bool {
	switch str {
	case "1", "t", "T", "true", "TRUE", "True":
		return true
	case "0", "f", "F", "false", "FALSE", "False":
		return false
	}
	return false
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



func get_close_wait_num(c string,L *log.Logger) (num int){
	err, sout ,_ := Shellout(c)
	if err != nil {
		L.Println(err)
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

func restart_service(c string,L *log.Logger)  {
	err, sout ,serr := Shellout(c)
	if err != nil {
		L.Println(err)
	}
	if sout != "" {
		L.Println(sout)
	}
	if serr != "" {
		L.Println(serr)
	}
}

func main()  {
	ticker := time.NewTicker(conf_vale.interval)
	var wg  sync.WaitGroup

	wg.Add(1)
	Logtofile.Println("service wather start")
	go func(L *log.Logger) {
		defer wg.Done()
		for {
			select {
			case <- ticker.C:
				num := get_close_wait_num(conf_vale.watch_command,L)
				if num > conf_vale.close_wait_max {
					restart_service(conf_vale.service_command,L)
				}
				L.Println("CLOSE_WAIT: ",num)
				srv_ipaddr := "127.0.0.1:" + conf_vale.service_port
				_, err := net.Dial("tcp", srv_ipaddr)
				if err == nil {
					L.Println("service is runing")
				}else{
					L.Println("service is closed")
					restart_service(conf_vale.service_command,L)
				}
			}
		}
	}(Logtofile)
	wg.Wait()
}
