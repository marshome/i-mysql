package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"
)

type Generator struct {
	Namespace string
	TableList []*Table

	buf *bytes.Buffer
}

func NewGenerator() *Generator {
	g := &Generator{}
	g.TableList = make([]*Table, 0)
	g.buf = bytes.NewBufferString("")

	return g
}

func (g *Generator) P(format string, a ...interface{}) {
	g.buf.WriteString(fmt.Sprintf(format, a...))
}

func (g *Generator) Pn(format string, a ...interface{}) {
	g.buf.WriteString(fmt.Sprintf(format+"\n", a...))
}

func (g *Generator) genConstants(t *Table) {
	g.Pn("const %s_TABLE_NAME = \"%s\"", strings.ToUpper(t.DbName), t.DbName)
	g.Pn("")

	var filedNameList []string
	for _, v := range t.ColumnList {
		g.Pn("const %s_FIELD_%s = \"%s\"", strings.ToUpper(t.DbName), strings.ToUpper(v.DbName), v.DbName)
		filedNameList = append(filedNameList, fmt.Sprintf("\"%s\"", v.DbName))
	}
	g.Pn("")
	g.Pn("var %s_ALL_FIELDS = []string{\n%s,\n}", strings.ToUpper(t.DbName), strings.Join(filedNameList, ",\n"))
}

func (g *Generator) genEntity(t *Table) {
	g.Pn("type %s struct{", t.GoName)
	for _, v := range t.ColumnList {
		if v.Size != "" {
			g.Pn("    %s %s//size=%s", v.GoName, v.GoType, v.Size)
		} else {
			g.Pn("    %s %s", v.GoName, v.GoType)
		}
	}
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genPrepareInsertStmt(t *Table) {
	var fieldNames []string
	var fieldValues []string
	for _, v := range t.ColumnList {
		if v.AutoIncrement {
			continue
		}
		fieldNames = append(fieldNames, v.DbName)
		fieldValues = append(fieldValues, "?")
	}

	g.Pn("func (dao *%sDao) prepareInsertStmt() (err error){", t.GoName)
	g.Pn("    dao.insertStmt,err=dao.db.Prepare(\"INSERT INTO %s (%s) VALUES (%s)\")",
		t.DbName,
		strings.Join(fieldNames, ","),
		strings.Join(fieldValues, ","))
	g.Pn("    if err!=nil{")
	g.Pn("        return err")
	g.Pn("    }")
	g.Pn("    ")
	g.Pn("    return nil")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genPrepareUpdateStmt(t *Table) {
	var fieldNames []string
	for _, v := range t.ColumnList {
		if v == t.PrimaryColumn {
			continue
		}
		fieldNames = append(fieldNames, v.DbName+"=?")
	}

	g.Pn("func (dao *%sDao) prepareUpdateStmt() (err error){", t.GoName)
	g.Pn("    dao.updateStmt,err=dao.db.Prepare(\"UPDATE %s SET %s WHERE %s=?\")", t.DbName, strings.Join(fieldNames, ","), t.PrimaryColumn.DbName)
	g.Pn("    if err!=nil{")
	g.Pn("        return err")
	g.Pn("    }")
	g.Pn("    ")
	g.Pn("    return nil")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genPrepareDeleteStmt(t *Table) {
	g.Pn("func (dao *%sDao) prepareDeleteStmt() (err error){", t.GoName)
	g.Pn("    dao.deleteStmt,err=dao.db.Prepare(\"DELETE FROM %s WHERE %s=?\")", t.DbName, t.PrimaryColumn.DbName)
	g.Pn("    if err!=nil{")
	g.Pn("        return err")
	g.Pn("    }")
	g.Pn("    ")
	g.Pn("    return nil")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genPrepareSelectStmtBy(t *Table, columnList []*Column) {
	var fieldNames []string
	for _, v := range t.ColumnList {
		fieldNames = append(fieldNames, v.DbName)
	}

	var namesAdd []string
	var conditionList []string
	for _, v := range columnList {
		namesAdd = append(namesAdd, v.GoName)
		conditionList = append(conditionList, fmt.Sprintf("%s=?", v.DbName))
	}

	g.Pn("func (dao *%sDao) prepareSelectStmtBy%s() (err error){", t.GoName, strings.Join(namesAdd, "And"))
	g.Pn("    dao.selectStmtBy%s,err=dao.db.Prepare(\"SELECT %s FROM %s WHERE %s\")",
		strings.Join(namesAdd, "And"),
		strings.Join(fieldNames, ","),
		t.DbName,
		strings.Join(conditionList, " AND "))
	g.Pn("    if err!=nil{")
	g.Pn("        return err")
	g.Pn("    }")
	g.Pn("    ")
	g.Pn("    return nil")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genPrepareSelectStmtAll(t *Table) {
	var fieldNames []string
	for _, v := range t.ColumnList {
		fieldNames = append(fieldNames, v.DbName)
	}

	g.Pn("func (dao *%sDao) prepareSelectStmtAll() (err error){", t.GoName)
	g.Pn("    dao.selectStmtAll,err=dao.db.Prepare(\"SELECT %s FROM %s\")",
		strings.Join(fieldNames, ","),
		t.DbName)
	g.Pn("    if err!=nil{")
	g.Pn("        return err")
	g.Pn("    }")
	g.Pn("    ")
	g.Pn("    return nil")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genPrepareSelectStmt(t *Table) {
	g.genPrepareSelectStmtAll(t)

	if t.PrimaryColumn != nil {
		g.genPrepareSelectStmtBy(t, []*Column{t.PrimaryColumn})
	}

	if t.IndexList != nil {
		for _, index := range t.IndexList {
			g.genPrepareSelectStmtBy(t, []*Column{index.Column})
		}
	}

	if t.UniqueIndexList != nil {
		for _, uniqueIndex := range t.UniqueIndexList {
			g.genPrepareSelectStmtBy(t, []*Column{uniqueIndex.Column})
		}
	}

	//联合唯一索引
	if t.UniqueUnionIndexList != nil {
		for _, uniqueUnionIndex := range t.UniqueUnionIndexList {
			for i := 0; i < len(uniqueUnionIndex.ColumnList); i++ {
				g.genPrepareSelectStmtBy(t, uniqueUnionIndex.ColumnList[:i+1])
			}
		}
	}

	//联合索引
	if t.UnionIndexList != nil {
		for _, unionIndex := range t.UnionIndexList {
			for i := 0; i < len(unionIndex.ColumnList); i++ {
				g.genPrepareSelectStmtBy(t, unionIndex.ColumnList[:i+1])
			}
		}
	}
}

func (g *Generator) genDaoInitSelectBy(t *Table, columnList []*Column) {
	var namesAnd []string
	for _, c := range columnList {
		namesAnd = append(namesAnd, c.GoName)
	}

	g.genDaoInitPrepareFunc(t, fmt.Sprintf("prepareSelectStmtBy%s", strings.Join(namesAnd, "And")))
}

func (g *Generator) genDaoInitSelectAll(t *Table) {
	g.genDaoInitPrepareFunc(t, fmt.Sprintf("prepareSelectStmtAll"))
}

func (g *Generator) genDaoInitPrepareSelectStmt(t *Table) {
	g.genDaoInitSelectAll(t)

	if t.PrimaryColumn != nil {
		g.genDaoInitSelectBy(t, []*Column{t.PrimaryColumn})
	}

	if t.IndexList != nil {
		for _, index := range t.IndexList {
			g.genDaoInitSelectBy(t, []*Column{index.Column})
		}
	}

	if t.UniqueIndexList != nil {
		for _, uniqueIndex := range t.UniqueIndexList {
			g.genDaoInitSelectBy(t, []*Column{uniqueIndex.Column})
		}
	}

	if t.UniqueUnionIndexList != nil {
		for _, uniqueUnionIndex := range t.UniqueUnionIndexList {
			for i := 0; i < len(uniqueUnionIndex.ColumnList); i++ {
				g.genDaoInitSelectBy(t, uniqueUnionIndex.ColumnList[:i+1])
			}
		}
	}

	if t.UnionIndexList != nil {
		for _, unionIndex := range t.UnionIndexList {
			for i := 0; i < len(unionIndex.ColumnList); i++ {
				g.genDaoInitSelectBy(t, unionIndex.ColumnList[:i+1])
			}
		}
	}
}

func (g *Generator) genDaoInitPrepareFunc(t *Table, funcName string) {
	g.Pn("    err=dao.%s()", funcName)
	g.Pn("    if err!=nil{")
	g.Pn("        return err")
	g.Pn("    }")
	g.Pn("    ")
}

func (g *Generator) genDaoInit(t *Table) {
	g.Pn("func (dao *%sDao) Init() (err error){", t.GoName)

	g.genDaoInitPrepareFunc(t, "prepareInsertStmt")
	g.genDaoInitPrepareFunc(t, "prepareUpdateStmt")
	g.genDaoInitPrepareFunc(t, "prepareDeleteStmt")
	g.genDaoInitPrepareSelectStmt(t)

	g.Pn("   return nil")
	g.Pn("}")
}

func (g *Generator) genDaoDefSelectListByIndexList(t *Table, columnList []*Column) {
	var namesAnd []string
	for _, c := range columnList {
		namesAnd = append(namesAnd, c.GoName)
	}

	g.Pn("    selectStmtBy%s *sql.Stmt", strings.Join(namesAnd, "And"))
}

func (g *Generator) genDaoDefSelectStmt(t *Table) {
	g.Pn("    selectStmtAll *sql.Stmt")

	if t.PrimaryColumn != nil {
		g.Pn("    selectStmtBy%s *sql.Stmt", t.PrimaryColumn.GoName)
	}

	if t.IndexList != nil {
		for _, index := range t.IndexList {
			g.Pn("    selectStmtBy%s *sql.Stmt", index.Column.GoName)
		}
	}

	if t.UniqueIndexList != nil {
		for _, uniqueIndex := range t.UniqueIndexList {
			g.Pn("    selectStmtBy%s *sql.Stmt", uniqueIndex.Column.GoName)
		}
	}

	//联合唯一索引
	if t.UniqueUnionIndexList != nil {
		for _, uniqueUnionIndex := range t.UniqueUnionIndexList {
			for i := 0; i < len(uniqueUnionIndex.ColumnList); i++ {
				g.genDaoDefSelectListByIndexList(t, uniqueUnionIndex.ColumnList[:i+1])
			}
		}
	}

	//联合索引
	if t.UnionIndexList != nil {
		for _, unionIndex := range t.UnionIndexList {
			for i := 0; i < len(unionIndex.ColumnList); i++ {
				g.genDaoDefSelectListByIndexList(t, unionIndex.ColumnList[:i+1])
			}
		}
	}
}

func (g *Generator) genDaoDef(t *Table) {
	g.Pn("type %sDao struct{", t.GoName)
	g.Pn("    db *sql.DB")
	g.Pn("    insertStmt *sql.Stmt")
	g.genDaoDefSelectStmt(t)
	g.Pn("    updateStmt *sql.Stmt")
	g.Pn("    deleteStmt *sql.Stmt")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genDaoNew(t *Table) {
	g.Pn("func New%sDao(db *sql.DB)(t *%sDao){", t.GoName, t.GoName)
	g.Pn("    t=&%sDao{}", t.GoName)
	g.Pn("    t.db=db")
	g.Pn("    return t")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genInsert(t *Table) {
	var insertParams []string
	for _, v := range t.ColumnList {
		if v.AutoIncrement {
			continue
		}
		insertParams = append(insertParams, "e."+v.GoName)
	}

	g.Pn("func (dao *%sDao)Insert(tx *sql.Tx,e *%s)(id int64,err error){", t.GoName, t.GoName)
	g.Pn("    stmt:=dao.insertStmt")
	g.Pn("    if tx!=nil{")
	g.Pn("        stmt=tx.Stmt(stmt)")
	g.Pn("    }")
	g.Pn("")
	g.Pn("    result,err:=stmt.Exec(%s)", strings.Join(insertParams, ","))
	g.Pn("    if err!=nil{")
	g.Pn("        return 0,err")
	g.Pn("    }")
	g.Pn("")
	g.Pn("    id,err=result.LastInsertId()")
	g.Pn("    if err!=nil{")
	g.Pn("        return 0,err")
	g.Pn("    }")
	g.Pn("")
	g.Pn("    return id,nil")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genUpdate(t *Table) {
	var updateParams []string
	for _, v := range t.ColumnList {
		if t.PrimaryColumn == v {
			continue
		}
		updateParams = append(updateParams, "e."+v.GoName)
	}

	g.Pn("func (dao *%sDao)Update(tx *sql.Tx,e *%s)(int64,error){", t.GoName, t.GoName)
	g.Pn("    stmt:=dao.updateStmt")
	g.Pn("    if tx!=nil{")
	g.Pn("        stmt=tx.Stmt(stmt)")
	g.Pn("    }")
	g.Pn("")
	g.Pn("    result,err:=stmt.Exec(%s,e.%s)", strings.Join(updateParams, ","), t.PrimaryColumn.GoName)
	g.Pn("    if err!=nil{")
	g.Pn("        return 0,err")
	g.Pn("    }")
	g.Pn("")
	g.Pn("    return result.RowsAffected()")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genDelete(t *Table) {
	g.Pn("func (dao *%sDao)Delete(tx *sql.Tx,%s %s)(rowsAffected int64,err error){", t.GoName, t.PrimaryColumn.DbName, t.PrimaryColumn.GoType)
	g.Pn("    stmt:=dao.deleteStmt")
	g.Pn("    if tx!=nil{")
	g.Pn("        stmt=tx.Stmt(stmt)")
	g.Pn("    }")
	g.Pn("")
	g.Pn("    result,err:=stmt.Exec(%s)", t.PrimaryColumn.DbName)
	g.Pn("    if err!=nil{")
	g.Pn("        return 0,err")
	g.Pn("    }")
	g.Pn("")
	g.Pn("    return result.RowsAffected()")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genScanRow(t *Table) {
	var scanParams []string
	for _, v := range t.ColumnList {
		scanParams = append(scanParams, "&e."+v.GoName)
	}

	g.Pn("func (dao *%sDao)ScanRow(row *sql.Row)(*%s,error){", t.GoName, t.GoName)
	g.Pn("    e:=&%s{}", t.GoName)
	g.Pn("    err:=row.Scan(%s)", strings.Join(scanParams, ","))
	g.Pn("    if err!=nil{")
	g.Pn("        if err==sql.ErrNoRows{")
	g.Pn("            return nil,nil")
	g.Pn("        }else{")
	g.Pn("            return nil,err")
	g.Pn("        }")
	g.Pn("    }")
	g.Pn("")
	g.Pn("    return e,nil")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genScanRows(t *Table) {
	var scanParams []string
	for _, v := range t.ColumnList {
		scanParams = append(scanParams, "&e."+v.GoName)
	}

	g.Pn("func (dao *%sDao)ScanRows(rows *sql.Rows)(list []*%s,err error){", t.GoName, t.GoName)
	g.Pn("    list=make([]*%s,0)", t.GoName)
	g.Pn("    for rows.Next(){")
	g.Pn("        e:=%s{}", t.GoName)
	g.Pn("        err=rows.Scan(%s)", strings.Join(scanParams, ","))
	g.Pn("        if err!=nil{")
	g.Pn("            return nil,err")
	g.Pn("        }")
	g.Pn("        list=append(list,&e)")
	g.Pn("    }")
	g.Pn("    if rows.Err()!=nil{")
	g.Pn("        return nil,rows.Err()")
	g.Pn("    }")
	g.Pn("")
	g.Pn("    return list,nil")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genSelectBy(t *Table, columnList []*Column) {
	var paramList []string
	var goNamesList []string
	for _, v := range columnList {
		paramList = append(paramList, fmt.Sprintf("%s %s", v.GoName, v.GoType))
		goNamesList = append(goNamesList, v.GoName)
	}

	g.Pn("func (dao *%sDao)SelectBy%s(tx *sql.Tx,%s)(*%s,error){",
		t.GoName, strings.Join(goNamesList, "And"), strings.Join(paramList, ","), t.GoName)
	g.Pn("    stmt:=dao.selectStmtBy%s", strings.Join(goNamesList, "And"))
	g.Pn("    if tx!=nil{")
	g.Pn("        stmt=tx.Stmt(stmt)")
	g.Pn("    }")
	g.Pn("")
	g.Pn("    row:=stmt.QueryRow(%s)", strings.Join(goNamesList, ","))
	g.Pn("")
	g.Pn("    return dao.ScanRow(row)")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genSelectListBy(t *Table, columnList []*Column) {
	var paramList []string
	var goNamesList []string
	for _, v := range columnList {
		paramList = append(paramList, fmt.Sprintf("%s %s", v.GoName, v.GoType))
		goNamesList = append(goNamesList, v.GoName)
	}

	g.Pn("func (dao *%sDao)SelectListBy%s(tx *sql.Tx,%s)(list []*%s,err error){",
		t.GoName, strings.Join(goNamesList, "And"), strings.Join(paramList, ","), t.GoName)
	g.Pn("    stmt:=dao.selectStmtBy%s", strings.Join(goNamesList, "And"))
	g.Pn("    if tx!=nil{")
	g.Pn("        stmt=tx.Stmt(stmt)")
	g.Pn("    }")
	g.Pn("")
	g.Pn("    rows,err:=stmt.Query(%s)", strings.Join(goNamesList, ","))
	g.Pn("    if err!=nil{")
	g.Pn("        return nil,err")
	g.Pn("    }")
	g.Pn("")
	g.Pn("    return dao.ScanRows(rows)")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genSelectAll(t *Table) {
	g.Pn("func (dao *%sDao)SelectAll(tx *sql.Tx)(list []*%s,err error){", t.GoName, t.GoName)
	g.Pn("    stmt:=dao.selectStmtAll")
	g.Pn("    if tx!=nil{")
	g.Pn("        stmt=tx.Stmt(stmt)")
	g.Pn("    }")
	g.Pn("")
	g.Pn("    rows,err:=stmt.Query()")
	g.Pn("    if err!=nil{")
	g.Pn("        return nil,err")
	g.Pn("    }")
	g.Pn("")
	g.Pn("    return dao.ScanRows(rows)")
	g.Pn("}")
	g.Pn("")
}

func (g *Generator) genSelect(t *Table) {
	g.genSelectAll(t)

	if t.PrimaryColumn != nil {
		g.genSelectBy(t, []*Column{t.PrimaryColumn})
	}

	if t.IndexList != nil {
		for _, index := range t.IndexList {
			g.genSelectBy(t, []*Column{index.Column})
			g.genSelectListBy(t, []*Column{index.Column})
		}
	}

	if t.UniqueIndexList != nil {
		for _, uniqueIndex := range t.UniqueIndexList {
			g.genSelectBy(t, []*Column{uniqueIndex.Column})
		}
	}

	if t.UniqueUnionIndexList != nil {
		for _, uniqueUnionIndex := range t.UniqueUnionIndexList {
			g.genSelectBy(t, uniqueUnionIndex.ColumnList)

			//次级联合索引
			for i := 0; i < len(uniqueUnionIndex.ColumnList)-1; i++ {
				g.genSelectListBy(t, uniqueUnionIndex.ColumnList[:i+1])
			}
		}
	}

	if t.UnionIndexList != nil {
		for _, unionIndex := range t.UnionIndexList {
			for i := 0; i < len(unionIndex.ColumnList); i++ {
				g.genSelectListBy(t, unionIndex.ColumnList[:i+1])
			}
		}
	}
}

func (g *Generator) genDao(t *Table) {
	g.genConstants(t)

	g.genEntity(t)

	g.genDaoDef(t)
	g.genDaoNew(t)
	g.genDaoInit(t)

	g.genPrepareInsertStmt(t)
	g.genPrepareUpdateStmt(t)
	g.genPrepareDeleteStmt(t)
	g.genPrepareSelectStmt(t)

	g.genInsert(t)
	g.genUpdate(t)
	g.genDelete(t)

	g.genScanRow(t)
	g.genScanRows(t)
	g.genSelect(t)
}

func (g *Generator) gen() {
	g.Pn("package %s", g.Namespace)
	g.Pn("")
	g.Pn("import(")
	g.Pn("    \"database/sql\"")
	g.Pn(")")
	g.Pn("")
	for _, v := range g.TableList {
		g.genDao(v)
	}
}

func (g *Generator) Gen(sql string, namespace string) (orm string, err error) {
	g.Namespace = namespace
	err = g.parse(sql)
	if err != nil {
		return "", nil
	}

	g.gen()

	data, err := format.Source(g.buf.Bytes())
	if err != nil {
		fmt.Println(err.Error())
		return g.buf.String(), nil
	}

	return string(data), nil
}
