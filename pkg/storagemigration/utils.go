package storagemigration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

func SwitchAllContainersStorageDriver(root, newStorageDriver string) error {
	containerDir := filepath.Join(root, "containers")
	containerIDs, err := loadIDs(containerDir)
	if err != nil {
		return fmt.Errorf("Error listing containers: %v", err)
	}
	logrus.Infof("migrating %v container(s) to %s", len(containerIDs), newStorageDriver)
	for _, containerID := range containerIDs {
		err := switchContainerStorageDriver(root, containerID, newStorageDriver)
		if err != nil {
			return fmt.Errorf("Error rewriting container config for %s: %v", containerID, err)
		}
		logrus.WithField("container_id", containerID).Debugf("reconfigured storage-driver to %s", newStorageDriver)
	}
	return nil
}

// switchContainerStorageDriver rewrites the container config to use a new storage driver,
// this is the only change needed to make it work after the migration
func switchContainerStorageDriver(root, containerID, newStorageDriver string) error {
	containerConfigPath := filepath.Join(root, "containers", containerID, "config.v2.json")
	f, err := os.OpenFile(containerConfigPath, os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	var containerConfig = make(map[string]interface{})
	err = json.NewDecoder(f).Decode(&containerConfig)
	if err != nil {
		return err
	}
	containerConfig["Driver"] = newStorageDriver

	err = f.Truncate(0)
	if err != nil {
		return err
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}
	err = json.NewEncoder(f).Encode(&containerConfig)
	if err != nil {
		return err
	}
	err = f.Sync()
	if err != nil {
		return err
	}
	return nil
}

// replicate hardlinks all files from sourceDir to targetDir, reusing the same
// file structure
func replicate(sourceDir, targetDir string) error {
	return filepath.Walk(sourceDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		var (
			targetPath = strings.Replace(path, sourceDir, targetDir, 1)
			logrus     = logrus.WithField("path", targetPath)
		)

		if fi.IsDir() {
			logrus.Debug("creating directory")
			err = os.MkdirAll(targetPath, os.ModeDir|0755)
			if err != nil {
				return err
			}
		} else {
			logrus.Debug("create hardlink")
			err = os.Link(path, targetPath)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func removeDirIfExists(path string) error {
	ok, err := exists(path, true)
	if err != nil {
		return err
	}
	if ok {
		logrus.Infof("removing %s", path)
		err = os.RemoveAll(path)
		if err != nil {
			return err
		}
	}
	return nil
}