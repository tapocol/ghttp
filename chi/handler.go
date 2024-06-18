package chi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"ghttp"

	"github.com/go-chi/chi/v5"
	"github.com/go-openapi/spec"
)

var (
	pathParamPattern = regexp.MustCompile("{([^}]+)}")
)

func HandlerFunc(r chi.Router) http.HandlerFunc {
	onceFn := sync.OnceValue(func() spec.Swagger {
		return initializeDoc(r)
	})
	return func(w http.ResponseWriter, req *http.Request) {
		doc := onceFn()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		enc := json.NewEncoder(w)
		if err := enc.Encode(doc); err != nil {
			fmt.Printf("Error encoding doc: %s\n", err.Error())
			return
		}
	}
}

func initializeDoc(r chi.Router) spec.Swagger {
	doc := spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger:     "2.0",
			Definitions: spec.Definitions{},
			Paths: &spec.Paths{
				Paths: map[string]spec.PathItem{},
			},
		},
	}
	chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if _, ok := doc.Paths.Paths[route]; !ok {
			doc.SwaggerProps.Paths.Paths[route] = spec.PathItem{}
		}
		operation := spec.NewOperation("")

		var pTyper ghttp.PayloadTyper
		pTyper, _ = handler.(ghttp.PayloadTyper)
		if pTyper != nil {
			pt := pTyper.PayloadType()
			addDefinition(doc, pt)
			parameter := spec.BodyParam(getName(pt), spec.RefProperty("#/definitions/"+getName(pt)))
			operation.AddParam(parameter)
		}

		var rTyper ghttp.ResponseTyper
		rTyper, _ = handler.(ghttp.ResponseTyper)
		if rTyper != nil {
			rt := rTyper.ResponseType()
			addDefinition(doc, rt)
			resp := spec.NewResponse()
			resp.Schema = spec.RefProperty("#/definitions/" + getName(rt))
			operation.RespondsWith(http.StatusOK, resp)
		}

		//var hAdder ghttp.HeaderAdder
		//hAdder, _ = handler.(ghttp.HeaderAdder)
		//if hAdder != nil {
		//	headers := hAdder.HeaderAdd()
		//	for _, header := range headers {
		//		parameter := spec.HeaderParam(header)
		//		operation.AddParam(parameter)
		//	}
		//}
		// TODO: Add Header through Middleware?

		pathParams := pathParamPattern.FindAllStringSubmatch(route, -1)
		for _, pathParam := range pathParams {
			parameter := spec.PathParam(pathParam[1])
			operation.AddParam(parameter)
		}

		pathItem := doc.SwaggerProps.Paths.Paths[route]
		switch method {
		case http.MethodGet:
			pathItem.PathItemProps.Get = operation
		case http.MethodPut:
			pathItem.PathItemProps.Put = operation
		case http.MethodPost:
			pathItem.PathItemProps.Post = operation
		case http.MethodDelete:
			pathItem.PathItemProps.Delete = operation
		case http.MethodOptions:
			pathItem.PathItemProps.Options = operation
		case http.MethodHead:
			pathItem.PathItemProps.Head = operation
		case http.MethodPatch:
			pathItem.PathItemProps.Patch = operation
		}
		doc.SwaggerProps.Paths.Paths[route] = pathItem
		return nil
	})
	return doc
}

func addDefinition(doc spec.Swagger, t reflect.Type) {
	if _, ok := doc.Definitions[getName(t)]; !ok {
		prop := getProperty(t)
		if prop != nil {
			doc.Definitions[getName(t)] = *prop
		}
	}
}

func getName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Pointer:
		return "*" + getName(t.Elem())
	default:
		return strings.ReplaceAll(t.Name(), "/", ".")
	}
}

func getProperty(t reflect.Type) *spec.Schema {
	switch t.String() {
	case "uuid.UUID":
		return spec.StrFmtProperty("uuid")
	case "date.DateString":
		return spec.DateProperty()
	}
	switch t.Kind() {
	//case reflect.Invalid:
	case reflect.Bool:
		return spec.BooleanProperty()
	case reflect.Int:
		return spec.Int64Property()
	case reflect.Int8:
		return spec.Int8Property()
	case reflect.Int16:
		return spec.Int16Property()
	case reflect.Int32:
		return spec.Int32Property()
	case reflect.Int64:
		return spec.Int64Property()
	case reflect.Uint:
		return spec.Int64Property()
	case reflect.Uint8:
		return spec.Int8Property()
	case reflect.Uint16:
		return spec.Int16Property()
	case reflect.Uint32:
		return spec.Int32Property()
	case reflect.Uint64:
		return spec.Int64Property()
	//case reflect.Uintptr:
	case reflect.Float32:
		return spec.Float32Property()
	case reflect.Float64:
		return spec.Float64Property()
	//case reflect.Complex64:
	//case reflect.Complex128:
	case reflect.Array:
		return spec.ArrayProperty(getProperty(t.Elem()))
	//case reflect.Chan:
	//case reflect.Func:
	//case reflect.Interface:
	//case reflect.Map:
	//	return spec.MapProperty()
	case reflect.Pointer:
		property := getProperty(t.Elem())
		property.Nullable = true
		return property
	case reflect.Slice:
		return spec.ArrayProperty(getProperty(t.Elem()))
	case reflect.String:
		return spec.StringProperty()
	case reflect.Struct:
		schema := spec.Schema{
			SchemaProps: spec.SchemaProps{
				Properties: spec.SchemaProperties{},
			},
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			property := getProperty(f.Type)
			if property != nil {
				schema.SchemaProps.Properties[strings.Split(f.Tag.Get("json"), ",")[0]] = *property
			}
		}
		return &schema
	//case reflect.UnsafePointer:
	default:
		log.Printf("Unknown kind for swagger property: %s %s\n", getName(t), t.Kind())
		return nil
	}
}
