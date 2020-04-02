package volume

import (
	"fmt"
	"io/ioutil"
	"os"
)

type Volume struct {
	root string
}

func New(root string) *Volume {
	return &Volume{root: root}
}

func (v *Volume) Store(name string, data []byte) error {
	path := fmt.Sprintf("%s/%s", v.root, name)
	err := ioutil.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("Error storing the binary: %s", name)
	}
	return nil
}

func (v *Volume) Delete(name string) error {
	path := fmt.Sprintf("%s/%s", v.root, name)
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("Error deleting the binary: %s", name)
	}
	return nil
}

func (v *Volume) Location(name string) (string, error) {
	path := fmt.Sprintf("%s/%s", v.root, name)
	os.Stat(path)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("Error locating the binary: %s", name)
	}
	return path, nil
}
