package main

import (
	"fmt"
)

func ExampleXml() {
	subs, _ := ReadXMLFile("subs.xml")
	sub := subs.Sub[3]
	fmt.Println(sub.From, sub.To, sub.Lines[0])
	// Output:
	// 29.96s 33s （猴急男女大街打野戰
}
