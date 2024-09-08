package gcs

import (
        "os"
        "log"
        "runtime/coverage"
        "net/http"
)

func dumpCoverage(w http.ResponseWriter, r *http.Request) {
        coverage.WriteCountersDir(os.Getenv("GOCOVERDIR"))
        coverage.ClearCounters()
}

func StartCoverageServer() {
        http.HandleFunc("/cover", dumpCoverage)
        log.Fatal(http.ListenAndServe(":6300", nil))
}
