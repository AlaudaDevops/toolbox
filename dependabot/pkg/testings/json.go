/*
Copyright 2025 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testings

import (
	"encoding/json"
	"os"
)

func LoadJson(filePath string, obj any) (err error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(fileData, obj)
}

func MustLoadJson(filePath string, obj any) {
	err := LoadJson(filePath, obj)
	if err != nil {
		panic(err)
	}
}
