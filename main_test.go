package main

import "fmt"

func ExampleLines() {
	fmt.Println(newlineRegexp.ReplaceAllString(`Die Spannungen in der Stadt sind
seit der Flüchtlingsdebatte extrem hoch.`, " $1"))
	// Output:
	// Die Spannungen in der Stadt sind seit der Flüchtlingsdebatte extrem hoch.
}
