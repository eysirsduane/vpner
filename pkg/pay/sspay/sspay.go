package sspay

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"just-vpn/pkg/util"
)

const (
	PayMethodFormJump = "formJump"
	PayMethodCodeImg  = "codeImg"
	PayMethodAlipay   = "alipayApp"
)

type PayInfo struct {
	PayOrderId string
	MchOrderNo string
	PayURL     string
	PayMethod  string
}

type CreateOrderRequest struct {
	APIURL     string
	Pid        string
	Type       string
	OutTradeNo string
	NotifyURL  string
	ReturnURL  string
	Name       string
	Money      string
	ClientIP   string
	Device     string
	SignType   string
	Key        string
}

type CreateOrderResponse struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	TradeNo   string `json:"trade_no"`
	PayURL    string `json:"payurl"`
	QrCode    string `json:"qrcode"`
	URLScheme string `json:"urlscheme"`
}

type QueryOrderResponse struct {
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
	TradeNo    string `json:"trade_no"`
	OutTradeNo string `json:"out_trade_no"`
	Type       string `json:"type"`
	Pid        int    `json:"pid"`
	Status     int    `json:"status"`
	Money      string `json:"money"`
	Sign       string `json:"sign"`
}

func Sign(params map[string]string, key string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || v == "" || k == "sign_type" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	signText := strings.Join(parts, "&") + key
	return strings.ToLower(util.Md5(signText))
}

func VerifySign(params map[string]string, key string) bool {
	sign := strings.TrimSpace(params["sign"])
	if sign == "" {
		return false
	}
	return strings.EqualFold(sign, Sign(params, key))
}

func CreateOrder(req CreateOrderRequest) (PayInfo, error) {
	if err := req.Validate(); err != nil {
		return PayInfo{}, err
	}
	if strings.TrimSpace(req.Type) == "" {
		req.Type = "alipay"
	}
	if strings.TrimSpace(req.Device) == "" {
		req.Device = "pc"
	}
	if strings.TrimSpace(req.Name) == "" {
		req.Name = req.OutTradeNo
	}
	if strings.TrimSpace(req.SignType) == "" {
		req.SignType = "MD5"
	}

	params := map[string]string{
		"pid":          req.Pid,
		"type":         req.Type,
		"out_trade_no": req.OutTradeNo,
		"notify_url":   req.NotifyURL,
		"return_url":   req.ReturnURL,
		"name":         req.Name,
		"money":        req.Money,
		"clientip":     req.ClientIP,
		"device":       req.Device,
		"sign_type":    req.SignType,
	}
	params["sign"] = Sign(params, req.Key)

	body, err := postForm(req.APIURL, "/mapi.php", params)
	if err != nil {
		return PayInfo{}, err
	}
	var res CreateOrderResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return PayInfo{}, err
	}
	if res.Code != 1 {
		msg := strings.TrimSpace(res.Msg)
		if msg == "" {
			msg = "create sspay order failed"
		}
		return PayInfo{}, errors.New(msg)
	}
	payURL := strings.TrimSpace(res.PayURL)
	method := PayMethodFormJump
	if payURL == "" && strings.TrimSpace(res.URLScheme) != "" {
		payURL = strings.TrimSpace(res.URLScheme)
		method = PayMethodAlipay
	}
	if payURL == "" && strings.TrimSpace(res.QrCode) != "" {
		// The client opens a URL. Let the provider's checkout render the QR code.
		delete(params, "clientip")
		delete(params, "device")
		params["sign"] = Sign(params, req.Key)
		form := url.Values{}
		for k, v := range params {
			if v != "" {
				form.Set(k, v)
			}
		}
		payURL = strings.TrimRight(req.APIURL, "/") + "/submit.php?" + form.Encode()
	}
	if payURL == "" {
		return PayInfo{}, errors.New("sspay pay url is empty")
	}
	return PayInfo{
		PayOrderId: res.TradeNo,
		MchOrderNo: req.OutTradeNo,
		PayURL:     payURL,
		PayMethod:  method,
	}, nil
}

func (req CreateOrderRequest) Validate() error {
	if strings.TrimSpace(req.APIURL) == "" {
		return errors.New("sspay api url is required")
	}
	if strings.TrimSpace(req.Pid) == "" {
		return errors.New("sspay pid is required")
	}
	if strings.TrimSpace(req.OutTradeNo) == "" {
		return errors.New("sspay order_no is required")
	}
	if req.Money == "" {
		return errors.New("sspay money is required")
	}
	if strings.TrimSpace(req.NotifyURL) == "" {
		return errors.New("sspay notify_url is required")
	}
	if strings.TrimSpace(req.Key) == "" {
		return errors.New("sspay key is required")
	}
	return nil
}

func QueryOrder(apiURL, pid, outTradeNo, key string) (QueryOrderResponse, error) {
	params := map[string]string{
		"act":          "order",
		"pid":          pid,
		"key":          key,
		"out_trade_no": outTradeNo,
	}
	body, err := getUrl(apiURL, "/api.php", params)
	if err != nil {
		return QueryOrderResponse{}, err
	}

	var res QueryOrderResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return QueryOrderResponse{}, err
	}
	if res.Code != 1 {
		msg := strings.TrimSpace(res.Msg)
		if msg == "" {
			msg = "query sspay order failed"
		}
		return QueryOrderResponse{}, errors.New(msg)
	}
	return res, nil
}

func VerifyJSONSign(body []byte, key string) bool {
	var raw map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return false
	}
	params := make(map[string]string, len(raw))
	for k, v := range raw {
		switch value := v.(type) {
		case nil:
			continue
		case string:
			params[k] = value
		case json.Number:
			params[k] = value.String()
		case bool:
			params[k] = fmt.Sprintf("%t", value)
		default:
			data, err := json.Marshal(value)
			if err != nil {
				continue
			}
			params[k] = string(data)
		}
	}
	return VerifySign(params, key)
}

func IsPaidStatus(status string) bool {
	return status == "TRADE_SUCCESS"
}

func getUrl(apiURL, path string, params map[string]string) ([]byte, error) {
	form := url.Values{}
	for k, v := range params {
		if v != "" {
			form.Set(k, v)
		}
	}
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", strings.TrimRight(apiURL, "/")+path+"?"+form.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("sspay get http status %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func postForm(apiURL, path string, params map[string]string) ([]byte, error) {
	form := url.Values{}
	for k, v := range params {
		if v != "" {
			form.Set(k, v)
		}
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.PostForm(strings.TrimRight(apiURL, "/")+path, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("sspay http status %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}
