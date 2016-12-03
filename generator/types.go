package generator

type Column struct {
	DbName        string
	GoName        string
	DbType        string
	GoType        string
	Size          string
	AutoIncrement bool
}

type Index struct {
	Name   string
	Column *Column
}

type UnionIndex struct {
	Name           string
	ColumnNameList []string
	ColumnList     []*Column
}

type Table struct {
	DbName               string
	GoName               string
	ColumnList           []*Column
	PrimaryColumn        *Column
	IndexList            []*Index
	UniqueIndexList      []*Index
	UnionIndexList       []*UnionIndex
	UniqueUnionIndexList []*UnionIndex
}

func newTable()(t *Table) {
	t = &Table{}
	t.ColumnList = make([]*Column, 0)

	return t
}

func (t *Table)AddColumn(c *Column) {
	t.ColumnList = append(t.ColumnList, c)
}