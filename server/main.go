package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
)

var (
	conns    = sync.Map{}
	clientip = ""
)

func main() {
	clientIP := flag.String("cip", "", "client ip")
	port := flag.String("p", "", "service port")
	flag.Parse()
	if len(*clientIP) == 0 || len(*port) == 0 {
		fmt.Println("clientip or port is null")
		os.Exit(1)
	}
	clientip = *clientIP
	listener, err := net.Listen("tcp", *port)
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Print(err)
			continue
		}
		r := bufio.NewReader(conn)
		for {
			line, err := r.ReadString('\n')
			if err != nil || err == io.EOF || len(line) == 0 {
				//当这个server服务被nginx等工具反向代理了，那么server会收到nginx发送的健康检查，会收到一个空请求，即line是空。
				// fmt.Printf("error: %v; len(line): %d\n", err, len(line))
				break
			}
			fmt.Println("main line: ", line)
			if strings.TrimSpace(line) == "myrole-bc3aa319-51b0-47a0-8b13-612d3bfe4d00-isclient" {
				fmt.Println("client connected")
				go handleClientConn(r, conn)
				conns.Store(clientip, conn)
			} else if strings.TrimSpace(line) == "myrole-bc3aa319-51b0-47a0-8b13-612d3bfe4d00-isminion" {
				fmt.Println("minion connected")
				go handleMinionConn(r, conn)
			}
			break

		}
	}
}

func handleClientConn(r *bufio.Reader, c net.Conn) {
	fmt.Println("ok, start handleClientConn")
	var wg sync.WaitGroup
	//r := bufio.NewReader(c)
	for {
		line, err := r.ReadString('\n')
		//当对端断开tcp连接后，己方就会收到“io.EOF”错误。
		if err != nil || err == io.EOF {
			fmt.Printf("handleClientConn is \"err != nil || err == io.EOF\", error: %v\n", err)
			break
		}
		fmt.Println("handleClientConn input:", line)
		if strings.TrimSpace(line) == "EOF" {
			fmt.Println("client is down status.")
		} else {
			minionip := strings.SplitN(line, "^", -1)[0]
			cmd := strings.SplitN(line, "^", -1)[1]
			if _, ok := conns.Load(minionip); !ok {
				errinfo := fmt.Sprintf("minionip(%s) is not invalid", minionip)
				fmt.Println("stderr: ", errinfo)
				fmt.Fprintln(c, errinfo)
				c.Close()
				return
			}
			if v, ok := conns.Load(minionip); ok {
				fmt.Fprintln(v.(net.Conn), cmd)
			}

		}
		// conns.Range(func(k, v interface{}) bool {
		// 	fmt.Println("conn key: ", k, "conn value: ", v)
		// 	return true
		// })
		fmt.Println("conns: ", conns)
	}
	wg.Wait()
	c.Close()
	fmt.Println("ok, stop handleClientConn", clientip)
	if _, ok := conns.Load(clientip); ok {
		conns.Delete(clientip)
	}
}

func handleMinionConn(r *bufio.Reader, c net.Conn) {
	var remoteaddr string
	var wg sync.WaitGroup
	//r := bufio.NewReader(c)
	for {
		line, err := r.ReadString('\n')
		if err != nil || err == io.EOF {
			//走到这里说明minion主动断开连接了，那需要通知client现在minion已经断开连接，client收到EOF错误后就不会继续等待；同时服务端需要在连接池里删掉client和minion
			fmt.Println("handleMinionConn error: ", err)
			if v, ok := conns.Load(clientip); ok {
				fmt.Fprintln(v.(net.Conn), err.Error())
				v.(net.Conn).Close()
				conns.Delete(clientip)
			}
			if _, ok := conns.Load(remoteaddr); ok {
				conns.Delete(remoteaddr)
			}
			return
		}
		if strings.HasPrefix(line, "mylocalip-bc3aa319-51b0-47a0-8b13-612d3bfe4d00-is:") {
			remoteaddr = strings.TrimSpace(strings.SplitN(line, ":", -1)[1])
			conns.Store(remoteaddr, c)
		}
		if strings.HasPrefix(line, "woshi-bc3aa319-51b0-47a0-8b13-612d3bfe4d00-keepalive, minion:") {
			fmt.Print(line)
		} else if strings.Contains(line, "woyijing-bc3aa319-51b0-47a0-8b13-612d3bfe4d00-jieshule") {
			//line == "woyijing-bc3aa319-51b0-47a0-8b13-612d3bfe4d00-jieshule" 说明没有内容了，就可以断开连接了
			if v, ok := conns.Load(clientip); ok {
				v.(net.Conn).Close()
				conns.Delete(clientip)
			}
		} else {
			line = strings.TrimSuffix(line, "\n")
			if v, ok := conns.Load(clientip); ok {
				// conns.Range(func(k, v interface{}) bool {
				// 	fmt.Println("conn key: ", k, "conn value: ", v)
				// 	return true
				// })
				fmt.Println("conns:", conns)
				fmt.Fprintln(v.(net.Conn), line)
				fmt.Println("minion result:", line)
			}
		}
		// conns.Range(func(k, v interface{}) bool {
		// 	fmt.Println("conn key: ", k, "conn value: ", v)
		// 	return true
		// })
		fmt.Println("conns: ", conns)
	}
	wg.Wait()
	c.Close()
	fmt.Println("ok, stop handleMinionConn", remoteaddr)
	if _, ok := conns.Load(remoteaddr); ok {
		conns.Delete(remoteaddr)
	}
}
