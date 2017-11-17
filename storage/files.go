package storage

import (
	"fmt"
	"io/ioutil"

	"github.com/Djoulzy/Tools/clog"
)

const (
	_FILEDB_PATH_ = "../../FileDB/"
)

func SaveUser(ID string, json []byte) {
	fileName := fmt.Sprintf("%s/users/%s.txt", _FILEDB_PATH_, ID)
	err := ioutil.WriteFile(fileName, json, 0644)
	if err != nil {
		clog.Error("storage", "SaveUser", "Can't save User %s : %s", ID, err)
	}
}

func LoadUser(ID string) ([]byte, error) {
	fileName := fmt.Sprintf("%s/users/%s.txt", _FILEDB_PATH_, ID)
	json, err := ioutil.ReadFile(fileName)
	if err != nil {
		clog.Error("storage", "SaveUser", "Can't read User %s : %s", ID, err)
	}
	return json, err
}
