package main

import (
	"encoding/base64"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type asset struct {
	Name string
	Data []byte
}

type assetsInfo struct {
	PackageName string
	Assets      []*asset
}

var (
	flagInput   = flag.String("input", "", "input directory")
	flagOutput  = flag.String("output", "", "output file")
	flagPackage = flag.String("pkg", "assets", "package name to use")
	logger      = log.New(os.Stdout, "", 0)
	funcMap     = template.FuncMap{"base64": base64.StdEncoding.EncodeToString}
	tmpl        = template.Must(template.New("assets").Funcs(funcMap).Parse(
		`package {{ .PackageName }}

import (
	"encoding/base64"
)

func Get() map[string][]byte {
	var assets = make(map[string][]byte, {{ .Assets | len }})
{{ range .Assets }}
	assets["{{ .Name }}"], _ = base64.StdEncoding.DecodeString("{{ .Data | base64 }}")
{{ end }}
	return assets
}
`))
)

func main() {
	flag.Parse()

	if stringIsNullOrEmpty(flagInput) {
		logger.Fatalln("error: no input directory specified")
	}

	if stringIsNullOrEmpty(flagOutput) {
		logger.Fatalln("error: no output file specified")
	}

	_, err := os.Stat(*flagOutput)
	outputExists := err == nil
	if outputExists {
		logger.Printf("warning: output file already exists and will be overwritten\n")
	}

	assets := []*asset{}
	err = filepath.Walk(*flagInput, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		//skip directories and hidden files
		if !info.Mode().IsRegular() || strings.HasPrefix(filepath.Base(path), ".") {
			logger.Printf("skipping %s\n", path)
			return nil
		}

		logger.Printf("adding %s\n", path)

		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		assets = append(assets, &asset{Name: path, Data: bytes})
		return nil
	})

	if err != nil {
		logger.Fatalf("error while walking %s: %s", *flagInput, err.Error())
	}

	var file *os.File
	if outputExists {
		file, err = os.OpenFile(*flagOutput, os.O_WRONLY, 0)
	} else {
		file, err = os.Create(*flagOutput)
	}
	defer file.Close()

	if err != nil {
		logger.Fatalf("error opening output %s: %s", *flagOutput, err.Error())
	}

	err = tmpl.Execute(file, assetsInfo{PackageName: *flagPackage, Assets: assets})
	if err != nil {
		logger.Fatalf("error executing template: %s", err.Error())
	}
}
