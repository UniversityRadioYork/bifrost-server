package bifrost

import (
	"fmt"
	"reflect"
	"strconv"
)

// Resource is a structure containing the path, type, and value of a RES response.
type Resource struct {
	Path  string
	Type  string
	Value string
}

// ToResource converts an URL and item to a list of resources.
func ToResource(url string, item interface{}) []Resource {
	val := reflect.ValueOf(item)
	typ := reflect.TypeOf(item)

	switch val.Kind() {
	case reflect.Struct:
		return structToResource(url, val, typ)
	case reflect.Array, reflect.Slice:
		return sliceToResource(url, val, typ)
	default:
		return []Resource{{url, "entry", fmt.Sprint(item)}}
	}
}

func structToResource(url string, val reflect.Value, typ reflect.Type) []Resource {
	nf := val.NumField()
	af := nf

	// First, announce the incoming directory.
	// We'll fix the value later.
	res := []Resource{{url, "directory", "?"}}

	// Now, recursively work out the fields.
	for i := 0; i < nf; i++ {
		fieldt := typ.Field(i)

		// We can't announce fields that aren't exported.
		// If this one isn't, knock one off the available fields and ignore it.
		if fieldt.PkgPath != "" {
			af--
			continue
		}

		// Work out the resource name from the field name/tag.
		tag := fieldt.Tag.Get("res")
		if tag == "" {
			tag = fieldt.Name
		}

		// Now, recursively emit and collate each resource.
		fieldv := val.Field(i)
		res = append(res, ToResource(url+"/"+tag, fieldv.Interface())...)
	}

	// Now fill in the final available fields count
	res[0].Value = strconv.Itoa(af)

	return res
}

func sliceToResource(url string, val reflect.Value, typ reflect.Type) []Resource {
	len := val.Len()

	// As before, but now with a list and indexes.
	res := []Resource{{url, "list", strconv.Itoa(len)}}

	for i := 0; i < len; i++ {
		fieldv := val.Index(i)
		res = append(res, ToResource(url+"/"+strconv.Itoa(i), fieldv.Interface())...)
	}

	return res
}
