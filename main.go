package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/Unknwon/goconfig"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
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
	service_config string
}

var conf_vale confstr
var logFile *os.File
var Logtofile *log.Logger


func init() {
	//支持参数
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

	//初始化配置文件
	cfg, err := goconfig.LoadConfigFile(c)
	if err != nil {
		log.Println("读取配置文件失败[config.ini]")
		panic(err)
	}

	//解析配置文件
	conf_vale.Getconf(cfg,err)

	//初始化日志
	if conf_vale.log {
		//输出日志到文件
		logFile, err = os.OpenFile(conf_vale.log_fle, os.O_WRONLY | os.O_CREATE | os.O_APPEND, 0644)
		if err != nil {
			log.Fatal(err)
		}
		Logtofile = log.New(logFile, "[watcher] ", log.Ldate|log.Ltime|log.LstdFlags)
	}else {
		//输出日志到标准输出
		Logtofile = log.New(os.Stdout, "[watcher] ", log.Ldate|log.Ltime|log.LstdFlags)
	}

	//判断配置文件是否存在。如果不存在说明没装服务。
	var serivce_dir = "/home/ss/config.php"
	dir_bool,err :=  PathExists(serivce_dir)
	if err != nil {
		Logtofile.Println(err)
	}

	if !dir_bool {
		Logtofile.Println("can't find service config file ,watcher exit.\ncheck service if serivce was installed .")
		os.Exit(1)
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
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

	this.service_config,err = c.GetValue("main","service_config")
	if err != nil {
		Logtofile.Panic(err)
	}

	this.service_port = readconf(conf_vale.service_config)

}

func readconf(fp string) string {
	port := ""
	fi, err := os.Open(fp)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
	defer fi.Close()
	br := bufio.NewReader(fi)
	for  {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		portconf_prefix := "$config['port']"
		if strings.HasPrefix(string(a),portconf_prefix) {
			b := strings.TrimSpace(string(a))
			c := strings.Fields(b)
			d := strings.Split(c[2],"'")
			port = d[1]
		}
	}
	return port
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
	Logtofile.Println("service wather start")
	ticker := time.NewTicker(conf_vale.interval)
	var wg  sync.WaitGroup

	wg.Add(1)
	go func(L *log.Logger,wg *sync.WaitGroup) {
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
	}(Logtofile,&wg)
	wg.Wait()
}
