package file_utilities

import (
    "io/ioutil"
    "os"
)

func ReadFile(path string) (string, error) {
    bytes, err := ioutil.ReadFile(path)
    if err != nil {
        return "", err
    }

    return string(bytes), nil
}

func FileExists(path string) bool {
    _, err := os.Stat(path)
    if err != nil {
        return false
    }

    return true
}
