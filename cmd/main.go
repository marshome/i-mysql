package main

import (
	"flag"
	"fmt"
	"github.com/marshome/i-mysql/generator"
	"io/ioutil"
)

func main() {
	sql_file_flag := flag.String("sql_file", "", "sql file")
	orm_file_flag := flag.String("orm_file", "", "orm file")
	package_name_flag := flag.String("package_name", "", "package name")
	flag.Parse()

	sql_file := *sql_file_flag
	orm_file := *orm_file_flag
	package_name := *package_name_flag

	if sql_file == "" {
		fmt.Println("sql file null")
		return
	}

	if orm_file == "" {
		fmt.Println("orm file null")
		return
	}

	sqlData, err := ioutil.ReadFile(sql_file)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	gen := generator.NewGenerator()
	orm, err := gen.Gen(string(sqlData), package_name)
	if err != nil {
		fmt.Println(err)
	}

	err = ioutil.WriteFile(orm_file, []byte(orm), 0)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println("OK")
}
