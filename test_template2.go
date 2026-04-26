package main

import (
"encoding/json"
"os"
htmltemplate "html/template"
)

func main() {
content := "body { color: red; }"

funcMap := htmltemplate.FuncMap{
"json": func(v interface{}) string {
b, _ := json.Marshal(v)
return string(b)
},
}

tmpl := htmltemplate.Must(htmltemplate.New("test").Funcs(funcMap).Parse("<script>var x = {{. | json}};</script>"))
tmpl.Execute(os.Stdout, content)
}
