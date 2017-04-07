package generator

import (
	"strings"
	"errors"
)

func goName(name string)(goName string) {
	goName = ""
	is_ := true
	for i := 0; i < len(name); i++ {
		if is_ {
			if name[i] == '_' {
				is_ = true
			} else {
				is_ = false
				goName += strings.ToUpper(string(name[i]))
			}
		} else {
			if name[i] == '_' {
				is_ = true
			} else {
				is_ = false
				goName += string(name[i])
			}
		}
	}
	return goName
}

func goType(typ string)(goType string) {
	if typ == "bigint" {
		return "int64"
	} else if typ == "varchar" {
		return "string"
	} else if typ == "int" {
		return "int32"
	} else if typ == "datetime" {
		return "string"
	} else if typ == "timestamp" {
		return "string"
	} else if typ == "tinyint" {
		return "int32"
	} else if (typ == "longtext") {
		return "string"
	} else {
		return typ
	}
}

func (g *Generator)parseColumnLine(line string)(c *Column,err error) {
	c = &Column{}
	tokens := strings.Split(line, " ")
	c.DbName = strings.Trim(tokens[0], "`")
	c.GoName = goName(c.DbName)
	c.DbType = tokens[1]

	if strings.HasSuffix(c.DbType, ")") {
		typTokens := strings.Split(c.DbType, "(")
		c.DbType = typTokens[0]
		sizeString := strings.TrimRight(typTokens[1], ")")
		c.Size = sizeString
	}

	c.GoType = goType(c.DbType)
	c.AutoIncrement = strings.Contains(line, "AUTO_INCREMENT")

	return c, nil
}

func (g *Generator)parsePrimaryKey(line string)(primaryKeyName string, err error) {
	primaryKeyName = line[strings.Index(line, "`")+1:strings.LastIndex(line, "`")]
	return primaryKeyName, nil
}

func (g *Generator)parseTable(lines []string, i *int)(t *Table,err error) {
	t = newTable()

	l := strings.TrimSpace(lines[*i])
	tokens := strings.Split(l, "`")
	t.DbName = tokens[1]
	t.GoName = goName(t.DbName)

	for ; *i < len(lines); *i++ {
		l = strings.TrimSpace(lines[*i])
		if strings.HasPrefix(l, ")") {
			return t, nil
		}

		if strings.HasPrefix(l, "`") {
			c, err := g.parseColumnLine(l)
			if err != nil {
				return nil, err
			}
			t.AddColumn(c)
		} else if strings.HasPrefix(l, "PRIMARY KEY") {
			primaryKeyName, err := g.parsePrimaryKey(l)
			if err != nil {
				return nil, err
			}
			for _, v := range t.ColumnList {
				if v.DbName == primaryKeyName {
					t.PrimaryColumn = v
					break
				}
			}
			if t.PrimaryColumn == nil {
				return nil, errors.New("primary key not found")
			}
		} else if strings.HasPrefix(l, "UNIQUE KEY") {
			names := l[strings.Index(l, "(`") + 2:strings.LastIndex(l, "`)")]
			if strings.Contains(names, ",") {
				unionIndex := &UnionIndex{}
				names = strings.Replace(names, "`", "", -1)
				unionIndex.ColumnNameList = strings.Split(names, ",")
				for _, cName := range unionIndex.ColumnNameList {
					for _, v := range t.ColumnList {
						if v.DbName == cName {
							unionIndex.ColumnList = append(unionIndex.ColumnList, v)
							break
						}
					}
				}
				t.UniqueUnionIndexList = append(t.UniqueUnionIndexList, unionIndex)
			} else {
				index := &Index{}
				index.Name = names
				for _, v := range t.ColumnList {
					if v.DbName == index.Name {
						index.Column = v
						break
					}
				}
				t.UniqueIndexList = append(t.UniqueIndexList, index)
			}
		} else if strings.HasPrefix(l, "KEY") {
			names := l[strings.Index(l, "(`") + 2:strings.LastIndex(l, "`)")]
			if strings.Contains(names, ",") {
				unionIndex := &UnionIndex{}
				names = strings.Replace(names, "`", "", -1)
				unionIndex.ColumnNameList = strings.Split(names, ",")
				for _, cName := range unionIndex.ColumnNameList {
					for _, v := range t.ColumnList {
						if v.DbName == cName {
							unionIndex.ColumnList = append(unionIndex.ColumnList, v)
							break
						}
					}
				}
				t.UnionIndexList = append(t.UnionIndexList, unionIndex)
			} else {
				index := &Index{}
				index.Name = names
				for _, v := range t.ColumnList {
					if v.DbName == index.Name {
						index.Column = v
						break
					}
				}
				t.IndexList = append(t.IndexList, index)
			}
		} else {
			//nothing
		}
	}

	return nil, errors.New("table no end")
}

func (g *Generator)parse(sql string) error {
	lines := strings.Split(sql, "\n")
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "CREATE TABLE ") {
			table, err := g.parseTable(lines, &i)
			if err != nil {
				return err
			}
			g.TableList = append(g.TableList, table)
		}
	}
	return nil
}
