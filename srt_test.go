package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func ExampleSrtParse() {
	pp(ParseSRT(`
1
00:00:02,360 --> 00:00:05,360
EINE NETFLIX ORIGINAL SERIE

2
00:00:05,440 --> 00:00:07,560
[ernste Musik]

3
00:00:12,000 --> 00:00:15.080
[Mann] Ich wusste immer,
dass der Tag meiner Abrechnung kommt.
`))
	// Output:
	// {
	// 	"Sub": [
	// 		{
	// 			"Number": 1,
	// 			"From": 2360000000,
	// 			"To": 5360000000,
	// 			"Lines": [
	// 				"EINE NETFLIX ORIGINAL SERIE"
	// 			]
	// 		},
	// 		{
	// 			"Number": 2,
	// 			"From": 5440000000,
	// 			"To": 7560000000,
	// 			"Lines": [
	// 				"[ernste Musik]"
	// 			]
	// 		},
	// 		{
	// 			"Number": 3,
	// 			"From": 12000000000,
	// 			"To": 15080000000,
	// 			"Lines": [
	// 				"[Mann] Ich wusste immer,",
	// 				"dass der Tag meiner Abrechnung kommt."
	// 			]
	// 		}
	// 	]
	// }
}

func ExampleSrtParse2() {
	pp(ParseSRT(`
1
00:00:31.520 --> 00:00:32.640  position:50.00%,middle  align:middle size:80.00%  line:84.67% 
[sopla el viento]

2
00:00:32.720 --> 00:00:35.240  position:50.00%,middle  align:middle size:80.00%  line:84.67% 
[hojas que crujen]
`))
	// Output:
	// {
	// 	"Sub": [
	// 		{
	// 			"Number": 1,
	// 			"From": 31520000000,
	// 			"To": 32640000000,
	// 			"Lines": [
	// 				"[sopla el viento]"
	// 			]
	// 		},
	// 		{
	// 			"Number": 2,
	// 			"From": 32720000000,
	// 			"To": 35240000000,
	// 			"Lines": [
	// 				"[hojas que crujen]"
	// 			]
	// 		}
	// 	]
	// }
}

func TestParseTimespan(t *testing.T) {
	s := `00:00:31.520 --> 00:00:32.640  position:50.00%,middle  align:middle size:80.00%  line:84.67% 
[sopla el viento]`
	t0, t1, i, _ := parseTimespan(s, 0)
	fmt.Println("t0", t0, "t1", t1)
	if s[i:] != "[sopla el viento]" {
		t.Fatal("bad", s[i:])
	}
}

func pp(x interface{}, e error) {
	if e != nil {
		fmt.Println(e)
		return
	} 
	b, e := json.MarshalIndent(x, "", "\t")
	if e != nil {
		fmt.Println(e)
	}
	fmt.Println(string(b))
}
