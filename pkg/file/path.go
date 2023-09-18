package file

import "os"

// PathExists checks if the given path exists.
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// PathExistOrCreate creates the directory at the given path if it does not exist.
func PathExistOrCreate(path string) error {
	exist, err := PathExists(path)
	if err == nil && !exist {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return err
}
