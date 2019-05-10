package main

import "fmt"

func ExampleSrtParse() {
	fmt.Println(ParseSRT(`
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
	// {[{1 2.36s 5.36s [EINE NETFLIX ORIGINAL SERIE]} {2 5.44s 7.56s [[ernste Musik]]} {3 12s 15.08s [[Mann] Ich wusste immer, dass der Tag meiner Abrechnung kommt.]}]} <nil>
}
