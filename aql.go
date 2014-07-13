package aranGO

import (
	"errors"
  "strconv"
)

// Aql query
type Query struct {
	// mandatory
	Aql string `json:"query,omitempty"`
	//Optional values
	Batch    int                    `json:"batchSize,omitempty"`
	Count    bool                   `json:"count,omitempty"`
	BindVars map[string]interface{} `json:"bindVars,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
  // opetions fullCount bool
  // Note that the fullCount sub-attribute will only be present in the result if the query has a LIMIT clause and the LIMIT clause is actually used in the query.

	// Control
	Validate bool   `json:"-"`
	ErrorMsg string `json:"errorMessage,omitempty"`
}

func NewQuery(query string) *Query {
  var q Query
  // alocate maps
  q.Options = make(map[string]interface{})
  q.BindVars= make(map[string]interface{})

	if query == "" {
		return &q
	} else {
		q.Aql = query
		return &q
	}
}

func (q *Query) Modify(query string) error {
	if query == "" {
		return errors.New("query must not be empty")
	} else {
		q.Aql = query
		return nil
	}
}

// Validate query before execution
func (q *Query) MustCheck() {
	q.Validate = true
	return
}

type AqlStructer interface{
  Generate()     string
}

// Basic structure
type AqlStruct struct {
  // main loop var
  main        string
  list        string
  lines       []AqlStructer
  vars        map[string]string
  // Return
  // could be string or AqlStruct
  //View        `json:"view"`
}

func (aq *AqlStruct) Generate() string{
  q:= "FOR "+aq.main+" IN "+aq.list

  for _,line :=range(aq.lines){
    q+= line.Generate()
  }

  return q
}

func (aq *AqlStruct) AddLoop(v string,list string) *AqlStruct {
  var l Loop
  if v != "" {
    l.Var = v
    l.List = list
  }
  aq.lines = append(aq.lines,l)
  return aq
}

func (aq *AqlStruct) SetView(v map[string]interface{}) *AqlStruct {
  var vie View
  vie = v
  aq.lines = append(aq.lines,vie)
  return aq
}

func (aq *AqlStruct) AddGroup(gs map[string]Var,into string) *AqlStruct{
  if gs != nil {
    var c Collects
    c.Collect = gs
    if into != ""{
      c.Gro = into
    }
    aq.lines = append(aq.lines,c)
  }
  return aq
}


func (aq *AqlStruct) SetList(obj string,list string) *AqlStruct{
  aq.main = obj
  aq.list = list
  return aq
}

func (aq *AqlStruct) AddFilter(key string,values []Pair) *AqlStruct{
  var fil Filters
  if key != "" && values != nil{
    fil.Key = key
    fil.Filter = values
    aq.lines = append(aq.lines,fil)
  }
  return aq
}

type View map[string]interface{}

func (v View) Generate() string{
  q:= `
RETURN { `
  i:=0
  for key,inte := range(v) {
    q += key+":"
    switch inte.(type) {
      case Var:
        q += inte.(Var).Obj+"."+inte.(Var).Name
      case string:
        q += "'"+inte.(string)+"'"
      case int:
        q += strconv.Itoa(inte.(int))
      case int32:
        q += strconv.FormatInt(inte.(int64),10)
      case int64:
        q += strconv.FormatInt(inte.(int64),10)
      case AqlStruct,*AqlStruct:
        q += "( "+inte.(*AqlStruct).Generate()+" )"
    }
    if len(v)-1 != i {
      q+=","
    }
    i++
  }
  q+= " }"
  return q
}

type Collects struct{
  // COLLECT key = Obj.Var,..., INTO Gro
  Collect map[string]Var `json:"collect"`
  Gro   string              `json:"group"`
}

type Group struct{
  Obj   string  `json:"obj"`
  Var   string  `json:"var"`
}

func (c Collects) Generate() string{
  if c.Collect == nil {
    return ""
  }
  q:= `
COLLECT `
  i:= 0
  for key,group := range(c.Collect){
    if i == len(c.Collect)-1 {
      q += key +"="+group.Obj+"."+group.Name
    }

    if i < len(c.Collect)-1 {
      q += key +"="+group.Obj+"."+group.Name+","
    }

    i++
  }
  if c.Gro != ""{
    q += " INTO "+c.Gro
  }
  return q
}

type Limits struct{
  Skip   int64 `json:"skip"`
  Limit  int64 `json:"limit"`
}

func (l Limits) Generate() string {
  skip := strconv.FormatInt(l.Skip,10)
  limit:= strconv.FormatInt(l.Limit,10)
  li := `
LIMIT `+skip+`,`+limit
  return li
}

func (aq *AqlStruct) AddLimit(skip,limit int64) *AqlStruct{
  var l Limits
  l.Skip = skip
  l.Limit = limit
  aq.lines = append(aq.lines,l)
  return aq
}

func (aq *AqlStruct) AddLet(v string,i interface{}) *AqlStruct{
  switch i.(type){
    case string:
    case *AqlStruct:
    default:
      return aq
  }
  var f Lets
  if v != ""{
    f.Var = v
    f.Comm = i
  }
  aq.lines = append(aq.lines,f)
  return aq
}

type Lets struct {
  Var     string         `json:"var"`
  Comm    interface{}    `json:"comm"`
}

func (l Lets) Generate() string {
  q := `
LET `+l.Var+` = (`
  switch l.Comm.(type) {
    case string:
        q += l.Comm.(string)
    case *AqlStruct:
        q += l.Comm.(*AqlStruct).Generate()
  }
  q += `)`
  return q
}

type Filters struct{
  Key    string  `json:"key"`
  Filter []Pair  `json:"filters"`
}

type Pair struct {
  Obj     string      `json:"obj"`
  Logic   string      `json:"op"`
  Value   interface{} `json:"val"`
}

func (f Filters) Generate() string{
  // check if filters available
  if len(f.Filter) == 0 {
    return ""
  }
  var oper      string

  lenmap := 0
  q := ""

  if f.Filter == nil{
    return ""
  }

  pairs := f.Filter
  key   := f.Key
  // iterate over filters
  // first
  q += `
FILTER ( `
  oper = "||"

  for i,pair := range(pairs){
    if i == len(pairs) -1 {
      oper = ""
    }
    switch pair.Value.(type) {
      case bool:
        q += key+"."+pair.Obj+" "+pair.Logic+" "+strconv.FormatBool(pair.Value.(bool))+" "+oper
      case int:
        q += key+"."+pair.Obj+" "+pair.Logic+" "+strconv.Itoa(pair.Value.(int))+" "+oper
      case int64:
        q += key+"."+pair.Obj+" "+pair.Logic+" "+strconv.FormatInt(pair.Value.(int64),10)+" "+oper
      case string:
        q += key+"."+pair.Obj+" "+pair.Logic+" '"+pair.Value.(string)+"' "+oper
      case float32,float64:
        q += key+"."+pair.Obj+" "+pair.Logic+" "+strconv.FormatFloat(pair.Value.(float64),'f',6,64)+" "+oper
      case Var:
        q += key+"."+pair.Obj+" "+pair.Logic+" "+pair.Value.(Var).Obj+"."+pair.Value.(Var).Name+" "+oper
    }
    if i == len(pairs)-1{
      q += ")"
    }
  }
  // next key
  lenmap++
  return q
}

type Loop struct {
  Var  string
  List string
}

func (l Loop) Generate() string{
  q := `
FOR `+l.Var+` IN `+l.List
  return q
}

// Variable into document
type Var  struct {
  Obj     string      `json:"obj"`
  Name    string      `json:"name"`
}
