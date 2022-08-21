/**
 * Copyright contributors to the ibm-storage-odf-operator project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	PoolConfigmapName     = "ibm-flashsystem-pools"
	FSCConfigmapMountPath = "/config"
	CsiIBMBlockDriver     = "block.csi.ibm.com"
	CsiIBMBlockScPool     = "pool"
)

type FlashSystemClusterMapContent struct {
	ScPoolMap map[string]string `json:"storageclass"`
	Secret    string            `json:"secret"`
}

type FSCConfigMapData struct {
	FlashSystemClusterMap map[string]FlashSystemClusterMapContent
}

func GenerateFSCConfigmapContent(sp FSCConfigMapData) (map[string]string, error) {
	var configMapContent = make(map[string]string)
	for FSCName, FSCMapContent := range sp.FlashSystemClusterMap {
		val, err := json.Marshal(FSCMapContent)
		if err != nil {
			return make(map[string]string), err
		}
		configMapContent[FSCName] = string(val)
	}

	return configMapContent, nil
}

func ReadPoolConfigMapFile() ([]FlashSystemClusterMapContent, error) {
	var fscContents []FlashSystemClusterMapContent
	var fscContent FlashSystemClusterMapContent
	fscPath := FSCConfigmapMountPath + "/"

	files, err := ioutil.ReadDir(fscPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
			fscContent, err = getFileContent(filepath.Join(fscPath, file.Name()))
			if err != nil {
				return fscContents, err
			} else {
				fscContents = append(fscContents, fscContent)
			}
		}
	}
	return fscContents, nil
}

func getFileContent(filePath string) (FlashSystemClusterMapContent, error) {
	var fscContent FlashSystemClusterMapContent
	fileReader, err := os.Open(filePath)
	if err != nil {
		return fscContent, err
	}

	fileContent, _ := ioutil.ReadAll(fileReader)
	err = json.Unmarshal(fileContent, &fscContent)
	return fscContent, err
}
