package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

var (
	conn    net.Conn
	r       *bufio.Reader
	nettype string
	server  string
	localip string
)

const shellToUse = "/bin/bash"

func forKeepAlive() {
	fmt.Fprintln(conn, "woshi-bc3aa319-51b0-47a0-8b13-612d3bfe4d00-keepalive, minion: "+localip)
}

func initConn() (err error) {
	registerinfo := "mylocalip-bc3aa319-51b0-47a0-8b13-612d3bfe4d00-is:" + localip + "\n"
	fmt.Println("registerinfo: ", registerinfo)
	conn, err = net.Dial(nettype, server)
	if err != nil {
		return
	}
	conn.Write([]byte("myrole-bc3aa319-51b0-47a0-8b13-612d3bfe4d00-isminion\n"))
	conn.Write([]byte(registerinfo))
	r = bufio.NewReader(conn)
	return
}

func main() {
	flocalip := flag.String("localip", "", "local ip")
	fserver := flag.String("server", "", "server address")
	flag.Parse()
	if len(*fserver) == 0 || len(*flocalip) == 0 {
		fmt.Println("server address or localip is null")
		os.Exit(1)
	}

	nettype = "tcp"
	server = *fserver
	localip = *flocalip
	err := initConn()
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			forKeepAlive()
			time.Sleep(time.Minute)
		}
	}()
	r = bufio.NewReader(conn)
	for {
		command, err := r.ReadString('\n')
		if err == io.EOF {
			fmt.Println("error is io.EOF")
			err := initConn()
			fmt.Println("reconnect error: ", err)
			time.Sleep(time.Second * 5)
			continue
		}
		if err != nil {
			fmt.Println("read command error: ", err)
			continue
		}
		command = strings.TrimSuffix(command, "\n")
		if len(command) == 0 {
			continue
		}
		fmt.Println("command:", command)
		execCommand(command)
	}
}

func execCommand(command string) {
	wg := sync.WaitGroup{}
	out := make(chan string)
	defer close(out)
	go func() {
		for {
			str, ok := <-out
			if !ok {
				fmt.Println("zhendeshi buxinglema ok: ", ok)
				break
			}
			// fmt.Println("shuchujieguo:", str)
			fmt.Fprintln(conn, str)
			fmt.Println(str)
		}
	}()
	args := []string{"-c", command}
	if err := RunCommand(&wg, out, "bash", args...); err != nil {
		errinfo := fmt.Sprintf("RunCommand(out, \"bash\", %v) error: %v", args, err)
		fmt.Println(errinfo)
		// fmt.Fprintln(conn, errinfo)
	}
	wg.Wait()

	doneinfo := fmt.Sprintf("RunCommand(out, \"bash\", %v) exec done", args)
	fmt.Println(doneinfo)
	fmt.Fprintln(conn, "woyijing-bc3aa319-51b0-47a0-8b13-612d3bfe4d00-jieshule")
}

// RunCommand run shell
func RunCommand(wg *sync.WaitGroup, out chan string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return err
	}
	// wg := sync.WaitGroup{}
	// defer wg.Wait()
	wg.Add(2)
	go readLog(wg, out, stdout)
	go readLog(wg, out, stderr)
	time.Sleep(time.Millisecond * 100)
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func readLog(wg *sync.WaitGroup, out chan string, reader io.ReadCloser) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r, string(debug.Stack()))
		}
	}()
	defer wg.Done()
	r := bufio.NewReader(reader)
	for {
		line, _, err := r.ReadLine()
		if err == io.EOF || err != nil {
			return
		}
		out <- string(line)
	}
}
