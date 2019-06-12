package cloudflare

import (
	"encoding/json"
	"fmt"
	"time"
)

func ExampleDuration() {
	d := Duration{1 * time.Second}
	fmt.Println(d)

	buf, err := json.Marshal(d)
	fmt.Println(string(buf), err)

	err = json.Unmarshal([]byte(`"5s"`), &d)
	fmt.Println(d, err)

	d.Duration += time.Second
	fmt.Println(d, err)

	// Output:
	// 1s
	// "1s" <nil>
	// 5s <nil>
	// 6s <nil>
}
