package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

type Configuration struct {
	AwsKey       string
	AwsSecretKey string
	AwsBucket    string
	AwsRegion    string `config:"us-east-1"`
	MetaEndpoint string
	Address      string `config:"tcp4://:8080"`
}

var Config = &Configuration{}

const keyPrefix = "HARBOUR"

func init() {
	te := reflect.TypeOf(Config).Elem()
	ve := reflect.ValueOf(Config).Elem()

	for i := 0; i < te.NumField(); i++ {
		sf := te.Field(i)
		name := sf.Name
		field := ve.FieldByName(name)

		envVar := strings.ToUpper(fmt.Sprintf("%s_%s", keyPrefix, name))
		env := os.Getenv(envVar)
		tag := sf.Tag.Get("config")

		if env == "" && tag != "" {
			env = tag
		}

		field.SetString(env)
	}
}
