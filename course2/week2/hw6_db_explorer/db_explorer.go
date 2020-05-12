package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

type colType int

const (
	cInt colType = iota
	cVarchar
	cText
)

type dbExplorer struct {
	db       *sql.DB
	Tables   map[string][]Column
	handlers []Handler
}

type Column struct {
	Name       string
	Type       colType
	IsPrimary  bool
	IsNullable bool
}

//type Table struct {
//	Name    string
//	Columns []Column
//}

func NewDbExplorer(d *sql.DB) (*dbExplorer, error) {

	db := &dbExplorer{
		db:     d,
		Tables: make(map[string][]Column),
	}
	// Получаем список таблиц
	rows, err := db.db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}

	var tn string
	for rows.Next() {
		err = rows.Scan(&tn)
		if err != nil {
			return nil, err
		}
		db.Tables[tn] = make([]Column, 0)
	}
	rows.Close()

	var colInfo struct {
		Field string
		Type  string
		Null  string
		Key   string
	}
	// Вытаскиваем структуру
	for k := range db.Tables {
		rows, err = db.db.Query("SHOW FULL COLUMNS FROM " + k)
		if err != nil {
			return nil, err
		}

		var tmpParam interface{}
		cols := make([]Column, 0)
		for rows.Next() {
			err = rows.Scan(&colInfo.Field,
				&colInfo.Type,
				&tmpParam,
				&colInfo.Null,
				&colInfo.Key,
				&tmpParam,
				&tmpParam,
				&tmpParam,
				&tmpParam,
			)
			if err != nil {
				return nil, err
			}
			col := Column{
				Name:       colInfo.Field,
				IsPrimary:  colInfo.Key == "PRI",
				IsNullable: colInfo.Null == "YES",
			}
			if strings.HasPrefix(colInfo.Type, "varchar") {
				col.Type = cVarchar
			}
			if strings.HasPrefix(colInfo.Type, "int") {
				col.Type = cInt
			}
			if strings.HasPrefix(colInfo.Type, "text") {
				col.Type = cText
			}
			cols = append(cols, col)
		}
		db.Tables[k] = cols
		rows.Close()
	}
	// Добавляем роутинг и хендлеры
	// Сюда впендюриваем наш роутинг

	db.AddHandler("NewRecord", "/", http.MethodPut, http.HandlerFunc(db.NewRecord))          // Вставка новой записи
	db.AddHandler("UpdateRecord", "/", http.MethodPost, http.HandlerFunc(db.UpdateRecord))   // Обновление существующей записи
	db.AddHandler("DeleteRecord", "/", http.MethodDelete, http.HandlerFunc(db.DeleteRecord)) // Удаление существующей записи
	db.AddHandler("List", "/", http.MethodGet, http.HandlerFunc(db.List))                    // Этот должен идти последним, т.к. охватывает слишком много урлов

	return db, nil
}

type Handler struct {
	Name   string
	Method string
	Route  string
	http.Handler
}

func (d *dbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	isFound := false
	for i := range d.handlers {
		if d.handlers[i].Method == r.Method {
			if strings.HasPrefix(r.URL.Path, d.handlers[i].Route) {
				d.handlers[i].ServeHTTP(w, r)
				isFound = true
				break
			}
		}
	}
	if !isFound {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))
	}
	return
}

func (d *dbExplorer) AddHandler(name string, route string, method string, f http.HandlerFunc) {
	d.handlers = append(d.handlers, Handler{
		Name:    name,
		Method:  method,
		Route:   route,
		Handler: f,
	})
}

type Response struct {
	Error    string                 `json:"error,omitempty"`
	Response map[string]interface{} `json:"response,omitempty"`
}

func (d *dbExplorer) DeleteRecord(w http.ResponseWriter, r *http.Request) {
	urlParams := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(urlParams) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(withResponse("", nil, "incorrect parameters"))
		return
	}
	tableName := urlParams[0]
	id, err := strconv.Atoi(urlParams[1])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(withResponse("", nil, err.Error()))
		return
	}

	// Вытаскиваем существующую запись
	//_, err = Get(d, tableName, id)
	//if err != nil {
	//	w.WriteHeader(http.StatusInternalServerError)
	//	w.Write(withResponse("", nil, "record not found"))
	//	return
	//}
	query := fmt.Sprintf("DELETE FROM %s WHERE id=?", tableName)
	res, err := d.db.Exec(query, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(withResponse("", nil, err.Error()))
		return
	}
	ra, _ := res.RowsAffected()
	w.Write(withResponse("deleted", ra, ""))
	return
}

func (d *dbExplorer) UpdateRecord(w http.ResponseWriter, r *http.Request) {
	urlParams := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(urlParams) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(withResponse("", nil, "incorrect parameters"))
		return
	}
	tableName := urlParams[0]
	id, err := strconv.Atoi(urlParams[1])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(withResponse("", nil, err.Error()))
		return
	}

	// Вытаскиваем существующую запись
	m, err := Get(d, tableName, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(withResponse("", nil, err.Error()))
		return
	}

	if cols, ok := d.Tables[tableName]; ok {
		// Такая таблица есть, вытаскиваем данные из тела
		bs, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(withResponse("", nil, err.Error()))
			return
		}
		defer r.Body.Close()
		var mUpd map[string]interface{}
		err = json.Unmarshal(bs, &mUpd)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(withResponse("", nil, err.Error()))
			return
		}

		if err := validate(cols, mUpd); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(withResponse("", nil, err.Error()))
			return
		}

		// Заполняем структуру новыми данными
		for j := range cols {
			if newVal, ok := mUpd[cols[j].Name]; ok {
				if !cols[j].IsPrimary {
					m[cols[j].Name] = newVal
				} else {
					w.WriteHeader(http.StatusBadRequest)
					w.Write(withResponse("", nil, fmt.Sprintf("field %s have invalid type", cols[j].Name)))
					return
				}
			}
		}

		// Пытаемся вставить
		var params []string
		var dd []interface{}
		primCol := "id"
		for i := range cols {
			if !cols[i].IsPrimary {
				params = append(params, fmt.Sprintf("%s=?", cols[i].Name))
				dd = append(dd, m[cols[i].Name])
			} else {
				primCol = cols[i].Name
			}
		}
		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s=%d",
			tableName,
			strings.Join(params, ", "),
			primCol,
			id,
		)
		res, err := d.db.Exec(query, dd...)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(withResponse("", nil, err.Error()))
			return
		}
		ra, _ := res.RowsAffected()
		w.Write(withResponse("updated", ra, ""))
		return
	}
	w.Write(withResponse("", nil, "not found"))
	w.WriteHeader(http.StatusNotFound)
}

func (d *dbExplorer) NewRecord(w http.ResponseWriter, r *http.Request) {
	tableName := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")[0]
	if cols, ok := d.Tables[tableName]; ok {
		// Такая таблица есть, вытаскиваем данные из тела
		bs, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(withResponse("", nil, err.Error()))
			return
		}
		defer r.Body.Close()
		m, err := populateMap(cols, bs)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(withResponse("", nil, err.Error()))
			return
		}
		// Пытаемся вставить
		var params []string
		var ph []string
		var dd []interface{}
		var prim = "id"
		for i := range cols {
			if !cols[i].IsPrimary {
				params = append(params, cols[i].Name)
				ph = append(ph, "?")
				dd = append(dd, m[cols[i].Name])
			} else {
				prim = cols[i].Name
			}
		}
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			tableName,
			strings.Join(params, ", "),
			strings.Join(ph, ", "),
		)
		res, err := d.db.Exec(query, dd...)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(withResponse("", nil, err.Error()))
			return
		}
		id, _ := res.LastInsertId()
		w.Write(withResponse(prim, id, ""))
		return
	}
	w.Write(withResponse("", nil, "not found"))
	w.WriteHeader(http.StatusNotFound)
}

func populateMap(cols []Column, data []byte) (map[string]interface{}, error) {
	var dataMap map[string]interface{}
	err := json.Unmarshal(data, &dataMap)
	if err != nil {
		return nil, err
	}
	resMap := make(map[string]interface{})
	for i := range cols {
		patchNil := false
		if !cols[i].IsNullable && dataMap[cols[i].Name] == nil {
			patchNil = true
		}
		switch cols[i].Type {
		case cVarchar, cText:
			if patchNil {
				resMap[cols[i].Name] = ""
			} else {
				resMap[cols[i].Name] = dataMap[cols[i].Name]
			}
		case cInt:
			if patchNil {
				resMap[cols[i].Name] = ""
			} else {
				resMap[cols[i].Name] = int(dataMap[cols[i].Name].(float64))
			}
		}
	}
	return resMap, nil
}

func validate(cols []Column, data map[string]interface{}) error {
	var val interface{}
	var ok bool
	for i := range cols {
		if val, ok = data[cols[i].Name]; !ok {
			continue
		}
		if val == nil {
			if !cols[i].IsNullable {
				return fmt.Errorf("field %s have invalid type", cols[i].Name)
			} else {
				continue
			}
		}

		if ok {
			switch cols[i].Type {
			case cText, cVarchar:
				if reflect.ValueOf(data[cols[i].Name]).Kind() != reflect.String {
					return fmt.Errorf("field %s have invalid type", cols[i].Name)
				}
			case cInt:
				if reflect.ValueOf(data[cols[i].Name]).Kind() != reflect.Float64 {
					return fmt.Errorf("field %s have invalid type", cols[i].Name)
				}
			}
		}
	}
	return nil
}

func (d *dbExplorer) List(w http.ResponseWriter, r *http.Request) {
	// Удаляем префикс и разбираем путь
	p := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")

	var resData interface{}
	var responseKey = ""
	switch len(p) {
	case 1:
		if p[0] == "" {
			// Возвращаем список всех таблиц
			tbls := make([]string, 0)
			for k := range d.Tables {
				tbls = append(tbls, k)
			}
			sort.Slice(tbls, func(i, j int) bool {
				return tbls[i] < tbls[j]
			})
			responseKey = "tables"
			resData = tbls
		} else {
			// Парсим строку
			limitStr := r.FormValue("limit")
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				limit = 5
			}
			offsetStr := r.FormValue("offset")
			offset, err := strconv.Atoi(offsetStr)
			if err != nil {
				offset = 0
			}

			// Ищем таблицу в списке
			isFound := false
			for k := range d.Tables {
				if k == p[0] {
					isFound = true
					break
				}
			}
			if !isFound {
				w.WriteHeader(http.StatusNotFound)
				w.Write(withResponse("", nil, "unknown table"))
				return
			}
			data, err := Select(d, p[0], limit, offset)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(withResponse("", nil, err.Error()))
				return
			}
			responseKey = "records"
			resData = data
		}
	case 2:
		table := p[0]
		id, err := strconv.Atoi(p[1])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(withResponse("", nil, err.Error()))
			return
		}
		data, err := Get(d, table, id)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write(withResponse("", nil, err.Error()))
			return
		}
		responseKey = "record"
		resData = data
	default:
		w.Write([]byte("неверный запрос"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Write(withResponse(responseKey, resData, ""))

}

func Select(d *dbExplorer, table string, limit, offset int) ([]map[string]interface{}, error) {
	var result = make([]map[string]interface{}, 0)
	query := fmt.Sprintf("SELECT * FROM %s ", table)
	if limit >= 0 {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}
	if limit >= 0 {
		query = fmt.Sprintf("%s OFFSET %d", query, offset)
	}

	rows, err := d.db.Query(query)
	if err != nil {
		return result, err
	}
	//Составляем адскую переменную
	tmpSlice := make([]interface{}, len(d.Tables[table]))
	for i := 0; i < len(d.Tables[table]); i++ {
		var ii interface{}
		tmpSlice[i] = &ii
	}
	index := 0
	for rows.Next() {
		index++
		rows.Scan(tmpSlice...)
		m := make(map[string]interface{})
		for i := range tmpSlice {
			var rawValue = *(tmpSlice[i].(*interface{}))
			if rawValue == nil {
				if d.Tables[table][i].IsNullable {
					m[d.Tables[table][i].Name] = nil
				} else {
					return result, fmt.Errorf("found null in non-nullable field")
				}
			} else {
				value := fmt.Sprintf("%s", rawValue)
				switch d.Tables[table][i].Type {
				case cVarchar, cText:
					m[d.Tables[table][i].Name] = value
				case cInt:
					vv, err := strconv.Atoi(value)
					if err != nil {
						return result, fmt.Errorf("value must be number, not string: %s", value)
					}
					m[d.Tables[table][i].Name] = vv
				}
			}
		}
		result = append(result, m)
	}
	return result, nil
}

func withResponse(responseKey string, data interface{}, error string) []byte {
	r := Response{
		Error: error,
	}
	if data != nil && responseKey != "" {
		r.Response = map[string]interface{}{responseKey: data}
	}
	bs, _ := json.Marshal(r)
	return bs
}

func Get(d *dbExplorer, table string, id int) (map[string]interface{}, error) {
	res, err := Select(d, table, -1, -1)
	if err != nil {
		return nil, err
	}
	for i := range res {
		for j := range d.Tables[table] {
			if d.Tables[table][j].IsPrimary {
				if fmt.Sprintf("%d", id) == fmt.Sprintf("%v", res[i][d.Tables[table][j].Name]) {
					return res[i], nil
				}
			}
		}
	}
	return nil, fmt.Errorf("record not found")
}
