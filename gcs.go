package gcs

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/coverage"
	"os/signal"
	"context"
)

const HOST = "localhost"
const TYPE = "tcp"
const PORT = "3001"

const PROF_FILE = "coverage.profile"

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

func getLCOV() []byte {
	//gen profile file
	cover_dir := os.Getenv("GOCOVERDIR")
	path_to_proffile := filepath.Join(cover_dir, PROF_FILE)
	cmd := exec.Command("go", "tool", "covdata", "textfmt", fmt.Sprintf("-i=%s", cover_dir), fmt.Sprintf("-o=%s", path_to_proffile))

	_, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	prof_file, err := os.Open(path_to_proffile)

	if err != nil {
		log.Fatal(err)
	}

	defer prof_file.Close()

	prof_reader := bufio.NewReader(prof_file)

	lcov_buffer := bytes.NewBuffer(nil)
	writer := bufio.NewWriter(lcov_buffer)

	ConvertCoverage(prof_reader, writer)

	removeGlob(fmt.Sprintf("%s/covcounters*", cover_dir))

	return lcov_buffer.Bytes()

}

func handleRequest(_ context.Context, conn net.Conn) {
	buffer := make([]byte, 8)
	_, err := conn.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}

	cmd := buffer[5]

	if int(cmd) == BLOCK_CMD_DUMP {
		coverage.WriteCountersDir(os.Getenv("GOCOVERDIR"))
		lcov := getLCOV()
		size := make([]byte, 4)
		binary.LittleEndian.PutUint32(size, uint32(len(lcov)))
		conn.Write(COVERAGE_INFO_RESPONSE)
		conn.Write(size)
		conn.Write(lcov)
	}

	reset_byte := buffer[7]

	if int(reset_byte) != 0 {
		coverage.ClearCounters()
	}

	conn.Write(CMD_OK_RESPONSE)
}

func init() {
	go startCoverageServer()
}

func startCoverageServer() {
	ctx := context.Background()
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt, os.Kill)

	lc := new(net.ListenConfig)
	listen, err := lc.Listen(ctx, TYPE, HOST+ ":" +PORT)
	if err != nil {
		log.Fatal(err)
	}
	defer listen.Close()

	conn, err := listen.Accept()
	if err != nil {
		log.Fatal(err)
	}

	for ctx.Err() == nil {
		handleRequest(ctx, conn)
	}
}
