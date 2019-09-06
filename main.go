package main

import (
	"bytes"
	"fmt"
	"github.com/Unknwon/goconfig"
	"log"
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
}

var conf_vale confstr
var logFile *os.File
var Logtofile *log.Logger


func init() {
	cfg, err := goconfig.LoadConfigFile("conf")
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

        logstr, err := c.GetValue("main","log")
        if err != nil {
                log.Panic(err)
        }
        this.log = ParseBool(logstr)

        this.log_fle, err = c.GetValue("main","log_file")
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
	L.Println("restart service \n",sout)
	L.Println("standerror :",serr)
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
			}
		}
	}(Logtofile)
	wg.Wait()
}
