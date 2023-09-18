package agent

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strconv"
)

// 设置文件的offset
func putLogFileOffset(dataPath string, filename string, offset int64) error {

	content := []byte(strconv.FormatInt(offset, 10))
	offilename := filepath.Join(dataPath, base64.StdEncoding.EncodeToString([]byte(filename))+".offset")
	err := os.WriteFile(offilename, content, 0644)
	return err
}

// 读取
func getLogFileOffset(dataPath string, filename string) (int64, error) {
	offilename := filepath.Join(dataPath, base64.StdEncoding.EncodeToString([]byte(filename))+".offset")
	var result int64
	offset, err := os.ReadFile(offilename)
	if err != nil {
		return result, err
	}

	result, err = strconv.ParseInt(string(offset), 10, 64)
	if err != nil {
		return result, err
	}

	return result, nil
}
