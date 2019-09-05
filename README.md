# close-wait-watcher

//执行系统命令
f, err := exec.LookPath("ls")  
if err != nil {  
    fmt.Println(err)  
}  
fmt.Println(f) //  /bin/ls  


netstat -n | awk '/^tcp/ {++S[$NF]} END {for(a in S) print a, S[a]}'

//定时器


func main() {
    ticker := time.NewTicker(5 * time.Second)
    quit := make(chan int)
    var wg  sync.WaitGroup
 
    wg.Add(1)
    go func() {
        defer wg.Done()
        fmt.Println("child goroutine bootstrap start")
        for {
            select {
                case <- ticker.C:
                    fmt.Println("ticker .")
                case <- quit:
                    fmt.Println("work well .")
                    ticker.Stop()
                    return
            }
        }
        fmt.Println("child goroutine bootstrap end")
    }()
    time.Sleep(10 * time.Second)
    quit <- 1
    wg.Wait()
}

//CLOSE_WAIT

conf

//secod
interval=5
//colose_wait number
close_wait_max=5000
//restart service
service_command="service restart ss"
