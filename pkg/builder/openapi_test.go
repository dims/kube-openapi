/*
Copyright 2016 The Kubernetes Authors.

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

package builder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"

	openapi "k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/util/jsontesting"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

// setUp is a convenience function for setting up for (most) tests.
func setUp(t *testing.T, fullMethods bool) (*openapi.Config, *restful.Container, *assert.Assertions) {
	assert := assert.New(t)
	config, container := getConfig(fullMethods)
	return config, container, assert
}

func noOp(request *restful.Request, response *restful.Response) {}

// Test input
type TestInput struct {
	// Name of the input
	Name string `json:"name,omitempty"`
	// ID of the input
	ID   int      `json:"id,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

// Test output
type TestOutput struct {
	// Name of the output
	Name string `json:"name,omitempty"`
	// Number of outputs
	Count int `json:"count,omitempty"`
}

type TestExtensionV2Schema struct{}

func (_ TestExtensionV2Schema) OpenAPIDefinition() openapi.OpenAPIDefinition {
	schema := spec.Schema{
		VendorExtensible: spec.VendorExtensible{
			Extensions: map[string]interface{}{
				openapi.ExtensionV2Schema: spec.Schema{
					SchemaProps: spec.SchemaProps{
						Type: []string{"integer"},
					},
				},
			},
		},
	}
	schema.Description = "Test extension V2 spec conversion"
	schema.Properties = map[string]spec.Schema{
		"apple": {
			SchemaProps: spec.SchemaProps{
				Description: "Name of the output",
				Type:        []string{"string"},
				Format:      "",
			},
		},
	}
	return openapi.OpenAPIDefinition{
		Schema:       schema,
		Dependencies: []string{},
	}
}

func (_ TestInput) OpenAPIDefinition() openapi.OpenAPIDefinition {
	schema := spec.Schema{}
	schema.Description = "Test input"
	schema.Properties = map[string]spec.Schema{
		"name": {
			SchemaProps: spec.SchemaProps{
				Description: "Name of the input",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"id": {
			SchemaProps: spec.SchemaProps{
				Description: "ID of the input",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"tags": {
			SchemaProps: spec.SchemaProps{
				Description: "",
				Type:        []string{"array"},
				Items: &spec.SchemaOrArray{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
				},
			},
		},
	}
	schema.Extensions = spec.Extensions{"x-test": "test"}
	return openapi.OpenAPIDefinition{
		Schema:       schema,
		Dependencies: []string{},
	}
}

func (_ TestOutput) OpenAPIDefinition() openapi.OpenAPIDefinition {
	schema := spec.Schema{}
	schema.Description = "Test output"
	schema.Properties = map[string]spec.Schema{
		"name": {
			SchemaProps: spec.SchemaProps{
				Description: "Name of the output",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"count": {
			SchemaProps: spec.SchemaProps{
				Description: "Number of outputs",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
	}
	return openapi.OpenAPIDefinition{
		Schema:       schema,
		Dependencies: []string{},
	}
}

var _ openapi.OpenAPIDefinitionGetter = TestInput{}
var _ openapi.OpenAPIDefinitionGetter = TestOutput{}

func getTestRoute(ws *restful.WebService, method string, additionalParams bool, opPrefix string) *restful.RouteBuilder {
	ret := ws.Method(method).
		Path("/test/{path:*}").
		Doc(fmt.Sprintf("%s test input", method)).
		Operation(fmt.Sprintf("%s%sTestInput", method, opPrefix)).
		Produces(restful.MIME_JSON).
		Consumes(restful.MIME_JSON).
		Param(ws.PathParameter("path", "path to the resource").DataType("string")).
		Param(ws.QueryParameter("pretty", "If 'true', then the output is pretty printed.")).
		Reads(TestInput{}).
		Returns(200, "OK", TestOutput{}).
		Writes(TestOutput{}).
		To(noOp)
	if additionalParams {
		ret.Param(ws.HeaderParameter("hparam", "a test head parameter").DataType("integer"))
		ret.Param(ws.FormParameter("fparam", "a test form parameter").DataType("number"))
	}
	return ret
}

func getConfig(fullMethods bool) (*openapi.Config, *restful.Container) {
	mux := http.NewServeMux()
	container := restful.NewContainer()
	container.ServeMux = mux
	ws := new(restful.WebService)
	ws.Path("/foo")
	ws.Route(getTestRoute(ws, "get", true, "foo"))
	if fullMethods {
		ws.Route(getTestRoute(ws, "post", false, "foo")).
			Route(getTestRoute(ws, "put", false, "foo")).
			Route(getTestRoute(ws, "head", false, "foo")).
			Route(getTestRoute(ws, "patch", false, "foo")).
			Route(getTestRoute(ws, "options", false, "foo")).
			Route(getTestRoute(ws, "delete", false, "foo"))

	}
	ws.Path("/bar")
	ws.Route(getTestRoute(ws, "get", true, "bar"))
	if fullMethods {
		ws.Route(getTestRoute(ws, "post", false, "bar")).
			Route(getTestRoute(ws, "put", false, "bar")).
			Route(getTestRoute(ws, "head", false, "bar")).
			Route(getTestRoute(ws, "patch", false, "bar")).
			Route(getTestRoute(ws, "options", false, "bar")).
			Route(getTestRoute(ws, "delete", false, "bar"))

	}
	container.Add(ws)
	return &openapi.Config{
		ProtocolList: []string{"https"},
		Info: &spec.Info{
			InfoProps: spec.InfoProps{
				Title:       "TestAPI",
				Description: "Test API",
				Version:     "unversioned",
			},
		},
		GetDefinitions: func(_ openapi.ReferenceCallback) map[string]openapi.OpenAPIDefinition {
			return map[string]openapi.OpenAPIDefinition{
				"k8s.io/kube-openapi/pkg/builder.TestInput":             TestInput{}.OpenAPIDefinition(),
				"k8s.io/kube-openapi/pkg/builder.TestOutput":            TestOutput{}.OpenAPIDefinition(),
				"k8s.io/kube-openapi/pkg/builder.TestExtensionV2Schema": TestExtensionV2Schema{}.OpenAPIDefinition(),
			}
		},
		GetDefinitionName: func(name string) (string, spec.Extensions) {
			friendlyName := name[strings.LastIndex(name, "/")+1:]
			return friendlyName, spec.Extensions{"x-test2": "test2"}
		},
	}, container
}

func getTestOperation(method string, opPrefix string) *spec.Operation {
	return &spec.Operation{
		OperationProps: spec.OperationProps{
			Description: fmt.Sprintf("%s test input", method),
			Consumes:    []string{"application/json"},
			Produces:    []string{"application/json"},
			Schemes:     []string{"https"},
			Parameters:  []spec.Parameter{},
			Responses:   getTestResponses(),
			ID:          fmt.Sprintf("%s%sTestInput", method, opPrefix),
		},
	}
}

func getTestPathItem(allMethods bool, opPrefix string) spec.PathItem {
	ret := spec.PathItem{
		PathItemProps: spec.PathItemProps{
			Get:        getTestOperation("get", opPrefix),
			Parameters: getTestCommonParameters(),
		},
	}
	ret.Get.Parameters = getAdditionalTestParameters()
	if allMethods {
		ret.Put = getTestOperation("put", opPrefix)
		ret.Put.Parameters = getTestParameters()
		ret.Post = getTestOperation("post", opPrefix)
		ret.Post.Parameters = getTestParameters()
		ret.Head = getTestOperation("head", opPrefix)
		ret.Head.Parameters = getTestParameters()
		ret.Patch = getTestOperation("patch", opPrefix)
		ret.Patch.Parameters = getTestParameters()
		ret.Delete = getTestOperation("delete", opPrefix)
		ret.Delete.Parameters = getTestParameters()
		ret.Options = getTestOperation("options", opPrefix)
		ret.Options.Parameters = getTestParameters()
	}
	return ret
}

func getRefSchema(ref string) *spec.Schema {
	return &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef(ref),
		},
	}
}

func getTestResponses() *spec.Responses {
	ret := spec.Responses{
		ResponsesProps: spec.ResponsesProps{
			StatusCodeResponses: map[int]spec.Response{},
		},
	}
	ret.StatusCodeResponses[200] = spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: "OK",
			Schema:      getRefSchema("#/definitions/builder.TestOutput"),
		},
	}
	return &ret
}

func getTestCommonParameters() []spec.Parameter {
	ret := make([]spec.Parameter, 2)
	ret[0] = spec.Parameter{
		Refable: spec.Refable{
			Ref: spec.MustCreateRef("#/parameters/path-z6Ciiujn"),
		},
	}
	ret[1] = spec.Parameter{
		Refable: spec.Refable{
			Ref: spec.MustCreateRef("#/parameters/pretty-nN7o5FEq"),
		},
	}
	return ret
}

func getTestParameters() []spec.Parameter {
	ret := make([]spec.Parameter, 1)
	ret[0] = spec.Parameter{
		ParamProps: spec.ParamProps{
			In:       "body",
			Name:     "body",
			Required: true,
			Schema:   getRefSchema("#/definitions/builder.TestInput"),
		},
	}
	return ret
}

func getAdditionalTestParameters() []spec.Parameter {
	ret := make([]spec.Parameter, 3)
	ret[0] = spec.Parameter{
		ParamProps: spec.ParamProps{
			In:       "body",
			Name:     "body",
			Required: true,
			Schema:   getRefSchema("#/definitions/builder.TestInput"),
		},
	}
	ret[1] = spec.Parameter{
		Refable: spec.Refable{
			Ref: spec.MustCreateRef("#/parameters/fparam-xCJg5kHS"),
		},
	}
	ret[2] = spec.Parameter{
		Refable: spec.Refable{
			Ref: spec.MustCreateRef("#/parameters/hparam-tx-jfxM1"),
		},
	}
	return ret
}

func getTestInputDefinition() spec.Schema {
	return spec.Schema{
		SchemaProps: spec.SchemaProps{
			Description: "Test input",
			Properties: map[string]spec.Schema{
				"id": {
					SchemaProps: spec.SchemaProps{
						Description: "ID of the input",
						Type:        spec.StringOrArray{"integer"},
						Format:      "int32",
					},
				},
				"name": {
					SchemaProps: spec.SchemaProps{
						Description: "Name of the input",
						Type:        spec.StringOrArray{"string"},
					},
				},
				"tags": {
					SchemaProps: spec.SchemaProps{
						Type: spec.StringOrArray{"array"},
						Items: &spec.SchemaOrArray{
							Schema: &spec.Schema{
								SchemaProps: spec.SchemaProps{
									Type: spec.StringOrArray{"string"},
								},
							},
						},
					},
				},
			},
		},
		VendorExtensible: spec.VendorExtensible{
			Extensions: spec.Extensions{
				"x-test":  "test",
				"x-test2": "test2",
			},
		},
	}
}

func getTestOutputDefinition() spec.Schema {
	return spec.Schema{
		SchemaProps: spec.SchemaProps{
			Description: "Test output",
			Properties: map[string]spec.Schema{
				"count": {
					SchemaProps: spec.SchemaProps{
						Description: "Number of outputs",
						Type:        spec.StringOrArray{"integer"},
						Format:      "int32",
					},
				},
				"name": {
					SchemaProps: spec.SchemaProps{
						Description: "Name of the output",
						Type:        spec.StringOrArray{"string"},
					},
				},
			},
		},
		VendorExtensible: spec.VendorExtensible{
			Extensions: spec.Extensions{
				"x-test2": "test2",
			},
		},
	}
}

func TestBuildOpenAPISpec(t *testing.T) {
	config, container, assert := setUp(t, true)
	expected := &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Title:       "TestAPI",
					Description: "Test API",
					Version:     "unversioned",
				},
			},
			Swagger: "2.0",
			Paths: &spec.Paths{
				Paths: map[string]spec.PathItem{
					"/foo/test/{path}": getTestPathItem(true, "foo"),
					"/bar/test/{path}": getTestPathItem(true, "bar"),
				},
			},
			Definitions: spec.Definitions{
				"builder.TestInput":  getTestInputDefinition(),
				"builder.TestOutput": getTestOutputDefinition(),
			},
			Parameters: map[string]spec.Parameter{
				"fparam-xCJg5kHS": {
					CommonValidations: spec.CommonValidations{
						UniqueItems: true,
					},
					SimpleSchema: spec.SimpleSchema{
						Type: "number",
					},
					ParamProps: spec.ParamProps{
						In:          "formData",
						Name:        "fparam",
						Description: "a test form parameter",
					},
				},
				"hparam-tx-jfxM1": {
					CommonValidations: spec.CommonValidations{
						UniqueItems: true,
					},
					SimpleSchema: spec.SimpleSchema{
						Type: "integer",
					},
					ParamProps: spec.ParamProps{
						In:          "header",
						Name:        "hparam",
						Description: "a test head parameter",
					},
				},
				"path-z6Ciiujn": {
					CommonValidations: spec.CommonValidations{
						UniqueItems: true,
					},
					SimpleSchema: spec.SimpleSchema{
						Type: "string",
					},
					ParamProps: spec.ParamProps{
						In:          "path",
						Name:        "path",
						Description: "path to the resource",
						Required:    true,
					},
				},
				"pretty-nN7o5FEq": {
					CommonValidations: spec.CommonValidations{
						UniqueItems: true,
					},
					SimpleSchema: spec.SimpleSchema{
						Type: "string",
					},
					ParamProps: spec.ParamProps{
						In:          "query",
						Name:        "pretty",
						Description: "If 'true', then the output is pretty printed.",
					},
				},
			},
		},
	}
	swagger, err := BuildOpenAPISpec(container.RegisteredWebServices(), config)
	if !assert.NoError(err) {
		return
	}
	expected_json, err := expected.MarshalJSON()
	if !assert.NoError(err) {
		return
	}
	actual_json, err := swagger.MarshalJSON()
	if !assert.NoError(err) {
		return
	}
	if err := jsontesting.JsonCompare(expected_json, actual_json); err != nil {
		t.Error(err)
	}
}

func TestBuildOpenAPIDefinitionsForResource(t *testing.T) {
	config, _, assert := setUp(t, true)
	expected := &spec.Definitions{
		"builder.TestInput": getTestInputDefinition(),
	}
	swagger, err := BuildOpenAPIDefinitionsForResource(TestInput{}, config)
	if !assert.NoError(err) {
		return
	}
	expected_json, err := json.Marshal(expected)
	if !assert.NoError(err) {
		return
	}
	actual_json, err := json.Marshal(swagger)
	if !assert.NoError(err) {
		return
	}
	if err := jsontesting.JsonCompare(expected_json, actual_json); err != nil {
		t.Error(err)
	}
}

func TestBuildOpenAPIDefinitionsForResourceWithExtensionV2Schema(t *testing.T) {
	config, _, assert := setUp(t, true)
	expected := &spec.Definitions{
		"builder.TestExtensionV2Schema": spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"integer"},
			},
		},
	}
	swagger, err := BuildOpenAPIDefinitionsForResource(TestExtensionV2Schema{}, config)
	if !assert.NoError(err) {
		return
	}
	expected_json, err := json.Marshal(expected)
	if !assert.NoError(err) {
		return
	}
	actual_json, err := json.Marshal(swagger)
	if !assert.NoError(err) {
		return
	}
	assert.Equal(string(expected_json), string(actual_json))
}
