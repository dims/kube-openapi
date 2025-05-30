/*
Copyright 2018 The Kubernetes Authors.

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

package idl_test

import (
	_ "k8s.io/kube-openapi/pkg/idl"
)

// This example shows how to use the structType atomic attribute to
// specify that this struct should be treated as a whole.
func ExampleStructType_atomic() {
	type SomeStruct struct {
		Name  string
		Value string
	}
	type SomeAPI struct {
		// +structType=atomic
		elements SomeStruct
	}
}
