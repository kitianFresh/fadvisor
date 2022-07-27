package credential

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

//const value
const (
	//moduleName      = "cls"
	defaultEndpoint = "sts.api.qcloud.com"
)

var defaultClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

// Credentials credential for yunapi auth
type Credentials struct {
	Token        string
	TmpSecretID  string
	TmpSecretKey string
	ExpiredTime  time.Time
}

//RequestParam request param
type RequestParam struct {
	Module  string
	Action  string
	Version string

	URL          string
	Timeout      time.Duration
	QueryParam   url.Values
	ExtraHeader  http.Header
	Request      interface{}
	RequestData  []byte
	Response     interface{}
	ResponseData []byte

	StatusCode      int
	StatusCodeCheck func(p *RequestParam) error
	// ResultCheck should set ResultCode and ResultMsg
	ResultCheck func(p *RequestParam, respData []byte) error
	ResultCode  string
	ResultMsg   string
	StartTime   time.Time

	CaCert  string
	KeyCert string
}

//Response sts credential response
type Response struct {
	Response *stsResponse
}

//SecretID return secret id
func (r *Response) SecretID() string {
	return r.Response.Credentials.TmpSecretID
}

//SecretKey return secret key
func (r *Response) SecretKey() string {
	return r.Response.Credentials.TmpSecretKey
}

//Token return token
func (r *Response) Token() string {
	return r.Response.Credentials.Token
}

//ExpiredTime return expired time
func (r *Response) ExpiredTime() int64 {
	return r.Response.ExpiredTime
}

type stsResponse struct {
	Error       *errorResponse
	Credentials *Credentials
	ExpiredTime int64
	Expiration  string
	RequestID   string
}

type errorResponse struct {
	Code    string
	Message string
}

//Config sts configure
type Config struct {
	Endpoint  string
	SecretID  string
	SecretKey string
}

//STS sts credential
type STS struct {
	longRegion       string
	cfg              *Config
	duration         time.Duration
	credentialsCache map[string]*Credentials //map[region:uin]Credential
}

//New return sts object
func NewSts(longRegion string, cfg Config) *STS {
	return &STS{
		longRegion:       longRegion,
		credentialsCache: map[string]*Credentials{},
		cfg: &Config{
			Endpoint:  cfg.Endpoint,
			SecretID:  cfg.SecretID,
			SecretKey: cfg.SecretKey,
		},
	}
}

//Get return tmpCredential
func (s *STS) Get(uin string) (*Credentials, error) {
	// 检查缓存中是否有数据
	cacheKey := fmt.Sprintf("%s:%s", s.longRegion, uin)
	cred := s.getCredentialFromCache(cacheKey)
	if cred != nil {
		return cred, nil
	}

	var reqURL string
	if strings.HasSuffix(s.cfg.Endpoint, "/") {
		reqURL = s.cfg.Endpoint
	} else {
		reqURL = s.cfg.Endpoint + "/"
	}

	params := url.Values{}
	params["RoleArn"] = []string{fmt.Sprintf("qcs::cam::uin/%s:roleName/%s", uin, "TKE_QCSRole")}
	params["RoleSessionName"] = []string{"TKE2CVM"}
	params["Version"] = []string{"2018-08-13"}
	params["Action"] = []string{"AssumeRole"}
	params["Region"] = []string{s.longRegion}
	params["Timestamp"] = []string{fmt.Sprintf("%d", time.Now().Unix())}
	params["Nonce"] = []string{strconv.Itoa(rand.Int())}
	params["SecretId"] = []string{s.cfg.SecretID}
	params["SignatureMethod"] = []string{"HmacSHA256"}
	params["Signature"] = []string{sign(s.cfg.SecretKey, reqURL, params)}
	resp := &Response{}
	p := &RequestParam{
		URL:        "https://" + reqURL,
		Module:     "tke",
		Action:     "AssumeRole",
		Version:    "n/a",
		Timeout:    time.Second * 30,
		QueryParam: params,
		Response:   resp,
		ResultCheck: func(p *RequestParam, respData []byte) error {
			if resp.Response == nil {
				return fmt.Errorf("empty response")
			}
			if resp.Response.Error != nil {
				p.ResultCode = fmt.Sprint(resp.Response.Error.Code)
				p.ResultMsg = resp.Response.Error.Message
				return fmt.Errorf(p.ResultMsg)
			}
			if resp.Response.Credentials == nil {
				return fmt.Errorf("empty credentials")
			}
			return nil
		},
	}

	if err := s.doGet(p); err != nil {
		klog.Errorf("get credential from remote failed. err:%v", err)
		return nil, err
	}
	cred = &Credentials{
		Token:        resp.Token(),
		TmpSecretKey: resp.SecretKey(),
		TmpSecretID:  resp.SecretID(),
		ExpiredTime:  time.Unix(resp.ExpiredTime(), 0),
	}
	s.duration = cred.ExpiredTime.Sub(time.Now())

	//存储凭证到缓存
	s.setCredentialToCache(cacheKey, cred)
	return cred, nil
}

func (s *STS) getCredentialFromCache(cacheKey string) *Credentials {
	cred, exists := s.credentialsCache[cacheKey]
	if !exists || cred == nil {
		return nil
	}
	if cred.ExpiredTime.Sub(time.Now()) < s.duration/2 {
		klog.Infof("sts credential is expired, expiredTime: %v, duration: %v", cred.ExpiredTime, s.duration)
		delete(s.credentialsCache, cacheKey)
		return nil
	}
	if time.Now().Add(2 * time.Minute).After(cred.ExpiredTime) { // 如果凭证还有2min过期，那就认为密钥过期了，重新获取密钥
		klog.Infof("sts credential is expired, expiredTime: %v", cred.ExpiredTime)
		delete(s.credentialsCache, cacheKey)
		return nil
	}

	return cred
}

func (s *STS) setCredentialToCache(cacheKey string, cred *Credentials) {
	if cred != nil {
		s.credentialsCache[cacheKey] = cred
	}
	return
}

func (s *STS) doGet(p *RequestParam) error {
	p.StartTime = time.Now()
	URL, err := url.Parse(p.URL)
	if err != nil {
		klog.Errorf("parse url failed. url:%s, err:%v", p.URL, err)
		return err
	}

	if len(p.QueryParam) != 0 {
		URL.RawQuery = p.QueryParam.Encode()
	}

	// response data
	resp, err := getResponse(URL.String(), http.MethodGet, p)
	if err != nil {
		return errors.Wrap(err, "get response failed")
	}

	return dealResponse(resp, p)
}

func sign(secretKey, url string, values url.Values) string {
	keys := make([]string, len(values))
	i := 0
	for k := range values {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	buf.WriteString("GET")
	buf.WriteString(url)
	buf.WriteString("?")
	for i, k := range keys {
		buf.WriteString(k)
		buf.WriteString("=")
		buf.WriteString(values[k][0])
		if i < len(keys)-1 {
			buf.WriteString("&")
		}
	}
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write(buf.Bytes())
	signByte := mac.Sum(nil)
	signature := base64.StdEncoding.EncodeToString(signByte)
	return signature
}

func dealResponse(resp *http.Response, p *RequestParam) error {
	defer func() { _ = resp.Body.Close() }()

	// check status code
	p.StatusCode = resp.StatusCode
	if p.StatusCodeCheck == nil {
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status code is %d", resp.StatusCode)
		}
	} else {
		if err := p.StatusCodeCheck(p); err != nil {
			return err
		}
	}

	// get response obj
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response data failed")
	}
	p.ResponseData = respData

	if p.Response != nil {
		if err := json.Unmarshal(respData, p.Response); err != nil {
			return fmt.Errorf("unmarshal json failed")
		}
	}

	// do checker
	if p.ResultCheck != nil {
		if err := p.ResultCheck(p, respData); err != nil {
			return fmt.Errorf(err.Error())
		}
	}

	return nil
}

func getResponse(url string, method string, p *RequestParam) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(p.RequestData))
	if err != nil {
		return nil, errors.Wrap(err, "new request failed")
	}

	ctx := context.Background()
	if p.Timeout == 0 {
		req = req.WithContext(ctx)
	} else {
		timeout, _ := context.WithTimeout(ctx, p.Timeout)
		req = req.WithContext(timeout)
	}

	for k, v := range p.ExtraHeader {
		req.Header[k] = v
	}

	client, err := getClient(p)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "client do request failed")
	}

	return resp, nil
}

func getClient(p *RequestParam) (*http.Client, error) {
	if p.CaCert == "" || p.KeyCert == "" {
		return defaultClient, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	cert, err := tls.LoadX509KeyPair(p.CaCert, p.KeyCert)
	if err != nil {
		return nil, err
	}
	tlsConfig.Certificates = []tls.Certificate{cert}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       tlsConfig,
	}

	client := &http.Client{
		Transport: transport,
	}

	if p.Timeout != 0 {
		client.Timeout = p.Timeout
	}

	return client, nil
}
