package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pterm/pterm"
)

type routineData struct {
	id      int
	status  atomic.Int32 // 0 = working ; 1 = completed ; 2 = waiting ; 3 = terminated
	current atomic.Int64 // current iteration
	domain  []int        // 2 values => 0 : start; 1 : end
}

func changeWidth(width, num int) int {
	strVal := fmt.Sprintf("%0*d", width, num)
	intVal, err := strconv.Atoi(strVal)
	if err != nil {
		fmt.Printf("%e", err)
		os.Exit(1)
	}
	return intVal
}

func getStatusLabel(code int32) string {
	switch code {
	case 0:
		return pterm.LightCyan("Working...")
	case 1:
		return pterm.LightGreen("CRACKED!")
	case 2:
		return pterm.Gray("Waiting")
	case 3:
		return pterm.LightRed("Terminated")
	default:
		return "Unknown"
	}
}

func renderTable(area *pterm.AreaPrinter, states []routineData, width int) {
	alternateStyle := pterm.NewStyle(pterm.BgDarkGray)

	table := pterm.TableData{
		[]string{"GoRoutine", "Range", "Current-Iteration", "Status"},
	}

	for i := range states {
		rangeStr := fmt.Sprintf("%0*d - %0*d", width, states[i].domain[0], width, states[i].domain[1]-1)
		currStr := fmt.Sprintf("%0*d", width, states[i].current.Load())

		row := []string{
			fmt.Sprintf("Worker-%d", states[i].id),
			rangeStr,
			currStr,
			getStatusLabel(states[i].status.Load()),
		}
		table = append(table, row)
	}

	// Srender() builds the table as a string so area.Update() can overwrite it in place
	tableStr, _ := pterm.DefaultTable.
		WithHasHeader().
		WithBoxed().
		WithData(table).
		WithAlternateRowStyle(alternateStyle).
		Srender()

	area.Update(tableStr)
}

func EndRageGen(digs int) int {
	strEnd := ""
	for range digs {
		strEnd += "9"
	}
	strVal, err := strconv.Atoi(strEnd)
	if err != nil {
		fmt.Println("error occoured for end rage gen ", err)
		os.Exit(1)
		return -1
	} else {
		return strVal
	}
}

func main() {
	pswdPtr := flag.Int("p", 999999, "Target password to crack")
	partitionPtr := flag.Int("w", 4, "Number of workers to allocate")

	flag.Parse()

	PSWD := *pswdPtr
	PswdLen := len(fmt.Sprintf("%d", PSWD))
	Partitions := *partitionPtr
	EndRange := EndRageGen(PswdLen)

	WorkersRange := int(EndRange / Partitions)
	result := make(chan int, 1)

	// Pre-allocate slice so memory addresses never shift while goroutines run
	routineDataSlice := make([]routineData, Partitions)

	fmt.Printf("Target Password :: %d \nUsing %d Workers each having %d range\n\n", PSWD, Partitions, WorkersRange)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	start := 0
	end := 0

	startTime := time.Now()
	for i := range Partitions {
		wg.Add(1)

		start = end
		if i == Partitions-1 {
			end = EndRange + 1
		} else {
			end = end + WorkersRange
		}

		// Initialize each worker's slot before launching its goroutine
		routineDataSlice[i].id = i
		routineDataSlice[i].status.Store(2) // 2 = waiting
		routineDataSlice[i].domain = []int{start, end}

		go func(ctx context.Context, startIndex, endIndex, i int) {
			defer wg.Done()

			routineDataSlice[i].status.Store(0) // 0 = working

			for tempPswd := startIndex; tempPswd < endIndex; tempPswd++ {
				select {
				case <-ctx.Done():
					routineDataSlice[i].status.Store(3) // 3 = terminated
					return

				default:
					// Update current iteration for the UI table
					routineDataSlice[i].current.Store(int64(tempPswd))

					currPswd := changeWidth(PswdLen, tempPswd)
					if currPswd == PSWD {
						routineDataSlice[i].status.Store(1) // 1 = completed/cracked
						cancel()
						result <- PSWD
						return
					}
				}
			}
		}(ctx, start, end, i)
	}

	// Start the live-updating area
	area, _ := pterm.DefaultArea.WithCenter().Start()

	// Run UI refresh loop in its own goroutine
	stopUI := make(chan struct{})
	uiDone := make(chan struct{})
	go func() {
		defer close(uiDone)
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopUI:
				renderTable(area, routineDataSlice, PswdLen) // Draw final frame
				return
			case <-ticker.C:
				renderTable(area, routineDataSlice, PswdLen)
			}
		}
	}()

	// Wait for all workers to finish
	wg.Wait()
	elapsed := time.Since(startTime)
	// Shut down UI cleanly before printing final output
	close(stopUI)
	<-uiDone
	area.Stop()

	crackedPswd := <-result
	fmt.Printf("\nGot the password: %d\nTime Taken to crack the password: %s", crackedPswd, elapsed)
}
