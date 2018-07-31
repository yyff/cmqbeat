package cmq

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/yyff/cmqbeat/config"
)

const (
	RecvAction      = "ReceiveMessage"
	DeleteMessage   = "DeleteMessage"
	SignatureMethod = "HmacSHA256"
	METHOD          = "GET"
)

type KeyValue struct {
	Key   string
	Value interface{}
}

type CMQAPI struct {
	URL                string
	QueueName          string
	Region             string
	SecretID           string
	SecretKey          string
	PollingWaitSeconds int
}

func NewCMQAPI(config *config.CMQConfig) *CMQAPI {
	t := &CMQAPI{
		URL:                config.URL,
		QueueName:          config.QueueName,
		Region:             config.Region,
		SecretID:           config.SecretID,
		SecretKey:          config.SecretKey,
		PollingWaitSeconds: config.PollingWaitSeconds,
	}
	return t
}

type ReceiveMessageRsp struct {
	Code          int
	Message       string
	MsgBody       string
	ReceiptHandle string
}

func (api *CMQAPI) RecvMsg() (string, string, error) {
	para := []KeyValue{
		KeyValue{"Action", RecvAction},
		KeyValue{"Region", api.Region},
		KeyValue{"Nonce", time.Now().Unix()},
		KeyValue{"SecretId", api.SecretID},
		KeyValue{"SignatureMethod", SignatureMethod},
		KeyValue{"Timestamp", time.Now().Unix()},
		KeyValue{"queueName", api.QueueName},
		KeyValue{"pollingWaitSeconds", api.PollingWaitSeconds},
	}
	hosturi := ""
	if len(api.URL) > len("http://") && api.URL[0:len("http://")] == "http://" {
		hosturi = api.URL[len("http://"):]
	} else if len(api.URL) > len("https://") && api.URL[0:len("https://")] == "https://" {
		hosturi = api.URL[len("https://"):]
	} else {
		return "", "", errors.New("Invalid url: " + api.URL)
	}

	sig := genSignature(METHOD, hosturi, api.SecretKey, para)
	logp.Debug("", "gen signature: "+sig)

	para = append(para, KeyValue{"Signature", sig})

	paraStr := genGetPara(para)
	reqUrl := api.URL + "?" + paraStr
	logp.Debug("", "req url: "+reqUrl)

	res, err := http.Get(reqUrl)
	if err != nil {
		logp.Err("%v", err)
		return "", "", fmt.Errorf("request error: %v, url: %v", err, reqUrl)
	}
	bodyBytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		logp.Err("%v", err)
		return "", "", fmt.Errorf("ReadAll error: ", err)

	}
	logp.Debug("", "rsp:"+string(bodyBytes))
	recvMsg := &ReceiveMessageRsp{}
	err = json.Unmarshal(bodyBytes, recvMsg)
	if err != nil {
		return "", "", err
	}
	if recvMsg.Code != 0 {
		return "", "", errors.New(recvMsg.Message)
	}
	if recvMsg.Code == 7000 { // no message
		return "", "", nil
	}
	logp.Info("ReceiveMessageRsp: %v", recvMsg)
	return recvMsg.MsgBody, recvMsg.ReceiptHandle, nil
}
func (api *CMQAPI) DeleteMsg(receiptHandle string) error {
	para := []KeyValue{
		KeyValue{"Action", DeleteMessage},
		KeyValue{"Region", api.Region},
		KeyValue{"Nonce", time.Now().Unix()},
		KeyValue{"SecretId", api.SecretID},
		KeyValue{"SignatureMethod", SignatureMethod},
		KeyValue{"Timestamp", time.Now().Unix()},
		KeyValue{"queueName", api.QueueName},
		KeyValue{"receiptHandle", receiptHandle},
	}

	sig := genSignature(METHOD, api.URL[len("https://"):], api.SecretKey, para)
	log.Print("sign str: ", sig)

	para = append(para, KeyValue{"Signature", sig})

	paraStr := genGetPara(para)
	reqUrl := api.URL + "?" + paraStr
	logp.Debug("", "req: "+reqUrl)

	res, err := http.Get(reqUrl)
	if err != nil {
		return fmt.Errorf("request error: %v, url: %v", err, reqUrl)
	}
	bodyBytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return fmt.Errorf("ReadAll error: %v", err)
	}
	logp.Debug("", "rsp: "+string(bodyBytes))
	recvMsg := &ReceiveMessageRsp{}
	err = json.Unmarshal(bodyBytes, recvMsg)
	if err != nil {
		return err
	}
	if recvMsg.Code != 0 {
		return errors.New(recvMsg.Message)
	}
	return nil

}

func genSignature(method, hosturi, secKey string, kvs []KeyValue) string {
	sort.Slice(kvs, func(i, j int) bool {
		return kvs[i].Key < kvs[j].Key
	})

	// paraStr := genGetPara(kvs)
	// no escape
	var paraStr string
	for i, kv := range kvs {
		if i != 0 {
			paraStr += "&"
		}
		paraStr += fmt.Sprintf("%v=%v", kv.Key, kv.Value)
	}
	originSigStr := method + hosturi + "?" + paraStr
	log.Print("origin sign str: ", originSigStr)
	mac := hmac.New(sha256.New, []byte(secKey))
	mac.Write([]byte(originSigStr))
	signByte := mac.Sum(nil)
	log.Print("sha256:", string(signByte))
	return base64.StdEncoding.EncodeToString(signByte)
}

func genGetPara(kvs []KeyValue) string {
	var result string
	for i, kv := range kvs {
		if i != 0 {
			result += "&"
		}
		v := url.QueryEscape(fmt.Sprint(kv.Value))
		result += fmt.Sprintf("%v=%v", kv.Key, v)
	}
	return result
}
