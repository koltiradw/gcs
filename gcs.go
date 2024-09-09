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
	"encoding/binary"
)

const HOST = "localhost"
const TYPE = "tcp"
const PORT = "3001"


const PROF_FILE = "coverage.profile"
const LCOV_FILE = "coverage.lcov"

// WuppieFuzz proto for lcov coverage client
const HEADER_SIZE = 8
var REQUEST_HEADER = [...]byte{0x01, 0xC0, 0xC0, 0x10, 0x07}
const BLOCK_CMD_DUMP = 0x40
var COVERAGE_INFO_RESPONSE = []byte{0x11}
var CMD_OK_RESPONSE = []byte{0x20}

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
	
	_, err := cmd.Output()
	if err != nil {
  		log.Fatal(err)
	}


	//gen lcov file
	path_to_lcov_file := filepath.Join(cover_dir, LCOV_FILE)
	cmd = exec.Command("gcov2lcov", fmt.Sprintf("-infile=%s", path_to_proffile), fmt.Sprintf("-outfile=%s", path_to_lcov_file))	
	
	_, err = cmd.Output()
	if err != nil {
		log.Fatal(err)
	}


	content, err := ioutil.ReadFile(path_to_lcov_file)

	if err != nil {
                log.Fatal(err)
        }
	
	removeGlob(fmt.Sprintf("%s/covcounters*", cover_dir))

	return content

}

func handleRequest(conn net.Conn){
	buffer := make([]byte, 8)
	_, err := conn.Read(buffer)
	if err != nil {
		return
	}
	
	cmd := buffer[5]

	if int(cmd) == BLOCK_CMD_DUMP {
		dumpCoverage()
		lcov := getLCOV()
		size := make([]byte, 4)
		binary.LittleEndian.PutUint32(size, uint32(len(lcov)))
		conn.Write(COVERAGE_INFO_RESPONSE)
		conn.Write(size)
		conn.Write(lcov)
		conn.Write(CMD_OK_RESPONSE)
	}
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

	conn, err := listen.Accept()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	for {
		handleRequest(conn)
	}
}
