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
)

const (
	PoolConfigmapName      = "ibm-flashsystem-pools"
	PoolConfigmapMountPath = "/config"
	PoolConfigmapKey       = "pools"

	CsiIBMBlockDriver = "block.csi.ibm.com"
	CsiIBMBlockScPool = "pool"
)

type ScPoolMap struct {
	ScPool map[string]string `json:"storageclass_pool,omitempty"`
}

func GeneratePoolConfigmapContent(sp ScPoolMap) (string, error) {

	data, err := json.Marshal(sp)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func readPoolConfigMapFile() ([]byte, error) {
	poolPath := filepath.Join(PoolConfigmapMountPath, PoolConfigmapKey)

	file, err := os.Open(poolPath) // For read access.
	if err != nil {
		//		if os.IsNotExist(err) {
		//			return "", nil
		//		} else {
		return nil, err
		//		}
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func GetPoolConfigmapContent() (ScPoolMap, error) {
	var sp ScPoolMap

	content, err := readPoolConfigMapFile()
	if err != nil {
		return sp, err
	}

	err = json.Unmarshal(content, &sp)
	return sp, err
}
