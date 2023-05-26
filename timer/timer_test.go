package timer

import (
	"fmt"
	"testing"
	"time"
)

func TestTimeOut(t *testing.T) {
	// fmt.Println("time out")

	Tm := MakeTimer(time.Millisecond * 100)
	Tm.SetTimeOut(time.Second, func() {
		fmt.Println("time", time.Now())
		Tm.SetTimeOut(time.Second, func() {
			fmt.Println("time1", time.Now())
			Tm.SetTimeOut(time.Second, func() {
				fmt.Println("time2", time.Now())
				Tm.SetTimeOut(time.Second, func() {
					fmt.Println("time3", time.Now())
				})
			})
		})
	})
	time.Sleep(time.Second * 5)
}
