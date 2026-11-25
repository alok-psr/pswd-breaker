package main

// change the endrange to normal
// get the tui working in the table in the pterm library
import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
)

func changeWidth(width, num int) int {
	strVal := fmt.Sprintf("%0*d", width, num)
	// calc the int versin of the strVal and return
	intVal, err := strconv.Atoi(strVal)
	if err != nil {
		fmt.Printf("%e", err)
		os.Exit(1)
	}
	return intVal
}

func main() {
	PSWD := 99999999
	PswdLen := len(fmt.Sprintf("%d", PSWD))
	Partitions := 4
	EndRange := PSWD
	//EndRange := 99999999
	WorkersRange := int(EndRange / Partitions)

	result := make(chan int, 1)

	fmt.Printf("Target Password :: %d \nUsing %d Workers each having %d range\n", PSWD, Partitions, WorkersRange)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	defer cancel()

	start := 0
	end := 0
	for i := range Partitions {
		wg.Add(1)

		// start,end define the range for the go routine
		// ctx is the context
		start = end
		if i == Partitions-1 {
			end = EndRange + 1
		} else {
			end = end + WorkersRange
		}

		go func(ctx context.Context, startIndex, endIndex, i int) {
			defer wg.Done()
			// logic for brute forcing the pswd
			// if we have the pswd then we set the channel finding to false
			// if the finding is false then we cancel the context --out of the for loop

			fmt.Printf("workder %d spawned successfully\n", i)
			for tempPswd := startIndex; tempPswd < endIndex; tempPswd++ {
				select {
				case <-ctx.Done():
					return

				default:
					currPswd := changeWidth(PswdLen, tempPswd)
					// if we have cracked the Password then cancel else itterate further
					if currPswd == PSWD {
						fmt.Println("got the password")
						cancel()
						result <- PSWD
						return
					}
				}
			}
		}(ctx, start, end, i)

	}
	fmt.Println("waiting wg")
	wg.Wait()
	fmt.Println("waiting complete")
	crackedPswd := <-result
	fmt.Printf("got the password %d", crackedPswd)
}
