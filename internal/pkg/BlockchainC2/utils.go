package BlockchainC2

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	rand2 "math/rand"
	"net/http"
	"strconv"
	"time"
)

// Contents of random strings to be generated
const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// infuraGetTransactionCount used to deserialize eth_TransactionCount response from Infura API
type infuraGetTransactionCount struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  string `json:"result"`
}

// RandStringBytes returns a random string of n length containing alphanumeric characters
func RandStringBytes(n int) string {
	rand2.Seed(time.Now().UTC().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand2.Intn(len(letterBytes))]
	}
	return string(b)
}

// splitSubN is used to return a string split into 'n' pieces
func splitSubN(s string, n int) []string {
	sub := ""
	subs := []string{}

	runes := bytes.Runes([]byte(s))
	l := len(runes)
	for i, r := range runes {
		sub = sub + string(r)
		if (i+1)%n == 0 {
			subs = append(subs, sub)
			sub = ""
		} else if (i + 1) == l {
			subs = append(subs, sub)
		}
	}

	return subs
}

// GetCurrentTransactionNonce returns the current transactionID from the Infura API for the provided wallet
func GetCurrentTransactionNonce(addr string) (int64, error) {
	resp, err := http.Post(
		"https://ropsten.infura.io/",
		"application/json",
		bytes.NewBuffer([]byte("{\"jsonrpc\":\"2.0\",\"method\":\"eth_getTransactionCount\",\"params\": [\""+addr+"\",\"latest\"],\"id\":1}")),
	)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var infuraResp infuraGetTransactionCount

	err = json.Unmarshal(body, &infuraResp)
	if err != nil {
		return 0, err
	}

	nonce, err := strconv.ParseInt(infuraResp.Result, 0, 64)
	if err != nil {
		return 0, err
	}
	return nonce, nil

}
