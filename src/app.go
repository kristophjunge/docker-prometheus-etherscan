package main

import (
    "io"
    "net/http"
    "log"
    "os"
    "strconv"
    "io/ioutil"
    "encoding/json"
    "errors"
    "strings"
)

const LISTEN_ADDRESS = ":9205"
const API_URL = "https://api.etherscan.io/api"

var testMode string
var accountIds string
var apiKey string

type EtherScanBalanceMulti struct {
    Status string `json:"status"`
    Message string `json:"message"`
    Result []struct {
        Account string `json:"account"`
        Balance string `json:"balance"`
    } `json:"result"`
}

func integerToString(value int) string {
    return strconv.Itoa(value)
}

func baseUnitsToEth(value string, precision int) string {
    if (len(value) < precision) {
        value = strings.Repeat("0", precision - len(value)) + value;
    }
    return value[:len(value)-precision+1] + "." + value[len(value)-precision+1:]
}

func formatValue(key string, meta string, value string) string {
    result := key;
    if (meta != "") {
        result += "{" + meta + "}";
    }
    result += " "
    result += value
    result += "\n"
    return result
}

func queryData() (string, error) {
    // Build URL
    url := API_URL + "?module=account&action=balancemulti&address=" + accountIds + "&tag=latest&apikey=" + apiKey

    // Perform HTTP request
    resp, httpErr := http.Get(url);
    if httpErr != nil {
        return "", httpErr;
    }

    // Parse response
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return "", errors.New("HTTP returned code " + integerToString(resp.StatusCode))
    }
    bodyBytes, ioErr := ioutil.ReadAll(resp.Body)
    bodyString := string(bodyBytes)
    if ioErr != nil {
        return "", ioErr;
    }

    return bodyString, nil;
}

func getTestData() (string, error) {
    dir, err := os.Getwd()
    if err != nil {
        log.Fatal(err)
    }
    body, err := ioutil.ReadFile(dir + "/test.json")
    if err != nil {
        log.Fatal(err)
    }
    return string(body), nil
}

func metrics(w http.ResponseWriter, r *http.Request) {
    log.Print("Serving /metrics")

    up := 1

    var jsonString string
    var err error
    if (testMode == "1") {
        jsonString, err = getTestData()
    } else {
        jsonString, err = queryData()
    }
    if err != nil {
        log.Print(err)
        up = 0
    }

    // Parse JSON
    jsonData := EtherScanBalanceMulti{}
    json.Unmarshal([]byte(jsonString), &jsonData)

    // Check response status
    if (jsonData.Status != "1") {
        log.Print("Received negative status in JSON response '" + jsonData.Status + "'")
        log.Print(jsonString)
        up = 0
    }

    // Output
    io.WriteString(w, formatValue("etherscan_up", "", integerToString(up)))
    for _, Account := range jsonData.Result {
        io.WriteString(w, formatValue("etherscan_up", "account=\"" + Account.Account + "\"", baseUnitsToEth(Account.Balance, 19)))
    }
}

func index(w http.ResponseWriter, r *http.Request) {
    log.Print("Serving /index")
    html := string(`<!doctype html>
<html>
    <head>
        <meta charset="utf-8">
        <title>Etherscan Exporter</title>
    </head>
    <body>
        <h1>Etherscan Exporter</h1>
        <p><a href="/metrics">Metrics</a></p>
    </body>
</html>
`)
    io.WriteString(w, html)
}

func main() {
    testMode = os.Getenv("TEST_MODE")
    if (testMode == "1") {
        log.Print("Test mode is enabled")
    }

    accountIds = os.Getenv("ACCOUNTS")
    log.Print("Monitoring account id's: " + accountIds)

    apiKey = os.Getenv("API_KEY")

    log.Print("Etherscan exporter listening on " + LISTEN_ADDRESS)
    http.HandleFunc("/", index)
    http.HandleFunc("/metrics", metrics)
    http.ListenAndServe(LISTEN_ADDRESS, nil)
}
