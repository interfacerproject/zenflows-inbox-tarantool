package main

import (
	"bytes"
	_ "embed"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	zenroom "github.com/dyne/Zenroom/bindings/golang/zenroom"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

//go:embed zenflows-crypto/src/gen_pubkey.zen
var SK2PK string

type WalletAgent struct {
	Keyring   map[string]string
	WalletUrl string
}

func (wa *WalletAgent) pkHeader() (string, string) {
	keys, _ := json.Marshal(map[string]interface{}{"keyring": wa.Keyring})
	result, success := zenroom.ZencodeExec(string(SK2PK), "", "", string(keys))
	if !success {
		panic(result.Logs)
	}
	var resDecoded map[string]string
	if err := json.Unmarshal([]byte(result.Output), &resDecoded); err != nil {
		panic(err)
	}
	return "did-pk", resDecoded["eddsa_public_key"]
}

func (wa *WalletAgent) signatureHeader(jsonData []byte) (string, string) {
	data := fmt.Sprintf(`{"gql": "%s"}`, b64.StdEncoding.EncodeToString(jsonData))
	keys, _ := json.Marshal(map[string]interface{}{"keyring": wa.Keyring})
	result, success := zenroom.ZencodeExec(SIGN, "", data, string(keys))
	if !success {
		panic(result.Logs)
	}
	var resDecoded map[string]string
	if err := json.Unmarshal([]byte(result.Output), &resDecoded); err != nil {
		panic(err)
	}
	return "did-sign", resDecoded["eddsa_signature"]
}

func (wa *WalletAgent) makeRequest(path string, query []byte) ([]byte, error) {
	url, err := url.Parse(fmt.Sprintf("%s/%s", wa.WalletUrl, path))
	if err != nil {
		log.Fatal(err)
	}
	r, err := http.NewRequest("POST", url.String(), bytes.NewReader(query))
	if err != nil {
		panic(err)
	}
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add(wa.signatureHeader(query))
	r.Header.Add(wa.pkHeader())
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (wa *WalletAgent) addPoints(amount uint64, id string, token string) error {
	query, err := json.Marshal(map[string]interface{}{
		"token":  token,
		"amount": strconv.FormatUint(amount, 10),
		"owner":  id,
	})

	body, err := wa.makeRequest("token", query)
	if err != nil {
		return err
	}

	var result map[string]bool
	json.Unmarshal(body, &result)

	if !result["success"] {
		return errors.New("Error in the response from wallet")
	}

	return nil
}

func (wa *WalletAgent) AddStrengthPoints(amount uint64, id string) error {
	return wa.addPoints(amount, id, "strength")
}

func (wa *WalletAgent) AddIdeaPoints(amount uint64, id string) error {
	return wa.addPoints(amount, id, "idea")
}
