/*
floppy_disk_image_file_to_serial_install
Copyright (C) 2024 github user bassjack1 <147515670+bassjack1@users.noreply.github.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

Communication with the author can be done via tagging @bassjack1 in github.com issues or by composing
private messages to user bassjack1 on reddit.com : https://www.reddit.com/message/compose/
*/

/*
floppy_disk_image_file_to_serial_install reads a disk image file and produces output
corresponding to a series of commands to the apple ][ system monitor which would result
in the writing of a single track of a floppy disk compatible with the Apple Disk II
floppy drive. The disk must have previously been formatted using the ProDOS disk formatting
utility. This format comprises 35 tracks of 16 sectors per track. Sectors contain 256 bytes.

Note : the output serial commands rely on the availabilty of the RWTS routine in memory. So the
apple ][ computer must have been booted into DOS before receiving the transmission of the series
of commands. The code has been seen to function correctly with the DOS3.3 RWTS routine, and
follows the example subroutine published in the Apple II "The DOS Manual" Copyright (c) 1980, 1981
by APPLE COMPUTER, INC. (pages 94-98)

The word Apple and The Apple Logo are registered trademarks of APPLE COMPUTER INC.

Usage: 
	floppy_disk_image_file_to_serial_install diskImageFilepath trackNum

diskImageFilepath must refer to a file in ProDOS sector order format (such as *.PO files)
trackNum must be an integer in the range [0,34]
*/
package main

import "bufio"
import "errors"
import "fmt"
import "io"
import "os"
import "strconv"
import "strings"

// readDiskImageFromFile fills the diskImage slice with data read directly from file diskImageFilePath.
// It also reports the count of read bytes to stderr.
func readDiskImageFromFile(diskImage *[]byte, diskImageFilepath string) {
	var f *os.File
	var err error
	f, err = os.Open(diskImageFilepath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var bufr *bufio.Reader = bufio.NewReader(f)
	for f != nil {
		var b byte
		b, err = bufr.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				f = nil
			} else {
				panic(err)
			}
		} else {
			*diskImage = append(*diskImage, b)
		}
	}
	fmt.Fprintf(os.Stderr, "read %d bytes from file %s\n", len(*diskImage), diskImageFilepath)
}

// diskImageStartPosOfTrackSector returns an integer offset corresponding to the start of a
// specified track/sector in a raw disk image.  trackNum must be in [0,34], sectorNum must be in [0,15].
func diskImageStartPosOfTrackSector(trackNum int, sectorNum int) int {
	//     0x000TTSBB              0x000TTSBB
	return 0x00001000 * trackNum + 0x00000100 * sectorNum
}

// Sector suffling section begin

// readSectorDataToBuffer fills the sectorBuffer slice with one sector of data
// from diskImage starting at the offset for track,sector.
func readSectorDataToBuffer(sectorBuffer *[0x0100]byte, diskImage []byte, track int, sector int) {
	var sourceBytesPos int = diskImageStartPosOfTrackSector(track, sector)
	var destinationPos int = 0
	for destinationPos < 0x0100 {
		sectorBuffer[destinationPos] = diskImage[sourceBytesPos]
		destinationPos = destinationPos + 1
		sourceBytesPos = sourceBytesPos + 1
	}
}

// writeSectorDataFromBuffer overwrites one sector of data in diskImage starting at the
// offset for track,sector with the data stored in the sectorBuffer.
func writeSectorDataFromBuffer(sectorBuffer *[0x0100]byte, diskImage []byte, track int, sector int) {
	var destinationBytesPos int = diskImageStartPosOfTrackSector(track, sector)
	var sourcePos int = 0
	for sourcePos < 0x0100 {
		diskImage[destinationBytesPos] = sectorBuffer[sourcePos]
		sourcePos = sourcePos + 1
		destinationBytesPos = destinationBytesPos + 1
	}
}

// writeSectorDataFromBuffer overwrites one sector of data in diskImage starting at the
// offset for track,destinationSector with the data from the diskImage starting at the
// offset for track,sourceSector.
func copySectorDataInImage(diskImage []byte, track int, sourceSector int, destinationSector int) {
	var sourceBytesPos int = diskImageStartPosOfTrackSector(track, sourceSector)
	var destinationBytesPos int = diskImageStartPosOfTrackSector(track, destinationSector)
	var bytesCopied = 0
	for bytesCopied < 0x0100 {
		diskImage[destinationBytesPos] = diskImage[sourceBytesPos]
		destinationBytesPos = destinationBytesPos + 1
		sourceBytesPos = sourceBytesPos + 1
		bytesCopied = bytesCopied + 1
	}
}

// convertDiskImageFromProdosOrderToDos33Order reorders the content of the passed in DiskImage by
// rearranging the sectors of each track into a new order. Exactly how this worked is still somehwat
// unclear. Several attempts at reordering were made before this one was found to be successful.
// Some of the online references which were helpful towards understanding the issue were:
// https://stason.org/TULARC/pc/apple2/faq/10-006-What-are-DSK-PO-DO-HDV-NIB-and-2MG-disk-image.html
// https://retrocomputing.stackexchange.com/questions/85/whats-the-difference-between-dos-ordered-and-prodos-ordered-disk-images
// https://nerdlypleasures.blogspot.com/2021/02/the-woz-format-accurate-preservation-of.html
// https://comp.sys.apple2.narkive.com/JY05JygH/reference-for-layout-of-prodos-and-dos-3-3-sector-ordering
// https://retrocomputing.stackexchange.com/questions/15056/converting-apple-ii-prodos-blocks-to-dos-tracks-and-sectors
// The last article has a comment by user "fadden" pointing to ciderpress code here:
// https://github.com/fadden/ciderpress/blob/master/diskimg/DiskImg.cpp
// This code was the most informative, although the issue is still confusing.
// Some of the documentation claims that prodos physical sector arrangment on the
// disk is non-sequential. However a nibble stream editor for tracks showed that
// a ProDOS formatted or a DOS3.3 formatted disk had the same physical sector ordering:
// 0x00,0x01,0x02,0x03,0x04,0x05,0x06,0x07,0x08,0x09,0x0A,0x0B,0x0C,0x0D,0x0E,0x0F
// But the translation of logical blocks (512 bytes per block) into sector pairs is
// somewhat opaque. It seems as though there is a re-ordering of sectors under prodos
// which differs from the web references above, or the .PO file format is not actually in 
// logical bock sequential order. The order which worked here is to write each track (16
// 256 byte sectors) in this physical sector ordering:
// 0x00,0x0E,0x0D,0x0C,0x0B,0x0A,0x09,0x08,0x07,0x06,0x05,0x04,0x03,0x02,0x01,0x0F
func convertDiskImageFromProdosOrderToDos33Order(diskImage []byte) {
	var sectorBuffer [0x0100]byte
	for track := 0x00; track < 0x23; track = track + 1 {
		// rotation group 1
		readSectorDataToBuffer(&sectorBuffer, diskImage, track, 0x01) // 0x01 -> 0x0E
		copySectorDataInImage(diskImage, track, 0x0E, 0x01) // 0x0E -> 0x01
		writeSectorDataFromBuffer(&sectorBuffer, diskImage, track, 0x0E)
		// rotation group 2
		readSectorDataToBuffer(&sectorBuffer, diskImage, track, 0x02) // 0x02 -> 0x0D
		copySectorDataInImage(diskImage, track, 0x0D, 0x02) // 0x0D -> 0x02
		writeSectorDataFromBuffer(&sectorBuffer, diskImage, track, 0x0D)
		// rotation group 3
		readSectorDataToBuffer(&sectorBuffer, diskImage, track, 0x03) // 0x03 -> 0x0C
		copySectorDataInImage(diskImage, track, 0x0C, 0x03) // 0x0C -> 0x03
		writeSectorDataFromBuffer(&sectorBuffer, diskImage, track, 0x0C)
		// rotation group 4
		readSectorDataToBuffer(&sectorBuffer, diskImage, track, 0x04) // 0x04 -> 0x0B
		copySectorDataInImage(diskImage, track, 0x0B, 0x04) // 0x0B -> 0x04
		writeSectorDataFromBuffer(&sectorBuffer, diskImage, track, 0x0B)
		// rotation group 5
		readSectorDataToBuffer(&sectorBuffer, diskImage, track, 0x05) // 0x05 -> 0x0A
		copySectorDataInImage(diskImage, track, 0x0A, 0x05) // 0x0A -> 0x05
		writeSectorDataFromBuffer(&sectorBuffer, diskImage, track, 0x0A)
		// rotation group 6
		readSectorDataToBuffer(&sectorBuffer, diskImage, track, 0x06) // 0x06 -> 0x09
		copySectorDataInImage(diskImage, track, 0x09, 0x06) // 0x09 -> 0x06
		writeSectorDataFromBuffer(&sectorBuffer, diskImage, track, 0x09)
		// rotation group 7
		readSectorDataToBuffer(&sectorBuffer, diskImage, track, 0x07) // 0x07 -> 0x08
		copySectorDataInImage(diskImage, track, 0x08, 0x07) // 0x08 -> 0x07
		writeSectorDataFromBuffer(&sectorBuffer, diskImage, track, 0x08)
	}
}
// Sector suffling section end

// generateLineStartPad creates a block of space characters to be prepended to each line to be
// sent over the serial connection. This pad is to allow for the loss of a variable number of
// bytes which are lost during the processing of the previous line by the apple ][ monitor.
// The pad is set into the string pointed to by lineStartPad.
func generateLineStartPad(lineStartPad *string) {
	const LINE_START_PAD_LENGTH = 16
	*lineStartPad = strings.Repeat(" ", LINE_START_PAD_LENGTH)
}

// generateMemoryAddress generates a hexadecimal formatted address for the apple ][ monitor.
// The targetStartAddress parameter holds the input address, and the output is stored in the
// string pointed to by memoryAddress.
func generateMemoryAddress(memoryAddress *string, targetStartAddress int) {
	if targetStartAddress < 0 {
		*memoryAddress = fmt.Sprintf("")
	} else {
		*memoryAddress = fmt.Sprintf("%02X", targetStartAddress)
	}
}

// generateByteWriteGroupStringFromBytes takes the byteWriteGroup slice as input and stores
// an appropriate string sequence of hexadecimal numbers for the apple ][ monitor, stored in
// the string pointed to by byteWriteGroupString.
func generateByteWriteGroupStringFromBytes(byteWriteGroupString *string, byteWriteGroup []byte) {
	var sb strings.Builder
	for i, b := range byteWriteGroup {
		var s string
		var err error
		if (i == len(byteWriteGroup) - 1) {
			s = fmt.Sprintf("%02X", b)
		} else {
			s = fmt.Sprintf("%02X ", b)
		}
		_, err = sb.WriteString(s)
		if err != nil {
			panic(err)
		}
	}
	*byteWriteGroupString = sb.String()
}

// writeCommandsToFillAppleMemorySegment outputs a carriage return terminated line of text which
// is a command to the apple ][ monitor which fills a block of memory starting at address
// targetStartAddress, with bytes from the sourceBytes slice starting at position sourceBytesStartPos
// and including the number of bytes specified in writeByteCount. Each line is prepended with lineStartPad.
func writeCommandsToFillAppleMemorySegment(sourceBytes []byte, lineStartPad string, targetStartAddress int, sourceBytesStartPos int, writeByteCount int) {
	var memoryAddress string
	generateMemoryAddress(&memoryAddress, targetStartAddress)
	var byteWriteGroupString string
	var sourceBytesEndPos int = sourceBytesStartPos + writeByteCount
	if sourceBytesEndPos > len(sourceBytes) {
		// make sure we don't run off the end of sourceBytes
		sourceBytesEndPos = len(sourceBytes)
	}
	var byteWriteGroup []byte = sourceBytes[sourceBytesStartPos : sourceBytesEndPos]
	generateByteWriteGroupStringFromBytes(&byteWriteGroupString, byteWriteGroup)
	fmt.Printf("%s%s:%s\r", lineStartPad, memoryAddress, byteWriteGroupString)
}

// writeCommandsToLoadDiskTrackToMemory outputs a sequence of commands to the apple ][ monitor which
// fill the 2KB of memory between address 0x1000 and memory address 0x1FFF with 16 sectors worth of
// data for transfer to the apple II disk. The 16 sectors correspond to 1 complete track from the
// diskImage slice corresponding to the trackNum input. Each command line fills SEGMENT_SIZE count
// of bytes. SEGMENT_SIZE was determened through trial and error to be best set at 8 ... when sending
// commands over a serial connection at 2400 baud with data format including 7 data bytes and 1 stop
// byte. The 16 space pad at the start of each line was adequate to avoid data loss during processing
// of each line, however, in order to sychronize timing of the loss of line start padding, commands
// of gradually increasing SEGMENT_SIZE was needed. So at the beginning of the transfer of a track,
// the first segment transfer command is repeated with byte count starting at 0 and ending at 8. This
// led to losing 12 or 13 characters from the 16 space pad regularly when executing each command.
// Use of hardware flow control might avoid the need for this pad.
func writeCommandsToLoadDiskTrackToMemory(diskImage []byte, trackNum int, SEGMENT_SIZE int) {
	if trackNum < 0x0 || trackNum > 0x22 {
		panic(fmt.Sprintf("illegal track number encountered: %d\n", trackNum))
	}
	var diskImageWriteByteCount int = 0x1000
	var sourceBytesStartPos int = diskImageStartPosOfTrackSector(trackNum, 0x00)
	var lineStartPad string
	generateLineStartPad(&lineStartPad)
	var bytesWritten int = 0
	var targetStartAddress = 0x2000
	var firstCommand bool = true
	for bytesWritten < diskImageWriteByteCount {
		if firstCommand {
			// ramp up data stream by doing access and extra dumplicated short writes .. to get the "rhythm" going
			writeCommandsToFillAppleMemorySegment(diskImage, lineStartPad, targetStartAddress, sourceBytesStartPos, SEGMENT_SIZE - 8)
			writeCommandsToFillAppleMemorySegment(diskImage, lineStartPad, targetStartAddress, sourceBytesStartPos, SEGMENT_SIZE - 7)
			writeCommandsToFillAppleMemorySegment(diskImage, lineStartPad, targetStartAddress, sourceBytesStartPos, SEGMENT_SIZE - 6)
			writeCommandsToFillAppleMemorySegment(diskImage, lineStartPad, targetStartAddress, sourceBytesStartPos, SEGMENT_SIZE - 5)
			writeCommandsToFillAppleMemorySegment(diskImage, lineStartPad, targetStartAddress, sourceBytesStartPos, SEGMENT_SIZE - 4)
			writeCommandsToFillAppleMemorySegment(diskImage, lineStartPad, targetStartAddress, sourceBytesStartPos, SEGMENT_SIZE - 3)
			writeCommandsToFillAppleMemorySegment(diskImage, lineStartPad, targetStartAddress, sourceBytesStartPos, SEGMENT_SIZE - 2)
			writeCommandsToFillAppleMemorySegment(diskImage, lineStartPad, targetStartAddress, sourceBytesStartPos, SEGMENT_SIZE - 1)
			writeCommandsToFillAppleMemorySegment(diskImage, lineStartPad, targetStartAddress, sourceBytesStartPos, SEGMENT_SIZE)
			firstCommand = false
		}
		writeCommandsToFillAppleMemorySegment(diskImage, lineStartPad, targetStartAddress, sourceBytesStartPos, SEGMENT_SIZE)
		targetStartAddress = targetStartAddress + SEGMENT_SIZE
		bytesWritten = bytesWritten + SEGMENT_SIZE
		sourceBytesStartPos = sourceBytesStartPos + SEGMENT_SIZE
	}
}

// writeCommandsToLoadRWTSClientProgramToMemory outputs a series of memory transfer commands to the
// apple ][ monitor which loads into memory and then executes a machine language program which calls
// the RWTS routine (already assumed to be loaded into memory and referenced indirectly by a vector
// at location 0x0D39) 16 times to write the 16 sectors worth of data which has been loaded into the
// memory range 0x1000 through 0x1FFF into sectors 0x00 through 0x0F of the apple II disk track which
// is input in parameter trackNum. The machine langague routine is transferred in commands which load
// segements of SEGMENT_SIZE, similar to the loading of the Disk Track buffer.
func writeCommandsToLoadRWTSClientProgramToMemory(trackNum int, SEGMENT_SIZE int) {
	var trackNumArray []byte = []byte{
			'\x00', '\x01', '\x02', '\x03', '\x04', '\x05', '\x06', '\x07', '\x08', '\x09', '\x0A', '\x0B', '\x0C', '\x0D', '\x0E', '\x0F',
			'\x10', '\x11', '\x12', '\x13', '\x14', '\x15', '\x16', '\x17', '\x18', '\x19', '\x1A', '\x1B', '\x1C', '\x1D', '\x1E', '\x1F',
			'\x20', '\x21', '\x22' }
	var trackNumByte = trackNumArray[trackNum]
	var clientProgram []byte = []byte{
			'\xA9', '\x0C', // load address of IOB for RWTS into A/Y
			'\xA0', '\x1C',
			'\x20', '\xD9', '\x03', // call RWTS
			'\xB0', '\x12', // break on error
			'\xA9', '\x0F', // we are done after writing final sector
			'\xCD', '\x21', '\x0C',
			'\xF0', '\x0A', //skip next iteration when done
			'\xEE', '\x21', '\x0C', // modify IOB : advance to write next sector (sector is in '\x0C21')
			'\xEE', '\x25', '\x0C', // modify IOB : advance to next memory page (buffer is in '\x0C25')
			'\xF0', '\xE8', //iterate
			'\xD0', '\xE6', //iterate
			'\x60', // return from client
			'\x00', // break
			'\x01', '\x60', '\x01', '\x00', trackNumByte, '\x00', // slot / drive / vol / track / sector
			'\x30', '\x0C', // DCT address is '\x0C2F
			'\x00', '\x20', // data buffer address (starts at 0x2000)
			'\x00', '\x00', '\x02', // write
			'\x00', '\x00', '\x60', '\x01', // actual volumne / previous slot / drive
			'\x00', '\x00', '\x00', // not used
			'\x00', '\x01', '\xEF', '\xD8' } // DCT table (constant)
	var clientWriteByteCount int = len(clientProgram)
	var sourceBytesStartPos int = 0
	var lineStartPad string
	generateLineStartPad(&lineStartPad)
	var bytesWritten int = 0
	var targetStartAddress = 0x0C00
	for bytesWritten < clientWriteByteCount {
		writeCommandsToFillAppleMemorySegment(clientProgram, lineStartPad, targetStartAddress, sourceBytesStartPos, SEGMENT_SIZE)
		targetStartAddress = targetStartAddress + SEGMENT_SIZE
		bytesWritten = bytesWritten + SEGMENT_SIZE
		sourceBytesStartPos = sourceBytesStartPos + SEGMENT_SIZE
	}
}

// executeClient outputs a command which executes the machine language program and
// reports the written track to stderr.
func executeClient(trackNum int) {
	fmt.Fprintf(os.Stderr, "executing binary client program to write track %d\n", trackNum)
	var lineStartPad string
	generateLineStartPad(&lineStartPad)
	fmt.Printf("%sC00G\r", lineStartPad)
}

// floppy_disk_image_file_to_serial_install main routine parses the desired track number and the
// disk image filepath from command line arguments. It outputs the full series of apple ][ monitor
// commands to load the apple ][ memory buffer with data for the requested track, and to load and
// execute the machine language routine which will write the data to the apple II Disk track via
// the Dos3.3 RWTS subroutine. Note that before transfer, the sector order is reordered for proper
// ProDOS block access during disk use.
func main() {
	const SEGMENT_SIZE = 8
	var diskImageFilepath string = os.Args[1]
	var trackNumString string = os.Args[2]
	var trackNumInt int
	trackNumInt, err := strconv.Atoi(trackNumString)
	if err != nil {
		panic(err)
	}
	var diskImage []byte
	readDiskImageFromFile(&diskImage, diskImageFilepath)
	convertDiskImageFromProdosOrderToDos33Order(diskImage)
	writeCommandsToLoadDiskTrackToMemory(diskImage, trackNumInt, SEGMENT_SIZE)
	writeCommandsToLoadRWTSClientProgramToMemory(trackNumInt, SEGMENT_SIZE)
	executeClient(trackNumInt)
}
