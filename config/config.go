package config

import (
	"fmt"
	"os"
)

const LOCAL_DB_PATH string = "./database/conferences.json"
const USER_DB_PATH string = "./database/userdata.json"

func GetSecret(key string) (string, error) {
	val, exist := os.LookupEnv(key)
	if exist {
		return val, nil
	}
	return "", fmt.Errorf("no env variable with key %v", key)
}
