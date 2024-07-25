package testtools

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

var (
	ErrFileNameEmpty = errors.New("file name is empty")
)

const (
	jsonExt = "json"
	slash   = "/"
)

func ConvertJSONFileIntoStruct(path, name string, v interface{}) error {
	if len(name) == 0 {
		return ErrFileNameEmpty
	}

	// makes sure the file name has a json file extension
	if !strings.HasSuffix(name, jsonExt) {
		name += jsonExt
	}

	if len(path) > 0 {
		// makes sure the path has "/" suffix if there is a path
		if !strings.HasSuffix(path, slash) {
			path += slash
		}

		// makes sure the path has not "/" as prefix
		path = strings.TrimPrefix(path, slash)
	}

	fname := path + name

	f, err := os.Open(fname)
	if err != nil {
		return fmt.Errorf("could not open file %s error %s", fname, err)
	}
	defer f.Close()

	buf, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("could not read file %s error %s", fname, err)
	}

	err = json.Unmarshal(buf, v)
	if err != nil {
		return fmt.Errorf("could not unmarshal file contents into struct. file %s error %s", fname, err)
	}

	return nil
}

func GenerateCtxWithJSONAndParams(t *testing.T, data map[string]interface{}, params []gin.Param) *gin.Context {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = params
	ctx.Request = httptest.NewRequest("POST", "http://localhost:8080", nil)

	jsonbytes, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	ctx.Request.Body = io.NopCloser(bytes.NewReader(jsonbytes))

	return ctx
}
