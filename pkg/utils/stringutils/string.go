package stringutils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"
const letterBytes = "abcdefghijklmnopqrstuvwxyz1234567890"

var re = regexp.MustCompile(ansi)

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func Split(str string, sep string) []string {
	if str == "" {
		return nil
	}
	return strings.Split(str, sep)
}

func StripAnsi(str string) string {
	return re.ReplaceAllString(str, "")
}

func ReplaceStringOnce(source, oldstr string) string {
	return strings.Replace(source, oldstr, "", 1)
}

func DataVolumeNameToDiskName(name string) string {
	//tpl-vol-xxxxxx    to vol-xxxx
	if len(name) > 8 {
		return name[4:]
	}
	return ""
}

func StringToInt(str string) (ret int) {
	ret, _ = strconv.Atoi(str)
	return ret
}

func GetNetWorkNameFromProvider(provider string) (ret string) {
	nets := strings.Split(provider, ".")
	if len(nets) == 3 {
		ret = nets[1] + "/" + nets[0]
	}
	return ret
}

func GetAuthorizationFromCookie(cookie string) string {
	ret := "Bearer "
	tokens := strings.Split(cookie, ";")
	for _, token := range tokens {
		if strings.Contains(token, "X-Ai-Authorization=") {
			ret += strings.Replace(token, "X-Ai-Authorization=", "", 1)
			return ret
		}
	}
	return ""
}

func CompareCpuMoreThan(reqCpu, limitCpu string) bool {
	var reqCpuInt int
	var limitCpuInt int
	if strings.Contains(reqCpu, "m") {
		reqCpuInt, _ = strconv.Atoi(strings.Replace(reqCpu, "m", "", 1))
	} else {
		reqCpuInt, _ = strconv.Atoi(reqCpu)
		reqCpuInt = 1000 * reqCpuInt
	}

	if strings.Contains(limitCpu, "m") {
		limitCpuInt, _ = strconv.Atoi(strings.Replace(limitCpu, "m", "", 1))
	} else {
		limitCpuInt, _ = strconv.Atoi(limitCpu)
		limitCpuInt = 1000 * limitCpuInt
	}

	return reqCpuInt > limitCpuInt
}

func CompareMemoryMoreThan(reqMem, limitMem string) bool {
	var reqMemInt int64
	var limitMemInt int64
	if strings.Contains(reqMem, "Ki") {
		reqMemInt, _ = strconv.ParseInt(strings.Replace(reqMem, "Ki", "", 1), 10, 64)
		reqMemInt = 1024 * reqMemInt
	} else if strings.Contains(reqMem, "Mi") {
		reqMemInt, _ = strconv.ParseInt(strings.Replace(reqMem, "Mi", "", 1), 10, 64)
		reqMemInt = 1024 * 1024 * reqMemInt
	} else if strings.Contains(reqMem, "Gi") {
		reqMemInt, _ = strconv.ParseInt(strings.Replace(reqMem, "Gi", "", 1), 10, 64)
		reqMemInt = 1024 * 1024 * 1024 * reqMemInt
	} else {
		reqMemInt, _ = strconv.ParseInt(reqMem, 10, 64)
	}

	if strings.Contains(limitMem, "Ki") {
		limitMemInt, _ = strconv.ParseInt(strings.Replace(limitMem, "Ki", "", 1), 10, 64)
		limitMemInt = 1024 * limitMemInt
	} else if strings.Contains(limitMem, "Mi") {
		limitMemInt, _ = strconv.ParseInt(strings.Replace(limitMem, "Mi", "", 1), 10, 64)
		limitMemInt = 1024 * 1024 * limitMemInt
	} else if strings.Contains(limitMem, "Gi") {
		limitMemInt, _ = strconv.ParseInt(strings.Replace(limitMem, "Gi", "", 1), 10, 64)
		limitMemInt = 1024 * 1024 * 1024 * limitMemInt
	} else {
		limitMemInt, _ = strconv.ParseInt(limitMem, 10, 64)
	}

	return reqMemInt > limitMemInt
}

func CompareCpuMoreOrEqual(reqCpu, limitCpu string) bool {
	var reqCpuInt int
	var limitCpuInt int
	if strings.Contains(reqCpu, "m") {
		reqCpuInt, _ = strconv.Atoi(strings.Replace(reqCpu, "m", "", 1))
	} else {
		reqCpuInt, _ = strconv.Atoi(reqCpu)
		reqCpuInt = 1000 * reqCpuInt
	}

	if strings.Contains(limitCpu, "m") {
		limitCpuInt, _ = strconv.Atoi(strings.Replace(limitCpu, "m", "", 1))
	} else {
		limitCpuInt, _ = strconv.Atoi(limitCpu)
		limitCpuInt = 1000 * limitCpuInt
	}

	return reqCpuInt >= limitCpuInt
}

func CompareMemoryMoreOrEqual(reqMem, limitMem string) bool {
	var reqMemInt int64
	var limitMemInt int64
	if strings.Contains(reqMem, "Ki") {
		reqMemInt, _ = strconv.ParseInt(strings.Replace(reqMem, "Ki", "", 1), 10, 64)
		reqMemInt = 1024 * reqMemInt
	} else if strings.Contains(reqMem, "Mi") {
		reqMemInt, _ = strconv.ParseInt(strings.Replace(reqMem, "Mi", "", 1), 10, 64)
		reqMemInt = 1024 * 1024 * reqMemInt
	} else if strings.Contains(reqMem, "Gi") {
		reqMemInt, _ = strconv.ParseInt(strings.Replace(reqMem, "Gi", "", 1), 10, 64)
		reqMemInt = 1024 * 1024 * 1024 * reqMemInt
	} else {
		reqMemInt, _ = strconv.ParseInt(reqMem, 10, 64)
	}

	if strings.Contains(limitMem, "Ki") {
		limitMemInt, _ = strconv.ParseInt(strings.Replace(limitMem, "Ki", "", 1), 10, 64)
		limitMemInt = 1024 * limitMemInt
	} else if strings.Contains(limitMem, "Mi") {
		limitMemInt, _ = strconv.ParseInt(strings.Replace(limitMem, "Mi", "", 1), 10, 64)
		limitMemInt = 1024 * 1024 * limitMemInt
	} else if strings.Contains(limitMem, "Gi") {
		limitMemInt, _ = strconv.ParseInt(strings.Replace(limitMem, "Gi", "", 1), 10, 64)
		limitMemInt = 1024 * 1024 * 1024 * limitMemInt
	} else {
		limitMemInt, _ = strconv.ParseInt(limitMem, 10, 64)
	}

	return reqMemInt >= limitMemInt
}

func GetTotalCPU(cpu string, replicas int) string {
	var reqCpuInt int
	if strings.Contains(cpu, "m") {
		reqCpuInt, _ = strconv.Atoi(strings.Replace(cpu, "m", "", 1))
	} else {
		reqCpuInt, _ = strconv.Atoi(cpu)
		reqCpuInt = 1000 * reqCpuInt
	}
	return strconv.Itoa(reqCpuInt*replicas) + "m"
}

func GetTotalMemory(memory string, replicas int) string {
	var reqMemInt int64
	if strings.Contains(memory, "Ki") {
		reqMemInt, _ = strconv.ParseInt(strings.Replace(memory, "Ki", "", 1), 10, 64)
		reqMemInt = 1024 * reqMemInt
	} else if strings.Contains(memory, "Mi") {
		reqMemInt, _ = strconv.ParseInt(strings.Replace(memory, "Mi", "", 1), 10, 64)
		reqMemInt = 1024 * 1024 * reqMemInt
	} else if strings.Contains(memory, "Gi") {
		reqMemInt, _ = strconv.ParseInt(strings.Replace(memory, "Gi", "", 1), 10, 64)
		reqMemInt = 1024 * 1024 * 1024 * reqMemInt
	} else {
		reqMemInt, _ = strconv.ParseInt(memory, 10, 64)
	}

	return strconv.FormatInt(reqMemInt*int64(replicas), 10)
}

func AesEncrypt(orig string, key string) string {
	// 转成字节数组
	origData := []byte(orig)
	k := []byte(key)

	// 分组秘钥
	block, err := aes.NewCipher(k)
	if err != nil {
		panic(fmt.Sprintf("key 长度必须 16/24/32长度: %s", err.Error()))
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 补全码
	origData = pkcs7Padding(origData, blockSize)
	// 加密模式
	blockMode := cipher.NewCBCEncrypter(block, k[:blockSize])
	// 创建数组
	cryted := make([]byte, len(origData))
	// 加密
	blockMode.CryptBlocks(cryted, origData)
	//使用RawURLEncoding 不要使用StdEncoding
	//不要使用StdEncoding  放在url参数中回导致错误
	return base64.RawURLEncoding.EncodeToString(cryted)

}

func AesDecrypt(cryted string, key string) string {
	//使用RawURLEncoding 不要使用StdEncoding
	//不要使用StdEncoding  放在url参数中回导致错误
	crytedByte, _ := base64.RawURLEncoding.DecodeString(cryted)
	k := []byte(key)

	// 分组秘钥
	block, err := aes.NewCipher(k)
	if err != nil {
		panic(fmt.Sprintf("key 长度必须 16/24/32长度: %s", err.Error()))
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 加密模式
	blockMode := cipher.NewCBCDecrypter(block, k[:blockSize])
	// 创建数组
	orig := make([]byte, len(crytedByte))
	// 解密
	blockMode.CryptBlocks(orig, crytedByte)
	// 去补全码
	orig = pkcs7UnPadding(orig)
	return string(orig)
}

// 补码
func pkcs7Padding(ciphertext []byte, blocksize int) []byte {
	padding := blocksize - len(ciphertext)%blocksize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// 去码
func pkcs7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
