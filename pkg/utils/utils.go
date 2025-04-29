package utils

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"unsafe"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
)

func UniqueID() string {
	uuidRequest, err := uuid.NewV7()
	if err != nil {
		return strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	return strings.ReplaceAll(uuidRequest.String(), "-", "")
}

func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", fmt.Errorf("failed to get interface addresses: %w", err)
	}

	for _, addr := range addrs {
		// Check if the address is IP network
		if ipnet, ok := addr.(*net.IPNet); ok {
			// Skip loopback addresses
			if ipnet.IP.IsLoopback() {
				continue
			}
			// We want IPv4 address
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no valid local IP address found")
}

// Marshal 对象转JSON字符串
func Marshal(v interface{}) ([]byte, error) {
	return sonic.Marshal(v)
}

// Unmarshal JSON字符串转interface{}
func Unmarshal(in []byte, body interface{}) error {
	return sonic.Unmarshal(in, body)
}

// MustToJSON 对象转JSON字符串
func MustToJSON(obj interface{}) string {
	str, _ := sonic.Marshal(obj)
	return Bytes2Str(str)
}

// MustToJSON 对象转JSON字符串
func MustToJSONByte(obj interface{}) string {
	str, _ := sonic.Marshal(obj)
	return Bytes2Str(str)
}

// MustFromJSON JSON字符串转interface{}
func MustFromJSON(data string) (m interface{}) {
	sonic.Unmarshal(Str2Bytes(data), &m)
	return m
}

func AnyToMap(data any) map[string]any {
	tmp, _ := Marshal(data)
	var m map[string]any
	_ = Unmarshal(tmp, &m)
	return m
}

func MapToAny[T any](data map[string]any) T {
	var m T
	tmp, _ := Marshal(data)
	_ = Unmarshal(tmp, &m)
	return m
}

// Bytes2Str converts byte slice to string.
func Bytes2Str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Str2Bytes converts string to byte slice.
func Str2Bytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&struct {
		string
		Cap int
	}{s, len(s)}))
}

func WriteResp(w http.ResponseWriter, httpStatus int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(body)
}

func WriteRespWithHttpStatus(w http.ResponseWriter, httpStatus int) {
	w.WriteHeader(httpStatus)
	fmt.Fprint(w, http.StatusText(httpStatus))
}
