package main

import (
	"fmt"
	"reflect"

	"github.com/openai/openai-go"
)

func main() {
	var u openai.EmbeddingNewParamsInputUnion
	t := reflect.TypeOf(u)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fmt.Printf("Field: %s, Type: %s\n", field.Name, field.Type)
	}
}

