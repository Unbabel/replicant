package tmpl

import (
	"bytes"
	"text/template"

	"github.com/Unbabel/replicant/transaction"
)

// Parse the script within the transaction with the defined transaction inputs
func Parse(config transaction.Config) (c transaction.Config, err error) {

	tpl, err := template.New(config.Name).Parse(config.Script)
	if err != nil {
		return config, err
	}

	var buf bytes.Buffer
	err = tpl.Execute(&buf, config.Inputs)
	if err != nil {
		return config, err
	}

	config.Script = buf.String()
	return config, nil

}
