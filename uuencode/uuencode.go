package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
)

func main() {
	cliArgs := os.Args
	if len(cliArgs) < 3 {
		fmt.Println("Format: uuencode <E|D> filename1 filename2 ..")
		fmt.Println("Use E for encoding and D for decoding")
		os.Exit(1)
	}

	encode := cliArgs[1] == "E"
	filenames := cliArgs[2:]

	for _, filename := range filenames {
		if file, err := os.Open(filename); err != nil {
			fmt.Println("Error opening file: %s", err.Error())
			os.Exit(1)
		} else {
			defer file.Close()

			if encode {
				encodedFilename := fmt.Sprintf("%s.uuencoded", filename)
				outfile, err := os.Create(encodedFilename)
				if err != nil {
					fmt.Println("Error creating output file: %s", err.Error())
					os.Exit(1)
				}

				encodedData := uuencode(file, filename)
				_, _ = outfile.Write(encodedData)
				_ = outfile.Close()
			} else {
				decodedData, _, filename := uudecode(file)
				outfile, err := os.Create(filename)
				if err != nil {
					fmt.Println("Error creating output file: %s", err.Error())
					os.Exit(1)
				}

				_, _ = outfile.Write(decodedData)
				_ = outfile.Close()
			}
		}
	}
}

func uuencode(input io.Reader, origFilename string) []byte {
	output := make([]byte, 0)
	output = append(output, fmt.Sprintf("begin 644 %s\n", origFilename)...)

	inputBlock := make([]byte, 45, 45)
	for {
		// Keep reading in 45 byte blocks, since our output needs to be 45 byte per line
		n, err := input.Read(inputBlock)
		if n == 0 && err != nil {
			if err != io.EOF {
				panic(err.Error())
			}
			break
		}

		if n < 45 {
			// Instead of counting how many 3-byte blocks we have available, just slice the input block, and the
			// 3-byte block reader will take care of the rest
			inputBlock = inputBlock[:n]
		}

		output = append(output, byte(n+32))

		// Read 3 block bytes from the 45 byte block we read from the input
		blockToEncode := make([]byte, 3)
		block3Reader := bytes.NewReader(inputBlock)
		for {
			n, err := block3Reader.Read(blockToEncode)
			if n == 0 && err != nil {
				break
			}

			// If we ever read less than 3 bytes add 0s to the end
			if n < 3 {
				for i := 2; i > n-1; i-- {
					blockToEncode[i] = 0
				}
			}

			output = append(output, encode3Block(blockToEncode)...)
		}

		output = append(output, '\n')
	}

	output = append(output, "`\nend\n"...)
	return output
}

func uudecode(input io.Reader) (fullData []byte, fileMode string, filename string) {
	scanner := bufio.NewScanner(input)

	// Check file header
	scanner.Scan()
	firstLine := scanner.Bytes()
	if !bytes.HasPrefix(firstLine, []byte("begin")) {
		panic("Invalid file header")
	}

	tmp := bytes.Split(firstLine, []byte(" "))
	fileMode = string(tmp[1])
	filename = string(tmp[2])

	fullData = make([]byte, 0)

	for scanner.Scan() {
		curLine := scanner.Bytes()
		if curLine[0] == '`' {
			// Get out of loop if this is the last line
			break
		}

		curLine = curLine[:] // Make a copy as we'll be modifying it below

		numBytesDecoded := int(curLine[0]) - 32
		decodedBytes := make([]byte, numBytesDecoded)

		// Since the encoder converts 3 byte block to 4 byte blocks, we read our input in 4 byte blocks
		curLine = curLine[1:]
		for len(curLine) > 0 {
			encoded4Bytes := curLine[:4]
			curLine = curLine[4:]

			decodedBytes = append(decodedBytes, decode4Block(encoded4Bytes)...)
		}

		fullData = append(fullData, decodedBytes...)
	}

	if err := scanner.Err(); err != nil {
		panic(err.Error())
	}
	return
}

/*
	decode4Block
	Decode the 4 byte encoded block back to the 3 decoded bytes
*/
func decode4Block(block []byte) []byte {
	// Easy access to encoded bytes
	var (
		b1 = block[0] - 32
		b2 = block[1] - 32
		b3 = block[2] - 32
		b4 = block[3] - 32
	)

	// Decoded bytes
	var d1, d2, d3 byte
	d1 = (b1 << 2) | ((b2 & 0x30) >> 4)
	d2 = (b2 << 4) | ((b3 >> 2) & 0x0F)
	d3 = (b3 << 6) | b4

	return []byte{d1, d2, d3}
}

/*
	encode3Block
	Returns the encoded version of the 3 byte block passed
*/
func encode3Block(block []byte) []byte {
	var p1, p2, p3, p4 byte
	p1 = block[0] >> 2
	p2 = ((block[0] << 6) >> 2) | ((block[1] & 0xF0) >> 4)
	p3 = ((block[1] & 0x0F) << 2) | (block[2] >> 6)
	p4 = block[2] & 0x3F

	var (
		out1 = p1 + 32
		out2 = p2 + 32
		out3 = p3 + 32
		out4 = p4 + 32
	)

	return []byte{out1, out2, out3, out4}
}
