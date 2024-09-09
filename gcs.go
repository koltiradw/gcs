package main

import (
        "os"
        "log"
        "runtime/coverage"
        "net"
	"os/exec"
	"fmt"
	"path/filepath"
	"io/ioutil"
)

const HOST = "localhost"
const TYPE = "tcp"
const PORT = "3001"


const PROF_FILE = "coverage.profile"
const LCOV_FILE = "coverage.lcov"

// WuppieFuzz proto for lcov coverage client
const HEADER_SIZE = 8
var REQUEST_HEADER = [...]byte{0x01, 0xC0, 0xC0, 0x10, 0x07}
const BLOCK_CMD_DUMP = 64
var COVERAGE_INFO_RESPONSE = [...]byte{0x11}
var CMD_OK_RESPONSE = [...]byte{0x20}

func removeGlob(path string) (err error) {
	contents, err := filepath.Glob(path)
	if err != nil {
        	return
	}
    	for _, item := range contents {
        	err = os.RemoveAll(item)
        	if err != nil {
            	return
        	}
    	}
    	return
}    

func dumpCoverage() {
	coverage.WriteCountersDir(os.Getenv("GOCOVERDIR"))
        coverage.ClearCounters()
}

func getLCOV() []byte {
	//gen profile file
	cover_dir := os.Getenv("GOCOVERDIR")
	path_to_proffile := filepath.Join(cover_dir, PROF_FILE)
	cmd := exec.Command("go", "tool", "covdata", "textfmt", fmt.Sprintf("-i=%s", cover_dir), fmt.Sprintf("-o=%s", path_to_proffile))
	
	output, err := cmd.Output()
	if err != nil {
  		log.Fatal(err)
	}
	fmt.Println(string(output))

	//gen lcov file
	path_to_lcov_file := filepath.Join(cover_dir, LCOV_FILE)
	cmd = exec.Command("gcov2lcov", fmt.Sprintf("-infile=%s", path_to_proffile), fmt.Sprintf("-outfile=%s", path_to_lcov_file))	
	
	output, err = cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(output))

	content, err := ioutil.ReadFile(path_to_lcov_file)

	if err != nil {
                log.Fatal(err)
        }
	
	removeGlob(fmt.Sprintf("%s/covcounters*", cover_dir))

	return content

}

func handleRequest(conn net.Conn){
	buffer := make([]byte, 64)
	_, err := conn.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}
	
	cmd := buffer[5]
	
	fmt.Printf("%s", buffer)

	if int(cmd) == BLOCK_CMD_DUMP {
		dumpCoverage()
		lcov := getLCOV()
		conn.Write(lcov)
	}

	conn.Close()

}

func init() {
	go startCoverageServer()
}

func startCoverageServer() {
	listen, err := net.Listen(TYPE, HOST + ":" + PORT)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	defer listen.Close()
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		go handleRequest(conn)
	}
}
