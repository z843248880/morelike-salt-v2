package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

type tcpClient struct {
	net.Conn
	r *bufio.Reader
}

func readContent(r *bufio.Reader) {
	for {
		line, err := r.ReadString('\n')
		//当对端断开tcp连接后，己方就会收到“io.EOF”错误。
		if err != nil || err == io.EOF {
			// fmt.Printf("readContent(r *bufio.Reader) is \"err != nil || err == io.EOF\", error: %v\n", err)
			break
		}
		//if len(strings.TrimSpace(line)) != 0 {
		if strings.TrimSpace(line) == "EOF" {
			fmt.Println("minion is down status.")
		} else {
			fmt.Print(line)
		}
		//}
	}
}

func (c *tcpClient) Run(cmd string) {
	c.Conn.Write([]byte("myrole-bc3aa319-51b0-47a0-8b13-612d3bfe4d00-isclient\n"))
	c.Conn.Write([]byte(cmd))
	readContent(c.r)
}

func main() {
	minionip := flag.String("minionip", "", "minion ip")
	masterip := flag.String("masterip", "", "master ip") //172.10.2.92:38080
	cmd := flag.String("c", "", "command")
	flag.Parse()
	if len(*cmd) == 0 || len(*minionip) == 0 || len(*masterip) == 0 {
		fmt.Println("command or minionip or clientip is null")
		os.Exit(1)
	}
	*cmd = *minionip + "^" + *cmd + "\n"
	conn, err := net.Dial("tcp", *masterip)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	r := bufio.NewReader(conn)
	tc := &tcpClient{conn, r}
	tc.Run(*cmd)
}
