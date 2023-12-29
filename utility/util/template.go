package util

import (
	"bytes"
	"github.com/gogf/gf/v2/os/gfile"
	"html/template"
)

func RenderTemplate(data any) (string, error) {

	tmpl, err := template.New("tmpl").Parse(gfile.GetContents("./resource/template/email/verify_code.tmpl"))
	if err != nil {
		return "", err
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return "", err
	}

	return body.String(), nil
}
