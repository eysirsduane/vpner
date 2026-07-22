package xxpay

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
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
	MchId      string
	AppId      string
	ProductId  string
	MchOrderNo string
	Amount     int
	ClientIP   string
	Device     string
	NotifyURL  string
	ReturnURL  string
	Subject    string
	Body       string
	Key        string
}

type CreateOrderResponse struct {
	RetCode    string      `json:"retCode"`
	RetMsg     string      `json:"retMsg"`
	PayOrderId string      `json:"payOrderId"`
	PayMethod  string      `json:"payMethod"`
	PayURL     string      `json:"payUrl"`
	PayJumpURL string      `json:"payJumpUrl"`
	PayParams  interface{} `json:"payParams"`
	CodeURL    string      `json:"codeUrl"`
	CodeImgURL string      `json:"codeImgUrl"`
	Sign       string      `json:"sign"`
}

type QueryOrderResponse struct {
	RetCode    string `json:"retCode"`
	RetMsg     string `json:"retMsg"`
	MchId      string `json:"mchId"`
	AppId      string `json:"appId"`
	ProductId  string `json:"productId"`
	PayOrderId string `json:"payOrderId"`
	MchOrderNo string `json:"mchOrderNo"`
	Amount     int    `json:"amount"`
	Status     string `json:"status"`
	Sign       string `json:"sign"`
}

func Sign(params map[string]string, key string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	signText := strings.Join(parts, "&") + "&key=" + key
	return strings.ToUpper(util.Md5(signText))
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
	if strings.TrimSpace(req.Device) == "" {
		req.Device = "WEB"
	}
	if strings.TrimSpace(req.Subject) == "" {
		req.Subject = req.MchOrderNo
	}
	if strings.TrimSpace(req.Body) == "" {
		req.Body = req.Subject
	}
	params := map[string]string{
		"mchId":      req.MchId,
		"appId":      req.AppId,
		"productId":  req.ProductId,
		"mchOrderNo": req.MchOrderNo,
		"amount":     util.ItoS(req.Amount),
		"currency":   "cny",
		"clientIp":   req.ClientIP,
		"device":     req.Device,
		"notifyUrl":  req.NotifyURL,
		"returnUrl":  req.ReturnURL,
		"subject":    req.Subject,
		"body":       req.Body,
		"reqTime":    time.Now().Format("20060102150405"),
		"version":    "1.0",
	}
	params["sign"] = Sign(params, req.Key)

	body, err := postForm(req.APIURL, "/api/pay/create_order", params)
	if err != nil {
		return PayInfo{}, err
	}
	var res CreateOrderResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return PayInfo{}, err
	}
	if res.RetCode != "0" {
		msg := strings.TrimSpace(res.RetMsg)
		if msg == "" {
			msg = "create xxpay order failed"
		}
		return PayInfo{}, errors.New(msg)
	}
	payURL := payURLFromResponse(res)
	if payURL == "" {
		return PayInfo{}, errors.New("xxpay pay url is empty")
	}
	return PayInfo{
		PayOrderId: res.PayOrderId,
		MchOrderNo: req.MchOrderNo,
		PayURL:     payURL,
		PayMethod:  res.PayMethod,
	}, nil
}

func (req CreateOrderRequest) Validate() error {
	if strings.TrimSpace(req.APIURL) == "" {
		return errors.New("xxpay api url is required")
	}
	if strings.TrimSpace(req.MchId) == "" {
		return errors.New("xxpay mch_id is required")
	}
	if strings.TrimSpace(req.ProductId) == "" {
		return errors.New("xxpay product_id is required")
	}
	if strings.TrimSpace(req.MchOrderNo) == "" {
		return errors.New("xxpay order_no is required")
	}
	if req.Amount <= 0 {
		return errors.New("xxpay amount must be greater than 0")
	}
	if strings.TrimSpace(req.NotifyURL) == "" {
		return errors.New("xxpay notify_url is required")
	}
	if strings.TrimSpace(req.Key) == "" {
		return errors.New("xxpay key is required")
	}
	return nil
}

func QueryOrder(apiURL, mchId, mchOrderNo, key string) (QueryOrderResponse, error) {
	params := map[string]string{
		"mchId":         mchId,
		"mchOrderNo":    mchOrderNo,
		"executeNotify": "false",
		"reqTime":       time.Now().Format("20060102150405"),
		"version":       "1.0",
	}
	params["sign"] = Sign(params, key)

	body, err := postForm(apiURL, "/api/pay/query_order", params)
	if err != nil {
		return QueryOrderResponse{}, err
	}
	if !VerifyJSONSign(body, key) {
		return QueryOrderResponse{}, errors.New("xxpay response sign error")
	}
	var res QueryOrderResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return QueryOrderResponse{}, err
	}
	if res.RetCode != "0" {
		msg := strings.TrimSpace(res.RetMsg)
		if msg == "" {
			msg = "query xxpay order failed"
		}
		return QueryOrderResponse{}, errors.New(msg)
	}
	return res, nil
}

func VerifyJSONSign(body []byte, key string) bool {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return false
	}
	params := make(map[string]string, len(raw))
	for k, v := range raw {
		switch value := v.(type) {
		case nil:
			continue
		case string:
			params[k] = value
		case float64:
			params[k] = fmt.Sprintf("%.0f", value)
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
	return status == "2" || status == "3"
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
		return nil, fmt.Errorf("xxpay http status %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func payURLFromResponse(res CreateOrderResponse) string {
	for _, value := range []string{res.PayJumpURL, res.PayURL, res.CodeURL, res.CodeImgURL} {
		if payURL := extractPayURL(value); payURL != "" {
			return payURL
		}
	}
	return ""
}

func extractPayURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(value), "http://") || strings.HasPrefix(strings.ToLower(value), "https://") {
		return value
	}
	matches := regexp.MustCompile(`https?://[^'"<>\s]+`).FindStringSubmatch(value)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}
