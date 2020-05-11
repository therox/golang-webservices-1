package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
	"text/template"
)

// код писать тут

const codePrefix = "// apigen:api"

type genParameters struct {
	URL    string `json:"url"`
	Auth   bool   `json:"auth"`
	Method string `json:"method"`
}

type Handler struct {
	Name     string // Название хэндлера
	NeedAuth bool   // Нужна ли авторизация
}

// Тут мы храним связь метода с хэндлерами для этого метода
type MethodHandlers struct {
	Name     string    // Название метода (GET, POST)"
	Handlers []Handler // Список хэндлеров
}

// Тут мы храним связь ссылки с методами (верхняя структура для отсылки в шаблон ServeHttp)
type URLMethods struct {
	Url     string
	Methods []MethodHandlers
}

type Method struct {
	Name          string // Название метода
	ParameterName string
	ParameterType string
}

type ServiceData struct {
	Name    string
	Methods []Method
	Data    []URLMethods
}

var tmplData = `package main

import (
	"context"
	"encoding/json"
	"net/http"
)
{{range .}}{{$srvName := .Name}} 
func (srv *{{$srvName}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var h http.Handler

	switch r.URL.Path { {{range .Data}}
		case "{{.Url}}":
		switch r.Method { {{range .Methods}}	
		case "{{.Name}}": {{ range .Handlers }}
				h = http.HandlerFunc(srv.{{.Name}}){{ if .NeedAuth}}
				h = AuthorizeWrappers(h){{end}}{{end}}{{end}}
		default:
			w.WriteHeader(http.StatusNotAcceptable)
			rd := responseData{
				Error: "bad method",
			}
			w.Write(rd.Create())
			return
		}{{end}}
	default:
		w.WriteHeader(http.StatusNotFound)
		rd := responseData{
			Error: "unknown method",
		}
		w.Write(rd.Create())
		return
	}
	h.ServeHTTP(w, r)
}

{{range .Methods}}
func (srv *{{$srvName}}) {{.Name}}Handler(w http.ResponseWriter, r *http.Request) {

	rd := &responseData{}
	
	
	ctx := context.Background()

	// Подготавливаем структуру для параметров
	{{.ParameterName}} := {{.ParameterType }}{}
	// Парсим форму
	r.ParseForm()
	//Заполняем параметры
	err := PopulateStruct(&{{.ParameterName}}, r.Form)
	if err != nil {
		rd.Error = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		w.Write(rd.Create())
		return
	}
	err = validateStruct({{.ParameterName}})
	if err != nil {
		rd.Error = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		w.Write(rd.Create())
		return
	}
	// Нужно в конечном итоге запустить
	user, err := srv.{{.Name}}(ctx, {{.ParameterName}})
	if err != nil {
		if e, ok := err.(ApiError); ok {
			rd.Error = e.Err.Error()
			http.Error(w, string(rd.Create()), e.HTTPStatus)
			return
		}
		rd.Error = err.Error()
		http.Error(w, string(rd.Create()), http.StatusInternalServerError)
		return
	}
	bs, _ := json.Marshal(user)
	rd.Response = bs
	w.Write(rd.Create())
}
{{end}}{{end}}`

func main() {
	f, err := os.Create(os.Args[2])

	if err != nil {
		panic(err)
	}
	defer f.Close()
	sd := ParseFile(os.Args[1])
	fmt.Printf("%#v", sd)

	fmt.Println("Генерация файла...")
	tmpl, err := template.New("serveHTTP").Parse(tmplData)
	if err != nil {
		panic(err)
	}

	tmpl.Execute(f, sd)
	fmt.Println("Обработка завершена")

}

func ParseFile(f string) []ServiceData {
	sd := make([]ServiceData, 0)

	// Парсим файл, ищем функции
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, f, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range node.Decls {
		f, ok := f.(*ast.FuncDecl)
		if !ok {
			fmt.Printf("SKIP %T is not *ast.FuncDecl\n", f)
			continue
		}
		fmt.Printf("%#v\n", f)

		if f.Doc == nil {
			fmt.Printf("SKIP struct %#v doesnt have comments\n", f.Name.Name)
			continue
		}
		var gp genParameters
		needCodegen := false

		for _, comment := range f.Doc.List {
			fmt.Printf("Comment found: %s\n", comment.Text)
			if strings.HasPrefix(comment.Text, codePrefix) {
				needCodegen = true
				err = json.Unmarshal([]byte(strings.TrimPrefix(comment.Text, codePrefix)), &gp)
				if err != nil {
					// Не удалось раскрыть параметры, не обрабатываем
					needCodegen = false
					continue
				}
				continue
			}
		}
		if !needCodegen {
			fmt.Printf("SKIP func %#v doesnt have apigen mark\n", f.Name.Name)
			continue
		}
		// Выцепляем название функции
		fmt.Printf("Found func %s\n", f.Name)
		// Данные для генерации хэндлеров

		m := Method{
			Name: f.Name.String(),
		}
		fmt.Printf("-> Creating method info: %s\n", m.Name)
		// Данные для генерации ServeHTTP
		hh := Handler{
			Name:     fmt.Sprintf("%sHandler", m.Name),
			NeedAuth: gp.Auth,
		}
		fmt.Printf("-> Creating handler info: %s\n", hh.Name)

		mh := make([]MethodHandlers, 0)
		if gp.Method != "POST" {
			fmt.Printf("-> Adding handler for GET method: %s\n", hh.Name)
			mh = append(mh, MethodHandlers{
				Name:     "GET",
				Handlers: []Handler{hh},
			})
		}
		if gp.Method != "GET" {
			fmt.Printf("-> Adding handler for POST method: %s\n", hh.Name)
			mh = append(mh, MethodHandlers{
				Name:     "POST",
				Handlers: []Handler{hh},
			})
		}
		for i := range f.Type.Params.List {
			if f.Type.Params.List[i].Names[0].Name != "ctx" {
				fmt.Printf("-> Found Params %v, %v\n", f.Type.Params.List[i].Type, f.Type.Params.List[i].Names[0].Name)
				m.ParameterName = f.Type.Params.List[i].Names[0].Name
				m.ParameterType = fmt.Sprintf("%s", f.Type.Params.List[i].Type)
				fmt.Printf("-> Added parameters %s: %s to %s\n", m.ParameterName, m.ParameterType, m.Name)
			}
		}

		// Тута у нас список ресиверов, выбираем основной
		curRcvr := ""
		for _, recv := range f.Recv.List {
			//fmt.Printf("-> Generating code for struct %#v %T\n", recv.Type, recv.Type)
			rr, ok := recv.Type.(*ast.StarExpr)
			if ok {
				x, ok := rr.X.(*ast.Ident)
				if ok {
					fmt.Printf("-> Searching struct %s data in config\n", x.Name)
					curRcvr = x.Name
					isFound := false
					for _, item := range sd {
						if item.Name == curRcvr {
							fmt.Printf("-> Found existing struct %s in config\n", x.Name)
							isFound = true
							break
						}
					}
					if !isFound {
						fmt.Println("-> Adding new receiver", curRcvr)
						sd = append(sd, ServiceData{
							Name:    curRcvr,
							Methods: make([]Method, 0),
							Data:    make([]URLMethods, 0),
						})
					}
				}
			}
		}
		// Мы нашли название структуры, для метода которой есть коммент на генерацию кода
		for i := range sd {
			if sd[i].Name == curRcvr {
				fmt.Printf("-> Found existing rcvr %s in config\n", curRcvr)
				// Мы в нужном нам верхнем разделе. Сначала разбираемся с Методами
				sd[i].Methods = append(sd[i].Methods, m)
				// Теперь разбираемся с хэндлерами для данного метода и ServeHTTP
				isFound := false
				urlIndex := -1
				fmt.Printf("-> Searching url %s in config\n", gp.URL)
				for j := range sd[i].Data {
					if sd[i].Data[j].Url == gp.URL {
						isFound = true
						urlIndex = j
						fmt.Printf("-> Found existing url %s in config\n", gp.URL)
						break
					}
				}
				if !isFound {
					fmt.Printf("-> Adding new methods section for url %s to config\n", gp.URL)
					sd[i].Data = append(sd[i].Data, URLMethods{
						Url:     gp.URL,
						Methods: make([]MethodHandlers, 0),
					})
					urlIndex = len(sd[i].Data) - 1
				}
				// А теперь ищем нужные методы
				fmt.Printf("-> Searching handlers section for new handlers config\n")
				for k := range mh {
					isFound = false
					for j := range sd[i].Data[urlIndex].Methods {
						if mh[k].Name == sd[i].Data[urlIndex].Methods[j].Name {
							sd[i].Data[urlIndex].Methods[j].Handlers = append(sd[i].Data[urlIndex].Methods[j].Handlers,
								mh[k].Handlers...)
							isFound = true
							break
						}
					}
					if !isFound {
						fmt.Printf("-> not found methods section for %s. Adding new\n", mh[k].Name)
						sd[i].Data[urlIndex].Methods = append(sd[i].Data[urlIndex].Methods, mh[k])
					}
				}
			}
		}

		//fmt.Printf("Информация для обработки: %#v", sd)
	}
	return sd
}
