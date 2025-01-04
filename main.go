package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"runtime/pprof"
	"strconv"
	"unicode"

	_ "github.com/joho/godotenv/autoload" // auto laod .env file
)

type Pieces struct {
	wB  int
	bB  int
	wN  int
	bN  int
	wP  int
	bP  int
	wQ  int
	bQ  int
	wR  int
	bR  int
	min int
	max int
}

// countPieces scans a FEN string and sets the number of pieces found
// returns the index of the last scanned byte
func countPieces(line *[]byte, pieces *Pieces) int {
	wB, bB, wN, bN, wP, bP, wQ, bQ, wR, bR := 0, 0, 0, 0, 0, 0, 0, 0, 0, 0

	fenStart := 8 // lines always start with {"fen": (7 chars)

	for i, b := range (*line)[fenStart:] {
		switch b {
		case 'B':
			wB += 1
		case 'b':
			bB += 1
		case 'N':
			wN += 1
		case 'n':
			bN += 1
		case 'P':
			wP += 1
		case 'p':
			bP += 1
		case 'Q':
			wQ += 1
		case 'q':
			bQ += 1
		case 'R':
			wR += 1
		case 'r':
			bR += 1
		case ' ':
			*pieces = Pieces{wB, bB, wN, bN, wP, bP, wQ, bQ, wR, bR, 0, 0}
			return fenStart + i
		}
	}
	// should never get here
	panic("did not find a space in the line, invalid input")
}

func main() {

	// cpu profiling
	f, err := os.Create("cpu.prof")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	// open input file
	filePath := os.Getenv("EVAL_FILE")
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	var pieces Pieces
	centiPawn := []byte("\"cp\"")
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Bytes()

		lastScanned := countPieces(&line, &pieces)
		line = line[lastScanned:]
		foundIndex := bytes.Index(line, centiPawn)
		minEval, maxEval := 99_999, -99_999
		for foundIndex != -1 {

			// shift slice 5 over
			// want to move past "cp": and start on the number
			line = line[foundIndex+5:]

			intEnd := 0
			negative := false

			if line[0] == '-' {
				// integer could start with a negative sign
				negative = true
				line = line[1:]
			}

			for j := range line {
				if !unicode.IsDigit(rune(line[j])) {
					intEnd = j
					break
				}
			}
			parsedInt, err := strconv.Atoi(string(line[:intEnd]))

			if err != nil {
				panic(fmt.Sprintf("int parse failed in line %d. parsing %s. remaining: %s\n", lineNum, string(line[:intEnd]), string(line)))
			}

			if negative {
				parsedInt *= -1
			}

			minEval = min(parsedInt, minEval)
			maxEval = max(parsedInt, maxEval)

			line = line[intEnd:]
			foundIndex = bytes.Index(line, centiPawn)
		}
		pieces.max = maxEval
		pieces.min = minEval

		lineNum += 1
		if lineNum%10_000_000 == 0 {
			fmt.Printf("%v %d\n", pieces, lineNum)
		}
	}

}
