package main

import (
"encoding/json"
"os"
htmltemplate "html/template"
"fmt"
)

func main() {
content := "body {\n color: red; \n}"

funcMap := htmltemplate.FuncMap{
"json": func(v interface{}) string {
b, _ := json.Marshal(v)
return string(b)
},
"js": func(v interface{}) htmltemplate.JS {
b, _ := json.Marshal(v)
return htmltemplate.JS(b)
},
}

tmpl := htmltemplate.Must(htmltemplate.New("test").Funcs(funcMap).Parse("<script>var x = {{. | json}}; var y = {{. | js}};</script>"))
tmpl.Execute(os.Stdout, content)
fmt.Println()
}
