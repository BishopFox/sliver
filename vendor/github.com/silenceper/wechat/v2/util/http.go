package util

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"golang.org/x/crypto/pkcs12"
)

// URIModifier URI修改器
type URIModifier func(uri string) string

var uriModifier URIModifier

// DefaultHTTPClient 默认httpClient
var DefaultHTTPClient = http.DefaultClient

// SetURIModifier 设置URI修改器
func SetURIModifier(fn URIModifier) {
	uriModifier = fn
}

// HTTPGet get 请求
func HTTPGet(uri string) ([]byte, error) {
	return HTTPGetContext(context.Background(), uri)
}

// HTTPGetContext get 请求
func HTTPGetContext(ctx context.Context, uri string) ([]byte, error) {
	if uriModifier != nil {
		uri = uriModifier(uri)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	response, err := DefaultHTTPClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http get error : uri=%v , statusCode=%v", uri, response.StatusCode)
	}
	return io.ReadAll(response.Body)
}

// HTTPPost post 请求
func HTTPPost(uri string, data string) ([]byte, error) {
	return HTTPPostContext(context.Background(), uri, []byte(data), nil)
}

// HTTPPostContext post 请求
func HTTPPostContext(ctx context.Context, uri string, data []byte, header map[string]string) ([]byte, error) {
	if uriModifier != nil {
		uri = uriModifier(uri)
	}
	body := bytes.NewBuffer(data)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, body)
	if err != nil {
		return nil, err
	}

	for key, value := range header {
		request.Header.Set(key, value)
	}

	response, err := DefaultHTTPClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http post error : uri=%v , statusCode=%v", uri, response.StatusCode)
	}
	return io.ReadAll(response.Body)
}

// PostJSONContext post json 数据请求
func PostJSONContext(ctx context.Context, uri string, obj interface{}) ([]byte, error) {
	if uriModifier != nil {
		uri = uriModifier(uri)
	}
	jsonBuf := new(bytes.Buffer)
	enc := json.NewEncoder(jsonBuf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(obj)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", uri, jsonBuf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	response, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http get error : uri=%v , statusCode=%v", uri, response.StatusCode)
	}
	return io.ReadAll(response.Body)
}

// PostJSON post json 数据请求
func PostJSON(uri string, obj interface{}) ([]byte, error) {
	return PostJSONContext(context.Background(), uri, obj)
}

// PostJSONWithRespContentType post json 数据请求，且返回数据类型
func PostJSONWithRespContentType(uri string, obj interface{}) ([]byte, string, error) {
	jsonBuf := new(bytes.Buffer)
	enc := json.NewEncoder(jsonBuf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(obj)
	if err != nil {
		return nil, "", err
	}

	response, err := DefaultHTTPClient.Post(uri, "application/json;charset=utf-8", jsonBuf)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("http get error : uri=%v , statusCode=%v", uri, response.StatusCode)
	}
	responseData, err := io.ReadAll(response.Body)
	contentType := response.Header.Get("Content-Type")
	return responseData, contentType, err
}

// PostFileByStream 上传文件
func PostFileByStream(fieldName, fileName, uri string, byteData []byte) ([]byte, error) {
	fields := []MultipartFormField{
		{
			IsFile:    false,
			Fieldname: fieldName,
			Filename:  fileName,
			Value:     byteData,
		},
	}
	return PostMultipartForm(fields, uri)
}

// PostFile 上传文件
func PostFile(fieldName, filePath, uri string) ([]byte, error) {
	fields := []MultipartFormField{
		{
			IsFile:    true,
			Fieldname: fieldName,
			FilePath:  filePath,
			Filename:  filePath,
		},
	}
	return PostMultipartForm(fields, uri)
}

// PostFileFromReader 上传文件，从 io.Reader 中读取
func PostFileFromReader(filedName, filePath, fileName, uri string, reader io.Reader) ([]byte, error) {
	fields := []MultipartFormField{
		{
			IsFile:     true,
			Fieldname:  filedName,
			FilePath:   filePath,
			Filename:   fileName,
			FileReader: reader,
		},
	}
	return PostMultipartForm(fields, uri)
}

// MultipartFormField 保存文件或其他字段信息
type MultipartFormField struct {
	IsFile     bool
	Fieldname  string
	Value      []byte
	FilePath   string
	Filename   string
	FileReader io.Reader
}

// PostMultipartForm 上传文件或其他多个字段
func PostMultipartForm(fields []MultipartFormField, uri string) (respBody []byte, err error) {
	if uriModifier != nil {
		uri = uriModifier(uri)
	}
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	for _, field := range fields {
		if field.IsFile {
			fileWriter, e := bodyWriter.CreateFormFile(field.Fieldname, field.Filename)
			if e != nil {
				err = fmt.Errorf("error writing to buffer , err=%v", e)
				return
			}

			if field.FileReader == nil {
				fh, e := os.Open(field.FilePath)
				if e != nil {
					err = fmt.Errorf("error opening file , err=%v", e)
					return
				}
				_, err = io.Copy(fileWriter, fh)
				_ = fh.Close()
				if err != nil {
					return
				}
			} else {
				if _, err = io.Copy(fileWriter, field.FileReader); err != nil {
					return
				}
			}
		} else {
			partWriter, e := bodyWriter.CreateFormFile(field.Fieldname, field.Filename)
			if e != nil {
				err = e
				return
			}
			valueReader := bytes.NewReader(field.Value)
			if _, err = io.Copy(partWriter, valueReader); err != nil {
				return
			}
		}
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, e := DefaultHTTPClient.Post(uri, contentType, bodyBuf)
	if e != nil {
		err = e
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http code error : uri=%v , statusCode=%v", uri, resp.StatusCode)
	}
	respBody, err = io.ReadAll(resp.Body)
	return
}

// PostXML perform a HTTP/POST request with XML body
func PostXML(uri string, obj interface{}) ([]byte, error) {
	if uriModifier != nil {
		uri = uriModifier(uri)
	}
	xmlData, err := xml.Marshal(obj)
	if err != nil {
		return nil, err
	}

	body := bytes.NewBuffer(xmlData)
	response, err := DefaultHTTPClient.Post(uri, "application/xml;charset=utf-8", body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http code error : uri=%v , statusCode=%v", uri, response.StatusCode)
	}
	return io.ReadAll(response.Body)
}

// httpWithTLS CA 证书
func httpWithTLS(rootCa, key string) (*http.Client, error) {
	var client *http.Client
	certData, err := os.ReadFile(rootCa)
	if err != nil {
		return nil, fmt.Errorf("unable to find cert path=%s, error=%v", rootCa, err)
	}
	cert := pkcs12ToPem(certData, key)
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// 安全地获取 *http.Transport
	var trans *http.Transport
	// 尝试从 DefaultHTTPClient 获取 Transport，如果失败则使用默认值
	if DefaultHTTPClient.Transport != nil {
		if t, ok := DefaultHTTPClient.Transport.(*http.Transport); ok {
			trans = t.Clone()
		}
	}
	// 如果无法获取有效的 Transport，使用默认值
	if trans == nil {
		trans = http.DefaultTransport.(*http.Transport).Clone()
	}

	trans.TLSClientConfig = config
	trans.DisableCompression = true
	client = &http.Client{Transport: trans}
	return client, nil
}

// pkcs12ToPem 将 Pkcs12 转成 Pem
func pkcs12ToPem(p12 []byte, password string) tls.Certificate {
	blocks, err := pkcs12.ToPEM(p12, password)
	defer func() {
		if x := recover(); x != nil {
			log.Print(x)
		}
	}()
	if err != nil {
		panic(err)
	}
	var pemData []byte
	for _, b := range blocks {
		pemData = append(pemData, pem.EncodeToMemory(b)...)
	}
	cert, err := tls.X509KeyPair(pemData, pemData)
	if err != nil {
		panic(err)
	}
	return cert
}

// PostXMLWithTLS perform a HTTP/POST request with XML body and TLS
func PostXMLWithTLS(uri string, obj interface{}, ca, key string) ([]byte, error) {
	if uriModifier != nil {
		uri = uriModifier(uri)
	}
	xmlData, err := xml.Marshal(obj)
	if err != nil {
		return nil, err
	}

	body := bytes.NewBuffer(xmlData)
	client, err := httpWithTLS(ca, key)
	if err != nil {
		return nil, err
	}
	response, err := client.Post(uri, "application/xml;charset=utf-8", body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http code error : uri=%v , statusCode=%v", uri, response.StatusCode)
	}
	return io.ReadAll(response.Body)
}
