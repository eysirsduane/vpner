package util

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	region "github.com/lionsoul2014/ip2region/binding/golang/ip2region"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type PhoneCode struct {
	PhoneCode string
	phone     string
}

// GetSpaceTime
// 获取剩余时间 (天,时,分)
func GetSpaceTime(vipTime int) string {
	now := GetNowInt()
	if vipTime <= now {
		return "未开通会员"
	}
	t := vipTime - now
	if t >= 86400 {
		return strconv.Itoa(F2i(math.Ceil(float64(t)/float64(86400)))) + " 天"
	}
	if t >= 3600 {
		return strconv.Itoa(F2i(math.Ceil(float64(t)/float64(3600)))) + " 小时"
	}
	return strconv.Itoa(F2i(math.Ceil(float64(t)/float64(60)))) + " 分钟"
}

func FormatTime(vipTime int) string {
	return time.Unix(int64(vipTime), 0).Format("2006-01-02")
}

func F2i(f float64) int {
	i, _ := strconv.Atoi(fmt.Sprintf("%1.0f", f))
	return i
}

func Int64ToStr(i int64) string {
	return fmt.Sprintf("%d", i)
}

func Float64ToStr(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

// 获取当前地址
func GetCurrPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}

// 检测邮箱格式
func CheckEmail(email string) bool {
	pattern := `\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*` //匹配电子邮箱
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(email)
}

// 检测自定义用户名格式
func CheckUsername(account string) bool {
	pattern := `^[a-zA-Z0-9]{6,20}$`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(account)
}

// 检测密码格式
func CheckPwd(password string) bool {
	pattern := `^[a-zA-Z0-9]{6,20}$`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(password)
}

// 检测手机号格式
func CheckPhone(phone string) bool {
	pattern := `^1[3456789]\d{9}$`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(phone)
}

// 检测QQ号格式
func CheckQq(qq string) bool {
	pattern := `[1-9][0-9]{4,14}`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(qq)
}

// 检测手机验证码格式
func VerifyPhone(phone string) bool {
	regular := "^1[3-9]\\d{9}$"
	reg := regexp.MustCompile(regular)
	return reg.MatchString(phone)
}

// 生成指定区间随机数（包括纯数字／纯字母／随机）
func Kand(size int, kind int) []byte {
	i_kind, kinds, result := kind, [][]int{[]int{10, 48}, []int{26, 97}, []int{26, 65}}, make([]byte, size)
	is_all := kind > 2 || kind < 0
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < size; i++ {
		if is_all { // random ikind
			i_kind = rand.Intn(3)
		}
		scope, base := kinds[i_kind][0], kinds[i_kind][1]
		result[i] = uint8(base + rand.Intn(scope))
	}
	return result
}

// 生成区间随机数
func RandInt(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return rand.Intn(max-min) + min
}

// md5加密
func Md5(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// sha1加密
func SHA1(s string) string {
	o := sha1.New()
	o.Write([]byte(s))
	return hex.EncodeToString(o.Sum(nil))
}

// 获取ip 地区信息
func GetIpInfo(filePath, ip string) region.IpInfo {
	regionSearcher, err := region.New(filePath)
	if err != nil {
		return region.IpInfo{}
	}
	defer regionSearcher.Close()
	ipInfo, _ := regionSearcher.MemorySearch(ip)
	return ipInfo
}

// 获取重定向信息
func HttpGetRedirect(url2 string) string {
	location := ""
	req, err := http.NewRequest("GET", url2, nil)
	if err != nil {
		return location
	}
	c := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 3 * time.Second,
	}
	resp, err := c.Do(req)

	if resp.StatusCode == 302 {
		location = resp.Header.Get("Location")
	}

	return location

}

// 支付使用，获取重定向后的url
func HttpGetReqUrl(url2 string) string {
	location := ""
	req, err := http.NewRequest("GET", url2, nil)
	if err != nil {
		return location
	}
	c := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := c.Do(req)

	if err != nil {
		return ""
	}
	if resp.StatusCode != 200 {
		return ""
	}
	return resp.Request.URL.String()

}

// jsonp 返回值
func JsonpReturn(c *gin.Context, code int, msg string, data interface{}) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	jsonpcallback := c.DefaultQuery("callback", "")
	if data == nil {
		data = gin.H{}
	}
	b, _ := json.Marshal(gin.H{
		"code": code,
		"msg":  msg,
		"data": data,
	})
	if jsonpcallback == "" {
		c.String(http.StatusOK, string(b))
	}
	c.String(http.StatusOK, fmt.Sprintf("%s(%s);", jsonpcallback, string(b)))
}

// content:请求放回的内容
func HttpPost(url string, data interface{}, contentType string) (content string) {
	jsonStr, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Add("content-type", contentType)
	if err != nil {
		panic(err)
	}
	defer req.Body.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	result, _ := ioutil.ReadAll(resp.Body)
	content = string(result)
	return
}

func HttpPostForm(postUrl string, param map[string]string) string {
	data := make(url.Values)
	for k, v := range param {
		data[k] = []string{v}
	}
	resp, err := http.PostForm(postUrl, data)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return string(body)
}

func HttpGET(url string) string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || err != nil {
		panic(err)
	}
	return string(body)
}

// time 转 Int
func Time2Int(t time.Time) int {
	return int(t.Unix())
}

// time 转 string
func Time2Str(t time.Time, formate string) string {
	i := Time2Int(t)
	return GetTimeStr(i, formate)
}

// 三元 实现
func Mif(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}

// 是否在数组内
func InArrar(s string, li []string) bool {
	if len(li) > 0 {
		for _, v := range li {
			if s == v {
				return true
			}
		}
	}
	return false
}
