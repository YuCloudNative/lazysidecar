package controllers

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"

	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestConstructEnvoyfilter(t *testing.T) {
	efVars := struct {
		ServiceName            string
		Name                   string
		Namespace              string
		Port                   int
		LazysidecarGateway     string
		LazysidecarGatewayPort string
	}{
		ServiceName:            "",
		Name:                   "demo",
		Namespace:              "",
		Port:                   0,
		LazysidecarGateway:     "",
		LazysidecarGatewayPort: "",
	}

	tpl, err := template.ParseFiles("../config/envoyfilter/workload_envoyfilter.tpl")
	if err != nil {
		panic(err)
	}

	var efBytes bytes.Buffer
	err = tpl.Execute(&efBytes, &efVars)
	if err != nil {
		fmt.Printf("go template execute failed.")
	}

	envoyfilter := &v1alpha3.EnvoyFilter{}
	err = yaml.Unmarshal(efBytes.Bytes(), envoyfilter)
	if err != nil {
		fmt.Printf("go template unmarshal failed.")
	}

	fmt.Printf("envoyfilter: %v", envoyfilter)
}
